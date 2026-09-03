package repository

import (
	"blog/models"
	"blog/models/vo"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 根据email修改用户密码
func (r *Repository) UpdateUserPassword(email, password string) error {
	err := r.db.Model(&models.User{}).
		Where("email = ?", email).
		Update("password", password).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetArticleByPage(page int, pageSize int) ([]vo.ArticleSimple, int64, error) {
	articleList := make([]vo.ArticleSimple, 0)
	var total int64
	err := r.db.Model(&models.Article{}).Where("status = ?", 2).Count(&total).Error
	if err != nil {
		zap.L().Error("GetArticleByPage count:" + err.Error())
		return articleList, total, err
	}
	err = r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.cover,
			article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Order("article.created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&articleList).Error
	return articleList, total, err
}

// GetArticleRanking 查询点赞数最高的已发布文章；参数 limit 为最多返回的文章数；返回文章摘要列表和查询错误。
func (r *Repository) GetArticleRanking(limit int) ([]vo.ArticleSimple, error) {
	articleList := make([]vo.ArticleSimple, 0)
	err := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.cover,
			article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Order("article.like_count DESC, article.created_at DESC").
		Limit(limit).
		Scan(&articleList).Error
	return articleList, err
}

func (r *Repository) GetArticleDetail(id uint64) (vo.ArticleDetail, error) {
	var detail vo.ArticleDetail
	result := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.content,
			article.cover, c.name AS category_name, article.view_count,
			article.like_count, article.comment_count, article.publish_time,
			article.tags, article.content_type
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Where("article.id = ?", id).
		Scan(&detail)
	if result.Error != nil {
		return detail, result.Error
	}
	if result.RowsAffected == 0 {
		return detail, gorm.ErrRecordNotFound
	}
	return detail, nil
}

func (r *Repository) SearchArticleByKey(keyword string) ([]vo.ArticleSimple, error) {
	articleList := make([]vo.ArticleSimple, 0)
	err := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.cover,
			article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Where("article.title like ? or article.summary like ? or article.content like ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Scan(&articleList).Error
	return articleList, err
}

// SearchArticles 查询已发布文章并带出正文；参数 keyword 为搜索词、limit 为返回上限，返回搜索文章和数据库错误。
func (r *Repository) SearchArticles(keyword string, limit int) ([]vo.ArticleSearch, error) {
	articleList := make([]vo.ArticleSearch, 0)
	query := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.content,
			article.cover, article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Where("article.title like ? or article.summary like ? or article.content like ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Order("article.created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	return articleList, query.Scan(&articleList).Error
}

func (r *Repository) GetArticleByCategory(categoryID uint64, page int, pageSize int) ([]vo.ArticleSimple, error) {
	articleList := make([]vo.ArticleSimple, 0)
	err := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.cover,
			article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Where("article.category_id = ?", categoryID).
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&articleList).Error
	return articleList, err
}

// GetPublishedArticlesByCategoryPage 查询分类下已发布文章并多取一条判断是否还有下一批；参数为分类 ID、页码和批量大小；返回文章列表、是否还有更多和查询错误。
func (r *Repository) GetPublishedArticlesByCategoryPage(categoryID uint64, page int, pageSize int) ([]vo.ArticleSimple, bool, error) {
	articleList := make([]vo.ArticleSimple, 0)
	err := r.db.Model(models.Article{}).
		Select(`
			article.id, article.title, article.summary, article.cover,
			article.category_id, c.name AS category_name,
			article.view_count, article.like_count, article.comment_count,
			article.tags, article.created_at, article.updated_at
		`).
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Where("article.category_id = ?", categoryID).
		Order("article.created_at DESC, article.id DESC").
		Limit(pageSize + 1).
		Offset((page - 1) * pageSize).
		Scan(&articleList).Error
	if err != nil {
		return articleList, false, err
	}
	hasMore := len(articleList) > pageSize
	if hasMore {
		articleList = articleList[:pageSize]
	}
	return articleList, hasMore, nil
}

func (r *Repository) IncrementViewCount(id uint64) error {
	return r.db.Model(&models.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", r.db.Raw("view_count + ?", 1)).Error
}

func (r *Repository) AdminGetArticles(page int, pageSize int, keyword string, status int8) ([]models.Article, int64, error) {
	articles := make([]models.Article, 0)
	var total int64
	query := r.db.Model(&models.Article{}).Preload("Category")
	if keyword != "" {
		query = query.Where("title like ? or summary like ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return articles, total, err
	}
	err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&articles).Error
	return articles, total, err
}

func (r *Repository) GetArticleByID(id uint64) (models.Article, error) {
	var article models.Article
	err := r.db.Preload("Category").First(&article, id).Error
	return article, err
}

func (r *Repository) CreateArticle(article *models.Article) error {
	return r.db.Create(article).Error
}

func (r *Repository) UpdateArticle(article *models.Article) error {
	return r.db.Save(article).Error
}

func (r *Repository) DeleteArticle(id uint64) error {
	return r.db.Delete(&models.Article{}, id).Error
}

func (r *Repository) GetDrafts(page int, pageSize int) ([]models.Article, int64, error) {
	articles := make([]models.Article, 0)
	var total int64
	if err := r.db.Model(&models.Article{}).Preload("Category").Where("status = ?", 1).Count(&total).Error; err != nil {
		return articles, total, err
	}
	err := r.db.Preload("Category").Where("status = ?", 1).
		Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&articles).Error
	return articles, total, err
}

func (r *Repository) GetAllTags() ([]string, error) {
	var articles []models.Article
	if err := r.db.Select("tags").Where("status = ? AND tags != ''", 2).Find(&articles).Error; err != nil {
		return nil, err
	}
	tagSet := make(map[string]struct{})
	for _, a := range articles {
		for _, t := range strings.Split(a.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagSet[t] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(tagSet))
	for t := range tagSet {
		result = append(result, t)
	}
	return result, nil
}

func (r *Repository) IncrementLikeCount(articleID uint64) error {
	return r.db.Model(&models.Article{}).Where("id = ?", articleID).
		UpdateColumn("like_count", r.db.Raw("like_count + ?", 1)).Error
}

func (r *Repository) UpdateArticleCommentCount(articleID uint64, delta int) error {
	return r.db.Model(&models.Article{}).Where("id = ?", articleID).
		UpdateColumn("comment_count", r.db.Raw("comment_count + ?", delta)).Error
}

func (r *Repository) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *Repository) GetUserByWechatOpenID(openID string) (models.User, error) {
	var user models.User
	err := r.db.Where("wechat_open_id = ?", openID).First(&user).Error
	return user, err
}

func (r *Repository) GetUserByPhone(phone string) (models.User, error) {
	var user models.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	return user, err
}

func (r *Repository) GetUserByID(id uint64) (models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	return user, err
}

// GetUserStatus 查询指定用户状态；参数为用户 ID；返回状态值和数据库错误。
func (r *Repository) GetUserStatus(userID uint64) (uint64, error) {
	var status uint64
	err := r.db.Model(&models.User{}).Select("status").Where("id = ?", userID).Scan(&status).Error
	return status, err
}

func (r *Repository) UpdateUserPhone(id uint64, phone string) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("phone", phone).Error
}

// UpdateUserAvatar 更新指定用户头像；参数为用户 ID 和头像 URL；返回数据库更新错误。
func (r *Repository) UpdateUserAvatar(id uint64, avatar string) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("avatar", avatar).Error
}

func (r *Repository) CreateUser(user models.User) error {
	return r.db.Create(&user).Error
}

func (r *Repository) GetUsersByPage(page int, pageSize int, keyword string, status uint64) ([]models.User, int64, error) {
	users := make([]models.User, 0)
	var total int64
	query := r.db.Model(&models.User{})
	if keyword != "" {
		query = query.Where("email like ? OR nickname like ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return users, total, err
	}
	err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&users).Error
	return users, total, err
}

func (r *Repository) UpdateUserStatus(id uint64, status uint64) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) DeleteUserByID(id uint64) error {
	return r.db.Delete(&models.User{}, id).Error
}

func (r *Repository) LoginVerification(username, password string) (models.Admin, error) {
	var ad models.Admin
	if err := r.db.Where("username = ?", username).First(&ad).Error; err != nil {
		return ad, err
	}
	//临时应急
	if ad.Password != password {
		return models.Admin{}, errors.New("password error")
	}
	//if !utils.CheckPassword(ad.Password, password) {
	//	return models.Admin{}, errors.New("password error")
	//}
	return ad, nil
}

func (r *Repository) GetAllCategories() ([]models.Category, error) {
	categories := make([]models.Category, 0)
	err := r.db.Order("sort DESC").Find(&categories).Error
	return categories, err
}

// 管理端：获取分类列表（带文章数量）
func (r *Repository) AdminGetCategories(keyword string) ([]map[string]interface{}, error) {
	var categories []models.Category
	query := r.db.Model(&models.Category{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := query.Order("sort DESC").Find(&categories).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(categories))
	for _, cat := range categories {
		var count int64
		r.db.Model(&models.Article{}).Where("category_id = ?", cat.ID).Count(&count)
		result = append(result, map[string]interface{}{
			"id":            cat.ID,
			"name":          cat.Name,
			"description":   cat.Description,
			"cover":         cat.Cover,
			"sort":          cat.Sort,
			"article_count": count,
			"created_at":    cat.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":    cat.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func (r *Repository) GetCategoryByID(id uint64) (models.Category, error) {
	var cat models.Category
	err := r.db.First(&cat, id).Error
	return cat, err
}

// GetCategoryByName 根据分类名称查询分类；参数 name 为分类名称；返回分类记录和查询错误。
func (r *Repository) GetCategoryByName(name string) (models.Category, error) {
	var cat models.Category
	err := r.db.Where("name = ?", name).First(&cat).Error
	return cat, err
}

// GetMaxCategorySort 查询当前分类的最大排序值；无参数；返回最大排序值和查询错误。
func (r *Repository) GetMaxCategorySort() (int, error) {
	var maxSort int
	err := r.db.Model(&models.Category{}).Select("COALESCE(MAX(sort), 0)").Scan(&maxSort).Error
	return maxSort, err
}

func (r *Repository) CreateCategory(cat *models.Category) error {
	return r.db.Create(cat).Error
}

func (r *Repository) UpdateCategory(cat *models.Category) error {
	return r.db.Model(cat).Updates(map[string]interface{}{
		"name":        cat.Name,
		"description": cat.Description,
		"cover":       cat.Cover,
	}).Error
}

func (r *Repository) UpdateCategorySort(id uint64, sort int) error {
	return r.db.Model(&models.Category{}).Where("id = ?", id).Update("sort", sort).Error
}

func (r *Repository) BatchUpdateCategorySort(ids []uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&models.Category{}).Where("id = ?", id).Update("sort", len(ids)-i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) DeleteCategory(id uint64) error {
	return r.db.Delete(&models.Category{}, id).Error
}

// DeleteCategoryWithArticles 事务删除分类及其全部文章；参数 id 为分类 ID；返回事务执行错误。
func (r *Repository) DeleteCategoryWithArticles(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("category_id = ?", id).Delete(&models.Article{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Category{}, id).Error
	})
}

func (r *Repository) GetCategoryArticleCount(id uint64) (int64, error) {
	var count int64
	err := r.db.Model(&models.Article{}).Where("category_id = ?", id).Count(&count).Error
	return count, err
}

func (r *Repository) TransferArticles(fromID, toID uint64) error {
	return r.db.Model(&models.Article{}).Where("category_id = ?", fromID).Update("category_id", toID).Error
}

// TransferArticlesAndDeleteCategory 事务迁移分类文章并删除原分类；参数为原分类和目标分类 ID；返回事务执行错误。
func (r *Repository) TransferArticlesAndDeleteCategory(fromID, toID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Article{}).Where("category_id = ?", fromID).Update("category_id", toID).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Category{}, fromID).Error
	})
}

