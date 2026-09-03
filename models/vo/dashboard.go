package vo

import "time"

// DashboardData 描述管理端运营面板的完整统计数据。
type DashboardData struct {
	TotalArticles         int64                     `json:"total_articles"`
	PublishedArticles     int64                     `json:"published_articles"`
	DraftArticles         int64                     `json:"draft_articles"`
	TotalComments         int64                     `json:"total_comments"`
	PendingComments       int64                     `json:"pending_comments"`
	HiddenComments        int64                     `json:"hidden_comments"`
	TotalUsers            int64                     `json:"total_users"`
	TotalViews            int64                     `json:"total_views"`
	TotalLikes            int64                     `json:"total_likes"`
	TotalCategories       int64                     `json:"total_categories"`
	UncategorizedArticles int64                     `json:"uncategorized_articles"`
	NewArticles           int64                     `json:"new_articles"`
	NewUsers              int64                     `json:"new_users"`
	NewComments           int64                     `json:"new_comments"`
	Trend                 []DashboardTrendPoint     `json:"trend"`
	TopArticles           []DashboardArticleMetric  `json:"top_articles"`
	Categories            []DashboardCategoryMetric `json:"categories"`
	RecentArticles        []DashboardRecentArticle  `json:"recent_articles"`
	RecentComments        []DashboardRecentComment  `json:"recent_comments"`
	RecentUsers           []DashboardRecentUser     `json:"recent_users"`
}

// DashboardTrendPoint 描述某一天的新增文章、用户与评论数量。
type DashboardTrendPoint struct {
	Date     string `json:"date"`
	Articles int64  `json:"articles"`
	Users    int64  `json:"users"`
	Comments int64  `json:"comments"`
}

// DashboardArticleMetric 描述文章在运营面板中的互动表现。
type DashboardArticleMetric struct {
	ID           uint64 `json:"id"`
	Title        string `json:"title"`
	CategoryName string `json:"category_name"`
	ViewCount    uint64 `json:"view_count"`
	LikeCount    uint64 `json:"like_count"`
	CommentCount uint64 `json:"comment_count"`
}

// DashboardCategoryMetric 描述分类下文章量与累计阅读量。
type DashboardCategoryMetric struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	ArticleCount int64  `json:"article_count"`
	ViewCount    int64  `json:"view_count"`
}

// DashboardRecentArticle 描述最近创建或更新的文章。
type DashboardRecentArticle struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	CategoryName string    `json:"category_name"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// DashboardRecentComment 描述最新评论动态。
type DashboardRecentComment struct {
	ID           uint64    `json:"id"`
	ArticleID    uint64    `json:"article_id"`
	ArticleTitle string    `json:"article_title"`
	Nickname     string    `json:"nickname"`
	Content      string    `json:"content"`
	Status       int8      `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// DashboardRecentUser 描述最近注册用户动态。
type DashboardRecentUser struct {
	ID        uint64    `json:"id"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Status    uint64    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
