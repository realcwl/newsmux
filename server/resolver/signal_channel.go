package resolver

import (
	"context"
	"errors"
	"sync"

	"github.com/Luismorlan/newsmux/model"
	"github.com/google/uuid"
)

// SignalChannels contains all structures that handles user's signal channel.
// All internal state should not be handled directly by hand by managed by its
// public receivers.
type SignalChannels struct {
	// connectionMap maps from user id to the user's active signal channels.
	// User's active channels are represented in the form of a map from channel
	// id (uuid) to the actual channel. This is needed so that deletion of channel
	// is O(1).
	// Each connectionMap entry will be deleted once all user's active channels
	// are closed.
	// Multiple user's devices cannot share the same channel and has to create its
	// own unique channel, I don't find a way of subscription sharing.
	connectionMap map[string]map[string]chan *model.Signal

	// Adding/Removing a new subscription must grab WriteLock, while all other
	// usage (e.g. pushing a new Signal) should grab a ReadLock. Ideally we
	// should create lock per-user but we can start from a shared lock in the
	// beginning for simplicity.
	mu sync.RWMutex
}

func NewSignalChannels() *SignalChannels {
	return &SignalChannels{
		connectionMap: make(map[string]map[string]chan *model.Signal),
		mu:            sync.RWMutex{},
	}
}

// cleanUp a single connection when the context terminates. If a user's all
// active connections terminates, clean up the user's top-level entry as well.
func (sc *SignalChannels) cleanUp(ctx context.Context, ch_id string, user_id string) {
	<-ctx.Done()

	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.connectionMap[user_id], ch_id)
	if len(sc.connectionMap[user_id]) == 0 {
		delete(sc.connectionMap, user_id)
	}
}

// Thread-safe
func (sc *SignalChannels) AddNewConnection(ctx context.Context, user_id string) (chan *model.Signal, string) {
	ch_id := "signal_channel_" + uuid.New().String()
	ch := make(chan *model.Signal, 1)

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if _, ok := sc.connectionMap[user_id]; !ok {
		sc.connectionMap[user_id] = make(map[string]chan *model.Signal)
	}

	sc.connectionMap[user_id][ch_id] = ch

	// Spin up a background grabage collector.
	go sc.cleanUp(ctx, ch_id, user_id)

	return ch, ch_id
}

// Thread-safe
func (sc *SignalChannels) GetActiveConnectionsCount() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	count := 0
	for _, mp := range sc.connectionMap {
		count += len(mp)
	}
	return count
}

// Thread-safe
func (sc *SignalChannels) PushSignalToUser(signal *model.Signal, userId string) error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if _, ok := sc.connectionMap[userId]; !ok {
		return errors.New("no active connection for user: " + userId)
	}
	userChannels := sc.connectionMap[userId]
	for _, ch := range userChannels {
		ch <- signal
	}
	return nil
}

// Thread-safe
func (sc *SignalChannels) PushSignalToSingleChannelForUser(signal *model.Signal, chId string, userId string) error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if _, ok := sc.connectionMap[userId]; !ok {
		return errors.New("no active connection for user: " + userId)
	}

	userChannels := sc.connectionMap[userId]
	if ch, ok := userChannels[chId]; ok {
		ch <- signal
	}

	return nil
}
