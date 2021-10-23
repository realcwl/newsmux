package sink

import "github.com/Luismorlan/newsmux/protocol"

type CollectedDataSink interface {
	Push(msg *protocol.CrawlerMessage) error
}
