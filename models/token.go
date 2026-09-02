package models

import "time"

type Token struct {
	ID uint64 `gorm:"primaryKey"`

	UserID uint64 `gorm:"uniqueIndex;not null;comment:token所属用户ID"`
	User   User

	Token string `gorm:"size:512;not null"`

	ExpiresAt time.Time `gorm:"not null;comment:到期时间"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
