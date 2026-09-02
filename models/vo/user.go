package vo

import "time"

type UserProfile struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	Nickname  string    `json:"nickname"`
	Phone     *string   `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}
type CaptchaResponse struct {
	CaptchaId string `json:"captchaId"` // 验证码 ID
	PicBase64 string `json:"picBase64"` // base64 图片数据（可直接 img.src）
}
