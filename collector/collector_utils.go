package collector

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Luismorlan/newsmux/collector/file_store"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

const (
	Jin10SourceId             = "a882eb0d-0bde-401a-b708-a7ce352b7392"
	WeiboSourceId             = "0129417c-4987-45c9-86ac-d6a5c89fb4f7"
	KuailansiSourceId         = "6e1f6734-985b-4a52-865f-fc39a9daa2e8"
	WallstreetNewsSourceId    = "66251821-ef9a-464c-bde9-8b2fd8ef2405"
	JinseSourceId             = "5891f435-d51e-4575-b4af-47cd4ede5607"
	CaUsSourceId              = "1c6ab31c-aebe-40ba-833d-7cc2d977e5a1"
	WeixinSourceId            = "0f90f563-7c95-4be0-a592-7e5666f02c33"
	WisburgSourceId           = "bb3c8ee2-c81e-43d9-8d98-7a6bb6ca0238"
	Kr36SourceId              = "c0ae802e-3c12-4144-86ca-ab0f8fe629ce"
	CaixinSourceId            = "cc2a61b1-721f-4529-8afc-6da686f23b36"
	WallstreetArticleSourceId = "66251821-ef9a-464c-bde9-8b2fd8ef2405"
	GelonghuiSourceId         = "3627b507-d28d-4627-8afd-a6168e6b10d3"
	ClsNewsSourceId           = "9ae67eea-4839-11ec-81d3-0242ac130003"
	TwitterSourceId           = "a19df1ae-3c80-4ffc-b8e6-cefb3a6a3c27"
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

func MarkAndLogCrawlError(task *protocol.PanopticTask, err error, moreInfo string) {
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
	case protocol.PanopticTask_COLLECTOR_USER_CUSTOMIZED_SOURCE:
		source = "customized"
	}

	task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
	Logger.Log.WithFields(
		logrus.Fields{"source": source},
	).Error(fmt.Sprintf("Error in data collector. [Source] %s. [Error] %s. [Type] %s. [Task_id] %s. [More Info] %s", source, err.Error(), task.DataCollectorId, task.TaskId, moreInfo))
}

func LogHtmlParsingError(task *protocol.PanopticTask, elem *colly.HTMLElement, err error) {
	html, _ := elem.DOM.Html()
	MarkAndLogCrawlError(task, err, fmt.Sprintf("[DOM Start] %s [DOM End].", html))
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
	case WeixinSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/weixin.png"
	case WisburgSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/wisburg.png"
	case Kr36SourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/36ke.png"
	case CaixinSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/caixin.png"
	case GelonghuiSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/gelonghui.png"
	case ClsNewsSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png"
	default:
		return ""
	}
}

func InitializeCrawlerResult(workingContext *working_context.CrawlerWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}
	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
	if workingContext.SubSource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
		workingContext.Result.Post.SubSource.ExternalId = workingContext.SubSource.ExternalId
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

	workingContext.Result.CrawlerIp = workingContext.Task.TaskMetadata.IpAddr
}

func InitializeRssCollectorResult(workingContext *working_context.RssCollectorWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}

	workingContext.Result.CrawledAt = timestamppb.Now()
	workingContext.Result.CrawlerVersion = "1"
	workingContext.Result.IsTest = !utils.IsProdEnv()

	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
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

func ParallelSubsourceApiCollect(task *protocol.PanopticTask, collector ParallelCollector) {
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

func IterateAllNodes(doc *goquery.Document, jquery string, callback func(*goquery.Selection)) {
	doc.Find(jquery).Each(func(i int, s *goquery.Selection) {
		callback(s)
	})
}

func OffloadImageSourceFromHtml(sourceHtml string, imageStore file_store.CollectedFileStore) (string, error) {

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sourceHtml))
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}

	IterateAllNodes(doc, "img", func(s *goquery.Selection) {
		originUrl, exists := s.Attr("src")
		if !exists {
			return
		}
		key, err := imageStore.FetchAndStore(originUrl, "")
		if err != nil {
			return
		}
		newUrl := imageStore.GetUrlFromKey(key)
		s.SetAttr("src", newUrl)
	})

	return doc.Html()
}
