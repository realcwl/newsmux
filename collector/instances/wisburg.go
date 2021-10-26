package collector_instances

import (
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/gocolly/colly"
)

type WisburgCrawler struct {
	Sink sink.CollectedDataSink
}

func (w WisburgCrawler) GetStartUrl(channelType protocol.WisburgParams_ChannelType) string {
	switch channelType {
	case protocol.WisburgParams_CHANNEL_TYPE_VIEWPOINT:
		return "https://wisburg.com/viewpoint"
	case protocol.WisburgParams_CHANNEL_TYPE_RESEARCH:
		return "https://wisburg.com/research"
	default:
		return ""
	}
}

func CollectAndPublishWisburgResearch() {

}

func (w WisburgCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	c := colly.NewCollector()

	c.OnHTML()
}
