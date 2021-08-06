package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/stretchr/testify/assert"
)

func TestSeedStateChannelCreation(t *testing.T) {
	scs := NewSeedStateChannels()
	ctx, cancel := context.WithCancel(context.Background())

	scs.AddNewConnection(ctx, "user_1")
	assert.Equal(t, 1, scs.GetActiveConnectionsCount())

	cancel()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, scs.GetActiveConnectionsCount())
}

func TestSeedStateChannelMultipleCreation(t *testing.T) {
	scs := NewSeedStateChannels()
	ctx_1, cancel_1 := context.WithCancel(context.Background())
	ctx_2, cancel_2 := context.WithCancel(context.Background())
	ctx_3, cancel_3 := context.WithCancel(context.Background())

	// User 1 signed in 2 devices.
	scs.AddNewConnection(ctx_1, "user_1")
	scs.AddNewConnection(ctx_2, "user_1")

	// User 2 signed in only 1 device.
	scs.AddNewConnection(ctx_3, "user_2")

	assert.Equal(t, 3, scs.GetActiveConnectionsCount())

	cancel_1()
	cancel_2()
	cancel_3()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, scs.GetActiveConnectionsCount())
}

func TestPushSeedStateToUser(t *testing.T) {
	scs := NewSeedStateChannels()
	ctx, cancel := context.WithCancel(context.Background())
	ch := scs.AddNewConnection(ctx, "user_id")

	done := make(chan interface{})
	go func() {
		state := <-ch
		assert.Equal(t, state, &model.SeedState{
			Username: "test",
		})
		done <- 0
	}()

	scs.PushSeedStateToUser(&model.SeedState{
		Username: "test",
	}, "user_id")
	<-done

	cancel()
	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Error(t, scs.PushSeedStateToUser(&model.SeedState{
		Username: "test",
	}, "user_id"))
}
