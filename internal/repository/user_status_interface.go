package repository

// UserStatusRepository 定义用户状态查询能力，供 Service 判断账号是否仍可使用。
type UserStatusRepository interface {
	// GetUserStatus 查询用户状态；参数为用户 ID；返回状态值和数据库错误。
	GetUserStatus(userID uint64) (uint64, error)
}
