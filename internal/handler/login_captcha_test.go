package handler

import (
	"blog/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestLoginRejectsInvalidCaptchaBeforePasswordCheck 验证无效图形验证码会在账号密码校验前被拒绝；无参数；通过测试结果返回业务码校验结论。
func TestLoginRejectsInvalidCaptchaBeforePasswordCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(service.New(nil, nil, nil))
	r := gin.New()
	r.POST("/api/login", h.User.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":"user@example.com","password":"password","captcha_id":"missing","captcha_code":"abcd"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":2006`) {
		t.Fatalf("expected invalid-captcha business code: %s", w.Body.String())
	}
}
