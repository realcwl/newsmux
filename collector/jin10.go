package collector

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type Jin10Crawler struct {
}

func NewJin10Crawler() *Jin10Crawler {
	return &Jin10Crawler{}
}

func (collector Jin10Crawler) GetLevel(t interface{}) protocol.PanopticSubSource_SubSourceType {
	if t.(*colly.HTMLElement).DOM.Find(".jin-flash-item").HasClass("is-important") {
		return protocol.PanopticSubSource_KEYNEWS
	}
	return protocol.PanopticSubSource_FLASHNEWS
}

// Process each html selection to get content
func (collector Jin10Crawler) GetContent(s *goquery.Selection) string {
	content := ""
	s.Children().Each(func(i int, s *goquery.Selection) {
		if len(s.Nodes) > 0 && s.Nodes[0].Data == "br" {
			content = content + "\n"
		}
		content = content + s.Text()
	})

	// goquery don't have a good way to get text without child elements'
	// remove children's text manually
	remove := s.Children().Text()
	text := s.Text()
	result := strings.Replace(text, remove, "", -1)
	content = content + result
	return content
}

func (collector Jin10Crawler) GetMessage(task *protocol.PanopticTask, elem *colly.HTMLElement) *protocol.CrawlerMessage {
	var requestedTypes map[protocol.PanopticSubSource_SubSourceType]bool

	for _, subsource := range task.TaskParams.SubSources {
		requestedTypes[subsource.Type] = true
	}

	level := collector.GetLevel(elem)
	// todo: check requested level, return only requested
	// if _, ok := requestedTypes[level]; !ok {
	// 	return nil
	// }

	fmt.Println("Time: ", elem.DOM.Find(".item-time").Text())
	// todo: deal with time

	s := elem.DOM.Find(".right-content > div")
	content := collector.GetContent(s)

	// there is only one image can be in jin10
	imageUrl := elem.DOM.Find(".img-container > img").AttrOr("src", "")
	var imageUrls []string
	if imageUrl != "" {
		imageUrls = append(imageUrls, imageUrl)
	}

	hasher := md5.New()
	hasher.Write([]byte(task.TaskParams.SourceId + content))

	res := &protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				Name:      SubsourceTypeToName(level),
				SourceId:  task.TaskParams.SourceId,
				AvatarUrl: "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", //todo: put in central place
			},
			Content:            content,
			ContentGeneratedAt: &timestamp.Timestamp{},
			DeduplicateId:      hex.EncodeToString(hasher.Sum(nil)),
			ImageUrls:          imageUrls,
		},
		CrawledAt:      &timestamp.Timestamp{},
		CrawlerIp:      "123", // todo: actual ip
		CrawlerVersion: "1",   // todo: actual version
		IsTest:         false,
	}
	return res
}

// todo: mock http response and test end to end Collect()
func (collector Jin10Crawler) Collect(task *protocol.PanopticTask) ([]*protocol.CrawlerMessage, error) {

	var res []*protocol.CrawlerMessage

	c := colly.NewCollector()

	// On every a element which has href attribute call callback
	c.OnHTML(`#jin_flash_list > .jin-flash-item-container`, func(e *colly.HTMLElement) {
		fmt.Println("解析金十")
		res = append(res, collector.GetMessage(task, e))
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		// todo: error should be put into metadata
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	err := c.Visit("https://www.jin10.com/index.html")
	return res, err
}
