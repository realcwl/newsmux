package collector

import (
	"sync"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
)

type DataCollectJobHandler struct{}

func (handler DataCollectJobHandler) Collect(job *protocol.PanopticJob) (err error) {
	Log.Info("Collect() with request: ", job)
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
			meta := handler.processTask(t, sink)
			t.TaskMetadata = meta
		}(t)
	}
	wg.Wait()
	Log.Info("Collect() response: ", job)
	return nil
}

func (hanlder DataCollectJobHandler) processTask(t *protocol.PanopticTask, sink CollectedDataSink) *protocol.TaskMetadata {
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
		Log.Error("Unknown task DataCollectorId")
		return nil
	}
	return RunCollector(collector, t)
}
