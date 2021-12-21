package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-redis/redis/v8"
)

type RedisStatusStore struct {
	inner     *redis.Client
	keyParser RedisKeyParser
}

const (
	// Redis only has string type, there is no boolean or int, so we use "1" to represent true
	RedisTrue  = "1"
	RedisFalse = "0"
)

var ctx = context.Background()

func GetRedisStatusStore() (*RedisStatusStore, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: os.Getenv("REDIS_PASSWD"),
		DB:       0, // use default DB
	})
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	return &RedisStatusStore{
		inner:     redisClient,
		keyParser: RedisKeyParser{delimiter: "__"},
	}, nil
}

type RedisKeyParser struct {
	delimiter string
}

func (r RedisKeyParser) DecodePostKey(key string) (string, string, error) {
	splits := strings.Split(key, r.delimiter)
	if (len(splits)) != 2 {
		return "", "", fmt.Errorf("invalid key: %s", key)
	}
	return splits[0], splits[1], nil
}

func (r RedisKeyParser) ValidateId(id string) bool {
	return !strings.Contains(id, r.delimiter)
}

func (r RedisKeyParser) EncodePostKey(userId string, postId string) (string, error) {
	if !r.ValidateId(userId) || !r.ValidateId(postId) {
		return "", fmt.Errorf("invalid userId or postId")
	}
	return fmt.Sprintf("%s%s%s", userId, r.delimiter, postId), nil
}

func (r RedisKeyParser) MustEncodePostKey(userId string, postId string) string {
	if !r.ValidateId(userId) || !r.ValidateId(postId) {
		panic(fmt.Errorf("invalid userId or postId with delimiter: %s, %s, %s", userId, postId, r.delimiter))
	}
	return fmt.Sprintf("%s%s%s", userId, r.delimiter, postId)
}

func (r *RedisStatusStore) GetItemsReadStatus(itemNodeIds []string, userId string) ([]bool, error) {
	if len(itemNodeIds) == 0 {
		return []bool{}, nil
	}

	postKeys := []string{}

	for _, pid := range itemNodeIds {
		postKeys = append(postKeys, r.keyParser.MustEncodePostKey(userId, pid))
	}

	res, err := r.inner.MGet(ctx, postKeys...).Result()
	status := []bool{}
	for _, v := range res {
		if v == nil {
			status = append(status, false)
			continue
		}

		// watchout
		if v == RedisTrue {
			status = append(status, true)
			continue
		}
		status = append(status, false)
	}
	return status, err
}

func (r RedisStatusStore) SetItemsReadStatus(itemNodeIds []string, userId string, read bool) error {
	if read {
		keyValues := []interface{}{}
		for _, pid := range itemNodeIds {
			keyValues = append(keyValues, r.keyParser.MustEncodePostKey(userId, pid))
			keyValues = append(keyValues, RedisTrue)
		}
		return r.inner.MSetNX(ctx, keyValues...).Err()
	}

	keyValues := []string{}
	for _, pid := range itemNodeIds {
		keyValues = append(keyValues, r.keyParser.MustEncodePostKey(userId, pid))
	}
	return r.inner.Del(ctx, keyValues...).Err()
}
