package dto

// UpdateSiteSettingRequest 描述管理员提交的站点配置；参数为前台开关和站点资料，返回值由接口统一响应。
type UpdateSiteSettingRequest struct {
	RegisterEnabled   bool   `json:"register_enabled"`
	CategoriesEnabled bool   `json:"categories_enabled"`
	ProfileEnabled    bool   `json:"profile_enabled"`
	CommentsEnabled   bool   `json:"comments_enabled"`
	SiteTitle         string `json:"site_title"`
	ProfileGithub     string `json:"profile_github"`
	ProfileEmail      string `json:"profile_email"`
	ProfileAvatar     string `json:"profile_avatar"`
	ProfileAbout      string `json:"profile_about"`
}
