package models

import "time"

// FriendLink 友情链接；Status 1=展示 0=隐藏，Sort 越小越靠前。
type FriendLink struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	URL         string    `gorm:"size:255;not null" json:"url"`
	Logo        string    `gorm:"size:255" json:"logo"`
	Description string    `gorm:"size:255" json:"description"`
	Sort        int       `gorm:"default:0" json:"sort"`
	Status      int8      `gorm:"default:1;comment:1展示 0隐藏" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
