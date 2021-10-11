package collector

import (
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type StdErrSink struct{}

func NewStdErrSink() *StdErrSink {
	return &StdErrSink{}
}

func (s *StdErrSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		return nil
	}
	Logger.Log.Info("mock pushed to SNS with msg: ", msg.String())
	return nil
}
