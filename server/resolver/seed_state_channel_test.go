package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/stretchr/testify/assert"
)

func TestSeedStateChannelCreation(t *testing.T) {
	ssc := NewSeedStateChannels()
	ctx, cancel := context.WithCancel(context.Background())

	ssc.AddNewConnection(ctx, "user_1")
	assert.Equal(t, 1, ssc.GetActiveConnectionsCount())

	cancel()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, ssc.GetActiveConnectionsCount())
}

func TestSeedStateChannelMultipleCreation(t *testing.T) {
	ssc := NewSeedStateChannels()
	ctx_1, cancel_1 := context.WithCancel(context.Background())
	ctx_2, cancel_2 := context.WithCancel(context.Background())
	ctx_3, cancel_3 := context.WithCancel(context.Background())

	// User 1 signed in 2 devices.
	ssc.AddNewConnection(ctx_1, "user_1")
	ssc.AddNewConnection(ctx_2, "user_1")

	// User 2 signed in only 1 device.
	ssc.AddNewConnection(ctx_3, "user_2")

	assert.Equal(t, 3, ssc.GetActiveConnectionsCount())

	cancel_1()
	cancel_2()
	cancel_3()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, ssc.GetActiveConnectionsCount())
}

func TestPushSeedStateToUser(t *testing.T) {
	ssc := NewSeedStateChannels()
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := ssc.AddNewConnection(ctx, "user_id")

	done := make(chan interface{})
	go func() {
		state := <-ch
		assert.Equal(t, state, &model.SeedState{
			UserSeedState: &model.UserSeedState{
				Name: "test",
			},
		})
		done <- 0
	}()

	ssc.PushSeedStateToUser(&model.SeedState{
		UserSeedState: &model.UserSeedState{
			Name: "test",
		},
	}, "user_id")
	<-done

	cancel()
	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Error(t, ssc.PushSeedStateToUser(&model.SeedState{
		UserSeedState: &model.UserSeedState{
			Name: "test",
		},
	}, "user_id"))
}

func TestPushSeedStateToSingleChannel(t *testing.T) {
	ssc := NewSeedStateChannels()
	ctx, cancel := context.WithCancel(context.Background())
	ch, chId := ssc.AddNewConnection(ctx, "user_id")

	done := make(chan interface{})
	go func() {
		state := <-ch
		assert.Equal(t, state, &model.SeedState{
			UserSeedState: &model.UserSeedState{
				Name: "test",
			},
		})
		done <- 0
	}()

	ssc.PushSeedStateToSingleChannelForUser(&model.SeedState{
		UserSeedState: &model.UserSeedState{
			Name: "test",
		},
	}, chId, "user_id")
	<-done

	cancel()
	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Error(t, ssc.PushSeedStateToUser(&model.SeedState{
		UserSeedState: &model.UserSeedState{
			Name: "test",
		},
	}, "user_id"))
}
