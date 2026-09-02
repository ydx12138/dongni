package models

import "time"

type Comment struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ArticleID uint64    `gorm:"comment:文章ID" json:"article_id"`
	Article   Article   `json:"-"`
	UserID    uint64    `gorm:"comment:评论所属用户ID" json:"user_id"`
	User      User      `json:"-"`
	ParentID  uint64    `gorm:"default:0;comment:父评论ID" json:"parent_id"`
	Content   string    `gorm:"type:text;not null;comment:评论内容" json:"content"`
	Status    int8      `gorm:"default:1;comment:1正常 2隐藏 3待审核" json:"status"`
	HitWords  string    `gorm:"type:text;comment:逗号分隔敏感词" json:"hit_words"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
