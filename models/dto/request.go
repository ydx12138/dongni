package dto

/*
	type Articles struct {
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
		Sort     string `json:"sort"` //默认日期排序，也可按浏览，点赞，评论数排序
		Keyword  string `json:"keywords"`
	}
*/
type PageQuery struct {
	Page int `form:"page" binding:"required,gte=1"`
}

type ArticleKeyWord struct {
	Keyword string `form:"keyword" binding:"required"`
}

type AdminLogin struct {
	Username    string `form:"username" json:"username" binding:"required"`
	Password    string `form:"password" json:"password" binding:"required"`
	CaptchaID   string `form:"captcha_id" json:"captcha_id" binding:"required"`
	CaptchaCode string `form:"captcha_code" json:"captcha_code" binding:"required,len=4"`
}

type UserRegister struct {
	Email      string `form:"email" json:"email" binding:"required,email"`
	Password   string `form:"password" json:"password" binding:"required,min=6,max=10"`
	Repassword string `form:"re_password" json:"re_password" binding:"required,min=6,max=10,eqfield=Password"`
	Nickname   string `form:"nickname" json:"nickname"`
	Code       string `form:"code" json:"code" binding:"required,len=6"`
}

type SendRegisterCodeReq struct {
	Email string `form:"email" json:"email" binding:"required,email"`
}

type UserLogin struct {
	Email       string `form:"email" json:"email" binding:"required,email"`
	Password    string `form:"password" json:"password" binding:"required"`
	CaptchaID   string `form:"captcha_id" json:"captcha_id" binding:"required"`
	CaptchaCode string `form:"captcha_code" json:"captcha_code" binding:"required,len=4"`
}

type WechatLoginReq struct {
	Code string `json:"code" binding:"required"`
}

type WechatPhoneReq struct {
	PhoneTicket string `json:"phone_ticket" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

type CreateCommentReq struct {
	ArticleID uint64 `form:"article_id" json:"article_id" binding:"required"`
	Content   string `form:"content" json:"content" binding:"required"`
	ParentID  uint64 `form:"parent_id" json:"parent_id"`
}

type CreateArticleReq struct {
	Title       string `json:"title" binding:"required"`
	Summary     string `json:"summary"`
	Content     string `json:"content"`
	ContentType int8   `json:"content_type"`
	Cover       string `json:"cover" binding:"required"`
	CategoryID  uint64 `json:"category_id"`
	Tags        string `json:"tags"`
	Status      int8   `json:"status"`
}

type UpdateArticleReq struct {
	ID          uint64  `json:"id"`
	Title       string  `json:"title" binding:"required"`
	Summary     string  `json:"summary"`
	Content     string  `json:"content"`
	ContentType int8    `json:"content_type"`
	Cover       string  `json:"cover"`
	CategoryID  uint64  `json:"category_id"`
	Tags        string  `json:"tags"`
	Status      int8    `json:"status"`
	ViewCount   *uint64 `json:"view_count"`
	LikeCount   *uint64 `json:"like_count"`
}

type PageQueryWithSize struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type CategoryArticlesQuery struct {
	CategoryID uint64 `form:"category_id" binding:"required"`
	Page       int    `form:"page"`
}

// CategoryArticlesPageQuery 接收分类文章无限加载参数；参数为分类 ID、页码和单批数量；返回绑定后的查询条件。
type CategoryArticlesPageQuery struct {
	CategoryID uint64 `form:"category_id" binding:"required"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type CommentListQuery struct {
	ArticleID uint64 `form:"article_id" binding:"required"`
	Page      int    `form:"page"`
}

type AdminArticleQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Status   int8   `form:"status"`
}

type AdminCommentQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type UserStatusReq struct {
	ID     uint64 `json:"id" binding:"required"`
	Status uint64 `json:"status" binding:"required"`
}

type IDReq struct {
	ID uint64 `form:"id" uri:"id" json:"id" binding:"required"`
}

// ArticleLikeReq 点赞请求
type ArticleLikeReq struct {
	ArticleID uint64 `json:"article_id" binding:"required"`
}

type UserUpdatePassword struct {
	Email      string `form:"email" json:"email" binding:"required,email"`
	Password   string `form:"password" json:"password" binding:"required,min=6,max=10"`
	RePassword string `form:"re_password" json:"re_password" binding:"required,min=6,max=10,eqfield=Password"`
	Code       string `form:"code" json:"code" binding:"required,len=6"`
}

type UserPutPhone struct {
	Email            string `form:"email" json:"email" binding:"required,email"`
	Phone            string `form:"phone" json:"phone" binding:"required"`
	VerificationCode string `form:"verification_code" json:"verification_code" binding:"required"`
}

// UpdateUserAvatarRequest 接收当前用户头像更新请求；参数为已上传的头像 URL；返回值由 Handler 统一响应。
type UpdateUserAvatarRequest struct {
	Avatar string `json:"avatar" binding:"required"`
}

// 分类管理
type CreateCategoryReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
	Sort        int    `json:"sort"`
}

type UpdateCategoryReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
}

type UpdateCategorySortReq struct {
	Sort int `json:"sort" binding:"required"`
}

type BatchUpdateSortReq struct {
	Ids []uint64 `json:"ids" binding:"required"`
}

// DeleteCategoryReq 描述删除分类时的安全确认与文章迁移参数；无；返回请求绑定结果。
type DeleteCategoryReq struct {
	Force            bool   `json:"force"`
	ConfirmText      string `json:"confirm_text"`
	TargetCategoryID uint64 `json:"target_category_id"`
}

type TransferArticlesReq struct {
	FromCategoryID uint64 `json:"from_category_id" binding:"required"`
	ToCategoryID   uint64 `json:"to_category_id" binding:"required"`
}
