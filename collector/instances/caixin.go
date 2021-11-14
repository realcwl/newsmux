package collector_instances

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Select top news.
	CaixinTopNewsSelector = "div.topNews"
	// Select all Flash News.
	CaixinFlashNewsSelector = "div.yaowen div.boxa"
	// The format we are parsing.
	CaixinTimeFormat = "2006-01-02 15:04:05"
)

type CaixinCollector struct {
	Sink sink.CollectedDataSink
}

func (cx CaixinCollector) UpdateTopNewsTitle(
	workingContext *working_context.CrawlerWorkingContext) error {
	title := workingContext.Element.DOM.Find("div.txt a").Text()
	if len(title) == 0 {
		return errors.New("empty title")
	}

	workingContext.Result.Post.Title = title

	return nil
}

func (cx CaixinCollector) UpdateTopNewsExternalPostId(
	workingContext *working_context.CrawlerWorkingContext) error {
	content := workingContext.Element.DOM.Find("div.txt a").AttrOr("href", "")
	tokens := strings.Split(content, "/")
	externalId := strings.Split(tokens[len(tokens)-1], ".")[0]
	workingContext.ExternalPostId = externalId
	return nil
}

func (cx CaixinCollector) UpdateTopNewsContent(workingContext *working_context.CrawlerWorkingContext) error {
	content := workingContext.Element.DOM.Find("div.txt p").Text()
	if len(content) == 0 {
		return errors.New("empty content")
	}
	workingContext.Result.Post.Content = content

	return nil
}

func (cx CaixinCollector) UpdateTopNewsDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	var token string
	if workingContext.ExternalPostId != "" {
		token = workingContext.ExternalPostId
	} else {
		token = workingContext.Result.Post.Title
	}
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.SubSource.SourceId + token)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (cx CaixinCollector) ParseTimeRawString(rawStr string) *timestamppb.Timestamp {
	// Parse Month
	re := regexp.MustCompile(`\d[\d,]*[月]`)
	match := re.FindStringSubmatch(rawStr)
	if len(match) == 0 {
		// Any failure during time parsing will be considered as a soft failure.
		return timestamppb.Now()
	}
	month := fmt.Sprintf("%02s", strings.ReplaceAll(match[0], "月", ""))

	// Parse Day
	re = regexp.MustCompile(`\d[\d,]*[日]`)
	match = re.FindStringSubmatch(rawStr)
	if len(match) == 0 {
		// Any failure during time parsing will be considered as a soft failure.
		return timestamppb.Now()
	}
	day := fmt.Sprintf("%02s", strings.ReplaceAll(match[0], "日", ""))

	// Match time in day
	timeInDay := "00:00"
	re = regexp.MustCompile(`\d[\d]:\d[\d]`)
	match = re.FindStringSubmatch(rawStr)
	if len(match) != 0 {
		timeInDay = match[0]
	}
	hourAndMin := strings.Split(timeInDay, ":")
	if len(hourAndMin) != 2 {
		return timestamppb.Now()
	}
	hour := fmt.Sprintf("%02s", hourAndMin[0])
	min := fmt.Sprintf("%02s", hourAndMin[1])

	location, err := time.LoadLocation(ChinaTimeZone)
	if err != nil {
		Logger.Log.Errorln("fail to parse ChinaTimeZone:", ChinaTimeZone)
		return timestamppb.Now()
	}
	year, _, _ := time.Now().In(location).Date()
	formatedTimeString := fmt.Sprintf("%d-%s-%s %s:%s:00", year, month, day, hour, min)

	t, err := time.ParseInLocation(CaixinTimeFormat, formatedTimeString, location)
	if err != nil {
		Logger.Log.Errorln("fail to parse Caixin time:", formatedTimeString)
		return timestamppb.Now()
	}

	return timestamppb.New(t)
}

func (cx CaixinCollector) UpdateTopNewsGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	rawStr := workingContext.Element.DOM.Find("div.txt span").Text()
	if rawStr == "" {
		// In the case that we cannot derive time string, we just use current time.
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return nil
	}
	t := cx.ParseTimeRawString(rawStr)
	workingContext.Result.Post.ContentGeneratedAt = t
	return nil
}

func (cx CaixinCollector) UpdateTopNewsImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imageUrl := workingContext.Element.DOM.Find("div.pic img").AttrOr("src", "")
	if imageUrl != "" {
		workingContext.Result.Post.ImageUrls = []string{imageUrl}
	}
	return nil
}

func (cx CaixinCollector) UpdateTopNewsSubSourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = "头条"
	return nil
}

func (cx CaixinCollector) UpdateTopNewsOriginalUrl(workingContext *working_context.CrawlerWorkingContext) error {
	url := workingContext.Element.DOM.Find("div.txt a").AttrOr("href", "")
	workingContext.Result.Post.OriginUrl = url
	return nil
}

// Parse an html element and get topnews as CrawlerMessage in context.
func (cx CaixinCollector) GetTopNewsMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		cx.UpdateTopNewsTitle,
		cx.UpdateTopNewsContent,
		cx.UpdateTopNewsExternalPostId,
		cx.UpdateTopNewsDedupId,
		cx.UpdateTopNewsGeneratedTime,
		cx.UpdateTopNewsImageUrls,
		cx.UpdateTopNewsSubSourceName,
		cx.UpdateTopNewsOriginalUrl,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	if workingContext.Result.Post.ContentGeneratedAt.AsTime().After(time.Now()) {
		workingContext.IntentionallySkipped = true
	}

	return nil
}

