package core

import (
	"blog/config"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func InitRedis() (*redis.Client, error) {
	var rdb *redis.Client
	redis1 := config.Cfg.Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redis1.Host, redis1.Port),
		Password: redis1.Password,
		DB:       redis1.DB,

		// 连接池（很重要）
		PoolSize:     redis1.PoolSize,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		_ = rdb.Close()
		zap.L().Error("redis ping error", zap.Error(err))
		//return nil, fmt.Errorf("redis connect failed: %w", err)
	}

	fmt.Println("Redis connected successfully")
	return rdb, nil
}
