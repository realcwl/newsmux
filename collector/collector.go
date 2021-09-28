package collector

import (
	"github.com/Luismorlan/newsmux/protocol"
)

// todo: implement data sink
type CollectedDataSink interface {
	Push([]*protocol.CrawlerMessage) error
}

type DataCollector interface {
	Collect(*protocol.PanopticTask) ([]*protocol.CrawlerMessage, error)
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
