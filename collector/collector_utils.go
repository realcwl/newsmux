package collector

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	Jin10SourceId          = "a882eb0d-0bde-401a-b708-a7ce352b7392"
	WeiboSourceId          = "0129417c-4987-45c9-86ac-d6a5c89fb4f7"
	KuailansiSourceId      = "6e1f6734-985b-4a52-865f-fc39a9daa2e8"
	WallstreetNewsSourceId = "66251821-ef9a-464c-bde9-8b2fd8ef2405"
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
	Logger.Log.Error(fmt.Sprintf("Error in data collector. [Error] %s. [Task] %s. [DOM Start] %s [DOM End].", err.Error(), task.String(), html))
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
	default:
		return ""
	}
}

func InitializeCrawlerResult(workingContext *working_context.CrawlerWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}
	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
	// subsource default logo will be source logo, unless overwirte
	// like weibo
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
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
			"Finished crawl weibo with 0 success msg, Task ", task.TaskId,
			"[url]", url,
			moreContext,
		)
	}
	if task.TaskMetadata.TotalMessageFailed > 0 {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.Error(
			"Finished crawl weibo with >0 failed msg, Task ", task.TaskId,
			"[url]", url,
			moreContext,
		)
	}
}

func CleanWeiboContent(content string) string {
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
	Logger.Log.Info("Finished collecting weibo users , Task", task)
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
