package collector

import (
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
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
