package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisMiddleRepo struct {
	Re *redis.Client
}

func NewRedisMiddleRepo(redis *redis.Client) RedisMiddleRepository {
	return &redisMiddleRepo{Re: redis}
}

// SaveUserSession 将指定用户当前的 PC 会话 ID 写入 Redis。
// 参数：userID 为用户 ID，sessionID 为随机会话 ID，duration 为 Redis 过期时间；返回 Redis 写入错误。
func (r *redisMiddleRepo) SaveUserSession(userID uint64, sessionID string, duration time.Duration) error {
	if r.Re == nil {
		return errors.New("redis client is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.Re.Set(ctx, pcUserSessionKey(userID), sessionID, duration).Err()
}

// GetUserSession 读取指定用户当前保存的 PC 会话 ID。
// 参数：userID 为用户 ID；返回当前会话 ID 和 Redis 读取错误，键不存在时返回空字符串和 nil。
func (r *redisMiddleRepo) GetUserSession(userID uint64) (string, error) {
	if r.Re == nil {
		return "", errors.New("redis client is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	value, err := r.Re.Get(ctx, pcUserSessionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

// pcUserSessionKey 生成用户 PC 单点登录会话在 Redis 中使用的键。
// 参数：userID 为用户 ID；返回固定格式的 Redis 键名。
func pcUserSessionKey(userID uint64) string {
	return "pc:user:session:" + strconv.FormatUint(userID, 10)
}
