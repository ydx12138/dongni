package models

import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Password     string    `gorm:"size:255;not null" json:"-"`
	Avatar       string    `gorm:"size:255;not null" json:"avatar"`
	Nickname     string    `gorm:"size:50" json:"nickname"`
	Phone        *string   `gorm:"size:50;uniqueIndex;default:null;comment:手机号" json:"phone"`
	Status       uint64    `gorm:"default:1;comment:1正常，2封禁" json:"status"`
	WechatOpenID string    `gorm:"size:64;uniqueIndex" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
