package redismethod

import "github.com/redis/go-redis/v9"

type RedisMethod struct {
	redis *redis.Client
}

func New(redis *redis.Client) *RedisMethod {
	return &RedisMethod{redis: redis}
}

func (rm *RedisMethod) RefreshIsExist() bool {
	return true
}
