package collector

import (
	"errors"
	"sync"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type DataCollectJobHandler struct{}

func (handler DataCollectJobHandler) Collect(job *protocol.PanopticJob) (err error) {
	Logger.Log.Info("Collect() with request: ", job)
	var (
		sink       CollectedDataSink
		imageStore CollectedFileStore
		wg         sync.WaitGroup
		httpClient HttpClient
	)
	ip, err := GetCurrentIpAddress(httpClient)
	if err != nil {
		Logger.Log.Error("ip fetching error: ", err)
	}
	Logger.Log.Info("ip address", ip)

	if !utils.IsProdEnv() {
		sink = NewStdErrSink()
		if imageStore, err = NewLocalFileStore("test"); err != nil {
			return err
		}
	} else {
		sink, err = NewSnsSink()
		if err != nil {
			return err
		}
		if imageStore, err = NewS3FileStore(ProdS3ImageBucket); err != nil {
			return err
		}
	}

	for ind := range job.Tasks {
		t := job.Tasks[ind]
		wg.Add(1)
		go func(t *protocol.PanopticTask) {
			defer wg.Done()
			if err := handler.processTask(t, sink, imageStore); err != nil {
				Logger.Log.Errorf("fail to process task: %s", err)
			}
			t.TaskMetadata.IpAddr = ip
		}(t)
	}
	wg.Wait()
	Logger.Log.Info("Collect() response: ", job)
	return nil
}

func (hanlder DataCollectJobHandler) processTask(t *protocol.PanopticTask, sink CollectedDataSink, imageStore CollectedFileStore) error {
	var (
		collector DataCollector
		builder   CollectorBuilder
	)
	// forward task to corresponding collector
	switch t.DataCollectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		// please follow this patter to get collector
		collector = builder.NewJin10Crawler(sink)
	case protocol.PanopticTask_COLLECTOR_WEIBO:
		collector = builder.NewWeiboApiCollector(sink, imageStore)
	default:
		return errors.New("unknown task data collector id")
	}
	RunCollector(collector, t)
	return nil
}
