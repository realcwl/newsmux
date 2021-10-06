package panoptic

import (
	"context"
	"log"
	"time"
)

const (
	GracefulRetryDelay = 3
)

func RunModuleWithGracefulRestart(ctx context.Context, module *Module) {
	for {
		err := (*module).RunModule(ctx)
		if err == nil {
			break
		}
		log.Printf(
			"Module %s exited with error %v, retry in %d seconds",
			(*module).Name(),
			err,
			GracefulRetryDelay)

		// Wait for a small amount of time and restart.
		time.Sleep(GracefulRetryDelay * time.Second)
	}
}

type Module interface {
	// RunModule contains the customized logic of the module. It takes in a
	// context object by which its lifecycle is managed. Return error if
	// encountered any error during execution.
	RunModule(ctx context.Context) error

	// Return name of the Module. Uniquely identifies the module instance. Note
	// that if there are multiple instances of the same module, each instance
	// should have a unique name instead of using the same name.
	Name() string
}
