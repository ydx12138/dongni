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
)

const (
	settingRegisterEnabled   = "register_enabled"
	settingCategoriesEnabled = "categories_enabled"
	settingProfileEnabled    = "profile_enabled"
	settingCommentsEnabled   = "comments_enabled"
	settingSiteTitle         = "site_title"
	settingProfileGithub     = "profile_github"
	settingProfileEmail      = "profile_email"
	settingProfileAvatar     = "profile_avatar"
	settingProfileAbout      = "profile_about"
)

var siteSettingKeys = []string{
	settingRegisterEnabled,
	settingCategoriesEnabled,
	settingProfileEnabled,
	settingCommentsEnabled,
	settingSiteTitle,
	settingProfileGithub,
	settingProfileEmail,
	settingProfileAvatar,
	settingProfileAbout,
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
		{Key: settingSiteTitle, Value: strings.TrimSpace(req.SiteTitle)},
		{Key: settingProfileGithub, Value: strings.TrimSpace(req.ProfileGithub)},
		{Key: settingProfileEmail, Value: strings.TrimSpace(req.ProfileEmail)},
		{Key: settingProfileAvatar, Value: strings.TrimSpace(req.ProfileAvatar)},
		{Key: settingProfileAbout, Value: strings.TrimSpace(req.ProfileAbout)},
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
	if value := strings.TrimSpace(values[settingSiteTitle]); value != "" {
		result.SiteTitle = value
	}
	result.ProfileGithub = strings.TrimSpace(values[settingProfileGithub])
	result.ProfileEmail = strings.TrimSpace(values[settingProfileEmail])
	result.ProfileAvatar = strings.TrimSpace(values[settingProfileAvatar])
	result.ProfileAbout = strings.TrimSpace(values[settingProfileAbout])
	return result
}

func parseSettingBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func validateSiteSetting(req dto.UpdateSiteSettingRequest) error {
	if title := strings.TrimSpace(req.SiteTitle); title == "" || len([]rune(title)) > 40 {
		return fmt.Errorf("%w: 网站名称不能为空且不能超过40个字符", ErrInvalidSiteSetting)
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
