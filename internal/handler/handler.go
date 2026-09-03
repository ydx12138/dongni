package handler

import (
	"blog/config"
	"blog/internal/service"
	"blog/internal/utils"
	"blog/models/dto"
	"blog/models/vo"
	"blog/pkg/code"
	"blog/pkg/response"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler struct {
	User  *UserHandler
	Admin *AdminHandler
}

// 其它地方访问不到svc未导出字段，只有结构体方法能够访问到svc，从而访问到service层方法
type UserHandler struct {
	svc *service.Service
}

type AdminHandler struct {
	svc *service.Service
}

func (h *UserHandler) Captcha(c *gin.Context) {
	var store = base64Captcha.DefaultMemStore
	// 验证码配置
	var driver = &base64Captcha.DriverString{
		Height:          80,
		Width:           200,
		NoiseCount:      0,                                      // 干扰点数量
		ShowLineOptions: 2,                                      // 干扰线数量（2 表示中等）
		Length:          4,                                      // 验证码长度
		Source:          "1234567890qwertyuioplkjhgfdsazxcvbnm", // 字符集
		Fonts:           nil,                                    // 使用默认字体
	}

	// 生成验证码
	captcha := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "生成验证码失败"})
		return
	}
	response.SuccessWithData(vo.CaptchaResponse{
		CaptchaId: id,
		PicBase64: b64s,
	}, c)
}

