package main

import (
	"blog/config"
	"blog/core"
	"blog/flags"
	"blog/internal/app"
	"blog/internal/router"
	"blog/internal/utils"
	"blog/seed"
	"log"

	"go.uber.org/zap"
)

func main() {
	//命令行参数
	flags.Parse()
	//加载settings.yaml配置文件
	cfg, err := config.LoadConfig()
	if err != nil {
		zap.L().Error("config load failed: " + err.Error())
		return
	}
	//加载日志
	core.LogInit()
	//加载数据库
	db, err := core.DataBaseInit()
	if err != nil {
		return
	}
	//迁移
	core.InitModel(db)
	//插入初始数据
	if flags.FlagOptions.Seed {
		seed.Run()
		zap.L().Info("seed data completed")
		return
	}
	//加载redis
	initRedis, err := core.InitRedis()
	if err != nil {
		zap.L().Error("redis init failed: " + err.Error())
		log.Print("redis加载失败")
	}

	//加载敏感词
	utils.InitSensitive("./high.txt")

	//加载依赖
	container := app.NewContainer(cfg, db, initRedis)
	//加载路由
	r := router.Register(container)

	zap.L().Debug("gin running at " + config.Cfg.SystemConfig.Address())
	//启动项目
	if err := r.Run(config.Cfg.SystemConfig.Address()); err != nil {
		zap.L().Error("router run failed: " + err.Error())
	}
}
