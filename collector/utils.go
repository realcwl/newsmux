package collector

import (
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
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

func RunCollector(collector DataCollector, task *protocol.PanopticTask) *protocol.TaskMetadata {
	meta := &protocol.TaskMetadata{}

	meta.TaskStartTime = timestamppb.Now()
	successCount, failCount := collector.CollectAndPublish(task)
	meta.TaskEndTime = timestamppb.Now()

	meta.TotalMessageCollected = successCount
	meta.TotalMessageFailed = failCount

	return meta
}

func LogHtmlParsingError(task *protocol.PanopticTask, elem *colly.HTMLElement, err error) {
	html, _ := elem.DOM.Html()
	Log.Error(fmt.Sprintf("Error in data collector. [Error] %s. [Task] %s. [DOM Start] %s [DOM End].", err.Error(), task.String(), html))
}