func (r *Repository) GetCategoryArticlesForAdmin(id uint64, page, pageSize int) ([]map[string]interface{}, int64, error) {
	var articles []models.Article
	var total int64
	query := r.db.Model(&models.Article{}).Where("category_id = ?", id)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&articles).Error; err != nil {
		return nil, 0, err
	}

	result := make([]map[string]interface{}, 0, len(articles))
	for _, a := range articles {
		pubTime := ""
		if a.PublishTime != nil {
			pubTime = a.PublishTime.Format("2006-01-02 15:04:05")
		}
		result = append(result, map[string]interface{}{
			"id":           a.ID,
			"title":        a.Title,
			"cover":        a.Cover,
			"publish_time": pubTime,
		})
	}
	return result, total, nil
}

func (r *Repository) GetOrCreateDefaultCategory() (models.Category, error) {
	var cat models.Category
	if err := r.db.Where("name = ?", "杂谈").First(&cat).Error; err == nil {
		return cat, nil
	}
	cat = models.Category{Name: "杂谈", Description: "未分类的杂谈文章", Cover: models.DefaultCategoryCover, Sort: 0}
	err := r.db.Create(&cat).Error
	return cat, err
}

func (r *Repository) CreateComment(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *Repository) GetCommentsByArticle(articleID uint64, page int, pageSize int) ([]vo.CommentVO, int64, error) {
	comments := make([]vo.CommentVO, 0)
	var total int64
	if err := r.db.Model(&models.Comment{}).Where("article_id = ? AND status = ?", articleID, 1).Count(&total).Error; err != nil {
		return comments, total, err
	}
	err := r.db.Model(&models.Comment{}).
		Select(`
			comment.id, comment.article_id, a.title AS article_title,
			comment.user_id, u.nickname, comment.content,
			comment.parent_id, comment.status, comment.created_at
		`).
		Joins("LEFT JOIN user u ON comment.user_id = u.id").
		Joins("LEFT JOIN article a ON comment.article_id = a.id").
		Where("comment.article_id = ? AND comment.status = ?", articleID, 1).
		Order("comment.created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&comments).Error
	return comments, total, err
}

func (r *Repository) GetAllComments(page int, pageSize int, keyword string, searchType string) ([]vo.CommentVO, int64, error) {
	comments := make([]vo.CommentVO, 0)
	var total int64
	query := r.db.Model(&models.Comment{}).
		Joins("LEFT JOIN user u ON comment.user_id = u.id").
		Joins("LEFT JOIN article a ON comment.article_id = a.id")
	if keyword != "" {
		if searchType == "nickname" {
			query = query.Where("u.nickname like ?", "%"+keyword+"%")
		} else {
			query = query.Where("comment.content like ?", "%"+keyword+"%")
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return comments, total, err
	}
	err := query.Select(`
			comment.id, comment.article_id, a.title AS article_title,
			comment.user_id, u.nickname, comment.content,
			comment.parent_id, comment.status, comment.created_at
		`).
		Order("comment.created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&comments).Error
	return comments, total, err
}

func (r *Repository) GetPendingComments(page int, pageSize int) ([]vo.CommentVO, int64, error) {
	comments := make([]vo.CommentVO, 0)
	var total int64
	if err := r.db.Model(&models.Comment{}).Where("status = ?", 3).Count(&total).Error; err != nil {
		return comments, total, err
	}
	err := r.db.Model(&models.Comment{}).
		Select(`
			comment.id, comment.article_id, a.title AS article_title,
			comment.user_id, u.nickname, comment.content,
			comment.parent_id, comment.status, comment.created_at
		`).
		Joins("LEFT JOIN user u ON comment.user_id = u.id").
		Joins("LEFT JOIN article a ON comment.article_id = a.id").
		Where("comment.status = ?", 3).
		Order("comment.created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&comments).Error
	return comments, total, err
}

func (r *Repository) UpdateCommentStatus(id uint64, status int8) error {
	return r.db.Model(&models.Comment{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) GetCommentByID(id uint64) (models.Comment, error) {
	var comment models.Comment
	err := r.db.First(&comment, id).Error
	return comment, err
}

func (r *Repository) DeleteComment(id uint64) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

func (r *Repository) GetDashboard() (vo.DashboardData, error) {
	var data vo.DashboardData
	if err := r.db.Model(&models.Article{}).Count(&data.TotalArticles).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Article{}).Where("status = ?", 2).Count(&data.PublishedArticles).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Article{}).Where("status = ?", 1).Count(&data.DraftArticles).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Comment{}).Count(&data.TotalComments).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Comment{}).Where("status = ?", 3).Count(&data.PendingComments).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.User{}).Count(&data.TotalUsers).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Comment{}).Where("status = ?", 2).Count(&data.HiddenComments).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Category{}).Count(&data.TotalCategories).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Article{}).Where("category_id = ?", 0).Count(&data.UncategorizedArticles).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Article{}).Select("COALESCE(SUM(view_count), 0)").Scan(&data.TotalViews).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Article{}).Select("COALESCE(SUM(like_count), 0)").Scan(&data.TotalLikes).Error; err != nil {
		return data, err
	}

	start := dashboardStartOfDay(time.Now(), 14)
	if err := r.db.Model(&models.Article{}).Where("created_at >= ?", start).Count(&data.NewArticles).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.User{}).Where("created_at >= ?", start).Count(&data.NewUsers).Error; err != nil {
		return data, err
	}
	if err := r.db.Model(&models.Comment{}).Where("created_at >= ?", start).Count(&data.NewComments).Error; err != nil {
		return data, err
	}

	articleCounts, err := r.dashboardDateCounts(&models.Article{}, start)
	if err != nil {
		return data, err
	}
	userCounts, err := r.dashboardDateCounts(&models.User{}, start)
	if err != nil {
		return data, err
	}
	commentCounts, err := r.dashboardDateCounts(&models.Comment{}, start)
	if err != nil {
		return data, err
	}
	data.Trend = buildDashboardTrend(start, 14, articleCounts, userCounts, commentCounts)

	data.TopArticles = make([]vo.DashboardArticleMetric, 0)
	if err := r.db.Model(&models.Article{}).
		Select("article.id, article.title, c.name AS category_name, article.view_count, article.like_count, article.comment_count").
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Where("article.status = ?", 2).
		Order("article.view_count DESC, article.like_count DESC, article.comment_count DESC, article.id DESC").
		Limit(5).
		Scan(&data.TopArticles).Error; err != nil {
		return data, err
	}

	data.Categories = make([]vo.DashboardCategoryMetric, 0)
	if err := r.db.Model(&models.Category{}).
		Select("category.id, category.name, COUNT(article.id) AS article_count, COALESCE(SUM(article.view_count), 0) AS view_count").
		Joins("LEFT JOIN article ON article.category_id = category.id").
		Group("category.id, category.name, category.sort").
		Order("article_count DESC, category.sort DESC, category.id DESC").
		Limit(6).
		Scan(&data.Categories).Error; err != nil {
		return data, err
	}

	data.RecentArticles = make([]vo.DashboardRecentArticle, 0)
	if err := r.db.Model(&models.Article{}).
		Select("article.id, article.title, c.name AS category_name, article.status, article.created_at").
		Joins("LEFT JOIN category c ON article.category_id = c.id").
		Order("article.created_at DESC, article.id DESC").
		Limit(5).
		Scan(&data.RecentArticles).Error; err != nil {
		return data, err
	}

	data.RecentComments = make([]vo.DashboardRecentComment, 0)
	if err := r.db.Model(&models.Comment{}).
		Select("comment.id, comment.article_id, article.title AS article_title, user.nickname, comment.content, comment.status, comment.created_at").
		Joins("LEFT JOIN article ON comment.article_id = article.id").
		Joins("LEFT JOIN user ON comment.user_id = user.id").
		Order("comment.created_at DESC, comment.id DESC").
		Limit(5).
		Scan(&data.RecentComments).Error; err != nil {
		return data, err
	}

	data.RecentUsers = make([]vo.DashboardRecentUser, 0)
	if err := r.db.Model(&models.User{}).
		Select("id, nickname, email, status, created_at").
		Order("created_at DESC, id DESC").
		Limit(5).
		Scan(&data.RecentUsers).Error; err != nil {
		return data, err
	}
	return data, nil
}

