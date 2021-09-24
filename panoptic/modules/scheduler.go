package modules

import (
	"context"
	"log"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

type SchedulerConfig struct {
	// Name of the orchestrator.
	Name string
}

type Scheduler struct {
	Config SchedulerConfig

	EventBus *gochannel.GoChannel
}

// Return a new instance of Scheduler.
func NewScheduler(config SchedulerConfig, e *gochannel.GoChannel) *Scheduler {
	return &Scheduler{
		Config:   config,
		EventBus: e,
	}
}

func (s *Scheduler) RunModule(ctx context.Context) error {
	// TODO(chenweilunster): Actually implement Scheduler.
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			log.Printf("Scheduler: %s sending message", s.Name())
			msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, world!"))
			s.EventBus.Publish("task.topic", msg)
			time.Sleep(3 * time.Second)
		}
	}
}

func (s *Scheduler) Name() string {
	return s.Config.Name
}
