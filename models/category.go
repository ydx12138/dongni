package models

import "time"

const DefaultCategoryCover = "/default-category-cover.jpg"

type Category struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Cover       string    `gorm:"size:255;not null;default:/default-category-cover.jpg" json:"cover"`
	Sort        int       `gorm:"default:0" json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
