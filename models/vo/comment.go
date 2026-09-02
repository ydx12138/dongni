package vo

import "time"

type CommentVO struct {
	ID           uint64    `json:"id"`
	ArticleID    uint64    `json:"article_id"`
	ArticleTitle string    `json:"article_title"`
	UserID       uint64    `json:"user_id"`
	Nickname     string    `json:"nickname"`
	Content      string    `json:"content"`
	ParentID     uint64    `json:"parent_id"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
