package models

import "time"

type Admin struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;uniqueIndex;not null;comment:用户名" json:"username"`
	Password  string    `gorm:"size:255;not null;comment:密码" json:"-"`
	Nickname  string    `gorm:"size:50;comment:昵称" json:"nickname"`
	Email     string    `gorm:"size:100;comment:邮箱" json:"email"`
	Phone     string    `gorm:"size:50;comment:手机号" json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
