package repository

import (
	"blog/models"
	"errors"

	"gorm.io/gorm"
)

// GetSettings 按配置键批量查询站点配置；参数为配置键列表，返回值为已存在的配置记录和查询错误。
func (r *Repository) GetSettings(keys []string) ([]models.Setting, error) {
	settings := make([]models.Setting, 0, len(keys))
	if len(keys) == 0 {
		return settings, nil
	}
	err := r.db.Where("`key` IN ?", keys).Find(&settings).Error
	return settings, err
}

// UpsertSettings 在一个事务中批量新增或更新站点配置；参数为待保存的配置记录，返回值为事务执行错误。
func (r *Repository) UpsertSettings(settings []models.Setting) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range settings {
			var existing models.Setting
			err := tx.Where("`key` = ?", setting.Key).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err = tx.Create(&setting).Error; err != nil {
					return err
				}
				continue
			}
			if err = tx.Model(&existing).Update("value", setting.Value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
