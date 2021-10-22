package collector_instances

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	sink "github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	KUAILANSI_URI         = "http://m.fbecn.com/24h/news_fbe0406.json?newsid=0"
	CHINA_TIMEZONE        = "Asia/Shanghai"
	KUAILANSI_TIME_FORMAT = "2006-01-02 15:04:05"

	IP_BAN_MESSAGE = "IP访问受限制"
)

type KuailansiApiCrawler struct {
	Sink sink.CollectedDataSink
}

type KuailansiPost struct {
	NewsId   string `json:"newsID"`
	Time     string `json:"time"`
	Content  string `json:"content"`
	Level    string `json:"Level"`
	Type     string `json:"Type"`
	Keywords string `json:"Keywords"`
}

type KuailansiApiResponse struct {
	List     []KuailansiPost `json:"list"`
	NextPage string          `json:"nextpage"`
}

func (k KuailansiApiCrawler) GetKuailansiResponse(task *protocol.PanopticTask) (*KuailansiApiResponse, error) {
	httpClient := collector.HttpClient{}
	httpResponse, err := httpClient.Get(KUAILANSI_URI)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	// Remove BOM before parsing, see https://en.wikipedia.org/wiki/Byte_order_mark for details.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	kuailansiResponse := &KuailansiApiResponse{}
	err = json.Unmarshal(body, kuailansiResponse)
	if err != nil {
		Logger.Log.Errorln("fail to parse Kuailansi response:", body)
		return nil, err
	}

	return kuailansiResponse, nil
}

// For kuailansi, if Level == 0, it's a important update.
func (k KuailansiApiCrawler) GetNewsTypeForPost(post *KuailansiPost) (protocol.PanopticSubSource_SubSourceType, error) {
	level, err := strconv.Atoi(post.Level)
	if err != nil {
		return protocol.PanopticSubSource_UNSPECIFIED, errors.Wrap(err, "cannot parse post.Level")
	}

	if level >= 1 {
		return protocol.PanopticSubSource_FLASHNEWS, nil
	}

	return protocol.PanopticSubSource_KEYNEWS, nil
}

func (k KuailansiApiCrawler) GetCrawledSubSourceNameFromPost(post *KuailansiPost) (string, error) {
	t, err := k.GetNewsTypeForPost(post)
	if err != nil {
		return "", errors.Wrap(err, "fail to get subsource type from post"+collector.PrettyPrint(post))
	}
	return collector.SubsourceTypeToName(t), nil
}

func (k KuailansiApiCrawler) ParseGenerateTime(post *KuailansiPost) (*timestamppb.Timestamp, error) {
	location, err := time.LoadLocation(CHINA_TIMEZONE)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse time zome: "+CHINA_TIMEZONE)
	}
	t, err := time.ParseInLocation(KUAILANSI_TIME_FORMAT, post.Time, location)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse Kuailansi post time: "+post.Time)
	}
	return timestamppb.New(t), nil
}

func (k KuailansiApiCrawler) ValidatePost(post *KuailansiPost) error {
	if strings.Contains(post.Content, IP_BAN_MESSAGE) {
		return errors.New("IP is banned")
	}
	return nil
}

func (k KuailansiApiCrawler) ProcessSinglePost(post *KuailansiPost,
	workingContext *working_context.ApiCollectorWorkingContext) error {
	if err := k.ValidatePost(post); err != nil {
		return err
	}

	subSourceType, err := k.GetNewsTypeForPost(post)
	if err != nil {
		return err
	}

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, subSourceType) {
		// Return nil if the post is not of requested type. Note that this is
		// intentionally not considered as failure.
		return nil
	}

	collector.InitializeApiCollectorResult(workingContext)

	ts, err := k.ParseGenerateTime(post)
	if err != nil {
		return err
	}

	name, err := k.GetCrawledSubSourceNameFromPost(post)
	if err != nil {
		return err
	}

	workingContext.Result.Post.ContentGeneratedAt = ts
	workingContext.Result.Post.Content = post.Content
	workingContext.Result.Post.SubSource.Name = name
	workingContext.Result.Post.SubSource.AvatarUrl = collector.GetSourceLogoUrl(
		workingContext.Task.TaskParams.SourceId)

	return nil
}

func (k KuailansiApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	res, err := k.GetKuailansiResponse(task)
	if err != nil {
		Logger.Log.Errorln("fail to get Kuailansi response:", err)
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return
	}

	for _, post := range res.List {
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext: working_context.SharedContext{Task: task},
			ApiUrl:        KUAILANSI_URI,
		}

		err := k.ProcessSinglePost(&post, workingContext)
		if err != nil {
			Logger.Log.Errorln("fail to process a single Kuailansi Post:", err,
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

	collector.SetTaskResultState(task)
}