func (cx CaixinCollector) CollectTopNews(task *protocol.PanopticTask, subSource *protocol.PanopticSubSource) {
	metadata := task.TaskMetadata

	c := colly.NewCollector()
	c.OnHTML(CaixinTopNewsSelector, func(elem *colly.HTMLElement) {
		var err error
		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: subSource.Link}
		if err = cx.GetTopNewsMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		if workingContext.Result == nil {
			return
		}
		if !workingContext.IntentionallySkipped {
			sink.PushResultToSinkAndRecordInTaskMetadata(cx.Sink, workingContext)
		}
	})

	c.Visit(subSource.Link)
}

func (cx CaixinCollector) UpdateFlashNewsTitle(
	workingContext *working_context.CrawlerWorkingContext) error {
	title := workingContext.Element.DOM.Find("h4 a").Text()
	if len(title) == 0 {
		return errors.New("empty title")
	}

	workingContext.Result.Post.Title = title

	return nil
}

func (cx CaixinCollector) UpdateFlashNewsExternalPostId(
	workingContext *working_context.CrawlerWorkingContext) error {
	content := workingContext.Element.DOM.Find("a").AttrOr("href", "")
	tokens := strings.Split(content, "/")
	externalId := strings.Split(tokens[len(tokens)-1], ".")[0]
	workingContext.ExternalPostId = externalId
	return nil
}

func (cx CaixinCollector) UpdateFlashNewsContent(workingContext *working_context.CrawlerWorkingContext) error {
	content := workingContext.Element.DOM.Find("p").Text()
	if len(content) == 0 {
		return errors.New("empty content")
	}
	workingContext.Result.Post.Content = content

	return nil
}

func (cx CaixinCollector) UpdateFlashNewsDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	var token string
	if workingContext.ExternalPostId != "" {
		token = workingContext.ExternalPostId
	} else {
		token = workingContext.Result.Post.Title
	}
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.SubSource.SourceId + token)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (cx CaixinCollector) UpdateFlashNewsGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	rawStr := workingContext.Element.DOM.Find("span").Text()
	if rawStr == "" {
		// In the case that we cannot derive time string, we just use current time.
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return nil
	}
	t := cx.ParseTimeRawString(rawStr)
	workingContext.Result.Post.ContentGeneratedAt = t
	return nil
}

func (cx CaixinCollector) UpdateFlashNewsImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imageUrl := workingContext.Element.DOM.Find("div.pic img").AttrOr("src", "")
	if imageUrl != "" {
		workingContext.Result.Post.ImageUrls = []string{imageUrl}
	}
	return nil
}

func (cx CaixinCollector) UpdateFlashNewsSubSourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
	return nil
}

func (cx CaixinCollector) UpdateFlashNewsOriginalUrl(workingContext *working_context.CrawlerWorkingContext) error {
	url := workingContext.Element.DOM.Find("a").AttrOr("href", "")
	workingContext.Result.Post.OriginUrl = url
	return nil
}

func (cx CaixinCollector) GetFlashNewsMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		cx.UpdateFlashNewsTitle,
		cx.UpdateFlashNewsContent,
		cx.UpdateFlashNewsExternalPostId,
		cx.UpdateFlashNewsDedupId,
		cx.UpdateFlashNewsGeneratedTime,
		cx.UpdateFlashNewsImageUrls,
		cx.UpdateFlashNewsSubSourceName,
		cx.UpdateFlashNewsOriginalUrl,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	if workingContext.Result.Post.ContentGeneratedAt.AsTime().After(time.Now()) {
		workingContext.IntentionallySkipped = true
	}

	return nil
}
func (cx CaixinCollector) CollectFlashNews(task *protocol.PanopticTask, subSource *protocol.PanopticSubSource) {
	metadata := task.TaskMetadata

	c := colly.NewCollector()

	c.OnHTML(CaixinFlashNewsSelector, func(elem *colly.HTMLElement) {
		var err error
		workingContext := &working_context.CrawlerWorkingContext{
			SubSource: subSource,
			SharedContext: working_context.SharedContext{
				Task: task, IntentionallySkipped: false,
			},
			Element:   elem,
			OriginUrl: subSource.Link,
		}
		if err = cx.GetFlashNewsMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		if workingContext.Result == nil {
			return
		}
		if !workingContext.IntentionallySkipped {
			sink.PushResultToSinkAndRecordInTaskMetadata(cx.Sink, workingContext)
		}
	})

	c.Visit(subSource.Link)
}

func (cx CaixinCollector) CollectOneSubSource(task *protocol.PanopticTask, subSource *protocol.PanopticSubSource) {
	// metadata := task.TaskMetadata
	cx.CollectTopNews(task, subSource)
	cx.CollectFlashNews(task, subSource)
}

func (cx CaixinCollector) CollectAndPublish(task *protocol.PanopticTask) {
	for _, subSource := range task.TaskParams.SubSources {
		cx.CollectOneSubSource(task, subSource)
	}
}
