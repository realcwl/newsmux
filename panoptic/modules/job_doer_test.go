package modules

import (
	"context"
	"testing"

	"github.com/Luismorlan/newsmux/panoptic"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestSchedulerJobDoer(t *testing.T) {
	job := GetDefaultSchedulerJob(t)

	eventbus := gochannel.NewGoChannel(
		gochannel.Config{},
		watermill.NewStdLogger(false, false),
	)
	ctx := context.Background()

	// Go channel receive and send cannot be in the same routine, otherwise it
	// will cause deadlock. Thus we need to asynchronously get back message.
	var receivedMsg *message.Message
	done := make(chan int)
	// Receiver
	messages, err := eventbus.Subscribe(
		ctx, panoptic.TOPIC_PENDING_TASK)
	assert.Nil(t, err)

	go func() {
		// Publisher
		schedulerJobDoer := NewSchedulerJobDoer(eventbus)
		assert.Nil(t, schedulerJobDoer.Do(job))
		assert.Equal(t, job.runCount, int64(1))
	}()

	go func() {
		for message := range messages {
			receivedMsg = message
			message.Ack()
			done <- 1
			break
		}
	}()

	// Wait for message to be received.
	<-done

	// Validate received message.
	assert.NotNil(t, receivedMsg)

	panopticJob := protocol.PanopticJob{}
	assert.Nil(t, proto.Unmarshal(receivedMsg.Payload, &panopticJob))

	assert.Equal(t, len(panopticJob.Tasks), 1)
	opts := []cmp.Option{
		protocmp.Transform(),
	}
	assert.Empty(t, cmp.Diff(
		job.panopticConfig.TaskParams,
		panopticJob.Tasks[0].TaskParams,
		opts...))
	assert.Equal(t,
		job.panopticConfig.DataCollectorId,
		panopticJob.Tasks[0].DataCollectorId)
}
