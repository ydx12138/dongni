package service

import (
	"blog/models"
	"blog/models/dto"
	"blog/models/vo"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	settingRegisterEnabled               = "register_enabled"
	settingCategoriesEnabled             = "categories_enabled"
	settingProfileEnabled                = "profile_enabled"
	settingCommentsEnabled               = "comments_enabled"
	settingLikeEnabled                   = "like_enabled"
	settingSiteTitle                     = "site_title"
	settingProfileGithub                 = "profile_github"
	settingProfileEmail                  = "profile_email"
	settingProfileAvatar                 = "profile_avatar"
	settingProfileAbout                  = "profile_about"
	settingUserAccessTokenExpireMinutes  = "user_access_token_expire_minutes"
	settingUserRefreshTokenExpireMinutes = "user_refresh_token_expire_minutes"
	settingAdminTokenExpireMinutes       = "admin_token_expire_minutes"
	// legacyUserTokenExpireMinutes 是旧版统一用户 Token 有效期的配置键，仅用于迁移读取，不再写入。
	legacyUserTokenExpireMinutes = "user_token_expire_minutes"
)

const (
	defaultUserAccessTokenExpireMinutes  = 15
	defaultUserRefreshTokenExpireMinutes = 7 * 24 * 60
	defaultAdminTokenExpireMinutes       = 7 * 24 * 60
	minTokenExpireMinutes                = 1
	maxTokenExpireMinutes                = 999999
)

var siteSettingKeys = []string{
	settingRegisterEnabled,
	settingCategoriesEnabled,
	settingProfileEnabled,
	settingCommentsEnabled,
	settingLikeEnabled,
	settingSiteTitle,
	settingProfileGithub,
	settingProfileEmail,
	settingProfileAvatar,
	settingProfileAbout,
	settingUserAccessTokenExpireMinutes,
	settingUserRefreshTokenExpireMinutes,
	settingAdminTokenExpireMinutes,
	legacyUserTokenExpireMinutes,
}

var (
	ErrFeatureDisabled    = errors.New("feature disabled")
	ErrInvalidSiteSetting = errors.New("invalid site setting")
)

// GetSiteSettings 获取前台站点配置；参数为无，返回值为带默认值的站点配置和查询错误。
func (s *Service) GetSiteSettings() (vo.SiteSettingVO, error) {
	settings, err := s.repo.GetSettings(siteSettingKeys)
	if err != nil {
		return vo.SiteSettingVO{}, err
	}
	return siteSettingFromRows(settings), nil
}

// GetAdminSiteSettings 获取管理端站点配置；参数为无，返回值为管理端可编辑的配置和查询错误。
func (s *Service) GetAdminSiteSettings() (vo.SiteSettingVO, error) {
	return s.GetSiteSettings()
}

// UserAccessTokenDuration 获取用户 Access Token 的有效时长；无参数；返回时长和查询错误。
func (s *Service) UserAccessTokenDuration() (time.Duration, error) {
	return s.tokenDuration(settingUserAccessTokenExpireMinutes, defaultUserAccessTokenExpireMinutes)
}

// UserRefreshTokenDuration 获取用户 Refresh Token 的有效时长；无参数；返回时长和查询错误。
// 优先读取新版配置键，若不存在则回退旧版统一配置键，保证历史数据不丢失。
func (s *Service) UserRefreshTokenDuration() (time.Duration, error) {
	settings, err := s.repo.GetSettings([]string{settingUserRefreshTokenExpireMinutes, legacyUserTokenExpireMinutes})
	if err != nil {
		return 0, err
	}
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	minutes := defaultUserRefreshTokenExpireMinutes
	if value, ok := values[settingUserRefreshTokenExpireMinutes]; ok {
		minutes = parseTokenExpireMinutes(value, defaultUserRefreshTokenExpireMinutes)
	} else if value, ok := values[legacyUserTokenExpireMinutes]; ok {
		minutes = parseTokenExpireMinutes(value, defaultUserRefreshTokenExpireMinutes)
	}
	return time.Duration(minutes) * time.Minute, nil
}

