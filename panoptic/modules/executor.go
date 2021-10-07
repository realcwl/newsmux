package modules

import (
	"context"

	"github.com/Luismorlan/newsmux/protocol"
)

// Executor is in charge of job execution. This is the common interface shared
// by different types of job executors.
type Executor interface {
	Execute(ctx context.Context, job *protocol.PanopticJob) (*protocol.PanopticJob, error)
}
