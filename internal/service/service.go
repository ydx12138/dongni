package service

import (
	"blog/config"
	"blog/internal/repository"
	"blog/internal/utils"
	"blog/internal/wechat"
	"blog/models"
	"blog/models/dto"
	"blog/models/vo"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 验证码过期时间
const registerCodeTTL = 60 * time.Second
const resetPasswordCodeTTL = 60 * time.Second
const phoneTicketTTL = 15 * time.Minute

var (
	ErrUserExists        = errors.New("user exists")
	ErrUserNotExists     = errors.New("user not exists")
	ErrPassword          = errors.New("password error")
	ErrUserDisabled      = errors.New("user disabled")
	ErrVerificationCode  = errors.New("verification code invalid or expired")
	ErrCoderepeated      = errors.New("verification codes cannot be obtained repeatedly")
	ErrWechatUnavailable = errors.New("wechat login is not configured")
	ErrPhoneTicket       = errors.New("wechat phone ticket is invalid or expired")
	ErrPhoneAlreadyBound = errors.New("phone number is already bound")
	ErrDefaultAvatar     = errors.New("default avatar is not configured")
	ErrSessionInvalid    = errors.New("pc login session is invalid or replaced")
	ErrInvalidUserAvatar = errors.New("invalid user avatar")
)

type Service struct {
	repo           *repository.Repository
	redis          *redis.Client
	wechat         wechat.Exchanger
	sessionRepo    repository.RedisMiddleRepository
	userStatusRepo repository.UserStatusRepository
}

func New(repo *repository.Repository, redis *redis.Client, exchanger wechat.Exchanger, sessionRepos ...repository.RedisMiddleRepository) *Service {
	var sessionRepo repository.RedisMiddleRepository
	if len(sessionRepos) > 0 {
		sessionRepo = sessionRepos[0]
	}
	return &Service{repo: repo, redis: redis, wechat: exchanger, sessionRepo: sessionRepo, userStatusRepo: repo}
}

// SetUserStatusRepository 替换用户状态查询仓储；参数为状态仓储；无返回值，主要用于依赖注入和测试。
func (s *Service) SetUserStatusRepository(statusRepo repository.UserStatusRepository) {
	s.userStatusRepo = statusRepo
}

func (s *Service) SetWechatExchanger(exchanger wechat.Exchanger) {
	s.wechat = exchanger
}

func BuildWechatUser(openID string) (models.User, error) {
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return models.User{}, err
	}
	nicknameBytes := make([]byte, 3)
	if _, err := rand.Read(nicknameBytes); err != nil {
		return models.User{}, err
	}
	password, err := utils.HashPassword(hex.EncodeToString(passwordBytes))
	if err != nil {
		return models.User{}, err
	}
	return models.User{
		Email:        "wechat-" + openID + "@wechat.local",
		Password:     password,
		Nickname:     "微信用户" + hex.EncodeToString(nicknameBytes),
		Status:       1,
		WechatOpenID: openID,
	}, nil
}

func BuildEmailUser(email, password, nickname, avatar string) (models.User, error) {
	avatar = strings.TrimSpace(avatar)
	if avatar == "" {
		return models.User{}, ErrDefaultAvatar
	}
	return models.User{
		Email:    email,
		Password: password,
		Avatar:   avatar,
		Nickname: nickname,
	}, nil
}

func (s *Service) WechatLogin(ctx context.Context, loginCode string) (map[string]interface{}, error) {
	//
	if s.wechat == nil {
		return nil, ErrWechatUnavailable
	}
	session, err := s.wechat.ExchangeCode(ctx, loginCode)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetUserByWechatOpenID(session.OpenID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user, err = BuildWechatUser(session.OpenID)
		if err != nil {
			return nil, err
		}
		if err = s.repo.CreateUser(user); err != nil {
			return nil, err
		}
		user, err = s.repo.GetUserByWechatOpenID(session.OpenID)
	}
	if err != nil {
		return nil, err
	}
	if user.Status == 2 {
		return nil, ErrUserDisabled
	}
	if user.Phone == nil || strings.TrimSpace(*user.Phone) == "" {
		ticket, err := s.savePhoneTicket(user.ID)
		if err != nil {
			return nil, err
		}
		return phoneRequiredResponse(ticket), nil
	}
	return s.issueUserTokens(user)
}

const pcSessionTTL = 7 * 24 * time.Hour

