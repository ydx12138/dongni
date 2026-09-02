package code

import "errors"

type ErrorCode struct {
	HttpCode int
	BizCode  int
	Message  string
}

func ErrorMsg(err error) ErrorCode {
	return ErrorCode{
		HttpCode: 200,
		BizCode:  0,
		Message:  err.Error(),
	}
}

var (
	Success             = ErrorCode{200, 0, "success"}
	BadRequest          = ErrorCode{200, 1001, "参数错误"}
	Unauthorized        = ErrorCode{200, 1002, "请先登录"}
	Forbidden           = ErrorCode{200, 1003, "无权限"}
	NotFound            = ErrorCode{200, 1004, "资源不存在"}
	InternalError       = ErrorCode{200, 1005, "服务器内部错误"}
	AccessTokenExpired  = ErrorCode{200, 1006, "access_token过期"}
	RefreshTokenExpired = ErrorCode{200, 1007, "refresh_token过期"}
	SessionReplaced     = ErrorCode{200, 1008, "账号已在其他地方登录"}
	UserBanned          = ErrorCode{200, 1010, "账号已被封禁"}
)

var FeatureDisabled = ErrorCode{200, 1009, "功能暂未开放"}

var (
	ErrUserExist        = ErrorCode{200, 2001, "用户已存在"}
	ErrUserNotFound     = ErrorCode{200, 2002, "用户不存在"}
	ErrPassword         = ErrorCode{200, 2003, "密码错误"}
	EmailRepeat         = ErrorCode{200, 2004, "邮箱已经存在"}
	ErrVerificationCode = ErrorCode{200, 2005, "验证码错误或已过期"}
	ErrCaptcha          = ErrorCode{200, 2006, "图形验证码错误或已过期"}
)

var (
	ErrArticleNotFound = ErrorCode{200, 3001, "文章不存在"}
)

var (
	ErrCommentNotFound = ErrorCode{200, 4001, "评论不存在"}
)

var (
	ErrFileType     = ErrorCode{200, 5001, "不支持的文件类型"}
	ErrFileTooLarge = ErrorCode{200, 5002, "文件大小超过限制"}
)

var (
	ErrNoPermission = ErrorCode{200, 6001, "无操作权限"}
)

var (
	ErrNotHaveToken = errors.New("token为空")
)
