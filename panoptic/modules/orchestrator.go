package modules

import (
	"context"

	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"google.golang.org/protobuf/proto"
)

type OrchestratorConfig struct {
	// Name of the orchestrator.
	Name string
}

type Orchestrator struct {
	panoptic.Module

	Config OrchestratorConfig

	executor Executor

	EventBus *gochannel.GoChannel
}

// Return a new instance of Orchestrator.
func NewOrchestrator(config OrchestratorConfig, executor Executor, e *gochannel.GoChannel) *Orchestrator {
	return &Orchestrator{
		Config:   config,
		executor: executor,
		EventBus: e,
	}
}

// After a job is executed successfully, publish it into an executed job
// channel for reporter to report to Datadog.
func (o *Orchestrator) PublishFinishedJob(job *protocol.PanopticJob) error {
	data, err := proto.Marshal(job)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), data)
	return o.EventBus.Publish(panoptic.TopicExecutedJob, msg)
}

func (o *Orchestrator) RunModule(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, err := o.EventBus.Subscribe(ctx, panoptic.TopicPendingJob)
	if err != nil {
		return err
	}

	for msg := range messages {
		msg.Ack()

		panopticJob := protocol.PanopticJob{}
		err := proto.Unmarshal(msg.Payload, &panopticJob)

		if err != nil {
			return err
		}

		go func(job *protocol.PanopticJob) {
			res, err := o.executor.Execute(ctx, &panopticJob)
			if err != nil {
				Logger.Log.Errorf("fail to execute job: %s, error: %s", panopticJob.String(), err)
				return
			}

			err = o.PublishFinishedJob(res)
			if err != nil {
				Logger.Log.Errorf("fail to publish job into executed job channel, error: %s", err)
				return
			}
		}(&panopticJob)

	}

	return nil
}

func (o *Orchestrator) Name() string {
	return o.Config.Name
}

func (o *Orchestrator) Shutdown() {
	o.executor.Shutdown()
	Logger.Log.Infoln("Module ", o.Config.Name, " gracefully shutdown")
}
