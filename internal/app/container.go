package app

import (
	"blog/config"
	"blog/internal/handler"
	"blog/internal/middleware"
	"blog/internal/redismethod"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/internal/wechat"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Config = *config.Config

type Container struct {
	Config     Config
	DB         *gorm.DB
	Redis      *redis.Client
	Repository *repository.Repository
	Service    *service.Service
	Handler    *handler.Handler
	RedisStore *redismethod.RedisMethod
}

func NewContainer(cfg Config, db *gorm.DB, redis *redis.Client) *Container {
	//repository
	repo := repository.New(db)
	authMiddleRepo := repository.NewRedisMiddleRepo(redis)
	middleware.SetRedisRepo(authMiddleRepo)
	//微信客户端
	var wechatClient wechat.Exchanger
	if strings.TrimSpace(cfg.Wechat.AppID) != "" && strings.TrimSpace(cfg.Wechat.AppSecret) != "" {
		wechatClient = wechat.NewClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret, "")
	}
	//service
	svc := service.New(repo, redis, wechatClient, authMiddleRepo)
	middleware.SetUserAccessChecker(svc)
	//handle
	h := handler.New(svc)
	//额外一层redismethod，这个层里写需要访问redis的方法，供auth中间件使用
	rm := redismethod.New(redis)
	return &Container{
		Config:     cfg,
		DB:         db,
		Redis:      redis,
		Repository: repo,
		Service:    svc,
		Handler:    h,
		RedisStore: rm,
	}
}
