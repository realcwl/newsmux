package panoptic

import (
	"context"
	"sync"

	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Engine manages shared resources and execution lifecycle of each module. It
// maintains a shared event bus
type Engine struct {
	// A list of modules that will be run in this Engine. Module's lifetime is
	// bound to Engine's lifetime. Each Module will be ran in a separate routine.
	Modules []Module

	// Root this engine is running on
	ctx context.Context

	// Cancel function for root context, used for graceful shutdown
	cancel context.CancelFunc

	// The EventBus this engine managed. For now we use a golang channel
	// implementation for the EventBus, but later when needed we could substitute
	// it with Kafka-based EventBus.
	EventBus *gochannel.GoChannel
}

// Create a new Engine given the provided modules and event bus.
func NewEngine(ms []Module, ctx context.Context, cancel context.CancelFunc, e *gochannel.GoChannel) *Engine {
	return &Engine{
		Modules:  ms,
		ctx:      ctx,
		cancel:   cancel,
		EventBus: e,
	}
}

// Execute all Engine modules and wait untils all modules to finish execution.
func (e *Engine) Run() {
	var wg sync.WaitGroup

	for idx := range e.Modules {
		wg.Add(1)
		go func(index int) {
			Logger.Log.Infof("start engine module %s", e.Modules[index].Name())
			defer wg.Done()
			RunModuleWithGracefulRestart(e.ctx, &e.Modules[index])
			Logger.Log.Infof("Module %s finished execution.", e.Modules[index].Name())
		}(idx)
	}

	// Block until all goroutine finished execution.
	wg.Wait()
}

func (e *Engine) Shutdown() {
	Logger.Log.Infoln("Starting graceful shutdown process. Goodbye!")
	e.cancel()

	var wg sync.WaitGroup
	for idx := range e.Modules {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			Logger.Log.Infof("shutdown engine module %s", e.Modules[index].Name())
			e.Modules[index].Shutdown()
			Logger.Log.Infof("Module %s shut down.", e.Modules[index].Name())
		}(idx)
	}

	// Block until all goroutine finished execution.
	wg.Wait()
}