// AdminTokenDuration 获取管理员 Token 的有效时长；无参数；返回时长和查询错误。
func (s *Service) AdminTokenDuration() (time.Duration, error) {
	return s.tokenDuration(settingAdminTokenExpireMinutes, defaultAdminTokenExpireMinutes)
}

// tokenDuration 读取并转换 Token 有效期配置；参数为配置键和兜底分钟数；返回时间长度和查询错误。
func (s *Service) tokenDuration(key string, fallback int) (time.Duration, error) {
	settings, err := s.repo.GetSettings([]string{key})
	if err != nil {
		return 0, err
	}
	minutes := fallback
	if len(settings) > 0 {
		minutes = parseTokenExpireMinutes(settings[0].Value, fallback)
	}
	return time.Duration(minutes) * time.Minute, nil
}

// UpdateSiteSettings 校验并保存站点配置；参数为管理员提交的配置，返回值为校验或数据库错误。
func (s *Service) UpdateSiteSettings(req dto.UpdateSiteSettingRequest) error {
	if err := validateSiteSetting(req); err != nil {
		return err
	}
	settings := []models.Setting{
		{Key: settingRegisterEnabled, Value: strconv.FormatBool(req.RegisterEnabled)},
		{Key: settingCategoriesEnabled, Value: strconv.FormatBool(req.CategoriesEnabled)},
		{Key: settingProfileEnabled, Value: strconv.FormatBool(req.ProfileEnabled)},
		{Key: settingCommentsEnabled, Value: strconv.FormatBool(req.CommentsEnabled)},
		{Key: settingLikeEnabled, Value: strconv.FormatBool(req.LikeEnabled)},
		{Key: settingSiteTitle, Value: strings.TrimSpace(req.SiteTitle)},
		{Key: settingProfileGithub, Value: strings.TrimSpace(req.ProfileGithub)},
		{Key: settingProfileEmail, Value: strings.TrimSpace(req.ProfileEmail)},
		{Key: settingProfileAvatar, Value: strings.TrimSpace(req.ProfileAvatar)},
		{Key: settingProfileAbout, Value: strings.TrimSpace(req.ProfileAbout)},
		{Key: settingUserAccessTokenExpireMinutes, Value: strconv.Itoa(req.UserAccessTokenExpireMinutes)},
		{Key: settingUserRefreshTokenExpireMinutes, Value: strconv.Itoa(req.UserRefreshTokenExpireMinutes)},
		{Key: settingAdminTokenExpireMinutes, Value: strconv.Itoa(req.AdminTokenExpireMinutes)},
	}
	return s.repo.UpsertSettings(settings)
}

// IsFeatureEnabled 查询功能开关；参数为配置键，返回值为功能是否开启和查询错误。
func (s *Service) IsFeatureEnabled(key string) (bool, error) {
	settings, err := s.repo.GetSettings([]string{key})
	if err != nil {
		return false, err
	}
	if len(settings) == 0 {
		return true, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(settings[0].Value))
	if err != nil {
		return true, nil
	}
	return value, nil
}

// RequireFeatureEnabled 检查指定功能是否开启；参数为配置键，返回值为功能关闭或配置读取错误。
func (s *Service) RequireFeatureEnabled(key string) error {
	enabled, err := s.IsFeatureEnabled(key)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrFeatureDisabled
	}
	return nil
}

