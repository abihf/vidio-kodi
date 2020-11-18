package main

import (
	"fmt"
	"sort"
	"text/template"

	"github.com/gin-gonic/gin"
)

var channelItemTemplate = template.Must(template.New("channel-item").Parse(`
#EXTINF:-1 tvg-id="channel-{{ .Item.ContentID }}" tvg-name="{{ .Item.ID }}" group-title="{{ .Item.Section }}" tvg-chno="{{ .Item.ContentID }}" tvg-logo="{{ .Item.Image }}" radio="{{ .Item.IsRadio }}" catchup-source="https://www.vidio.com/videos/{catchup-id}/common_tokenized_playlist.m3u8", {{ .Item.ContentTitle }}
http://{{ .Host }}/stream/{{ .Item.ContentID }}
`))

func handleList(c *gin.Context) {

	channels, err := getChannelList()
	if err != nil {
		c.Error(err)
		return
	}

	orderedChannels := make([]*tvChannel, len(channels))
	i := 0
	for _, ch := range channels {
		orderedChannels[i] = ch
		i++
	}
	sort.SliceStable(orderedChannels, func(i, j int) bool {
		return orderedChannels[i].ContentID < orderedChannels[j].ContentID
	})

	fmt.Fprintf(c.Writer, "#EXTM3U x-tvg-url=\"http://%s/guide.xml\"\n", c.Request.Host)
	for _, ch := range orderedChannels {
		data := map[string]interface{}{
			"Host": c.Request.Host,
			"Item": ch,
		}
		channelItemTemplate.Execute(c.Writer, data)
	}
}
