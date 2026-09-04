package vo

// SiteSettingVO 返回前台和管理端使用的站点配置；参数为数据库中的配置记录，返回值为脱离数据库结构的配置视图。
type SiteSettingVO struct {
	RegisterEnabled               bool   `json:"register_enabled"`
	CategoriesEnabled             bool   `json:"categories_enabled"`
	ProfileEnabled                bool   `json:"profile_enabled"`
	CommentsEnabled               bool   `json:"comments_enabled"`
	LikeEnabled                   bool   `json:"like_enabled"`
	SiteTitle                     string `json:"site_title"`
	ProfileGithub                 string `json:"profile_github"`
	ProfileEmail                  string `json:"profile_email"`
	ProfileAvatar                 string `json:"profile_avatar"`
	ProfileAbout                  string `json:"profile_about"`
	UserAccessTokenExpireMinutes  int    `json:"user_access_token_expire_minutes"`
	UserRefreshTokenExpireMinutes int    `json:"user_refresh_token_expire_minutes"`
	AdminTokenExpireMinutes       int    `json:"admin_token_expire_minutes"`
}
