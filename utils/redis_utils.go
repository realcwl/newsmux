package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	inner *redis.Client
}

const (
	// Redis only has string type, there is no boolean or int, so we use "1" to represent true
	REDIS_TRUE = "1"
)

var ctx = context.Background()

func GetRedisClient() *RedisClient {
	return &RedisClient{
		inner: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
			Password: os.Getenv("REDIS_PASSWD"),
			DB:       0, // use default DB
		})}
}

func PostKey(userId string, postId string) string {
	return fmt.Sprintf("%s_%s", userId, postId)
}

func (r RedisClient) GetPostsReadStatus(postIds []string, userId string) ([]bool, error) {
	postKeys := []string{}

	for _, pid := range postIds {
		postKeys = append(postKeys, PostKey(userId, pid))
	}

	res, err := r.inner.MGet(ctx, postKeys...).Result()
	status := []bool{}
	for _, v := range res {
		if v == nil {
			status = append(status, false)
		}
		status = append(status, v.(bool))
	}
	return status, err
}

func (r RedisClient) MarkPostsAsRead(postIds []string, userId string) error {
	keyValues := []interface{}{}
	for _, pid := range postIds {
		keyValues = append(keyValues, PostKey(userId, pid))
		keyValues = append(keyValues, REDIS_TRUE)
	}
	return r.inner.MSetNX(ctx, keyValues...).Err()
}
