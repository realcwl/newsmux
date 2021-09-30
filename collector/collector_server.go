package collector

import (
	"context"
	"sync"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
)

type DataCollectServerHandler struct {
	protocol.UnimplementedDataCollectServer
}

func (c DataCollectServerHandler) Collect(context context.Context, job *protocol.PanopticJob) (*protocol.PanopticJob, error) {
	Log.Info("RPC call Collect() request: ", job)

	var (
		sink CollectedDataSink
		wg   sync.WaitGroup
	)
	if flag.IsDevelopment {
		sink = NewStdErrSink()
	} else {
		sink = NewSnsSink()
	}

	for ind := range job.Tasks {
		t := job.Tasks[ind]
		wg.Add(1)
		go func(t *protocol.PanopticTask) {
			defer wg.Done()
			meta := processTask(t, sink)
			t.TaskMetadata = meta
		}(t)
	}

	wg.Wait()
	Log.Info("RPC call Collect() response: ", job)
	return job, nil
}

func processTask(t *protocol.PanopticTask, sink CollectedDataSink) *protocol.TaskMetadata {
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
