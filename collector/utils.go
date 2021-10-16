package collector

import (
	"fmt"
	"io/ioutil"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func RunCollector(collector DataCollector, task *protocol.PanopticTask) {
	if task.TaskMetadata == nil {
		task.TaskMetadata = &protocol.TaskMetadata{}
	}

	task.TaskMetadata.TaskStartTime = timestamppb.Now()
	collector.CollectAndPublish(task)
	task.TaskMetadata.TaskEndTime = timestamppb.Now()
}

func LogHtmlParsingError(task *protocol.PanopticTask, elem *colly.HTMLElement, err error) {
	html, _ := elem.DOM.Html()
	Logger.Log.Error(fmt.Sprintf("Error in data collector. [Error] %s. [Task] %s. [DOM Start] %s [DOM End].", err.Error(), task.String(), html))
}

func GetCurrentIpAddress(client HttpClient) (ip string, err error) {
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	resp.Body.Close()
	return string(body), err
}

func GetSourceLogoUrl(sourceId string) string {
	switch sourceId {
	// Jin10
	case "a882eb0d-0bde-401a-b708-a7ce352b7392":
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png"
	// Weibo
	case "0129417c-4987-45c9-86ac-d6a5c89fb4f7":
		return ""
	default:
		return ""
	}
}

func InitializeCrawlerResult(workingContext *CrawlerWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}
	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId
	// subsource default logo will be source logo, unless overwirte
	// like weibo
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
	workingContext.Result.CrawledAt = &timestamp.Timestamp{}
	workingContext.Result.CrawlerVersion = "1"
	workingContext.Result.IsTest = !utils.IsProdEnv()
	workingContext.Result.Post.OriginUrl = workingContext.OriginUrl
	var httpClient HttpClient
	ip, err := GetCurrentIpAddress(httpClient)
	if err != nil {
		Logger.Log.Error("ip fetching error: ", err)
	}
	workingContext.Result.CrawlerIp = ip

}

func InitializeApiCollectorResult(workingContext *ApiCollectorWorkingContext) {
	workingContext.Result = &protocol.CrawlerMessage{Post: &protocol.CrawlerMessage_CrawledPost{}}

	workingContext.Result.CrawledAt = &timestamp.Timestamp{}
	workingContext.Result.CrawlerVersion = "1"
	workingContext.Result.IsTest = !utils.IsProdEnv()

	workingContext.Result.Post.SubSource = &protocol.CrawledSubSource{}
	// subsource default logo will be source logo, unless overwirte
	// like weibo
	workingContext.Result.Post.SubSource.AvatarUrl = GetSourceLogoUrl(workingContext.Task.TaskParams.SourceId)
	workingContext.Result.Post.SubSource.SourceId = workingContext.Task.TaskParams.SourceId

	var httpClient HttpClient
	ip, err := GetCurrentIpAddress(httpClient)
	if err != nil {
		Logger.Log.Error("ip fetching error: ", err)
	}
	workingContext.Result.CrawlerIp = ip
}

func SetErrorBasedOnCounts(task *protocol.PanopticTask, url string, moreContext string) {
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
