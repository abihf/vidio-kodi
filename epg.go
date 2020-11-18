package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

var epgProgramTemplate = template.Must(template.New("epg-program").Parse(`
<programme start="{{ .Date }}{{ .TimeStart }}00 +0700" 
	stop="{{ .Date }}{{ .TimeStop }}00 +0700" 
	channel="channel-{{ .ChannelID }}" catchup-id="{{ .VideoID }}">
<title>{{ html .Title }}</title>
<date>{{ .Date }}</date>
</programme>
`))

const epgHeader = `<?xml version="1.0" encoding="ISO-8859-1"?>
<!DOCTYPE tv SYSTEM "xmltv.dtd">

<tv source-info-url="http://www.vidio.com/" source-info-name="Vidio" generator-info-name="XMLTV/$Id: vidio-kodi $" generator-info-url="http://www.xmltv.org/">
`

type epgProgramData struct {
	Date      string
	TimeStart string
	TimeStop  string
	ChannelID int
	VideoID   string
	Title     string
}

func handleEPG(c *gin.Context) {
	list, err := getChannelList()
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("content-type", "application/xml")
	w := c.Writer
	fmt.Fprint(w, epgHeader)
	for id, ch := range list {
		fmt.Fprintf(w, `<channel id="channel-%d"><display-name>%s</display-name></channel>`, ch.ContentID, ch.ContentTitle)
		func() {
			file, err := os.Open(fmt.Sprintf("cache/epg-%d", id))
			if err != nil {
				return
			}
			defer file.Close()
			io.Copy(w, file)
		}()
	}
	fmt.Fprint(w, "</tv>")
}

func scheduleEPGUpdate() error {
	list, err := getChannelList()
	if err != nil {
		return err
	}

	for id := range list {
		time.Sleep(1 * time.Second)
		crawlEPG(id)
	}

	return nil
}

func crawlEPG(id int) error {
	cacheFileName := fmt.Sprintf("cache/epg-%d", id)
	stat, err := os.Stat(cacheFileName)
	if err == nil {
		if stat.ModTime().After(time.Now().Add(-24 * time.Hour)) {
			return nil
		}
	}

	res, err := httpClient.Get(fmt.Sprintf("https://www.vidio.com/live/%d/schedules?locale=id", id))
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return err
	}

	file, err := os.Create(cacheFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	doc.Find(".b-livestreaming-daily-schedule__content").Each(func(_ int, dateNode *goquery.Selection) {
		dateID, ok := dateNode.Attr("id")
		if !ok {
			// id not found
			return
		}

		// schedule-content-20201112
		datePart := dateID[17:25]
		dateNode.Find(".b-livestreaming-daily-schedule__item").Each(func(_ int, itemNode *goquery.Selection) {
			title := itemNode.Find(".b-livestreaming-daily-schedule__item-content-title").First().Text()

			timeStr := itemNode.Find(".b-livestreaming-daily-schedule__item-content-caption").First().Text()
			timeSplitted := strings.Split(strings.ReplaceAll(strings.Replace(timeStr, " WIB", "", 1), ":", ""), " - ")

			href := itemNode.Parent().AttrOr("href", "-")
			vidID := ""
			if href != "#" {
				vidID = strings.Replace(strings.SplitN(href, "-", 2)[0], "/watch/", "", 1)
			}

			data := epgProgramData{
				Date:      datePart,
				TimeStart: timeSplitted[0],
				TimeStop:  timeSplitted[1],
				ChannelID: id,
				VideoID:   vidID,
				Title:     title,
			}
			epgProgramTemplate.Execute(file, data)
		})
	})

	return nil
}
