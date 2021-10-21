package collector_instances

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"time"

	Collector "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	KUAILANSI_URI         = "http://m.fbecn.com/24h/news_fbe0406.json?newsid=0"
	CHINA_TIMEZONE        = "Asia/Shanghai"
	KUAILANSI_TIME_FORMAT = "2006-01-02 15:04:05"
)

type KuailansiApiCrawler struct {
	Sink Collector.CollectedDataSink
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

func GetKuailansiResponse(task *protocol.PanopticTask) (*KuailansiApiResponse, error) {
	httpClient := Collector.HttpClient{}
	httpResponse, err := httpClient.Get(KUAILANSI_URI)

	if err != nil {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return nil, err
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return nil, err
	}

	// Remove BOM before parsing, see https://en.wikipedia.org/wiki/Byte_order_mark for details.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	kuailansiResponse := &KuailansiApiResponse{}
	err = json.Unmarshal(body, kuailansiResponse)
	if err != nil {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.Errorln("fail to parse Kuailansi response:", body)
		return nil, err
	}

	return kuailansiResponse, nil
}

// For kuailansi, if Level == 0, it's a important update.
func GetNewsTypeForPost(post *KuailansiPost) (protocol.PanopticSubSource_SubSourceType, error) {
	level, err := strconv.Atoi(post.Level)
	if err != nil {
		return protocol.PanopticSubSource_UNSPECIFIED, errors.Wrap(err, "cannot parse post.Level")
	}

	if level >= 1 {
		return protocol.PanopticSubSource_FLASHNEWS, nil
	}

	return protocol.PanopticSubSource_KEYNEWS, nil
}

func ParseGenerateTime(post *KuailansiPost) (*timestamppb.Timestamp, error) {
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

func ProcessSinglePost(post *KuailansiPost,
	workingContext *Collector.ApiCollectorWorkingContext) error {
	subSourceType, err := GetNewsTypeForPost(post)
	if err != nil {
		return err
	}

	if !Collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, subSourceType) {
		// Return nil if the post is not of requested type. Note that this is
		// intentionally not considered as failure.
		return nil
	}

	ts, err := ParseGenerateTime(post)
	if err != nil {
		return err
	}

	res := &protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			Content:            post.Content,
			ContentGeneratedAt: ts,
		},
		CrawlerIp: workingContext.Task.TaskMetadata.IpAddr,
		CrawledAt: timestamppb.Now(),
	}

	workingContext.Result = res

	return nil
}

func (k KuailansiApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	res, err := GetKuailansiResponse(task)
	if err != nil {
		return
	}

	for _, post := range res.List {
		workingContext := &Collector.ApiCollectorWorkingContext{
			SharedContext: Collector.SharedContext{Task: task},
			ApiUrl:        KUAILANSI_URI,
		}

		err := ProcessSinglePost(&post, workingContext)
		if err != nil {
			Logger.Log.Errorln("fail to process a single Kuailansi Post: ", err)
			workingContext.Task.TaskMetadata.TotalMessageFailed++
			continue
		}

		// Returning nil in ProcessSinglePost doesn't necessarily mean success, it
		// could just be that we're skiping that post (e.g. subsource type doesn't
		// match)
		if workingContext.Result != nil {
			workingContext.Task.TaskMetadata.TotalMessageCollected++
			k.Sink.Push(workingContext.Result)
		}
	}
}
