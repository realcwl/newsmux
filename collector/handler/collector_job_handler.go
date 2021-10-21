package collector_job_handler

import (
	"errors"
	"sync"

	. "github.com/Luismorlan/newsmux/collector"
	. "github.com/Luismorlan/newsmux/collector/builder"
	. "github.com/Luismorlan/newsmux/collector/instances"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/golang/protobuf/proto"
)

type DataCollectJobHandler struct{}

func (handler DataCollectJobHandler) Collect(job *protocol.PanopticJob) (err error) {
	Logger.Log.Info("Collect() with request: \n", proto.MarshalTextString(job))
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
	Logger.Log.Info("ip address: ", ip)

	if !utils.IsProdEnv() || job.Debug {
		sink = NewStdErrSink()
		if imageStore, err = NewLocalFileStore("test"); err != nil {
			return err
		}
		defer imageStore.CleanUp()
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
	Logger.Log.Info("Collect() response: \n", proto.MarshalTextString(job))
	return nil
}

func (hanlder DataCollectJobHandler) processTask(t *protocol.PanopticTask, sink CollectedDataSink, imageStore CollectedFileStore) error {
	var (
		collector DataCollector
		builder   CollectorBuilder
	)
	zsxqFileStore, err := GetZsxqS3FileStore(t, utils.IsProdEnv())
	if err != nil {
		return err
	}
	// forward task to corresponding collector
	switch t.DataCollectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		// please follow this patter to get collector
		collector = builder.NewJin10Crawler(sink)
	case protocol.PanopticTask_COLLECTOR_WEIBO:
		collector = builder.NewWeiboApiCollector(sink, imageStore)
	case protocol.PanopticTask_COLLECTOR_ZSXQ:
		collector = builder.NewZsxqApiCollector(sink, imageStore, zsxqFileStore)
	case protocol.PanopticTask_COLLECTOR_WALLSTREET_NEWS:
		collector = builder.NewWallstreetNewsApiCollector(sink)
	case protocol.PanopticTask_COLLECTOR_KUAILANSI:
		collector = builder.NewKuailansiApiCollector(sink)
	default:
		return errors.New("unknown task data collector id")
	}
	RunCollector(collector, t)
	return nil
}
