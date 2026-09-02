package middleware

import (
	"blog/internal/utils"
	"blog/pkg/code"
	"blog/pkg/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type sessionRepoStub struct {
	value string
}

type userAccessCheckerStub struct {
	active bool
	err    error
}

// IsUserActive 返回测试用户是否可用；参数为用户 ID；返回预设状态和错误。
func (s *userAccessCheckerStub) IsUserActive(uint64) (bool, error) {
	return s.active, s.err
}

func (s *sessionRepoStub) SaveUserSession(uint64, string, time.Duration) error {
	return nil
}

func (s *sessionRepoStub) GetUserSession(uint64) (string, error) {
	return s.value, nil
}

func TestJWTAuthRejectsReplacedSession(t *testing.T) {
	redisMiddleRepo = &sessionRepoStub{value: "session-new"}
	userAccessChecker = &userAccessCheckerStub{active: true}
	token, err := utils.GenerateUserTokenWithSession(7, time.Minute, "access", "session-old")
	if err != nil {
		t.Fatalf("GenerateUserTokenWithSession returned error: %v", err)
	}

	r := newTestRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !containsResponseCode(recorder.Body.Bytes(), 1008) {
		t.Fatalf("expected session replaced response, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestJWTAuthAcceptsCurrentSession(t *testing.T) {
	redisMiddleRepo = &sessionRepoStub{value: "session-current"}
	userAccessChecker = &userAccessCheckerStub{active: true}
	token, err := utils.GenerateUserTokenWithSession(7, time.Minute, "access", "session-current")
	if err != nil {
		t.Fatalf("GenerateUserTokenWithSession returned error: %v", err)
	}

	r := newTestRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !containsResponseCode(recorder.Body.Bytes(), 0) {
		t.Fatalf("expected protected response, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestJWTAuthRejectsBannedUser(t *testing.T) {
	redisMiddleRepo = &sessionRepoStub{value: "session-current"}
	userAccessChecker = &userAccessCheckerStub{active: false}
	token, err := utils.GenerateUserTokenWithSession(7, time.Minute, "access", "session-current")
	if err != nil {
		t.Fatalf("GenerateUserTokenWithSession returned error: %v", err)
	}

	r := newTestRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !containsResponseCode(recorder.Body.Bytes(), code.UserBanned.BizCode) {
		t.Fatalf("expected banned user response, got %d %s", recorder.Code, recorder.Body.String())
	}
}

// TestJWTAuthRejectsAccessTokenWithoutSessionID 确保没有 PC 会话 ID 的旧访问令牌不能绕过单点登录校验。
func TestJWTAuthRejectsAccessTokenWithoutSessionID(t *testing.T) {
	redisMiddleRepo = &sessionRepoStub{value: "session-current"}
	userAccessChecker = &userAccessCheckerStub{active: true}
	token, err := utils.GenerateUserToken(7, time.Minute, "access")
	if err != nil {
		t.Fatalf("GenerateUserToken returned error: %v", err)
	}

	r := newTestRouter()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !containsResponseCode(recorder.Body.Bytes(), code.SessionReplaced.BizCode) {
		t.Fatalf("expected token without session ID to be rejected, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func newTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/protected", JWTAuth(), func(c *gin.Context) {
		response.SuccessWithMsg("ok", c)
	})
	return r
}

func containsResponseCode(body []byte, expected int) bool {
	var result response.Response
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Code == expected
}
