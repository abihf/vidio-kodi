package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func handleStream(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(strings.Replace(idStr, ".m3u8", "", 1))
	// c.Header("content-type")
	err := writeStreamURL(c, id)
	if err != nil {
		c.Error(err)
		return
	}
}

func writeStreamURL(c *gin.Context, id int) error {
	w := c.Writer

	st, err := getStreamType(id)
	if err != nil {
		return err
	}

	if st == dashStream {
		dashURL, err := getDashURL(id)
		if err != nil {
			return err
		}
		c.Header("content-type", "application/x-mpegurl")
		fmt.Fprintln(w, "#EXTM3U")
		fmt.Fprintln(w, "#EXTINF:-1,")
		fmt.Fprintln(w, "#KODIPROP:inputstreamaddon=inputstream.adaptive")
		fmt.Fprintln(w, "#KODIPROP:inputstream.adaptive.manifest_type=mpd")
		fmt.Fprintln(w, dashURL)

		return nil
	}
	hlsURL, err := getHlsURL(id)
	if err != nil {
	}
	upstreamRes, err := httpClient.Get(hlsURL)
	if err != nil {
		return err
	}
	defer upstreamRes.Body.Close()
	if upstreamRes.StatusCode != 200 {
		return fmt.Errorf("Can not get m3u")
	}
	c.Header("content-type", upstreamRes.Header.Get("content-type"))
	io.Copy(w, upstreamRes.Body)

	return nil

}

type streamType int

const (
	hlsStream  streamType = 1
	dashStream streamType = 2
)

var streamTypeCache map[int]streamType

const streamTypeCacheFile = "cache/stream-type.json"

func getStreamType(id int) (streamType, error) {
	if streamTypeCache == nil {
		file, err := os.Open(streamTypeCacheFile)
		if err == nil {
			json.NewDecoder(file).Decode(&streamTypeCache)
			file.Close()
		} else {
			streamTypeCache = map[int]streamType{}
		}
	}

	st, ok := streamTypeCache[id]
	if ok {
		return st, nil
	}

	res, err := httpClient.Get(fmt.Sprintf("https://www.vidio.com/live/%d", id))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return 0, fmt.Errorf("Open stream page: status %d", res.StatusCode)
	}

	streamTypeCache[id] = dashStream
	buff := make([]byte, 4096)
	for {
		n, _ := res.Body.Read(buff)
		if n <= 0 {
			break
		}
		if strings.Contains(string(buff), `hls-url="http`) {
			streamTypeCache[id] = hlsStream
		}
	}

	if file, err := os.Create(streamTypeCacheFile); err == nil {
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "\t")
		encoder.Encode(streamTypeCache)
	}

	return streamTypeCache[id], nil
}

type tokenInfo struct {
	Token  string `json:"token"`
	Expire time.Time
}

var hlsTokenCache = map[int]*tokenInfo{}
var dashTokenCache = map[int]*tokenInfo{}

func getHlsURL(id int) (string, error) {
	token, err := getHlsToken(id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://app-etslive-2.vidio.com/live/%d/master.m3u8?%s", id, token.Token), nil
}

func getHlsToken(id int) (*tokenInfo, error) {
	token, ok := hlsTokenCache[id]
	if ok && token.Expire.After(time.Now()) {
		return token, nil
	}
	tokenURL := fmt.Sprintf("https://www.vidio.com/live/%d/tokens", id)
	tokenRes, err := httpClient.Post(tokenURL, "", bytes.NewReader([]byte{}))
	if err != nil {
		return nil, err
	}
	defer tokenRes.Body.Close()
	if tokenRes.StatusCode != 200 {
		return nil, fmt.Errorf("Invalid status %d when fetching hls token", tokenRes.StatusCode)
	}

	var tokenObj tokenInfo

	json.NewDecoder(tokenRes.Body).Decode(&tokenObj)
	tokenObj.Expire = time.Now().Add(5 * time.Minute)
	hlsTokenCache[id] = &tokenObj
	return &tokenObj, nil
}

func getDashURL(id int) (string, error) {
	token, err := getDashToken(id)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://etslive-2-vidio-com.akamaized.net/%s/vp9/%d_stream.mpd", token.Token, id), nil
}

func getDashToken(id int) (*tokenInfo, error) {
	token, ok := dashTokenCache[id]
	if ok {
		return token, nil
	}

	tokenURL := fmt.Sprintf("https://www.vidio.com/live/%d/tokens?type=dash", id)
	tokenRes, err := httpClient.Post(tokenURL, "", bytes.NewReader([]byte{}))
	if err != nil {
		return nil, err
	}
	defer tokenRes.Body.Close()
	if tokenRes.StatusCode != 200 {
		return nil, fmt.Errorf("Invalid status %d when fetching dash token", tokenRes.StatusCode)
	}

	var inf tokenInfo
	json.NewDecoder(tokenRes.Body).Decode(&inf)
	dashTokenCache[id] = &inf
	return &inf, nil
}