// ValidateSession 校验 Token 中的 PC 会话 ID 是否仍是 Redis 中的当前会话。
// 参数：userID 为用户 ID，sessionID 为 Token 携带的会话 ID；返回 nil 表示有效，返回 ErrSessionInvalid 表示已被替换或不存在，其他错误表示 Redis 访问失败。
func (s *Service) ValidateSession(userID uint64, sessionID string) error {
	if s.sessionRepo == nil || strings.TrimSpace(sessionID) == "" {
		return ErrSessionInvalid
	}
	currentSession, err := s.sessionRepo.GetUserSession(userID)
	if err != nil {
		return err
	}
	if currentSession == "" || subtle.ConstantTimeCompare([]byte(currentSession), []byte(sessionID)) != 1 {
		return ErrSessionInvalid
	}
	return nil
}

// IsUserActive 检查用户账号是否仍可使用；参数为用户 ID；返回可用状态和查询错误。
func (s *Service) IsUserActive(userID uint64) (bool, error) {
	if s.userStatusRepo == nil || userID == 0 {
		return false, errors.New("user status repository is nil")
	}
	status, err := s.userStatusRepo.GetUserStatus(userID)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}

// generateSessionID 使用密码学安全随机数生成 PC 单点登录会话 ID。
// 参数：无；返回 64 位十六进制会话 ID和生成错误。
func generateSessionID() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

// issuePCUserTokens 为 PC 邮箱登录生成带同一 sessionID 的 access 和 refresh Token，并保存当前会话。
// 参数：user 为已通过密码校验的用户；返回登录响应数据和签发或保存失败错误。
func (s *Service) issuePCUserTokens(user models.User) (map[string]interface{}, error) {
	if s.sessionRepo == nil {
		return nil, errors.New("session repository is nil")
	}
	accessDuration, err := s.UserAccessTokenDuration()
	if err != nil {
		return nil, err
	}
	refreshDuration, err := s.UserRefreshTokenDuration()
	if err != nil {
		return nil, err
	}
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateUserTokenWithSession(user.ID, refreshDuration, "refresh", sessionID)
	if err != nil {
		return nil, err
	}
	accessToken, err := utils.GenerateUserTokenWithSession(user.ID, accessDuration, "access", sessionID)
	if err != nil {
		return nil, err
	}
	// 登录会话与 Refresh Token 同生命周期，避免 Refresh 仍有效但会话被提前清除。
	if err := s.sessionRepo.SaveUserSession(user.ID, sessionID, refreshDuration); err != nil {
		return nil, err
	}
	if err := s.SaveRefreshToken(refreshTokenKey(user.ID), refreshToken, refreshDuration); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"email":         user.Email,
		"avatar":        user.Avatar,
		"nickname":      user.Nickname,
		"id":            user.ID,
	}, nil
}

func phoneRequiredResponse(ticket string) map[string]interface{} {
	return map[string]interface{}{
		"phone_required": true,
		"phone_ticket":   ticket,
	}
}

