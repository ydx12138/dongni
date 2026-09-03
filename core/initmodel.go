package core

import (
	model2 "blog/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func InitModel(db *gorm.DB) {
	if db.Migrator().HasTable(&model2.User{}) {
		if err := db.Exec("UPDATE `user` SET phone = NULL WHERE phone = ''").Error; err != nil {
			zap.L().Panic("normalize empty user phones failed: " + err.Error())
		}
	}
	err := db.AutoMigrate(
		&model2.User{},
		&model2.Admin{},
		&model2.Category{},
		&model2.Article{},
		&model2.Comment{},
		&model2.Token{},
		&model2.Setting{},
	)
	if err != nil {
		zap.L().Panic("migrate tables failed: " + err.Error())
	}
	if err := db.Model(&model2.Category{}).Where("cover = ? OR cover IS NULL", "").Update("cover", model2.DefaultCategoryCover).Error; err != nil {
		zap.L().Panic("normalize empty category covers failed: " + err.Error())
	}
}
