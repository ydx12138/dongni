package core

import (
	"blog/config"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB is kept for legacy code paths such as seed scripts. New runtime code should use dependency injection.
var DB *gorm.DB

func DataBaseInit() (*gorm.DB, error) {
	var level logger.LogLevel = logger.Info
	switch config.Cfg.MysqlConfig.Log_level {
	case "info":
		level = logger.Info
	case "warn":
		level = logger.Warn
	case "error":
		level = logger.Error
	case "silent":
		level = logger.Silent
	}

	db, err := gorm.Open(mysql.Open(config.Cfg.MysqlConfig.DSN()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(level),
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		zap.L().Error("database connection failed: " + err.Error())
		zap.L().Error(config.Cfg.MysqlConfig.Password)
		return nil, err
	}
	DB = db
	return db, nil
}
