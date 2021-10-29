package collector_instances

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	Kr36StartingUrl = "https://36kr.com/newsflashes"
)

type Kr36FlashNewsPost struct {
	ItemID           int64 `json:"itemId"`
	ItemType         int   `json:"itemType"`
	TemplateMaterial struct {
		ItemID         int64  `json:"itemId"`
		TemplateType   int    `json:"templateType"`
		PublishTime    int64  `json:"publishTime"`
		WidgetTitle    string `json:"widgetTitle"`
		WidgetContent  string `json:"widgetContent"`
		SourceURLRoute string `json:"sourceUrlRoute"`
	} `json:"templateMaterial"`
	Route string `json:"route"`
}

type Kr36ApiResponse struct {
	NewsflashCatalogData struct {
		Data struct {
			NewsflashList struct {
				Code int `json:"code"`
				Data struct {
					ItemList     []Kr36FlashNewsPost `json:"itemList"`
					PageCallback string              `json:"pageCallback"`
					HasNextPage  int                 `json:"hasNextPage"`
				} `json:"data"`
			} `json:"newsflashList"`
			Hotlist struct {
				Code int `json:"code"`
				Data []struct {
					ItemID           int64 `json:"itemId"`
					ItemType         int   `json:"itemType"`
					TemplateMaterial struct {
						ItemID       int64  `json:"itemId"`
						TemplateType int    `json:"templateType"`
						WidgetImage  string `json:"widgetImage"`
						PublishTime  int64  `json:"publishTime"`
						WidgetTitle  string `json:"widgetTitle"`
					} `json:"templateMaterial"`
					Route string `json:"route"`
				} `json:"data"`
			} `json:"hotlist"`
		} `json:"data"`
	} `json:"newsflashCatalogData"`
}

type Kr36ApiCollector struct {
	Sink sink.CollectedDataSink
}

func (k Kr36ApiCollector) Get36KrFlashCardResponse(task *protocol.PanopticTask) (string, error) {
	httpClient := collector.NewHttpClientFromTaskParams(task)
	res, err := httpClient.Get(Kr36StartingUrl)
	if err != nil {
		return "", err
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	// Wisburg's 9-th <script> tag contains all data we need.
	jsCodeText := doc.Find("script").Eq(8).Text()
	body := strings.Split(
		jsCodeText,
		"window.initialState=")[1]
	return body, nil
}

func (k Kr36ApiCollector) GetKr36ApiResponseStruct(task *protocol.PanopticTask) (*Kr36ApiResponse, error) {
	body, err := k.Get36KrFlashCardResponse(task)
	if err != nil {
		return nil, err
	}

	kr36Response := &Kr36ApiResponse{}
	err = json.Unmarshal([]byte(body), kr36Response)
	if err != nil {
		Logger.Log.Errorf("fail to parse response: %s, type: %T", body, kr36Response)
		return nil, err
	}
	return kr36Response, nil
}

func (k Kr36ApiCollector) ProcessSinglePost(post *Kr36FlashNewsPost,
	workingContext *working_context.ApiCollectorWorkingContext) error {
	collector.InitializeApiCollectorResult(workingContext)

	workingContext.Result.Post.Content = post.TemplateMaterial.WidgetContent
	workingContext.Result.Post.Title = post.TemplateMaterial.WidgetTitle
	workingContext.Result.Post.OriginUrl = fmt.Sprintf("https://36kr.com/newsflashes/%s", strconv.FormatInt(post.ItemID, 10))
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(time.Unix(post.TemplateMaterial.PublishTime/1000, 0))

	// There's only a single subsource for 36kr. Thus we default to the first one's name.
	workingContext.Result.Post.SubSource.Name = workingContext.Task.TaskParams.SubSources[0].Name

	workingContext.Result.Post.SubSource.AvatarUrl = collector.GetSourceLogoUrl(collector.Kr36SourceId)

	dedupId, err := utils.TextToMd5Hash(collector.Kr36SourceId + strconv.FormatInt(post.ItemID, 10))

	if err != nil {
		return fmt.Errorf("fail to get dedup if from post %s, err :%w", collector.PrettyPrint(post), err)
	}
	workingContext.Result.Post.DeduplicateId = dedupId

	return nil
}

func (k Kr36ApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	res, err := k.GetKr36ApiResponseStruct(task)
	if err != nil {
		Logger.Log.Errorf("fail to get Kr36 response, error: %s", err)
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
	}

	for _, post := range res.NewsflashCatalogData.Data.NewsflashList.Data.ItemList {
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false},
			ApiUrl:        Kr36StartingUrl,
		}

		err := k.ProcessSinglePost(&post, workingContext)
		if err != nil {
			Logger.Log.WithFields(logrus.Fields{"source": "kuailansi"}).Errorln("fail to process a single Kr36 Post:", err,
				"\npost content:\n", collector.PrettyPrint(post))
			workingContext.Task.TaskMetadata.TotalMessageFailed++
			continue
		}

		// Returning nil in ProcessSinglePost doesn't necessarily mean success, it
		// could just be that we're skiping that post (e.g. subsource type doesn't
		// match)
		if workingContext.Result != nil {
			sink.PushResultToSinkAndRecordInTaskMetadata(k.Sink, workingContext)
		}
	}
}