func (s *Service) CompleteWechatPhoneLogin(ctx context.Context, ticket, phoneCode string) (map[string]interface{}, error) {
	if s.wechat == nil {
		return nil, ErrWechatUnavailable
	}
	userID, err := s.getPhoneTicketUserID(ticket)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user.Status == 2 {
		return nil, ErrUserDisabled
	}
	phone, err := s.wechat.ExchangePhoneCode(ctx, phoneCode)
	if err != nil {
		return nil, err
	}
	boundUser, err := s.repo.GetUserByPhone(phone)
	if err == nil && boundUser.ID != user.ID {
		return nil, ErrPhoneAlreadyBound
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err = s.repo.UpdateUserPhone(user.ID, phone); err != nil {
		return nil, err
	}
	if err = s.redis.Del(ctx, phoneTicketKey(ticket)).Err(); err != nil {
		return nil, err
	}
	user.Phone = &phone
	return s.issueUserTokens(user)
}

func (s *Service) savePhoneTicket(userID uint64) (string, error) {
	if s.redis == nil {
		return "", errors.New("redis client is nil")
	}
	ticketBytes := make([]byte, 32)
	if _, err := rand.Read(ticketBytes); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(ticketBytes)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.redis.Set(ctx, phoneTicketKey(ticket), strconv.FormatUint(userID, 10), phoneTicketTTL).Err(); err != nil {
		return "", err
	}
	return ticket, nil
}

func (s *Service) getPhoneTicketUserID(ticket string) (uint64, error) {
	if s.redis == nil || strings.TrimSpace(ticket) == "" {
		return 0, ErrPhoneTicket
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	value, err := s.redis.Get(ctx, phoneTicketKey(ticket)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrPhoneTicket
	}
	if err != nil {
		return 0, err
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		return 0, ErrPhoneTicket
	}
	return userID, nil
}

func phoneTicketKey(ticket string) string {
	return "wechat:phone-ticket:" + ticket
}

// refreshToken是否还在
func (s *Service) RefreshTokenIsExist(userid string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	uid, err := strconv.Atoi(userid)
	if err != nil {
		zap.L().Error("Invalid user id" + err.Error())
		return false
	}
	//
	refreshToken := s.redis.Get(ctx, refreshTokenKey(uint64(uid))).Val()
	if refreshToken == "" {
		return false
	}
	return true

}

// 修改电话号码
func (s *Service) UpdatePhoneNumber(email, phone, verificationcode string) error {
	//这个手机号是否已经被其他账号绑定
	//
	return nil
}

func (s *Service) UpdatePasswordByCode(email, password string, code string) error {
	//email对应的验证码是否存在
	result, err := s.CheckCode(email)
	if err != nil {
		return err
	}
	if result == false {
		return ErrVerificationCode
	}
	//检查重置密码验证码
	res, err := s.VerifyResetPasswordCode(email, code)
	if err != nil {
		return err
	}
	if res == false {
		return ErrVerificationCode
	}
	//修改密码
	hashPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	if err = s.repo.UpdateUserPassword(email, hashPassword); err != nil {
		return err
	}
	return nil
}

// 验证重置密码的验证码是否正确
func (s *Service) VerifyResetPasswordCode(email string, code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	//检查验证码是否存在
	result, err := s.redis.ZRangeWithScores(ctx, resetPasswordCodeKey(email), 0, -1).Result()
	if err != nil {
		return false, err
	}
	if len(result) == 0 {
		return false, ErrVerificationCode
	}
	//是否一样,如果不一样，次数减一
	for _, value := range result {
		if value.Member != code {
			err = s.redis.ZIncrBy(ctx, resetPasswordCodeKey(email), -1, value.Member.(string)).Err()
			if err != nil {
				return false, err
			}
			if err = s.DeleteCodeEfftive(email); err != nil {
				return false, err
			}
			return false, ErrVerificationCode
		}
	}
	//如果次数-1之后为0，删除这个key

	//验证码通过
	return true, nil
}

// 如果次数为0，删除验证码
func (s *Service) DeleteCodeEfftive(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := s.redis.ZRangeWithScores(ctx, resetPasswordCodeKey(email), 0, -1).Result()
	if err != nil {
		return err
	}
	for _, value := range res {
		if value.Score == 0 {
			err = s.redis.Del(ctx, resetPasswordCodeKey(email)).Err()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// 给忘记密码的邮箱发验证码
func (s *Service) SendCodeForgetPwd(email string) error {
	email = normalizeEmail(email)
	//邮箱是否已经存在，如果不存在，返回
	user, err := s.repo.GetUserByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if user.ID == 0 {
		return ErrUserNotExists
	}
	//上次已发的验证码是否还在有效期内
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := s.CheckCode(email)
	if err != nil {
		return err
	}
	if result == true {
		return ErrCoderepeated
	}
	//生成验证码
	verifyCode, err := utils.GenerateCode()
	if err != nil {
		return err
	}
	//存到redis
	if err = s.SaveCodeForgetPwd(email, verifyCode, 5); err != nil {
		return err
	}
	//发验证码，如果发送失败，把刚刚存的验证码删除
	key := resetPasswordCodeKey(email)
	if err = utils.SendEmailToQQ(email, "YDX Blog 重置密码验证码", verifyCode); err != nil {
		_ = s.redis.Del(ctx, key).Err()
		return err
	}
	return nil
}

// 保存重置密码验证码
func (s *Service) SaveCodeForgetPwd(email, verifyCode string, effective float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	key := resetPasswordCodeKey(email)
	//存到zset,5次访问机会
	if err := s.redis.ZAdd(ctx, key, redis.Z{Score: effective, Member: verifyCode}).Err(); err != nil {
		return err
	}
	//60秒过期
	err := s.redis.Expire(ctx, key, resetPasswordCodeTTL).Err()
	if err != nil {
		return err
	}
	return nil
}

// 检查是否已经存在重置密码验证码
func (s *Service) CheckCode(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	key := resetPasswordCodeKey(email)
	result, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if result == 0 {
		return false, nil
	}
	return true, nil
}

func (s *Service) GetArticles(page, pageSize int) ([]vo.ArticleSimple, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return s.repo.GetArticleByPage(page, pageSize)
}

// GetArticleRanking 获取浏览量最高的已发布文章；参数 limit 为最多返回数量；返回文章摘要列表和查询错误。
func (s *Service) GetArticleRanking(limit int) ([]vo.ArticleSimple, error) {
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	return s.repo.GetArticleRanking(limit)
}

func (s *Service) GetArticle(id uint64) (vo.ArticleDetail, error) {
	detail, err := s.repo.GetArticleDetail(id)
	if err != nil {
		return detail, err
	}
	_ = s.repo.IncrementViewCount(id)
	// 合并 Redis 中尚未同步到 MySQL 的点赞增量，让详情页点赞数实时准确。
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if delta, err := s.redis.Get(ctx, articleLikeKey(id)).Int64(); err == nil && delta > 0 {
			detail.LikeCount += uint64(delta)
		}
		cancel()
	}
	return detail, nil
}

func (s *Service) SearchArticle(keyword string) ([]vo.ArticleSimple, error) {
	return s.repo.SearchArticleByKey(keyword)
}

// SearchArticleResults 搜索完整结果并生成命中片段；参数 keyword 为搜索词，返回结果列表和业务错误。
func (s *Service) SearchArticleResults(keyword string) ([]vo.ArticleSearch, error) {
	return s.searchArticles(keyword, 0)
}

// SearchArticleSuggestions 查询搜索下拉建议；参数 keyword 为搜索词，返回最多十条真实文章和业务错误。
func (s *Service) SearchArticleSuggestions(keyword string) ([]vo.ArticleSearch, error) {
	return s.searchArticles(keyword, 10)
}

// searchArticles 统一查询搜索文章并生成命中片段；参数 keyword 为搜索词、limit 为返回上限，返回处理后的搜索结果。
func (s *Service) searchArticles(keyword string, limit int) ([]vo.ArticleSearch, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []vo.ArticleSearch{}, nil
	}
	articles, err := s.repo.SearchArticles(keyword, limit)
	if err != nil {
		return nil, err
	}
	for index := range articles {
		articles[index].SearchExcerpt = utils.BuildArticleSearchExcerpt(
			articles[index].Title,
			articles[index].Summary,
			articles[index].Content,
			keyword,
		)
		articles[index].Content = ""
	}
	return articles, nil
}

func (s *Service) SendRegisterCode(req dto.SendRegisterCodeReq) error {
	if err := s.RequireFeatureEnabled(settingRegisterEnabled); err != nil {
		return err
	}
	email := normalizeEmail(req.Email)
	//查找邮箱
	existUser, err := s.repo.GetUserByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	//如果邮箱存在
	if existUser.ID != 0 {
		return ErrUserExists
	}
	//如果redis客户端为nil
	if s.redis == nil {
		return errors.New("redis client is nil")
	}
	//生成验证码
	verifyCode, err := utils.GenerateCode()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	//key:verifyCode
	key := registerCodeKey(email)
	//验证码存redis
	if err := s.redis.Set(ctx, key, verifyCode, registerCodeTTL).Err(); err != nil {
		return err
	}
	//发验证码
	if err := utils.SendEmailToQQ(email, "YDX Blog 注册验证码", verifyCode); err != nil {
		_ = s.redis.Del(ctx, key).Err()
		return err
	}
	return nil
}

// 注册
func (s *Service) Register(req dto.UserRegister) error {
	if err := s.RequireFeatureEnabled(settingRegisterEnabled); err != nil {
		return err
	}
	email := normalizeEmail(req.Email)
	//判断验证码是否无误
	if err := s.verifyRegisterCode(email, req.Code); err != nil {
		return err
	}
	//用户是否已经存在
	existUser, err := s.repo.GetUserByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existUser.ID != 0 {
		return ErrUserExists
	}
	//加密密码
	hashPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}
	//创建新用户
	avatar := ""
	if config.Cfg != nil {
		avatar = config.Cfg.OssConfig.DefaultAvatar
	}
	user, err := BuildEmailUser(email, hashPassword, req.Nickname, avatar)
	if err != nil {
		return err
	}
	if err := s.repo.CreateUser(user); err != nil {
		return err
	}
	//删除验证码
	_ = s.deleteRegisterCode(email)
	return nil
}

// 登录
func (s *Service) UserLogin(req dto.UserLogin) (map[string]interface{}, error) {
	//根据邮箱查用户
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	//密码是否正确
	if !utils.CheckPassword(user.Password, req.Password) {
		return nil, ErrPassword
	}
	//用户是否被禁用
	if user.Status == 2 {
		return nil, ErrUserDisabled
	}
	return s.issuePCUserTokens(user)
}

func (s *Service) issueUserTokens(user models.User) (map[string]interface{}, error) {
	//生成refreshToken，放入用户ID和用户身份(user)
	refreshToken, err := utils.GenerateUserToken(user.ID, 7*24*time.Hour, "refresh")
	if err != nil {
		return nil, err
	}
	//生成accessToken，放入用户ID和用户身份(user)
	accessToken, err := utils.GenerateUserToken(user.ID, 15*time.Minute, "access")
	if err != nil {
		return nil, err
	}
	//保存refreshToken
	if err = s.SaveRefreshToken(refreshTokenKey(user.ID), refreshToken, 7*24*time.Hour); err != nil {
		return nil, err
	}
	//accessToken和refreshToken全部返回
	return map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"email":         user.Email,
		"nickname":      user.Nickname,
		"id":            user.ID,
	}, nil
}

// 把refreshToken存入redis
func (s *Service) SaveRefreshToken(key, token string, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.redis.Set(ctx, key, token, duration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetCategories() ([]vo.CategoryWithStats, error) {
	if err := s.RequireFeatureEnabled(settingCategoriesEnabled); err != nil {
		return nil, err
	}
	return s.repo.GetAllCategories()
}

func (s *Service) GetCategoryArticles(categoryID uint64, page int) ([]vo.ArticleSimple, error) {
	if err := s.RequireFeatureEnabled(settingCategoriesEnabled); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	return s.repo.GetArticleByCategory(categoryID, page, 10)
}

// GetCategoryArticlesPage 获取分类文章的下一批数据；参数为分类 ID、页码和批量大小；返回无限加载响应或业务错误。
func (s *Service) GetCategoryArticlesPage(categoryID uint64, page, pageSize int) (vo.ArticleInfinitePage, error) {
	if err := s.RequireFeatureEnabled(settingCategoriesEnabled); err != nil {
		return vo.ArticleInfinitePage{}, err
	}
	page, pageSize = normalizePage(page, pageSize)
	if pageSize > 30 {
		pageSize = 30
	}
	articles, hasMore, err := s.repo.GetPublishedArticlesByCategoryPage(categoryID, page, pageSize)
	if err != nil {
		return vo.ArticleInfinitePage{}, err
	}
	return vo.ArticleInfinitePage{
		List:     articles,
		HasMore:  hasMore,
		NextPage: page + 1,
	}, nil
}

// 管理端分类操作
func (s *Service) AdminGetCategories(keyword string) ([]map[string]interface{}, error) {
	return s.repo.AdminGetCategories(keyword)
}

func (s *Service) CreateCategory(req dto.CreateCategoryReq) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len([]rune(req.Name)) > 50 {
		return errors.New("分类名称不合法")
	}
	if _, err := s.repo.GetCategoryByName(req.Name); err == nil {
		return errors.New("分类名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if req.Sort == 0 {
		maxSort, err := s.repo.GetMaxCategorySort()
		if err != nil {
			return err
		}
		req.Sort = maxSort + 1
	}
	cat := models.Category{
		Name:        req.Name,
		Description: req.Description,
		Cover:       normalizeCategoryCover(req.Cover),
		Sort:        req.Sort,
	}
	return s.repo.CreateCategory(&cat)
}

func (s *Service) UpdateCategory(id uint64, req dto.UpdateCategoryReq) error {
	cat, err := s.repo.GetCategoryByID(id)
	if err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len([]rune(req.Name)) > 50 {
		return errors.New("分类名称不合法")
	}
	if existing, err := s.repo.GetCategoryByName(req.Name); err == nil && existing.ID != id {
		return errors.New("分类名称已存在")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cat.Name = req.Name
	cat.Description = req.Description
	cat.Cover = normalizeCategoryCover(req.Cover)
	return s.repo.UpdateCategory(&cat)
}

// normalizeCategoryCover 统一分类封面地址；参数为请求提交的封面 URL；返回非空 URL，空值时返回系统默认封面。
func normalizeCategoryCover(cover string) string {
	cover = strings.TrimSpace(cover)
	if cover == "" {
		return models.DefaultCategoryCover
	}
	return cover
}

func (s *Service) UpdateCategorySort(id uint64, sort int) error {
	return s.repo.UpdateCategorySort(id, sort)
}

func (s *Service) BatchUpdateCategorySort(ids []uint64) error {
	return s.repo.BatchUpdateCategorySort(ids)
}

func (s *Service) DeleteCategory(id uint64, req dto.DeleteCategoryReq) error {
	count, err := s.repo.GetCategoryArticleCount(id)
	if err != nil {
		return err
	}
	if count == 0 {
		return s.repo.DeleteCategory(id)
	}
	if req.TargetCategoryID > 0 {
		if req.TargetCategoryID == id {
			return errors.New("不能迁移到当前分类")
		}
		if _, err := s.repo.GetCategoryByID(req.TargetCategoryID); err != nil {
			return err
		}
		return s.repo.TransferArticlesAndDeleteCategory(id, req.TargetCategoryID)
	}
	if req.Force && strings.TrimSpace(req.ConfirmText) == "确认删除" {
		return s.repo.DeleteCategoryWithArticles(id)
	}
	return errors.New("该分类下还有文章，请确认删除或迁移文章")
}

func (s *Service) GetCategoryArticleCount(id uint64) (int64, error) {
	return s.repo.GetCategoryArticleCount(id)
}

func (s *Service) TransferArticles(fromID, toID uint64) error {
	if fromID == toID {
		return errors.New("不能迁移到当前分类")
	}
	if _, err := s.repo.GetCategoryByID(toID); err != nil {
		return err
	}
	return s.repo.TransferArticles(fromID, toID)
}

func (s *Service) AdminGetCategoryArticles(id uint64, page, pageSize int) ([]map[string]interface{}, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.GetCategoryArticlesForAdmin(id, page, pageSize)
}

func (s *Service) GetTags() ([]string, error) {
	return s.repo.GetAllTags()
}

// LikeArticle 为文章点赞；参数 articleID 为文章 ID；返回业务或存储错误。
// 点赞先写入 Redis 计数，由后台任务定时批量回写 MySQL，避免高并发点赞直接压垮数据库。
func (s *Service) LikeArticle(articleID uint64) error {
	if err := s.RequireFeatureEnabled(settingLikeEnabled); err != nil {
		return err
	}
	if articleID == 0 {
		return errors.New("invalid article id")
	}
	// Redis 可用时走缓冲计数。
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		key := articleLikeKey(articleID)
		if err := s.redis.Incr(ctx, key).Err(); err == nil {
			// 设置兜底过期时间，防止后台任务异常时 key 永久残留。
			_ = s.redis.Expire(ctx, key, likeKeyTTL).Err()
			return nil
		}
		// Redis 写失败降级为直写 MySQL，保证点赞功能可用。
	}
	return s.repo.IncrementLikeCount(articleID)
}

// SyncLikeCounts 将 Redis 中累积的点赞增量批量写回 MySQL；参数 ctx 为调用方上下文；返回同步错误。
// 每个 key 通过 Lua 原子读取并删除，避免重复计数；写库失败的增量会重新放回 Redis 等待下次重试。
func (s *Service) SyncLikeCounts(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}
	deltas := make(map[uint64]int64)
	var cursor uint64
	for {
		keys, nextCursor, err := s.redis.Scan(ctx, cursor, articleLikeKeyPrefix+"*", likeSyncScanBatch).Result()
		if err != nil {
			return err
		}
		for _, key := range keys {
			articleID := parseArticleLikeKey(key)
			if articleID == 0 {
				continue
			}
			delta, err := getAndDelLikeDelta(ctx, s.redis, key)
			if err != nil {
				return err
			}
			if delta != 0 {
				deltas[articleID] += delta
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	for articleID, delta := range deltas {
		if err := s.repo.AddLikeCount(articleID, delta); err != nil {
			// 写库失败则把增量放回 Redis，并刷新 TTL，等待下次同步重试。
			key := articleLikeKey(articleID)
			if incrErr := s.redis.IncrBy(ctx, key, delta).Err(); incrErr == nil {
				_ = s.redis.Expire(ctx, key, likeKeyTTL).Err()
			}
			zap.L().Error("sync like count to mysql failed: " + err.Error())
		}
	}
	return nil
}

// StartLikeSyncWorker 启动点赞计数后台同步任务；参数 ctx 为生命周期上下文、interval 为同步间隔；无返回值。
func (s *Service) StartLikeSyncWorker(ctx context.Context, interval time.Duration) {
	if s.redis == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.SyncLikeCounts(ctx); err != nil {
				zap.L().Error("sync like counts: " + err.Error())
			}
		}
	}
}

const (
	articleLikeKeyPrefix = "article:like:"
	likeKeyTTL           = 2 * time.Hour
	likeSyncScanBatch    = 200
)

// articleLikeKey 生成文章点赞计数在 Redis 中的键；参数 articleID 为文章 ID；返回固定格式的键名。
func articleLikeKey(articleID uint64) string {
	return articleLikeKeyPrefix + strconv.FormatUint(articleID, 10)
}

// parseArticleLikeKey 从 Redis 键名解析文章 ID；参数 key 为键名；返回文章 ID，非法时返回 0。
func parseArticleLikeKey(key string) uint64 {
	if !strings.HasPrefix(key, articleLikeKeyPrefix) {
		return 0
	}
	articleID, err := strconv.ParseUint(strings.TrimPrefix(key, articleLikeKeyPrefix), 10, 64)
	if err != nil {
		return 0
	}
	return articleID
}

// getAndDelLikeDelta 原子读取并删除点赞计数键；参数为上下文、Redis 客户端和键名；返回计数增量和错误。
func getAndDelLikeDelta(ctx context.Context, client *redis.Client, key string) (int64, error) {
	val, err := likeGetDelScript.Run(ctx, client, []string{key}).Result()
	if err == redis.Nil || val == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	switch v := val.(type) {
	case int64:
		return v, nil
	case string:
		parsed, parseErr := strconv.ParseInt(v, 10, 64)
		if parseErr != nil {
			return 0, parseErr
		}
		return parsed, nil
	}
	return 0, nil
}

// likeGetDelScript 用 Lua 保证「读取并删除」的原子性，兼容各版本 Redis。
var likeGetDelScript = redis.NewScript(`local v = redis.call('GET', KEYS[1]) if v then redis.call('DEL', KEYS[1]) end return v`)

func (s *Service) GetComments(articleID uint64, page int) ([]vo.CommentVO, int64, error) {
	if page < 1 {
		page = 1
	}
	return s.repo.GetCommentsByArticle(articleID, page, 10)
}

// 保存评论
func (s *Service) CreateComment(req dto.CreateCommentReq, userID uint64) error {
	if err := s.RequireFeatureEnabled(settingCommentsEnabled); err != nil {
		return err
	}
	//处理敏感词
	var words []string
	if utils.Has(req.Content) == true {
		words = utils.FindAll(req.Content)
		req.Content = utils.Replace(req.Content)
	}

	comment := models.Comment{
		ArticleID: req.ArticleID,
		UserID:    userID,
		Content:   req.Content,
		ParentID:  req.ParentID,
		Status:    1,
		HitWords:  strings.Join(words, ","),
	}

	if err := s.repo.CreateComment(&comment); err != nil {
		return err
	}
	_ = s.repo.UpdateArticleCommentCount(comment.ArticleID, 1)
	return nil
}

func (s *Service) AdminLogin(req dto.AdminLogin) (map[string]interface{}, error) {
	admin, err := s.repo.LoginVerification(req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	tokenDuration, err := s.AdminTokenDuration()
	if err != nil {
		return nil, err
	}
	token, err := utils.GenerateAdminToken(admin.ID, tokenDuration)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"token":    token,
		"nickname": admin.Nickname,
		"username": admin.Username,
	}, nil
}

func (s *Service) Dashboard() (vo.DashboardData, error) {
	return s.repo.GetDashboard()
}

func (s *Service) AdminArticles(q dto.AdminArticleQuery) ([]models.Article, int64, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	return s.repo.AdminGetArticles(page, pageSize, q.Keyword, q.Status)
}

func (s *Service) AdminArticle(id uint64) (models.Article, error) {
	return s.repo.GetArticleByID(id)
}

func (s *Service) CreateArticle(req dto.CreateArticleReq, authorID uint64) error {
	categoryID := req.CategoryID
	if categoryID == 0 {
		defaultCat, err := s.repo.GetOrCreateDefaultCategory()
		if err == nil {
			categoryID = defaultCat.ID
		}
	}
	article := models.Article{
		Title:       req.Title,
		Summary:     req.Summary,
		Content:     req.Content,
		ContentType: req.ContentType,
		Cover:       req.Cover,
		CategoryID:  categoryID,
		Tags:        req.Tags,
		Status:      req.Status,
		AuthorID:    authorID,
	}
	if req.Status == 2 {
		now := time.Now()
		article.PublishTime = &now
	}
	return s.repo.CreateArticle(&article)
}

func (s *Service) UpdateArticle(req dto.UpdateArticleReq) error {
	article, err := s.repo.GetArticleByID(req.ID)
	if err != nil {
		return err
	}
	article.Title = req.Title
	article.Summary = req.Summary
	article.Content = req.Content
	article.ContentType = req.ContentType
	article.Cover = req.Cover
	article.CategoryID = req.CategoryID
	article.Tags = req.Tags
	article.Status = req.Status
	if req.ViewCount != nil {
		article.ViewCount = *req.ViewCount
	}
	if req.LikeCount != nil {
		article.LikeCount = *req.LikeCount
	}
	if req.Status == 2 && article.PublishTime == nil {
		now := time.Now()
		article.PublishTime = &now
	}
	return s.repo.UpdateArticle(&article)
}

func (s *Service) DeleteArticle(id uint64) error {
	return s.repo.DeleteArticle(id)
}

func (s *Service) Drafts(q dto.AdminArticleQuery) ([]models.Article, int64, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	return s.repo.GetDrafts(page, pageSize)
}

func (s *Service) PublishArticle(id uint64) error {
	article, err := s.repo.GetArticleByID(id)
	if err != nil {
		return err
	}
	article.Status = 2
	now := time.Now()
	article.PublishTime = &now
	return s.repo.UpdateArticle(&article)
}

func (s *Service) AllComments(page, pageSize int, keyword string, searchType string) ([]vo.CommentVO, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.GetAllComments(page, pageSize, keyword, searchType)
}

func (s *Service) PendingComments(page, pageSize int) ([]vo.CommentVO, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.GetPendingComments(page, pageSize)
}

func (s *Service) SetCommentStatus(id uint64, status int8) error {
	old, err := s.repo.GetCommentByID(id)
	if err != nil {
		return err
	}
	if old.Status == 1 && status != 1 {
		_ = s.repo.UpdateArticleCommentCount(old.ArticleID, -1)
	}
	if old.Status != 1 && status == 1 {
		_ = s.repo.UpdateArticleCommentCount(old.ArticleID, 1)
	}
	return s.repo.UpdateCommentStatus(id, status)
}

func (s *Service) DeleteComment(id uint64) error {
	comment, err := s.repo.GetCommentByID(id)
	if err == nil && comment.Status == 1 {
		_ = s.repo.UpdateArticleCommentCount(comment.ArticleID, -1)
	}
	return s.repo.DeleteComment(id)
}

func (s *Service) Users(page, pageSize int, keyword string, status uint64) ([]models.User, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	return s.repo.GetUsersByPage(page, pageSize, keyword, status)
}

func UserProfileFromUser(user models.User) vo.UserProfile {
	return vo.UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Nickname:  user.Nickname,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
	}
}

func (s *Service) CurrentUserProfile(userID uint64) (vo.UserProfile, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return vo.UserProfile{}, err
	}
	return UserProfileFromUser(user), nil
}

// UpdateCurrentUserAvatar 校验并保存当前用户头像；参数为用户 ID 和头像 URL；返回参数或数据库错误。
func (s *Service) UpdateCurrentUserAvatar(userID uint64, avatar string) error {
	avatar = strings.TrimSpace(avatar)
	parsed, err := url.ParseRequestURI(avatar)
	if userID == 0 || err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ErrInvalidUserAvatar
	}
	return s.repo.UpdateUserAvatar(userID, avatar)
}

