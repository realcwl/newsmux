package collector

import (
	"log"
	"os"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/gocolly/colly"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// todo: implement data sink
type CollectedDataSink interface {
	Push(msg *protocol.CrawlerMessage) error
}

type DataCollector interface {
	CollectAndPublish(task *protocol.PanopticTask) (successCount int32, failCount int32)

	// All implementation functions should output error
	// errors will be reported for debugging
	GetContent(task *protocol.PanopticTask, elem *colly.HTMLElement) (string, error)
	GetDedupId(task *protocol.PanopticTask, content string) (string, error)
	GetGeneratedTime(task *protocol.PanopticTask, elem *colly.HTMLElement) (time.Time, error)
	GetLevel(elem *colly.HTMLElement) (protocol.PanopticSubSource_SubSourceType, error)
	GetImageUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error)
	GetFileUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error)

	IsRequested(task *protocol.PanopticTask, level protocol.PanopticSubSource_SubSourceType) bool
}

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

func RunColector(collector DataCollector, task *protocol.PanopticTask) *protocol.TaskMetadata {
	meta := &protocol.TaskMetadata{}

	meta.TaskStartTime = timestamppb.Now()
	successCount, failCount := collector.CollectAndPublish(task)
	meta.TaskEndTime = timestamppb.Now()

	meta.TotalMessageCollected = successCount
	meta.TotalMessageFailed = failCount

	return meta
}

type StdErrSink struct{}

func NewStdErrSink() *StdErrSink {
	return &StdErrSink{}
}

func (s *StdErrSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		return nil
	}
	l := log.New(os.Stderr, "[Test DataCollector Sink]", 0)
	l.Println(msg.String())
	return nil
}
