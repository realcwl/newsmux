package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRedisStatusStore(t *testing.T) {
	_, err := GetRedisStatusStore()
	assert.Nil(t, err)
}

func TestRedisKeyParser(t *testing.T) {
	p := &RedisKeyParser{delimiter: "_"}
	validUserId := "valid-user-id"
	validPostId := "valid-post-id"
	expectedKey := "valid-user-id_valid-post-id"

	invalidUserId := "invalid_user_id"
	invalidPostId := "invalid_post_id"

	assert.True(t, p.ValidateId(validUserId))
	assert.True(t, p.ValidateId(validPostId))
	assert.False(t, p.ValidateId(invalidPostId))
	assert.False(t, p.ValidateId(invalidUserId))

	k, err := p.EncodePostKey(validUserId, validPostId)
	assert.Equal(t, k, expectedKey)
	assert.Nil(t, err)

	_, err = p.EncodePostKey(invalidUserId, invalidPostId)
	assert.NotNil(t, err)

	uId, pId, err := p.DecodePostKey(expectedKey)
	assert.Nil(t, err)
	assert.Equal(t, uId, validUserId)
	assert.Equal(t, pId, validPostId)
}

func TestRedisStatusStore(t *testing.T) {
	r, err := GetRedisStatusStore()
	assert.Nil(t, err)

	userId := "user-id"
	wrongId := "wrong-id"
	readItems := []string{"read1", "read2"}
	unreadItems := []string{"unread1", "unread2", "unread3"}
	r.SetItemsReadStatus(readItems, userId, true)
	r.SetItemsReadStatus(unreadItems, userId, false)

	status, err := r.GetItemsReadStatus(readItems, userId)
	assert.Nil(t, err)
	assert.Equal(t, len(readItems), len(status))
	for _, s := range status {
		assert.True(t, s)
	}

	status, err = r.GetItemsReadStatus(unreadItems, userId)
	assert.Nil(t, err)
	assert.Equal(t, len(unreadItems), len(status))
	for _, s := range status {
		assert.False(t, s)
	}

	status, err = r.GetItemsReadStatus(readItems, wrongId)
	assert.Equal(t, len(readItems), len(status))
	assert.Nil(t, err)
	for _, s := range status {
		assert.False(t, s)
	}
}
