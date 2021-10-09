package collector

import (
	"errors"
	"sync"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/flag"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type DataCollectJobHandler struct{}

func (handler DataCollectJobHandler) Collect(job *protocol.PanopticJob) (err error) {
	Logger.Log.Info("Collect() with request: ", job)
	var (
		sink CollectedDataSink
		wg   sync.WaitGroup
	)
	if flag.IsDevelopment {
		sink = NewStdErrSink()
	} else {
		sink, err = NewSnsSink()
		if err != nil {
			return err
		}
	}

	for ind := range job.Tasks {
		t := job.Tasks[ind]
		wg.Add(1)
		go func(t *protocol.PanopticTask) {
			defer wg.Done()
			if err := handler.processTask(t, sink); err != nil {
				Logger.Log.Errorf("fail to process task: %s", err)
			}
		}(t)
	}
	wg.Wait()
	Logger.Log.Info("Collect() response: ", job)
	return nil
}

func (hanlder DataCollectJobHandler) processTask(t *protocol.PanopticTask, sink CollectedDataSink) error {
	var (
		collector DataCollector
		builder   CollectorBuilder
	)
	// forward task to corresponding collector
	switch t.DataCollectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		// please follow this patter to get collector
		collector = builder.NewJin10Crawler(sink)
	default:
		return errors.New("unknown task data collector id")
	}
	RunCollector(collector, t)
	return nil
}
