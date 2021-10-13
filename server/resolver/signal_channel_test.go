package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/stretchr/testify/assert"
)

func TestSignalChannelCreation(t *testing.T) {
	sigChan := NewSignalChannels()
	ctx, cancel := context.WithCancel(context.Background())

	sigChan.AddNewConnection(ctx, "user_1")
	assert.Equal(t, 1, sigChan.GetActiveConnectionsCount())

	cancel()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, sigChan.GetActiveConnectionsCount())
}

func TestSignalChannelMultipleCreation(t *testing.T) {
	sigChan := NewSignalChannels()
	ctx_1, cancel_1 := context.WithCancel(context.Background())
	ctx_2, cancel_2 := context.WithCancel(context.Background())
	ctx_3, cancel_3 := context.WithCancel(context.Background())

	// User 1 signed in 2 devices.
	sigChan.AddNewConnection(ctx_1, "user_1")
	sigChan.AddNewConnection(ctx_2, "user_1")

	// User 2 signed in only 1 device.
	sigChan.AddNewConnection(ctx_3, "user_2")

	assert.Equal(t, 3, sigChan.GetActiveConnectionsCount())

	cancel_1()
	cancel_2()
	cancel_3()

	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, sigChan.GetActiveConnectionsCount())
}

func TestPushSignalToUser(t *testing.T) {
	sigChan := NewSignalChannels()
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := sigChan.AddNewConnection(ctx, "user_id")

	done := make(chan interface{})
	go func() {
		state := <-ch
		assert.Equal(t, state, &model.Signal{
			SignalType: model.SignalTypeSeedState})
		done <- 0
	}()

	sigChan.PushSignalToUser(&model.Signal{
		SignalType: model.SignalTypeSeedState}, "user_id")
	<-done

	cancel()
	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Error(t, sigChan.PushSignalToUser(&model.Signal{
		SignalType: model.SignalTypeSeedState,
	}, "test"))
}

func TestPushSignalToSingleChannel(t *testing.T) {
	ssc := NewSignalChannels()
	ctx, cancel := context.WithCancel(context.Background())
	ch, chId := ssc.AddNewConnection(ctx, "user_id")

	done := make(chan interface{})
	go func() {
		state := <-ch
		assert.Equal(t, state, &model.Signal{
			SignalType: model.SignalTypeSeedState})
		done <- 0
	}()

	ssc.PushSignalToSingleChannelForUser(&model.Signal{
		SignalType: model.SignalTypeSeedState},
		chId, "user_id")
	<-done

	cancel()
	// Force trigger an long IO operation to context swiching to clean up.
	time.Sleep(1 * time.Second)
	assert.Error(t, ssc.PushSignalToUser(&model.Signal{
		SignalType: model.SignalTypeSeedState}, "user_id"))
}
