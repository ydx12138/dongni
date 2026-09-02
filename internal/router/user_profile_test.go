package router

import (
	"blog/config"
	"blog/internal/app"
	"blog/internal/handler"
	"blog/internal/middleware"
	"blog/internal/service"
	"blog/internal/utils"
	"blog/pkg/code"
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type activeUserCheckerStub struct{}

type routerSessionRepoStub struct{}

// IsUserActive 为路由测试返回正常账号状态；参数为用户 ID；返回 true 和 nil。
func (activeUserCheckerStub) IsUserActive(uint64) (bool, error) {
	return true, nil
}

// GetUserSession 为头像路由测试返回当前会话；参数为用户 ID；返回固定会话 ID 和 nil。
func (routerSessionRepoStub) GetUserSession(uint64) (string, error) {
	return "router-test-session", nil
}

// SaveUserSession 满足会话仓储接口；参数为用户 ID、会话 ID 和过期时间；测试中不执行保存并返回 nil。
func (routerSessionRepoStub) SaveUserSession(uint64, string, time.Duration) error {
	return nil
}

func TestCurrentUserProfileRouteRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalConfig := config.Cfg
	config.Cfg = &config.Config{CORS: config.CORSConfig{
		AllowOrigins: []string{"http://localhost"},
		AllowMethods: []string{http.MethodGet},
		AllowHeaders: []string{"Authorization"},
	}}
	t.Cleanup(func() { config.Cfg = originalConfig })
	container := &app.Container{Handler: handler.New(service.New(nil, nil, nil))}
	router := Register(container)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	expectedCode := fmt.Sprintf(`"code":%d`, code.Unauthorized.BizCode)
	if !strings.Contains(recorder.Body.String(), expectedCode) {
		t.Fatalf("expected unauthorized business code: %s", recorder.Body.String())
	}
}

// TestUserAvatarRoutesRejectInvalidInput 确保头像接口在调用 OSS 或数据库前拒绝非法输入。
func TestUserAvatarRoutesRejectInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware.SetUserAccessChecker(activeUserCheckerStub{})
	middleware.SetRedisRepo(routerSessionRepoStub{})
	originalConfig := config.Cfg
	config.Cfg = &config.Config{CORS: config.CORSConfig{
		AllowOrigins: []string{"http://localhost"},
		AllowMethods: []string{http.MethodPost, http.MethodPut},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}}
	t.Cleanup(func() { config.Cfg = originalConfig })
	container := &app.Container{Handler: handler.New(service.New(nil, nil, nil))}
	router := Register(container)
	token, err := utils.GenerateUserTokenWithSession(7, time.Hour, "access", "router-test-session")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	t.Run("invalid avatar URL", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/api/users/avatar", strings.NewReader(`{"avatar":"javascript:alert(1)"}`))
		request.Header.Set("Authorization", token)
		request.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(recorder, request)

		expectedCode := fmt.Sprintf(`"code":%d`, code.BadRequest.BizCode)
		if !strings.Contains(recorder.Body.String(), expectedCode) {
			t.Fatalf("invalid URL should be rejected: %s", recorder.Body.String())
		}
	})

	t.Run("invalid image extension", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "avatar.txt")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		_, _ = part.Write([]byte("not an image"))
		_ = writer.Close()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/users/avatar/upload", body)
		request.Header.Set("Authorization", token)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(recorder, request)

		expectedCode := fmt.Sprintf(`"code":%d`, code.BadRequest.BizCode)
		if !strings.Contains(recorder.Body.String(), expectedCode) {
			t.Fatalf("invalid image should be rejected: %s", recorder.Body.String())
		}
	})

	t.Run("spoofed image content", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "avatar.png")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		_, _ = part.Write([]byte("plain text disguised as png"))
		_ = writer.Close()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/users/avatar/upload", body)
		request.Header.Set("Authorization", token)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(recorder, request)

		expectedCode := fmt.Sprintf(`"code":%d`, code.BadRequest.BizCode)
		if !strings.Contains(recorder.Body.String(), expectedCode) {
			t.Fatalf("spoofed image should be rejected: %s", recorder.Body.String())
		}
	})
}

// TestUserAvatarRoutesRejectMissingToken 确保用户头像上传和保存接口都必须经过用户登录中间件。
func TestUserAvatarRoutesRejectMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalConfig := config.Cfg
	config.Cfg = &config.Config{CORS: config.CORSConfig{
		AllowOrigins: []string{"http://localhost"},
		AllowMethods: []string{http.MethodPost, http.MethodPut},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}}
	t.Cleanup(func() { config.Cfg = originalConfig })
	container := &app.Container{Handler: handler.New(service.New(nil, nil, nil))}
	router := Register(container)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/users/avatar/upload"},
		{method: http.MethodPut, path: "/api/users/avatar"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, nil)

		router.ServeHTTP(recorder, request)

		expectedCode := fmt.Sprintf(`"code":%d`, code.Unauthorized.BizCode)
		if !strings.Contains(recorder.Body.String(), expectedCode) {
			t.Fatalf("%s %s should reject missing token: %s", test.method, test.path, recorder.Body.String())
		}
	}
}
