package models

import "time"

type Article struct {
	ID      uint64 `json:"id" gorm:"primaryKey"`
	Title   string `json:"title" gorm:"size:200;not null;comment:标题"`
	Summary string `json:"summary" gorm:"size:500;comment:摘要"`

	Content     string `json:"content" gorm:"type:longtext;comment:内容"`
	ContentType int8   `json:"content_type" gorm:"default:1;comment:1富文本 2markdown"`

	Cover string `json:"cover" gorm:"size:255;comment:封面"`

	CategoryID uint64   `json:"category_id" gorm:"comment:类别ID"`
	Category   Category //不在表里

	AuthorID uint64 `json:"author_id" gorm:"comment:作者ID"`
	Author   Admin  //不在表里

	ViewCount    uint64 `json:"view_count" gorm:"default:0;comment:浏览数"`
	LikeCount    uint64 `json:"like_count" gorm:"default:0;comment:点赞数"`
	CommentCount uint64 `json:"comment_count" gorm:"default:0;comment:评论数"`

	Status int8 `json:"status" gorm:"default:1;comment:1草稿 2发布"`

	PublishTime *time.Time `json:"publish_time"`

	Tags string `json:"tags" gorm:"size:255;comment:标签，逗号分隔"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
