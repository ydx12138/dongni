package repository

import "time"

type RedisMiddleRepository interface {
	// SaveUserSession 保存用户当前 PC 会话 ID，并设置过期时间。
	SaveUserSession(userID uint64, sessionID string, duration time.Duration) error
	// GetUserSession 读取用户当前 PC 会话 ID；键不存在时返回空字符串和 nil。
	GetUserSession(userID uint64) (string, error)
}
