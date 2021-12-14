package sink

import (
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/encoding/prototext"
)

type StdErrSink struct{}

func NewStdErrSink() *StdErrSink {
	return &StdErrSink{}
}

func (s *StdErrSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		return nil
	}
	Logger.Log.Info("=== mock pushed to SNS with CrawlerMessage === \n", prototext.Format(msg))
	return nil
}
