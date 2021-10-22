package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ()

type WallstreetApiCollector struct {
	Sink CollectedDataSink
}

type WallstreetItem struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentText string `json:"content_text"`
	DisplayTime int    `json:"display_time"`
	ID          int    `json:"id"`
	Score       int    `json:"score"`
}

type WallstreetApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []WallstreetItem `json:"items"`
	} `json:"data"`
}

func (collector WallstreetApiCollector) UpdateFileUrls(workingContext *ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (collector WallstreetApiCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *PaginationInfo) string {
	// backup url: https://api-one.wallstcn.com/apiv1/content/lives?channel=us-stock-channel&client=pc&limit=20
	return fmt.Sprintf("https://api.wallstcn.com/apiv1/content/lives?channel=%s&client=pc&limit=%d",
		paginationInfo.NextPageId,
		task.TaskParams.GetWallstreetNewsTaskParams().Limit,
	)
}

func (collector WallstreetApiCollector) UpdateDedupId(post *protocol.CrawlerMessage_CrawledPost) error {
	md5, err := utils.TextToMd5Hash(post.SubSource.SourceId + post.SubSource.ExternalId)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.DeduplicateId = md5
	return nil
}

func (collector WallstreetApiCollector) UpdateResultFromItem(item *WallstreetItem, workingContext *ApiCollectorWorkingContext) error {
	generatedTime := time.Unix(int64(item.DisplayTime), 0)
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(generatedTime)
	workingContext.Result.Post.SubSource.ExternalId = fmt.Sprint(item.ID)
	if err := collector.UpdateDedupId(workingContext.Result.Post); err != nil {
		return utils.ImmediatePrintError(err)
	}
	if item.Title == "" {
		workingContext.Result.Post.Content = item.Title + item.ContentText
	} else {
		workingContext.Result.Post.Content = item.Title + " " + item.ContentText
	}
	newsType := protocol.PanopticSubSource_FLASHNEWS
	if item.Score != 1 {
		newsType = protocol.PanopticSubSource_KEYNEWS
	}
	workingContext.NewsType = newsType
	workingContext.Result.Post.SubSource.Name = SubsourceTypeToName(newsType)
	return nil
}

func (collector WallstreetApiCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
	paginationInfo *PaginationInfo,
) error {
	var client HttpClient
	url := collector.ConstructUrl(task, subsource, paginationInfo)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	body, err := io.ReadAll(resp.Body)
	res := &WallstreetApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	if res.Message != "OK" {
		return errors.New(fmt.Sprintf("response not success: %v", res))
	}

	for _, item := range res.Data.Items {
		// working context for each message
		workingContext := &ApiCollectorWorkingContext{
			Task:           task,
			PaginationInfo: paginationInfo,
			ApiUrl:         url,
			Result:         &protocol.CrawlerMessage{},
			Subsource:      subsource,
		}
		InitializeApiCollectorResult(workingContext)
		err := collector.UpdateResultFromItem(&item, workingContext)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		} else {
			if !IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, workingContext.NewsType) {
				workingContext.Result = nil
				return nil
			}
			if err = collector.Sink.Push(workingContext.Result); err != nil {
				task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
				task.TaskMetadata.TotalMessageFailed++
				return utils.ImmediatePrintError(err)
			}
		}
		task.TaskMetadata.TotalMessageCollected++
		Logger.Log.Debug(workingContext.Result.Post.Content)
	}

	SetErrorBasedOnCounts(task, url, fmt.Sprintf("subsource: %s, body: %s", subsource.Name, string(body)))
	return nil
}

// Support configable multi-page API call
func (collector WallstreetApiCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	// Wallstreet uses channels and only know subsource after each message if fetched
	for ind, channel := range task.TaskParams.GetWallstreetNewsTaskParams().Channels {
		collector.CollectOneSubsourceOnePage(task, subsource, &PaginationInfo{
			CurrentPageCount: ind,
			NextPageId:       channel,
		})
	}
	return nil
}

func (collector WallstreetApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_SUCCESS
	collector.CollectOneSubsource(task, &protocol.PanopticSubSource{})
}
