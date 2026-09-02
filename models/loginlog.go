package models

import "time"

type LoginLog struct {
	ID uint64 `gorm:"primaryKey;autoIncrement;comment:主键"`

	UserID *uint64 `gorm:"index:idx_user;comment:用户ID，登录失败时可为空"`

	Username string `gorm:"size:50;comment:登录时使用的用户名"`

	LoginType uint8 `gorm:"not null;comment:登录方式：1密码 2邮箱验证码 3手机验证码"`

	Status uint8 `gorm:"not null;index:idx_status;comment:登录结果：1成功 2失败"`

	IP string `gorm:"size:45;index:idx_ip;comment:客户端IP"`

	Country string `gorm:"size:50;comment:国家"`

	Province string `gorm:"size:50;comment:省份"`

	City string `gorm:"size:50;comment:城市"`

	UserAgent string `gorm:"size:512;comment:完整User-Agent"`

	Browser string `gorm:"size:50;comment:浏览器"`

	BrowserVersion string `gorm:"size:30;comment:浏览器版本"`

	OS string `gorm:"size:50;comment:操作系统"`

	DeviceType uint8 `gorm:"comment:设备类型：1PC 2Android 3iPhone 4iPad 5Mac"`

	Reason string `gorm:"size:255;comment:失败原因或备注"`

	TokenID string `gorm:"size:64;comment:RefreshToken唯一标识(JTI)"`

	CreatedAt time.Time `gorm:"index:idx_time;autoCreateTime;comment:登录时间"`
}