func siteSettingFromRows(settings []models.Setting) vo.SiteSettingVO {
	result := vo.SiteSettingVO{
		RegisterEnabled:   true,
		CategoriesEnabled: true,
		ProfileEnabled:    true,
		CommentsEnabled:   true,
		LikeEnabled:       true,
		SiteTitle:         "懂你",
	}
	values := make(map[string]string, len(settings))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	result.RegisterEnabled = parseSettingBool(values[settingRegisterEnabled], true)
	result.CategoriesEnabled = parseSettingBool(values[settingCategoriesEnabled], true)
	result.ProfileEnabled = parseSettingBool(values[settingProfileEnabled], true)
	result.CommentsEnabled = parseSettingBool(values[settingCommentsEnabled], true)
	result.LikeEnabled = parseSettingBool(values[settingLikeEnabled], true)
	if value := strings.TrimSpace(values[settingSiteTitle]); value != "" {
		result.SiteTitle = value
	}
	result.ProfileGithub = strings.TrimSpace(values[settingProfileGithub])
	result.ProfileEmail = strings.TrimSpace(values[settingProfileEmail])
	result.ProfileAvatar = strings.TrimSpace(values[settingProfileAvatar])
	result.ProfileAbout = strings.TrimSpace(values[settingProfileAbout])
	result.UserAccessTokenExpireMinutes = parseTokenExpireMinutes(values[settingUserAccessTokenExpireMinutes], defaultUserAccessTokenExpireMinutes)
	result.AdminTokenExpireMinutes = parseTokenExpireMinutes(values[settingAdminTokenExpireMinutes], defaultAdminTokenExpireMinutes)
	// Refresh 有效期优先读新键，历史数据回退旧版统一配置键。
	result.UserRefreshTokenExpireMinutes = parseTokenExpireMinutes(values[settingUserRefreshTokenExpireMinutes], defaultUserRefreshTokenExpireMinutes)
	if _, ok := values[settingUserRefreshTokenExpireMinutes]; !ok {
		if legacy, exists := values[legacyUserTokenExpireMinutes]; exists {
			result.UserRefreshTokenExpireMinutes = parseTokenExpireMinutes(legacy, defaultUserRefreshTokenExpireMinutes)
		}
	}
	return result
}

func parseSettingBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

// parseTokenExpireMinutes 解析并校验 Token 有效期分钟数；参数为配置文本和兜底分钟数；返回合法分钟数。
func parseTokenExpireMinutes(value string, fallback int) int {
	minutes, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || minutes < minTokenExpireMinutes || minutes > maxTokenExpireMinutes {
		return fallback
	}
	return minutes
}

func validateSiteSetting(req dto.UpdateSiteSettingRequest) error {
	if req.UserAccessTokenExpireMinutes < minTokenExpireMinutes || req.UserAccessTokenExpireMinutes > maxTokenExpireMinutes {
		return fmt.Errorf("%w: 用户Access Token有效期必须是1-999999的整数", ErrInvalidSiteSetting)
	}
	if req.UserRefreshTokenExpireMinutes < minTokenExpireMinutes || req.UserRefreshTokenExpireMinutes > maxTokenExpireMinutes {
		return fmt.Errorf("%w: 用户Refresh Token有效期必须是1-999999的整数", ErrInvalidSiteSetting)
	}
	if req.UserAccessTokenExpireMinutes >= req.UserRefreshTokenExpireMinutes {
		return fmt.Errorf("%w: Access Token有效期必须小于Refresh Token有效期", ErrInvalidSiteSetting)
	}
	if req.AdminTokenExpireMinutes < minTokenExpireMinutes || req.AdminTokenExpireMinutes > maxTokenExpireMinutes {
		return fmt.Errorf("%w: 管理员Token有效期必须是1-999999的整数", ErrInvalidSiteSetting)
	}
	if title := strings.TrimSpace(req.SiteTitle); title == "" || len([]rune(title)) > 6 {
		return fmt.Errorf("%w: 网站名称不能为空且不能超过6个字符", ErrInvalidSiteSetting)
	}
	if err := validateOptionalURL(req.ProfileGithub, "GitHub地址"); err != nil {
		return err
	}
	if err := validateOptionalURL(req.ProfileAvatar, "头像地址"); err != nil {
		return err
	}
	if email := strings.TrimSpace(req.ProfileEmail); email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return fmt.Errorf("%w: Email格式不正确", ErrInvalidSiteSetting)
		}
	}
	if len([]rune(strings.TrimSpace(req.ProfileAbout))) > 2000 {
		return fmt.Errorf("%w: 关于我不能超过2000个字符", ErrInvalidSiteSetting)
	}
	return nil
}

func validateOptionalURL(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%w: %s格式不正确", ErrInvalidSiteSetting, field)
	}
	return nil
}
