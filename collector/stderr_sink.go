package collector

import (
	"log"
	"os"

	"github.com/Luismorlan/newsmux/protocol"
)

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