func (s *Service) BanUser(id uint64) error {
	return s.repo.UpdateUserStatus(id, 2)
}

func (s *Service) UnbanUser(id uint64) error {
	return s.repo.UpdateUserStatus(id, 1)
}

func (s *Service) DeleteUser(id uint64) error {
	return s.repo.DeleteUserByID(id)
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return page, pageSize
}

// 邮箱对应的验证码是否正确?
func (s *Service) verifyRegisterCode(email string, input string) error {
	if s.redis == nil {
		return errors.New("redis client is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stored, err := s.redis.Get(ctx, registerCodeKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrVerificationCode
	}
	if err != nil {
		return err
	}
	input = strings.TrimSpace(input)
	if subtle.ConstantTimeCompare([]byte(stored), []byte(input)) != 1 {
		return ErrVerificationCode
	}
	return nil
}

// 删除邮箱对应的验证码
func (s *Service) deleteRegisterCode(email string) error {
	if s.redis == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.redis.Del(ctx, registerCodeKey(email)).Err()
}

func registerCodeKey(email string) string {
	return "register:email_code:" + normalizeEmail(email)
}
func resetPasswordCodeKey(email string) string {
	return "reset_password:email_code:" + normalizeEmail(email)
}

// refreshToken存入redis时的key
func refreshTokenKey(userid uint64) string {
	return strconv.Itoa(int(userid)) + ":refreshToken:"
}

// 把邮件规范化（大写改成小写）
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
