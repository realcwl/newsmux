package panoptic

import (
	"context"
	"log"
	"sync"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Engine manages shared resources and execution lifecycle of each module. It
// maintains a shared event bus
type Engine struct {
	// A list of modules that will be run in this Engine. Module's lifetime is
	// bound to Engine's lifetime. Each Module will be ran in a separate routine.
	Modules []Module

	// The EventBus this engine managed. For now we use a golang channel
	// implementation for the EventBus, but later when needed we could substitute
	// it with Kafka-based EventBus.
	EventBus *gochannel.GoChannel
}

// Create a new Engine given the provided modules and event bus.
func NewEngine(ms []Module, e *gochannel.GoChannel) *Engine {
	return &Engine{
		Modules:  ms,
		EventBus: e,
	}
}

// Execute all Engine modules and wait untils all modules to finish execution.
func (e *Engine) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for idx := range e.Modules {
		wg.Add(1)
		go func(index int) {
			log.Printf("start engine module %s", e.Modules[index].Name())
			defer wg.Done()
			RunModuleWithGracefulRestart(ctx, &e.Modules[index])
			log.Printf("Module %s finished execution.", e.Modules[index].Name())
		}(idx)
	}

	// Block until all goroutine finished execution.
	wg.Wait()
}