// GetSiteSettings 获取前台站点配置；参数为 Gin 请求上下文，返回值通过统一响应写入配置或错误信息。
func (h *UserHandler) GetSiteSettings(c *gin.Context) {
	c.Request.Context()
	settings, err := h.svc.GetSiteSettings()
	if err != nil {
		zap.L().Error("GetSiteSettings: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(settings, c)
}

// GetSiteSettings 获取管理端站点配置；参数为 Gin 请求上下文，返回值通过统一响应写入配置或错误信息。
func (h *AdminHandler) GetSiteSettings(c *gin.Context) {
	settings, err := h.svc.GetAdminSiteSettings()
	if err != nil {
		zap.L().Error("AdminGetSiteSettings: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(settings, c)
}

// UpdateSiteSettings 更新管理端站点配置；参数为 Gin 请求上下文中的配置 JSON，返回值通过统一响应写入保存结果。
func (h *AdminHandler) UpdateSiteSettings(c *gin.Context) {
	var req dto.UpdateSiteSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.UpdateSiteSettings(req); err != nil {
		if errors.Is(err, service.ErrInvalidSiteSetting) {
			response.ErrWithMsg(code.BadRequest, c)
			return
		}
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("站点配置保存成功", c)
}

func New(svc *service.Service) *Handler {
	return &Handler{
		User:  &UserHandler{svc: svc},
		Admin: &AdminHandler{svc: svc},
	}
}

// 刷新token
func (h *UserHandler) TokenRefresh(c *gin.Context) {
	//从context中得到token
	token := utils.GetTokenFromContext(c)
	if token == "" {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//解析token，得到Data
	data, err := utils.GetDataFromToken(token)
	if data == nil || err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//解析token，得到claim
	var claim *utils.CustomClaims
	if claim = utils.GetClaimFromData(data); claim == nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if claim.SessionID == "" {
		response.ErrWithMsg(code.SessionReplaced, c)
		return
	}
	if err := h.svc.ValidateSession(claim.UserID, claim.SessionID); err != nil {
		if errors.Is(err, service.ErrSessionInvalid) {
			response.ErrWithMsg(code.SessionReplaced, c)
		} else {
			zap.L().Error("validate pc login session failed: " + err.Error())
			response.ErrWithMsg(code.InternalError, c)
		}
		return
	}
	active, err := h.svc.IsUserActive(claim.UserID)
	if err != nil {
		zap.L().Error("check refresh token user status failed: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	if !active {
		response.ErrWithMsg(code.UserBanned, c)
		return
	}
	tokenDuration, err := h.svc.UserTokenDuration()
	if err != nil {
		zap.L().Error("read user token duration failed: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	//如果type==refresh,有效，且redis里存在，则创建新accessToken，Abort+return
	if claim.Type == "refresh" && data.Valid && h.svc.RefreshTokenIsExist(strconv.FormatUint(claim.UserID, 10)) == true {
		accessToken, err := utils.GenerateUserTokenWithSession(claim.UserID, tokenDuration, "access", claim.SessionID)
		if err != nil {
			zap.L().Error("generate access token failed" + err.Error())
			response.ErrWithMsg(code.InternalError, c)
			return
		}

		response.SuccessWithData(map[string]interface{}{"access_token": accessToken}, c)
		return
	}
	//如果type==refresh,无效，或者redis里不存在，则401，Abort+return
	if claim.Type == "refresh" && (data.Valid == false && h.svc.RefreshTokenIsExist(claim.ID) == false) {
		response.ErrWithMsg(code.RefreshTokenExpired, c)
		return
	}
}

// 主要：验证token过期否
func (h *UserHandler) UsersMe(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}

	profile, err := h.svc.CurrentUserProfile(userID)
	if err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrUserNotFound, c)
			return
		}
		zap.L().Error("UsersMe:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}

	response.SuccessWithData(profile, c)
}

// UploadAvatar 上传当前用户裁剪后的头像；参数来自 multipart 文件和登录上下文；返回 OSS 图片 URL。
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			zap.L().Error("UploadAvatar close file: " + err.Error())
		}
	}()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] || header.Size > 10*1024*1024 {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	headerBytes := make([]byte, 512)
	readSize, err := file.Read(headerBytes)
	if err != nil || readSize == 0 {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	contentType := http.DetectContentType(headerBytes[:readSize])
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" && contentType != "image/webp" {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	filename := fmt.Sprintf("user_%d_avatar_%d%s", userID, time.Now().UnixNano(), ext)
	avatarURL, err := utils.UploadToOss(file, config.Cfg.OssConfig.Image_path, filename)
	if err != nil {
		zap.L().Error("UploadAvatar: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]string{"url": avatarURL}, c)
}

// UpdateAvatar 保存当前用户头像地址；参数来自 JSON 和登录上下文；返回更新结果。
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}
	var req dto.UpdateUserAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.UpdateCurrentUserAvatar(userID, req.Avatar); err != nil {
		if errors.Is(err, service.ErrInvalidUserAvatar) {
			response.ErrWithMsg(code.BadRequest, c)
			return
		}
		zap.L().Error("UpdateAvatar: " + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]string{"avatar": strings.TrimSpace(req.Avatar)}, c)
}

// 修改手机号
func (h *UserHandler) UpdatePhoneNumber(c *gin.Context) {
	var q dto.UserPutPhone
	if err := c.ShouldBind(&q); err != nil {
		zap.L().Error("ForgetPassword" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.UpdatePhoneNumber(q.Email, q.Phone, q.VerificationCode); err != nil {

	}
	response.ErrWithMsg(code.Success, c)
}

// 重置密码--1.发验证码
func (h *UserHandler) SendCodeForgetPwd(c *gin.Context) {
	var q dto.SendRegisterCodeReq
	err := c.ShouldBind(&q)
	if err != nil {
		zap.L().Error("ForgetPassword" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//发验证码
	if err = h.svc.SendCodeForgetPwd(q.Email); err != nil {
		zap.L().Error("ForgetPassword:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("验证码发送成功", c)
}

// 重置密码--2.修改密码
func (h *UserHandler) UpdatePasswordByCode(c *gin.Context) {
	//参数
	var q dto.UserUpdatePassword
	err := c.ShouldBind(&q)
	if err != nil {
		zap.L().Error("UpdatePasswordByCode" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//修改密码
	if err = h.svc.UpdatePasswordByCode(q.Email, q.Password, q.Code); err != nil {
		zap.L().Error("UpdatePasswordByCode:" + err.Error())
		response.ErrWithMsg(code.ErrorMsg(err), c)
		return
	}
	response.SuccessWithMsg("密码重置成功", c)
}

func (h *UserHandler) GetArticles(c *gin.Context) {
	var q dto.PageQueryWithSize
	if err := c.ShouldBindQuery(&q); err != nil {
		q.Page = 1
	}
	articles, total, err := h.svc.GetArticles(q.Page, q.PageSize)
	if err != nil {
		zap.L().Error("GetArticles:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": articles, "total": total}, c)
}

// GetArticleRanking 返回点赞数最高的文章；接收可选 limit 查询参数；响应文章摘要列表或统一错误信息。
func (h *UserHandler) GetArticleRanking(c *gin.Context) {
	var q struct {
		Limit int `form:"limit"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	articles, err := h.svc.GetArticleRanking(q.Limit)
	if err != nil {
		zap.L().Error("GetArticleRanking:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(articles, c)
}

func (h *UserHandler) GetArticle(c *gin.Context) {
	var q struct {
		ID uint64 `form:"id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	article, err := h.svc.GetArticle(q.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrArticleNotFound, c)
			return
		}
		zap.L().Error("GetArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(article, c)
}

func (h *UserHandler) SearchArticle(c *gin.Context) {
	var q dto.ArticleKeyWord
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	articles, err := h.svc.SearchArticle(q.Keyword)
	if err != nil {
		zap.L().Error("SearchArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(articles, c)
}

// SearchArticleResults 搜索独立结果页文章；参数为包含 keyword 查询参数的 Gin 上下文，返回文章结果或统一错误响应。
func (h *UserHandler) SearchArticleResults(c *gin.Context) {
	var q dto.ArticleKeyWord
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	articles, err := h.svc.SearchArticleResults(q.Keyword)
	if err != nil {
		zap.L().Error("SearchArticleResults:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(articles, c)
}

// SearchArticleSuggestions 获取搜索框真实建议文章；参数为包含 keyword 查询参数的 Gin 上下文，返回最多十条建议或统一错误响应。
func (h *UserHandler) SearchArticleSuggestions(c *gin.Context) {
	var q dto.ArticleKeyWord
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	articles, err := h.svc.SearchArticleSuggestions(q.Keyword)
	if err != nil {
		zap.L().Error("SearchArticleSuggestions:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(articles, c)
}

// 注册
func (h *UserHandler) Register(c *gin.Context) {
	//参数
	var req dto.UserRegister
	if err := c.ShouldBind(&req); err != nil {
		zap.L().Error("Register:" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//注册
	if err := h.svc.Register(req); err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		//用户已存在
		if errors.Is(err, service.ErrUserExists) {
			response.ErrWithMsg(code.ErrUserExist, c)
			return
		}
		//验证码无效
		if errors.Is(err, service.ErrVerificationCode) {
			response.ErrWithMsg(code.ErrVerificationCode, c)
			return
		}
		//其它原因
		zap.L().Error("Register:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("注册成功", c)
}

// 发送验证码
func (h *UserHandler) SendRegisterCode(c *gin.Context) {
	var req dto.SendRegisterCodeReq
	if err := c.ShouldBind(&req); err != nil {
		zap.L().Error("SendRegisterCode:" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//
	if err := h.svc.SendRegisterCode(req); err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		if errors.Is(err, service.ErrUserExists) {
			response.ErrWithMsg(code.ErrUserExist, c)
			return
		}
		zap.L().Error("SendRegisterCode:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("验证码已发送", c)
}

// 登录
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.UserLogin
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if !base64Captcha.DefaultMemStore.Verify(req.CaptchaID, req.CaptchaCode, true) {
		response.ErrWithMsg(code.ErrCaptcha, c)
		return
	}
	//登录
	data, err := h.svc.UserLogin(req)
	if err != nil {
		switch {
		//用户不存在
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.ErrWithMsg(code.ErrUserNotFound, c)
		//密码错误
		case errors.Is(err, service.ErrPassword):
			response.ErrWithMsg(code.ErrPassword, c)
		//用户被禁用
		case errors.Is(err, service.ErrUserDisabled):
			response.ErrWithMsg(code.Forbidden, c)
		default:
			//服务器错误
			zap.L().Error("Login:" + err.Error())
			response.ErrWithMsg(code.InternalError, c)
		}
		return
	}
	response.SuccessWithData(data, c)
}

// 微信登陆
func (h *UserHandler) WechatLogin(c *gin.Context) {
	//获取前端传来的code参数
	var req dto.WechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err)
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//调用微信登录函数，用code取交换open_id
	data, err := h.svc.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		switch {
		//用户被禁用
		case errors.Is(err, service.ErrUserDisabled):
			response.ErrWithMsg(code.Forbidden, c)
		//微信不可用
		case errors.Is(err, service.ErrWechatUnavailable):
			response.ErrWithMsg(code.InternalError, c)
		default:
			fmt.Println(err)
			response.ErrWithMsg(code.BadRequest, c)
		}
		return
	}
	response.SuccessWithData(data, c)
}

func (h *UserHandler) CompleteWechatPhoneLogin(c *gin.Context) {
	var req dto.WechatPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	data, err := h.svc.CompleteWechatPhoneLogin(c.Request.Context(), req.PhoneTicket, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserDisabled):
			response.ErrWithMsg(code.Forbidden, c)
		case errors.Is(err, service.ErrPhoneTicket), errors.Is(err, service.ErrPhoneAlreadyBound):
			response.ErrWithMsg(code.BadRequest, c)
		case errors.Is(err, service.ErrWechatUnavailable):
			response.ErrWithMsg(code.InternalError, c)
		default:
			response.ErrWithMsg(code.BadRequest, c)
		}
		return
	}
	response.SuccessWithData(data, c)
}

func (h *UserHandler) GetCategories(c *gin.Context) {
	categories, err := h.svc.GetCategories()
	if err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		zap.L().Error("GetCategories:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(categories, c)
}

func (h *UserHandler) GetCategoryArticles(c *gin.Context) {
	var q dto.CategoryArticlesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	articles, err := h.svc.GetCategoryArticles(q.CategoryID, q.Page)
	if err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		zap.L().Error("GetCategoryArticles:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(articles, c)
}

// GetCategoryArticlesPage 处理分类文章无限加载请求；参数来自查询字符串；返回当前批次文章和是否还有下一批。
func (h *UserHandler) GetCategoryArticlesPage(c *gin.Context) {
	var q dto.CategoryArticlesPageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	data, err := h.svc.GetCategoryArticlesPage(q.CategoryID, q.Page, q.PageSize)
	if err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		zap.L().Error("GetCategoryArticlesPage:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(data, c)
}

func (h *UserHandler) GetTags(c *gin.Context) {
	tags, err := h.svc.GetTags()
	if err != nil {
		zap.L().Error("GetTags:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(tags, c)
}

func (h *UserHandler) LikeArticle(c *gin.Context) {
	var req dto.ArticleLikeReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.LikeArticle(req.ArticleID); err != nil {
		zap.L().Error("LikeArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("点赞成功", c)
}

func (h *UserHandler) GetComments(c *gin.Context) {
	var q dto.CommentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	comments, total, err := h.svc.GetComments(q.ArticleID, q.Page)
	if err != nil {
		zap.L().Error("GetComments:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": comments, "total": total}, c)
}

func (h *UserHandler) CreateComment(c *gin.Context) {
	var req dto.CreateCommentReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	// 获取userid
	userID, ok := c.Get("userID")
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}
	uid, ok := userID.(uint64)
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}
	// 保存评论
	if err := h.svc.CreateComment(req, uid); err != nil {
		if errors.Is(err, service.ErrFeatureDisabled) {
			response.ErrWithMsg(code.FeatureDisabled, c)
			return
		}
		zap.L().Error("CreateComment:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("评论成功", c)
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req dto.AdminLogin
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if !base64Captcha.DefaultMemStore.Verify(req.CaptchaID, req.CaptchaCode, true) {
		response.ErrWithMsg(code.ErrCaptcha, c)
		return
	}
	data, err := h.svc.AdminLogin(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrUserNotFound, c)
		} else if err.Error() == "password error" {
			response.ErrWithMsg(code.ErrPassword, c)
		} else {
			zap.L().Error("AdminLogin:" + err.Error())
			response.ErrWithMsg(code.InternalError, c)
		}
		return
	}
	response.SuccessWithData(data, c)
}

// 获取仪表盘
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	data, err := h.svc.Dashboard()
	if err != nil {
		zap.L().Error("GetDashboard:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(data, c)
}

// 获取文章列表
func (h *AdminHandler) GetArticles(c *gin.Context) {
	var q dto.AdminArticleQuery
	_ = c.ShouldBindQuery(&q)
	articles, total, err := h.svc.AdminArticles(q)
	if err != nil {
		zap.L().Error("AdminGetArticles:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": articles, "total": total}, c)
}

func (h *AdminHandler) GetArticle(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	article, err := h.svc.AdminArticle(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrArticleNotFound, c)
			return
		}
		zap.L().Error("AdminGetArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(article, c)
}

func (h *AdminHandler) CreateArticle(c *gin.Context) {
	var req dto.CreateArticleReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	authorID, ok := currentUserID(c)
	if !ok {
		response.ErrWithMsg(code.Unauthorized, c)
		return
	}
	if err := h.svc.CreateArticle(req, authorID); err != nil {
		zap.L().Error("CreateArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("创建成功", c)
}

func (h *AdminHandler) UpdateArticle(c *gin.Context) {
	var req dto.UpdateArticleReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if req.ID == 0 {
		id, err := parseParamID(c)
		if err != nil {
			response.ErrWithMsg(code.BadRequest, c)
			return
		}
		req.ID = id
	}
	if err := h.svc.UpdateArticle(req); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrArticleNotFound, c)
			return
		}
		zap.L().Error("UpdateArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("更新成功", c)
}

func (h *AdminHandler) DeleteArticle(c *gin.Context) {
	id, err := idFromRequest(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.DeleteArticle(id); err != nil {
		zap.L().Error("DeleteArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("删除成功", c)
}

func (h *AdminHandler) GetDrafts(c *gin.Context) {
	var q dto.AdminArticleQuery
	_ = c.ShouldBindQuery(&q)
	articles, total, err := h.svc.Drafts(q)
	if err != nil {
		zap.L().Error("GetDrafts:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": articles, "total": total}, c)
}

func (h *AdminHandler) PublishArticle(c *gin.Context) {
	id, err := idFromRequest(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.PublishArticle(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrArticleNotFound, c)
			return
		}
		zap.L().Error("PublishArticle:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("发布成功", c)
}

func (h *AdminHandler) GetAllComments(c *gin.Context) {
	var q dto.AdminCommentQuery
	_ = c.ShouldBindQuery(&q)
	comments, total, err := h.svc.AllComments(q.Page, q.PageSize, c.Query("keyword"), c.Query("type"))
	if err != nil {
		zap.L().Error("GetAllComments:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": comments, "total": total}, c)
}

func (h *AdminHandler) GetPendingComments(c *gin.Context) {
	var q dto.AdminCommentQuery
	_ = c.ShouldBindQuery(&q)
	comments, total, err := h.svc.PendingComments(q.Page, q.PageSize)
	if err != nil {
		zap.L().Error("GetPendingComments:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": comments, "total": total}, c)
}

func (h *AdminHandler) ApproveComment(c *gin.Context) {
	h.setCommentStatus(c, 1, "审核通过")
}

func (h *AdminHandler) RejectComment(c *gin.Context) {
	h.setCommentStatus(c, 3, "已驳回")
}

func (h *AdminHandler) SetCommentStatus(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	var body struct {
		Status int8 `json:"status"`
	}
	if c.ShouldBind(&body) != nil || (body.Status != 1 && body.Status != 3) {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.SetCommentStatus(id, body.Status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrCommentNotFound, c)
			return
		}
		zap.L().Error("SetCommentStatus:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("状态已更新", c)
}

func (h *AdminHandler) DeleteComment(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.DeleteComment(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrCommentNotFound, c)
			return
		}
		zap.L().Error("DeleteComment:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("删除成功", c)
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	var q dto.AdminArticleQuery
	_ = c.ShouldBindQuery(&q)
	var status uint64
	if statusStr := c.Query("status"); statusStr != "" {
		status, _ = strconv.ParseUint(statusStr, 10, 64)
	}
	users, total, err := h.svc.Users(q.Page, q.PageSize, c.Query("keyword"), status)
	if err != nil {
		zap.L().Error("GetUsers:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	type safeUser struct {
		ID        uint64 `json:"id"`
		Email     string `json:"email"`
		Nickname  string `json:"nickname"`
		Status    uint64 `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	safeUsers := make([]safeUser, 0, len(users))
	for _, u := range users {
		safeUsers = append(safeUsers, safeUser{
			ID:        u.ID,
			Email:     u.Email,
			Nickname:  u.Nickname,
			Status:    u.Status,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	response.SuccessWithData(map[string]interface{}{"list": safeUsers, "total": total}, c)
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	h.setUserStatus(c, true)
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	h.setUserStatus(c, false)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.DeleteUser(id); err != nil {
		zap.L().Error("DeleteUser:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("删除成功", c)
}

// 分类管理
func (h *AdminHandler) GetCategories(c *gin.Context) {
	keyword := c.Query("keyword")
	categories, err := h.svc.AdminGetCategories(keyword)
	if err != nil {
		zap.L().Error("AdminGetCategories:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(categories, c)
}

func (h *AdminHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.CreateCategory(req); err != nil {
		zap.L().Error("CreateCategory:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("创建成功", c)
}

func (h *AdminHandler) UpdateCategory(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	var req dto.UpdateCategoryReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.UpdateCategory(id, req); err != nil {
		zap.L().Error("UpdateCategory:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("更新成功", c)
}

func (h *AdminHandler) UpdateCategorySort(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	var req dto.UpdateCategorySortReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.UpdateCategorySort(id, req.Sort); err != nil {
		zap.L().Error("UpdateCategorySort:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("排序已更新", c)
}

func (h *AdminHandler) BatchUpdateCategorySort(c *gin.Context) {
	var req dto.BatchUpdateSortReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.BatchUpdateCategorySort(req.Ids); err != nil {
		zap.L().Error("BatchUpdateCategorySort:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("排序已更新", c)
}

func (h *AdminHandler) DeleteCategory(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	var req dto.DeleteCategoryReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrWithMsg(code.BadRequest, c)
			return
		}
	}
	if err := h.svc.DeleteCategory(id, req); err != nil {
		zap.L().Error("DeleteCategory:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("删除成功", c)
}

func (h *AdminHandler) GetCategoryArticleCount(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	count, err := h.svc.GetCategoryArticleCount(id)
	if err != nil {
		zap.L().Error("GetCategoryArticleCount:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]int64{"count": count}, c)
}

func (h *AdminHandler) TransferArticles(c *gin.Context) {
	var req dto.TransferArticlesReq
	if err := c.ShouldBind(&req); err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.TransferArticles(req.FromCategoryID, req.ToCategoryID); err != nil {
		zap.L().Error("TransferArticles:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg("转移成功", c)
}

func (h *AdminHandler) GetCategoryArticles(c *gin.Context) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	var q dto.PageQueryWithSize
	_ = c.ShouldBindQuery(&q)
	articles, total, err := h.svc.AdminGetCategoryArticles(id, q.Page, q.PageSize)
	if err != nil {
		zap.L().Error("GetCategoryArticles:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithData(map[string]interface{}{"list": articles, "total": total}, c)
}

var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// 上传图片
func (h *AdminHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		zap.L().Error("UploadImage:" + err.Error())
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	defer func(file multipart.File) {
		if err := file.Close(); err != nil {
			zap.L().Error("UploadImage:" + err.Error())
		}
	}(file)
	//图片大小10MB以下，且必须是指定的后缀
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] || header.Size > 10*1024*1024 {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	//图片名字
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), strings.TrimSuffix(header.Filename, ext), ext)
	//上传，获得url
	url, err := utils.UploadToOss(file, config.Cfg.OssConfig.Image_path, filename)
	if err != nil {
		zap.L().Error("UploadImage:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	//返回url
	c.JSON(http.StatusOK, response.Response{Code: 0, Message: "上传成功", Data: map[string]string{"url": url}})
}

func (h *AdminHandler) setCommentStatus(c *gin.Context, status int8, msg string) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if err := h.svc.SetCommentStatus(id, status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.ErrWithMsg(code.ErrCommentNotFound, c)
			return
		}
		zap.L().Error("SetCommentStatus:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	response.SuccessWithMsg(msg, c)
}

func (h *AdminHandler) setUserStatus(c *gin.Context, banned bool) {
	id, err := parseParamID(c)
	if err != nil {
		response.ErrWithMsg(code.BadRequest, c)
		return
	}
	if banned {
		err = h.svc.BanUser(id)
	} else {
		err = h.svc.UnbanUser(id)
	}
	if err != nil {
		zap.L().Error("SetUserStatus:" + err.Error())
		response.ErrWithMsg(code.InternalError, c)
		return
	}
	if banned {
		response.SuccessWithMsg("封禁成功", c)
	} else {
		response.SuccessWithMsg("解封成功", c)
	}
}

func parseParamID(c *gin.Context) (uint64, error) {
	return strconv.ParseUint(c.Param("id"), 10, 64)
}

func idFromRequest(c *gin.Context) (uint64, error) {
	if id, err := parseParamID(c); err == nil && id > 0 {
		return id, nil
	}
	var req dto.IDReq
	if err := c.ShouldBind(&req); err != nil {
		return 0, err
	}
	return req.ID, nil
}

func currentUserID(c *gin.Context) (uint64, bool) {
	userID, ok := c.Get("userID")
	if !ok {
		return 0, false
	}
	uid, ok := userID.(uint64)
	return uid, ok
}
