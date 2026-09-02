package response

import (
	"blog/pkg/code"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`    // 业务错误码，0 表示成功
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 响应数据
}

// Success 成功响应
func Success(data any, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMsg 成功响应（自定义msg）
func SuccessWithMsg(msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: msg,
		Data:    "",
	})
}

// SuccessWithData 成功响应（自定义data）
func SuccessWithData(data any, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "成功",
		Data:    data,
	})
}

// ErrWithMsg 失败响应（预定义code和msg）
func ErrWithMsg(e code.ErrorCode, c *gin.Context) {
	c.JSON(e.HttpCode, Response{
		Code:    e.BizCode,
		Message: e.Message,
		Data:    nil,
	})
}
