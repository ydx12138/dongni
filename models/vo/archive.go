package vo

import "time"

// ArchiveItem 归档页里单篇文章的精简信息（避免重复 ArticleSimple 中的大字段）。
type ArchiveItem struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	Cover        string    `json:"cover"`
	CategoryName string    `json:"category_name"`
	Tags         string    `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
}

// ArchiveYear 归档页按年份聚合的节点：年份 + 该年文章数 + 文章列表。
type ArchiveYear struct {
	Year     int           `json:"year"`
	Count    int           `json:"count"`
	Articles []ArchiveItem `json:"articles"`
}

// ArchiveResult 归档页整体响应：所有年份节点（按年份倒序）。
type ArchiveResult struct {
	Total int           `json:"total"`
	Years []ArchiveYear `json:"years"`
}
