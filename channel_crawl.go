package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var channelList map[int]*tvChannel

type tvChannel struct {
	Action          string `json:"action"`
	CategoryID      int    `json:"category_id"`
	CategoryName    string `json:"category_name"`
	Section         string `json:"section"`
	SectionID       int    `json:"section_id"`
	SectionPosition int    `json:"section_position"`
	DataSource      string `json:"data_source"`
	ContentID       int    `json:"content_id"`
	ContentTitle    string `json:"content_title"`
	ContentType     string `json:"content_type"`
	ContentPosition int    `json:"content_position"`

	Href  string
	Image string
}

func (c *tvChannel) ID() string {
	return strings.ReplaceAll(c.Href, "/live/", "")
}

func (c *tvChannel) IsRadio() bool {
	return c.Section == "Radio"
}

const channelCacheFileName = "cache/channel.json"

func getChannelList() (map[int]*tvChannel, error) {
	if channelList == nil {
		var err error
		channelList, err = crawlChannelList()
		return channelList, err
	}
	return channelList, nil
}

func crawlChannelList() (map[int]*tvChannel, error) {
	channels := map[int]*tvChannel{}

	if file, err := os.Open(channelCacheFileName); err == nil {
		defer file.Close()

		err = json.NewDecoder(file).Decode(&channels)
		return channels, err
	}

	res, err := httpClient.Get("https://www.vidio.com/live")
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return nil, err
	}

	doc.Find("li.home-grid__item[data-ahoy-props]").Each(func(i int, node *goquery.Selection) {
		propStr, ok := node.Attr("data-ahoy-props")
		if !ok {
			return
		}
		// {"action":"click","category_id":55,"category_name":"Live","section":"TV Nasional","section_id":123,"section_position":4,"data_source":"none","segments":[],"content_id":874,"content_title":"Kompas TV","content_type":"livestreaming","content_position":6}
		var prop tvChannel
		json.Unmarshal([]byte(propStr), &prop)
		if prop.Section == "Siaran Langsung" || prop.Section == "Trending" || prop.DataSource != "none" {
			return
		}

		prop.Href = node.Find("a").First().AttrOr("href", "")
		prop.Image = node.Find("img").First().AttrOr("data-src", "")

		channels[prop.ContentID] = &prop
	})

	if file, err := os.Create(channelCacheFileName); err == nil {
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "\t")
		encoder.Encode(channels)
	}

	return channels, nil
}
