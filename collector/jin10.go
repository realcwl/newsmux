package collector

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	jin10DateFormat = "20060102-15:04:05"
)

type Jin10Crawler struct {
}

func NewJin10Crawler() *Jin10Crawler {
	return &Jin10Crawler{}
}

func (collector Jin10Crawler) GetLevel(elem *colly.HTMLElement) protocol.PanopticSubSource_SubSourceType {
	if elem.DOM.Find(".jin-flash-item").HasClass("is-important") {
		return protocol.PanopticSubSource_KEYNEWS
	}
	return protocol.PanopticSubSource_FLASHNEWS
}

func (collector Jin10Crawler) GetContent(task *protocol.PanopticTask, elem *colly.HTMLElement) string {
	var sb strings.Builder
	selection := elem.DOM.Find(".right-content > div")
	selection.Children().Each(func(i int, s *goquery.Selection) {
		if len(s.Nodes) > 0 && s.Nodes[0].Data == "br" {
			sb.WriteString(" ")
		}
		sb.WriteString(s.Text())
	})

	// goquery don't have a good way to get text without child elements'
	// remove children's text manually
	remove := selection.Children().Text()
	text := selection.Text()
	result := strings.Replace(text, remove, "", -1)
	sb.WriteString(result)
	return sb.String()
}

func (collector Jin10Crawler) GetGeneratedTime(task *protocol.PanopticTask, elem *colly.HTMLElement) time.Time {
	id := elem.DOM.AttrOr("id", "")
	timeText := elem.DOM.Find(".item-time").Text()
	dateStr := id[5:13] + "-" + timeText
	time, err := time.Parse(jin10DateFormat, dateStr)
	if err != nil {
		return time.UTC()
	}
	return time
}

func (collector Jin10Crawler) GetDedupId(task *protocol.PanopticTask, content string) string {
	hasher := md5.New()
	hasher.Write([]byte(task.TaskParams.SourceId + content))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (collector Jin10Crawler) GetImageUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) []string {
	// there is only one image can be in jin10
	imageUrl := elem.DOM.Find(".img-container > img").AttrOr("src", "")
	var imageUrls []string
	if imageUrl != "" {
		imageUrls = append(imageUrls, imageUrl)
	}

	return imageUrls
}

// Process each html selection to get content
func (collector Jin10Crawler) IsValid(task *protocol.PanopticTask, level protocol.PanopticSubSource_SubSourceType) bool {
	requestedTypes := make(map[protocol.PanopticSubSource_SubSourceType]bool)

	for _, subsource := range task.TaskParams.SubSources {
		s := subsource
		requestedTypes[s.Type] = true
	}

	if _, ok := requestedTypes[level]; !ok {
		fmt.Println("NOT LEGAL, actual level: ", level)
		return false
	}

	return true
}

func (collector Jin10Crawler) GetMessage(task *protocol.PanopticTask, elem *colly.HTMLElement) *protocol.CrawlerMessage {

	level := collector.GetLevel(elem)

	if !collector.IsValid(task, level) {
		return nil
	}

	generatedTime := collector.GetGeneratedTime(task, elem)

	content := collector.GetContent(task, elem)

	deduplicateId := collector.GetDedupId(task, content)

	imageUrls := collector.GetImageUrls(task, elem)

	return &protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				Name:      SubsourceTypeToName(level),
				SourceId:  task.TaskParams.SourceId,
				AvatarUrl: "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", //todo: put in central place
			},
			Content:            content,
			ContentGeneratedAt: timestamppb.New(generatedTime),
			DeduplicateId:      deduplicateId,
			ImageUrls:          imageUrls,
		},
		CrawledAt:      &timestamp.Timestamp{},
		CrawlerIp:      "123", // todo: actual ip
		CrawlerVersion: "1",   // todo: actual version
		IsTest:         false,
	}
}

// todo: mock http response and test end to end Collect()
func (collector Jin10Crawler) Collect(task *protocol.PanopticTask) ([]*protocol.CrawlerMessage, error) {

	var res []*protocol.CrawlerMessage

	c := colly.NewCollector()

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
