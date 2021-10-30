package collector

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	Jin10SourceId             = "a882eb0d-0bde-401a-b708-a7ce352b7392"
	WeiboSourceId             = "0129417c-4987-45c9-86ac-d6a5c89fb4f7"
	KuailansiSourceId         = "6e1f6734-985b-4a52-865f-fc39a9daa2e8"
	WallstreetNewsSourceId    = "66251821-ef9a-464c-bde9-8b2fd8ef2405"
	JinseSourceId             = "5891f435-d51e-4575-b4af-47cd4ede5607"
	CaUsSourceId              = "1c6ab31c-aebe-40ba-833d-7cc2d977e5a1"
	WisburgSourceId           = "bb3c8ee2-c81e-43d9-8d98-7a6bb6ca0238"
	Kr36SourceId              = "c0ae802e-3c12-4144-86ca-ab0f8fe629ce"
	WallstreetArticleSourceId = "66251821-ef9a-464c-bde9-8b2fd8ef2405"
)

// Hard code subsource type to name
func SubsourceTypeToName(t protocol.PanopticSubSource_SubSourceType) string {
	if t == protocol.PanopticSubSource_FLASHNEWS {
		return "快讯"
	}
	if t == protocol.PanopticSubSource_KEYNEWS {
		return "要闻"
	}
	return "其他"
}

func LogHtmlParsingError(task *protocol.PanopticTask, elem *colly.HTMLElement, err error) {
	html, _ := elem.DOM.Html()

	source := "undefined"
	switch task.DataCollectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		source = "jin10"
	case protocol.PanopticTask_COLLECTOR_JINSE:
		source = "jinse"
	case protocol.PanopticTask_COLLECTOR_WEIBO:
		source = "weibo"
	case protocol.PanopticTask_COLLECTOR_KUAILANSI:
		source = "kuailansi"
	case protocol.PanopticTask_COLLECTOR_WALLSTREET_NEWS:
		source = "wallstreet"
	}

	Logger.Log.WithFields(
		logrus.Fields{"source": source},
	).Error(fmt.Sprintf("Error in data collector. [Error] %s. [Type] %s. [Task_id] %s. [DOM Start] %s [DOM End].", err.Error(), task.DataCollectorId, task.TaskId, html))
}

func GetSourceLogoUrl(sourceId string) string {
	switch sourceId {
	// Jin10
	case Jin10SourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png"
	// Weibo
	case WeiboSourceId:
		return ""
	case WallstreetNewsSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/wallstrt.png"
	case KuailansiSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/kuailansi.png"
	case JinseSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jinse.png"
	case CaUsSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/caus.png"
	case WisburgSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/wisburg.png"
	case Kr36SourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/36ke.png"
	default:
		return ""
	}
}

func InitializeCrawlerResult(workingContext *working_context.CrawlerWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}
	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
	if workingContext.Subsource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.Subsource.Name
		workingContext.Result.Post.SubSource.ExternalId = workingContext.Subsource.ExternalId
	}
	workingContext.Result.CrawledAt = timestamppb.Now()
	workingContext.Result.CrawlerVersion = "1"
	workingContext.Result.IsTest = !utils.IsProdEnv()
	workingContext.Result.Post.OriginUrl = workingContext.OriginUrl

	workingContext.Result.CrawlerIp = workingContext.Task.TaskMetadata.IpAddr
}

func InitializeApiCollectorResult(workingContext *working_context.ApiCollectorWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}

	workingContext.Result.CrawledAt = timestamppb.Now()
	workingContext.Result.CrawlerVersion = "1"
	workingContext.Result.IsTest = !utils.IsProdEnv()

	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	// subsource default logo will be source logo, unless overwirte
	// like weibo
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
	workingContext.Result.Post.OriginUrl = workingContext.ApiUrl

	workingContext.Result.CrawlerIp = workingContext.Task.TaskMetadata.IpAddr
}

func SetErrorBasedOnCounts(task *protocol.PanopticTask, url string, moreContext ...string) {
	if task.TaskMetadata.TotalMessageCollected == 0 {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.Error(
			"Finished crawl with 0 success msg, Task ", task.TaskId,
			"[url]", url,
			moreContext,
		)
	}
	if task.TaskMetadata.TotalMessageFailed > 0 {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.Error(
			"Finished crawl with >0 failed msg, Task ", task.TaskId,
			"[url]", url,
			moreContext,
		)
	}
}

func LineBreakerToSpace(content string) string {
	return strings.ReplaceAll(content, "\n", " ")
}

func ParallelSubsourceApiCollect(task *protocol.PanopticTask, collector ApiCollector) {
	task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_SUCCESS

	var wg sync.WaitGroup
	for _, subsource := range task.TaskParams.SubSources {
		wg.Add(1)
		ss := subsource
		go func(ss *protocol.PanopticSubSource) {
			defer wg.Done()
			err := collector.CollectOneSubsource(task, ss)
			if err != nil {
				task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
			}
		}(ss)
	}
	wg.Wait()
	Logger.Log.Info("Finished collecting subsources, Task", task)
}

// Process each html selection to get content
func IsRequestedNewsType(subSources []*protocol.PanopticSubSource, newstype protocol.PanopticSubSource_SubSourceType) bool {
	requestedTypes := make(map[protocol.PanopticSubSource_SubSourceType]bool)

	for _, subsource := range subSources {
		s := subsource
		requestedTypes[s.Type] = true
	}

	if _, ok := requestedTypes[newstype]; !ok {
		fmt.Println("Not requested, actual level: ", newstype, " requested ", requestedTypes)
		return false
	}

	return true
}

func PrettyPrint(data interface{}) string {
	var p []byte
	//    var err := error
	p, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s \n", p)
}

func HtmlToText(html string) (string, error) {
	reader := strings.NewReader(html)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return "", utils.ImmediatePrintError(
			errors.New(fmt.Sprintf("fail to convert full rich-html text to node: %v", html)))
	}
	// goquery Text() will not replace br with newline
	// to keep consistent with prod crawler, we need to
	// add newline
	doc.Find("br").AfterHtml("\n")
	return doc.Text(), nil
}

func ParseGenerateTime(timeString string, format string, timeZoneString string, cralwer string) (*timestamppb.Timestamp, error) {
	location, err := time.LoadLocation(timeZoneString)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse time zome for "+cralwer+" : "+timeZoneString)
	}
	t, err := time.ParseInLocation(format, timeString, location)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse "+cralwer+" post time: "+timeString)
	}
	return timestamppb.New(t), nil
}
