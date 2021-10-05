package collector

import (
	"context"
	"sync"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
)

type DataCollectJobHandler struct{}

func (handler DataCollectJobHandler) Collect(context context.Context, job *protocol.PanopticJob) (err error) {
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
	var c DataCollector

	// forward task to corresponding collector
	switch t.DataCollectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		c = NewJin10Crawler(sink)
	default:
		Log.Error("Unknown task DataCollectorId")
		return nil
	}
	return RunColector(c, t)
}
