package vo

import (
	"time"
)

type ArticleSimple struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Cover        string    `json:"cover"`
	CategoryID   uint64    `json:"-"`
	CategoryName string    `json:"category_name"`
	ViewCount    uint64    `json:"view_count"`
	LikeCount    uint64    `json:"like_count"`
	CommentCount uint64    `json:"comment_count"`
	Tags         string    `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ArticleInfinitePage 返回分类文章无限加载结果；参数无；返回文章列表、是否还有下一批和下一页页码。
type ArticleInfinitePage struct {
	List     []ArticleSimple `json:"list"`
	HasMore  bool            `json:"has_more"`
	NextPage int             `json:"next_page"`
}

// ArticleSearch 是搜索结果文章；Content 只在后端内部用于生成命中片段，不会返回给前端。
type ArticleSearch struct {
	ID            uint64    `json:"id"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	SearchExcerpt string    `json:"search_excerpt"`
	Content       string    `json:"-"`
	Cover         string    `json:"cover"`
	CategoryID    uint64    `json:"-"`
	CategoryName  string    `json:"category_name"`
	ViewCount     uint64    `json:"view_count"`
	LikeCount     uint64    `json:"like_count"`
	CommentCount  uint64    `json:"comment_count"`
	Tags          string    `json:"tags"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ArticleDetail struct {
	ID           uint64     `json:"id" form:"id"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Content      string     `json:"content"`
	ContentType  int8       `json:"content_type"`
	Cover        string     `json:"cover"`
	CategoryName string     `json:"category_name"`
	ViewCount    uint64     `json:"view_count"`
	LikeCount    uint64     `json:"like_count"`
	CommentCount uint64     `json:"comment_count"`
	PublishTime  *time.Time `json:"publish_time"`
	Tags         string     `json:"tags"`
}
