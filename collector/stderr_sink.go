package collector

import (
	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/log"
)

type StdErrSink struct{}

func NewStdErrSink() *StdErrSink {
	return &StdErrSink{}
}

func (s *StdErrSink) Push(msg *protocol.CrawlerMessage) error {
	if msg == nil {
		return nil
	}
	Log.Info(msg.String())
	return nil
}
