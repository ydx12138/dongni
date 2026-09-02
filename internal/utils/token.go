package utils

import (
	"blog/pkg/code"
	"blog/pkg/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// 密钥
var SecretKey = []byte("your-secret-key")

type CustomClaims struct {
	UserID               uint64 `json:"user_id"` //要放到token里的信息
	Role                 string `json:"role"`
	Type                 string `json:"type"`
	SessionID            string `json:"session_id,omitempty"`
	jwt.RegisteredClaims        //jwt标准字段
}

// 生成Admintoken
func GenerateAdminToken(userID uint64, duration time.Duration) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // 7天过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),               //签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),               //生效时间
			Issuer:    "blog",                                       //发行者
		},
	}
	//token格式 ==> header.payload.signature
	//SigningMethodHS256决定header里的alg
	//claims决定payload
	//SignedString决定signature
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey) //使用密钥加密
}

// 生成Usertoken
func GenerateUserToken(userID uint64, duration time.Duration, typel string) (string, error) {
	return GenerateUserTokenWithSession(userID, duration, typel, "")
}

// GenerateUserTokenWithSession 生成带 PC 会话 ID 的用户 JWT。
// 参数：userID 为用户 ID，duration 为 Token 有效期，typel 为 access 或 refresh，sessionID 为当前登录会话 ID；返回 JWT 字符串和生成错误。
func GenerateUserTokenWithSession(userID uint64, duration time.Duration, typel, sessionID string) (string, error) {
	claims := CustomClaims{
		UserID:    userID,
		Role:      "user",
		Type:      typel,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // 7天过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),               //签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),               //生效时间
			Issuer:    "blog",                                       //发行者
		},
	}
	//token格式 ==> header.payload.signature
	//SigningMethodHS256决定header里的alg
	//claims决定payload
	//SignedString决定signature
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SecretKey) //使用密钥加密
}

// 校验token是否有效，如果有效，返回claim
func ParseToken(tokenString string) (*CustomClaims, error) {
	//从token里解析出信息，并验证签名
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return SecretKey, nil
		},
	)

	if err != nil {
		return nil, err
	}
	//验证签名是否还有效
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	//无效则返回err
	return nil, jwt.ErrTokenInvalidClaims
}

// parseClaims 从请求 Header 中解析 JWT Token，并返回解析后的自定义 Claims
// 返回值：
//   - *utils.CustomClaims：解析成功后的用户信息
//   - bool：是否解析成功（true=成功，false=失败）
func ParseClaims(c *gin.Context) (*CustomClaims, bool) {
	// 从请求头中获取 Authorization 字段
	// 一般格式：Bearer xxx.yyy.zzz
	tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
	// 如果没有携带 token，直接返回未授权
	if tokenStr == "" {
		response.ErrWithMsg(code.Unauthorized, c)
		c.Abort()
		return nil, false
	}
	// 兼容 "Bearer xxx" 格式（忽略大小写）
	// 如果有 Bearer 前缀，则截取真实 token
	if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		tokenStr = strings.TrimSpace(tokenStr[7:])
	}
	// 解析 JWT Token，获取 claims（用户信息 + 自定义字段）
	claims, err := ParseToken(tokenStr)
	if err != nil {
		// token 无效或过期，返回未授权
		response.ErrWithMsg(code.Unauthorized, c)
		c.Abort()
		return nil, false
	}
	// 解析成功，返回 claims
	return claims, true
}

//	func GetToken(c *gin.Context) string {
//		tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
//		// 无token，代表未授权
//		if tokenStr == "" {
//			return ""
//		}
//		// 兼容 "Bearer xxx" 格式（忽略大小写）
//		if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
//			tokenStr = strings.TrimSpace(tokenStr[7:])
//		}
//		return tokenStr
//		/*// 解析 JWT Token，获取 claims（用户信息 + 自定义字段）
//		claims, err := ParseToken(tokenStr)
//		if err != nil {
//			// token 无效或过期，返回未授权
//			response.ErrWithMsg(code.Unauthorized, c)
//			c.Abort()
//			return nil, false
//		}
//		// 解析成功，返回 claims
//		return claims, true*/
//	}
//
// // 根据token获得claim
//
//	func GetClaimFromToken(token string) (*CustomClaims, error) {
//		// 解析 JWT Token，获取 claims
//		claims, err := ParseToken(token)
//		if err != nil {
//			// token 无效或过期，返回未授权
//			return nil, err
//		}
//		// 解析成功，返回 claims
//		return claims, nil
//	}
func GetTokenFromContext(c *gin.Context) string {
	token := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return token
}

func GetDataFromToken(token string) (*jwt.Token, error) {
	//从token里解析出data
	data, err := jwt.ParseWithClaims(
		token,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return SecretKey, nil
		},
	)

	if err != nil {
		zap.L().Error("GetDataFromToken:" + err.Error())
		return nil, err
	}
	return data, nil
}

func GetClaimFromData(data *jwt.Token) *CustomClaims {
	if claim, ok := data.Claims.(*CustomClaims); ok {
		return claim
	}
	return nil
}
