package collector

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
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
	sink CollectedDataSink
}

func (collector Jin10Crawler) GetFileUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error) {
	return []string{}, errors.New("GetFileUrls not implemented, should not be called")
}

func (collector Jin10Crawler) GetLevel(elem *colly.HTMLElement) (protocol.PanopticSubSource_SubSourceType, error) {
	selection := elem.DOM.Find(".jin-flash-item")
	if len(selection.Nodes) == 0 {
		return protocol.PanopticSubSource_UNSPECIFIED, errors.New("Jin10 news item not found")
	}
	if selection.HasClass("is-important") {
		return protocol.PanopticSubSource_KEYNEWS, nil
	}
	return protocol.PanopticSubSource_FLASHNEWS, nil
}

func (collector Jin10Crawler) GetContent(task *protocol.PanopticTask, elem *colly.HTMLElement) (string, error) {
	var sb strings.Builder
	selection := elem.DOM.Find(".right-content > div")
	if len(selection.Nodes) == 0 {
		return "", errors.New("Jin10 news DOM not found")
	}
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
	return sb.String(), nil
}

func (collector Jin10Crawler) GetGeneratedTime(task *protocol.PanopticTask, elem *colly.HTMLElement) (time.Time, error) {
	id := elem.DOM.AttrOr("id", "")
	timeText := elem.DOM.Find(".item-time").Text()
	if len(id) <= 13 {
		return time.Now().UTC(), errors.New("Jin10 news DOM id length is smaller than expected")
	}

	dateStr := id[5:13] + "-" + timeText
	generatedTime, err := time.Parse(jin10DateFormat, dateStr)
	if err != nil {
		return generatedTime.UTC(), err
	}
	return generatedTime, nil
}

func (collector Jin10Crawler) GetDedupId(task *protocol.PanopticTask, content string) (string, error) {
	hasher := md5.New()
	_, err := hasher.Write([]byte(task.TaskParams.SourceId + content))
	return hex.EncodeToString(hasher.Sum(nil)), err
}

func (collector Jin10Crawler) GetImageUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error) {
	// there is only one image can be in jin10
	selection := elem.DOM.Find(".img-container > img")
	if len(selection.Nodes) == 0 {
		return []string{}, nil
	}

	imageUrl := selection.AttrOr("src", "")
	if len(imageUrl) == 0 {
		return []string{}, errors.New("Image DOM exist but src not found")
	}
	return []string{imageUrl}, nil
}

// Process each html selection to get content
func (collector Jin10Crawler) IsRequested(task *protocol.PanopticTask, level protocol.PanopticSubSource_SubSourceType) bool {
	requestedTypes := make(map[protocol.PanopticSubSource_SubSourceType]bool)

	for _, subsource := range task.TaskParams.SubSources {
		s := subsource
		requestedTypes[s.Type] = true
	}

	if _, ok := requestedTypes[level]; !ok {
		fmt.Println("Not requested, actual level: ", level, " requested ", requestedTypes)
		return false
	}

	return true
}

func (collector Jin10Crawler) GetMessage(task *protocol.PanopticTask, elem *colly.HTMLElement) (*protocol.CrawlerMessage, error) {

	content, err := collector.GetContent(task, elem)
	if err != nil {
		return nil, err
	}

	deduplicateId, err := collector.GetDedupId(task, content)
	if err != nil {
		return nil, err
	}

	level, err := collector.GetLevel(elem)
	if err != nil {
		return nil, err
	}

	if !collector.IsRequested(task, level) {
		return nil, errors.New("Not requested level")
	}

	generatedTime, err := collector.GetGeneratedTime(task, elem)
	if err != nil {
		return nil, err
	}

	imageUrls, err := collector.GetImageUrls(task, elem)
	if err != nil {
		return nil, err
	}

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
	}, nil
}

// todo: mock http response and test end to end Collect()
func (collector Jin10Crawler) CollectAndPublish(task *protocol.PanopticTask) (successCount int32, failCount int32) {
	c := colly.NewCollector()

	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(`#jin_flash_list > .jin-flash-item-container`, func(elem *colly.HTMLElement) {
		var (
			msg *protocol.CrawlerMessage
			err error
		)
		if msg, err = collector.GetMessage(task, elem); err != nil {
			failCount++
			return
		}
		if err = collector.sink.Push(msg); err != nil {
			failCount++
			return
		}
		successCount++
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		// todo: error should be put into metadata
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.Visit("https://www.jin10.com/index.html")
	return
}
