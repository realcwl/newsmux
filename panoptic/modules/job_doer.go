package modules

import (
	"log"

	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// JobDoer execute the SchedulerJob with customized logic. We create this
// abstraction so that we could inject different JobDoer implementation into
// scheduler for the easy of testing and debugging.
type JobDoer interface {
	// Performs a SchedulerJob, return error if there's any.
	Do(job *SchedulerJob) error
}

type SchedulerJobDoer struct {
	EventBus *gochannel.GoChannel
}

func NewSchedulerJobDoer(e *gochannel.GoChannel) *SchedulerJobDoer {
	return &SchedulerJobDoer{
		EventBus: e,
	}
}

// Convert SchedulerJob to PanopticJob and publish to event bus.
func (d *SchedulerJobDoer) Do(job *SchedulerJob) error {
	panopticJob := &protocol.PanopticJob{}
	panopticJob.JobId = uuid.NewString()
	panopticJob.Tasks = append(panopticJob.Tasks, &protocol.PanopticTask{
		TaskId:          uuid.NewString(),
		DataCollectorId: job.panopticConfig.DataCollectorId,
		TaskParams:      job.panopticConfig.TaskParams,
		TaskMetadata: &protocol.TaskMetadata{
			ConfigName: job.panopticConfig.Name,
		},
	})

	// We set the panoptic job to be debug only mode iff:
	// - in DEV environment.
	// - when config explicitly marked it as dry_run mode.
	panopticJob.Debug = !utils.IsProdEnv() || job.panopticConfig.DryRun

	data, err := proto.Marshal(panopticJob)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	d.EventBus.Publish(panoptic.TOPIC_PENDING_JOB, msg)

	job.IncrementRunCount()

	return nil
}

// Test only, print the to-be executed job
type PrinterJobDoer struct{}

func (d *PrinterJobDoer) Do(job *SchedulerJob) error {
	log.Println(job.panopticConfig.String())

	job.IncrementRunCount()

	return nil
}
