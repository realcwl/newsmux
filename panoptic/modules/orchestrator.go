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

	// Number of Lambdas maintained at a given time.
	LambdaPoolSize int32

	// Lambda life span in milli-second. Any lambda function that exceed this
	// value will be cleaned up and replaced with a new one.
	LambdaLifeSpanMilliSec int32
}

type Orchestrator struct {
	panoptic.Module

	Config OrchestratorConfig

	EventBus *gochannel.GoChannel
}

// Return a new instance of Orchestrator.
func NewOrchestrator(config OrchestratorConfig, e *gochannel.GoChannel) *Orchestrator {
	return &Orchestrator{
		Config:   config,
		EventBus: e,
	}
}

func (o *Orchestrator) RunModule(ctx context.Context) error {
	// TODO(chenweilunster): Actually implement Orchestrator.
	messages, err := o.EventBus.Subscribe(ctx, panoptic.TOPIC_PENDING_TASK)
	if err != nil {
		return err
	}

	for msg := range messages {
		log.Printf("Orchestrator %s received message", o.Name())
		panopticJob := protocol.PanopticJob{}
		err := proto.Unmarshal(msg.Payload, &panopticJob)

		if err != nil {
			return err
		}

		log.Println(panopticJob.String())
		msg.Ack()
	}

	return nil
}

func (o *Orchestrator) Name() string {
	return o.Config.Name
}
