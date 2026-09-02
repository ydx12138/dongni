package middleware

import (
	"blog/internal/repository"
	"blog/internal/utils"
	"blog/pkg/code"
	"blog/pkg/response"
	"crypto/subtle"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var redisMiddleRepo repository.RedisMiddleRepository

type UserAccessChecker interface {
	// IsUserActive 检查用户账号是否可用；参数为用户 ID；返回可用状态和查询错误。
	IsUserActive(userID uint64) (bool, error)
}

var userAccessChecker UserAccessChecker

func SetRedisRepo(authMiddleRepo repository.RedisMiddleRepository) {
	redisMiddleRepo = authMiddleRepo
}

// SetUserAccessChecker 注入用户状态检查器；参数为 Service 提供的状态检查能力；无返回值。
func SetUserAccessChecker(checker UserAccessChecker) {
	userAccessChecker = checker
}

// 用户token
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		//从context中得到token
		token := utils.GetTokenFromContext(c)
		if token == "" {
			zap.L().Info("从context中没有得到token")
			response.ErrWithMsg(code.Unauthorized, c)
			c.Abort()
			return
		}
		//解析token，得到Data
		data, err := utils.GetDataFromToken(token)
		if data == nil || err != nil {
			zap.L().Info("从token中没有得到Data")
			response.ErrWithMsg(code.Unauthorized, c)
			c.Abort()
			return
		}
		//解析data，得到claim
		var claim *utils.CustomClaims
		if claim = utils.GetClaimFromData(data); claim == nil {
			zap.L().Info("从data中没有得到claim")
			response.ErrWithMsg(code.Unauthorized, c)
			c.Abort()
			return
		}
		//如果type==access,data.Valid==true有效，则通过
		if claim.Type == "access" && data.Valid {
			if claim.SessionID == "" || redisMiddleRepo == nil {
				response.ErrWithMsg(code.SessionReplaced, c)
				c.Abort()
				return
			}
			currentSession, err := redisMiddleRepo.GetUserSession(claim.UserID)
			if err != nil {
				zap.L().Error("read pc login session failed: " + err.Error())
				response.ErrWithMsg(code.InternalError, c)
				c.Abort()
				return
			}
			if currentSession == "" || subtle.ConstantTimeCompare([]byte(currentSession), []byte(claim.SessionID)) != 1 {
				response.ErrWithMsg(code.SessionReplaced, c)
				c.Abort()
				return
			}
			if userAccessChecker == nil {
				response.ErrWithMsg(code.InternalError, c)
				c.Abort()
				return
			}
			active, err := userAccessChecker.IsUserActive(claim.UserID)
			if err != nil {
				zap.L().Error("check user status failed: " + err.Error())
				response.ErrWithMsg(code.InternalError, c)
				c.Abort()
				return
			}
			if !active {
				response.ErrWithMsg(code.UserBanned, c)
				c.Abort()
				return
			}
			c.Set("userID", claim.UserID)
			c.Set("role", claim.Role)
			c.Set("type", claim.Type)
			return
		}
		//如果type==access,无效，401，Abort+return
		if claim.Type == "access" && data.Valid == false {
			zap.L().Info("accessToken无效")
			response.ErrWithMsg(code.AccessTokenExpired, c)
			c.Abort()
			return
		}
	}

}

// 检测管理员token
func JWTAuthForAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := utils.ParseClaims(c)
		if !ok {
			return
		}
		if claims.Role != "admin" {
			response.ErrWithMsg(code.Forbidden, c)
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

//双token
