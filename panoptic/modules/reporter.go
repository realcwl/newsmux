package modules

import (
	"context"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"google.golang.org/protobuf/proto"
)

type ReporterConfig struct {
	Name string
}

// Reporter's job is to listen to different channels and aggregate results,
// sending to Datadog (Or other service if there's any) for monitoring purpose.
//
type Reporter struct {
	panoptic.Module

	Config ReporterConfig

	Statsd *statsd.Client

	EventBus *gochannel.GoChannel
}

func NewReporter(config ReporterConfig, statsd *statsd.Client, e *gochannel.GoChannel) *Reporter {
	return &Reporter{
		Config:   config,
		Statsd:   statsd,
		EventBus: e,
	}
}

// Report task result state to datadog.
func ReportResultState(job *protocol.PanopticJob, statsd *statsd.Client) {
	for _, task := range job.Tasks {
		err := statsd.Incr(panoptic.DDOG_TASK_STATE_COUNTER,
			[]string{
				task.TaskMetadata.ConfigName,
				task.DataCollectorId.String(),
				task.TaskMetadata.IpAddr,
				task.TaskMetadata.ResultState.String(),
			}, 1)
		if err != nil {
			Logger.Log.Infoln("cannot report result state")
		}
	}
}

func (r *Reporter) ProcessPanopticJobs(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, err := r.EventBus.Subscribe(ctx, panoptic.TOPIC_EXECUTED_JOB)
	if err != nil {
		return err
	}

	for msg := range messages {
		msg.Ack()

		job := protocol.PanopticJob{}
		err := proto.Unmarshal(msg.Payload, &job)

		if err != nil {
			return err
		}

		ReportResultState(&job, r.Statsd)
	}

	return nil
}

func (r *Reporter) RunModule(ctx context.Context) error {
	r.ProcessPanopticJobs(ctx)
	return nil
}

func (r *Reporter) Name() string {
	return r.Config.Name
}
