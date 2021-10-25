package collector_instances

import (
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/protocol"
)

type WisburgCrawler struct {
	Sink sink.CollectedDataSink
}

func (w WisburgCrawler) CollectAndPublish(task *protocol.PanopticTask) {

}