type dashboardDateCount struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

// dashboardDateCounts 按日期汇总指定模型的新增记录；参数为 GORM 模型和起始时间；返回日期到数量的映射及查询错误。
func (r *Repository) dashboardDateCounts(model interface{}, start time.Time) (map[string]int64, error) {
	counts := make([]dashboardDateCount, 0)
	if err := r.db.Model(model).
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("created_at >= ?", start).
		Group("DATE(created_at)").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(counts))
	for _, count := range counts {
		result[count.Date] = count.Count
	}
	return result, nil
}

// dashboardStartOfDay 计算趋势统计的起始自然日；参数为当前时间和统计天数；返回本地时区零点时间。
func dashboardStartOfDay(now time.Time, days int) time.Time {
	localNow := now.In(time.Local)
	return time.Date(localNow.Year(), localNow.Month(), localNow.Day()-days+1, 0, 0, 0, 0, time.Local)
}

// buildDashboardTrend 将各类日期统计补齐为连续趋势；参数为起始日期、天数及三类统计映射；返回完整趋势序列。
func buildDashboardTrend(start time.Time, days int, articles, users, comments map[string]int64) []vo.DashboardTrendPoint {
	trend := make([]vo.DashboardTrendPoint, 0, days)
	for offset := 0; offset < days; offset++ {
		date := start.AddDate(0, 0, offset).Format("2006-01-02")
		trend = append(trend, vo.DashboardTrendPoint{
			Date:     date,
			Articles: articles[date],
			Users:    users[date],
			Comments: comments[date],
		})
	}
	return trend
}

func (r *Repository) SaveToken(userID uint64, token string) error {
	tk := models.Token{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	return r.db.Where("user_id = ?", userID).Assign(tk).FirstOrCreate(&tk).Error
}
