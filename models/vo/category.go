package vo

import "time"

// CategoryWithStats 前台分类列表扩展数据；包含文章数与总浏览数；便于分类索引页展示。
type CategoryWithStats struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Cover        string    `json:"cover"`
	Sort         int       `json:"sort"`
	ArticleCount int64     `json:"article_count"`
	ViewCount    int64     `json:"view_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
