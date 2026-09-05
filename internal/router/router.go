package router

import (
	"blog/internal/app"
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Register(container *app.Container) *gin.Engine {
	h := container.Handler
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Static("/uploads", "./uploads")

	// 健康检查接口（不依赖任何中间件和数据库）
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// RSS 2.0 订阅源
	r.GET("/feed.xml", h.User.FeedXML)

	api := r.Group("/api")
	{
		public := api.Group("")
		public.GET("/articles", h.User.GetArticles)
		public.GET("/articles/ranking", h.User.GetArticleRanking)
		public.GET("/articles/detail", h.User.GetArticle)
		public.GET("/articles/search/results", h.User.SearchArticleResults)
		public.GET("/articles/search/suggestions", h.User.SearchArticleSuggestions)
		public.GET("/articles/search", h.User.SearchArticle)
		public.GET("/categories", h.User.GetCategories)
		public.GET("/categories/articles", h.User.GetCategoryArticles)
		public.GET("/categories/articles/page", h.User.GetCategoryArticlesPage)
		public.GET("/comments", h.User.GetComments)
		public.POST("/register/code", h.User.SendRegisterCode)
		public.POST("/register", h.User.Register)
		public.POST("/login", h.User.Login)
		// 1. 生成验证码
		public.GET("/captcha", h.User.Captcha)
		//微信登陆
		public.POST("/wechat/login", h.User.WechatLogin)
		//根据授权获取手机号
		public.POST("/wechat/phone", h.User.CompleteWechatPhoneLogin)
		public.POST("/articles/like", h.User.LikeArticle)
		public.GET("/tags", h.User.GetTags)
		public.GET("/tags/cloud", h.User.GetTagCloud)
		public.GET("/tags/articles", h.User.GetArticlesByTag)
		public.GET("/articles/archive", h.User.GetArchive)
		public.GET("/articles/related", h.User.GetRelatedArticles)
		public.GET("/links", h.User.GetLinks)
		public.GET("/settings/site", h.User.GetSiteSettings)
		public.POST("/sendpwdcode", h.User.SendCodeForgetPwd)
		public.POST("/updatePasswordByCode", h.User.UpdatePasswordByCode)
		//refreshToken刷新
		public.POST("/token/refresh", h.User.TokenRefresh)
	}
	{
		apiAuth := api.Group("")
		apiAuth.Use(middleware.JWTAuth())
		apiAuth.GET("/users/me", h.User.UsersMe)
		apiAuth.POST("/users/avatar/upload", h.User.UploadAvatar)
		apiAuth.PUT("/users/avatar", h.User.UpdateAvatar)
		apiAuth.POST("/comments", h.User.CreateComment)
		apiAuth.POST("/updatephonenumber", h.User.UpdatePhoneNumber)
	}

	adminGroup := r.Group("/api/admin")
	adminGroup.POST("/login", h.Admin.Login)

	adminAuth := adminGroup.Group("")
	adminAuth.Use(middleware.JWTAuthForAdmin())
	adminAuth.GET("/dashboard", h.Admin.GetDashboard)
	adminAuth.GET("/settings/site", h.Admin.GetSiteSettings)
	adminAuth.PUT("/settings/site", h.Admin.UpdateSiteSettings)
	adminAuth.GET("/articles", h.Admin.GetArticles)
	adminAuth.GET("/articles/:id", h.Admin.GetArticle)
	adminAuth.POST("/articles", h.Admin.CreateArticle)
	adminAuth.PUT("/articles/:id", h.Admin.UpdateArticle)
	adminAuth.DELETE("/articles/:id", h.Admin.DeleteArticle)
	adminAuth.GET("/drafts", h.Admin.GetDrafts)
	adminAuth.PUT("/articles/:id/publish", h.Admin.PublishArticle)
	adminAuth.POST("/upload", h.Admin.UploadImage)
	adminAuth.GET("/comments", h.Admin.GetAllComments)
	adminAuth.GET("/comments/pending", h.Admin.GetPendingComments)
	adminAuth.PUT("/comments/:id/approve", h.Admin.ApproveComment)
	adminAuth.PUT("/comments/:id/reject", h.Admin.RejectComment)
	adminAuth.DELETE("/comments/:id", h.Admin.DeleteComment)
	adminAuth.PUT("/comments/:id/status", h.Admin.SetCommentStatus)
	adminAuth.GET("/users", h.Admin.GetUsers)
	adminAuth.PUT("/users/:id/ban", h.Admin.BanUser)
	adminAuth.PUT("/users/:id/unban", h.Admin.UnbanUser)
	adminAuth.DELETE("/users/:id", h.Admin.DeleteUser)

	// 分类管理
	adminAuth.GET("/categories", h.Admin.GetCategories)
	adminAuth.POST("/categories", h.Admin.CreateCategory)
	adminAuth.PUT("/categories/:id", h.Admin.UpdateCategory)
	adminAuth.PUT("/categories/:id/sort", h.Admin.UpdateCategorySort)
	adminAuth.PUT("/categories/sort", h.Admin.BatchUpdateCategorySort)
	adminAuth.DELETE("/categories/:id", h.Admin.DeleteCategory)
	adminAuth.GET("/categories/:id/articles", h.Admin.GetCategoryArticles)
	adminAuth.GET("/categories/:id/article-count", h.Admin.GetCategoryArticleCount)
	adminAuth.POST("/categories/transfer", h.Admin.TransferArticles)

	// 友链管理
	adminAuth.GET("/friend-links", h.Admin.AdminListFriendLinks)
	adminAuth.POST("/friend-links", h.Admin.AdminCreateFriendLink)
	adminAuth.PUT("/friend-links/:id", h.Admin.AdminUpdateFriendLink)
	adminAuth.DELETE("/friend-links/:id", h.Admin.AdminDeleteFriendLink)

	return r
}
