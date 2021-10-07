package modules

import (
	"context"
	"log"

	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
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

func (o *Orchestrator) RunModule(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// TODO(chenweilunster): Actually implement Orchestrator.
	messages, err := o.EventBus.Subscribe(ctx, panoptic.TOPIC_PENDING_TASK)
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
				log.Printf("fail to execute job: %s, error: %s", panopticJob.String(), err)
				return
			}
			log.Printf("successfully executed job: %s", res)
		}(&panopticJob)

	}

	return nil
}

func (o *Orchestrator) Name() string {
	return o.Config.Name
}
