package models

import "time"

type Setting struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement;comment:主键"`
	Key       string `gorm:"size:50;uniqueIndex;comment:键"`
	Value     string `gorm:"type:text;not null;comment:值"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
