package handler

import (
	"blog/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWechatLoginRejectsEmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(service.New(nil, nil, nil))
	r := gin.New()
	r.POST("/api/wechat/login", h.User.WechatLogin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/wechat/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":1001`) {
		t.Fatalf("expected bad-request business code: %s", w.Body.String())
	}
}

func TestWechatPhoneRejectsMissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(service.New(nil, nil, nil))
	r := gin.New()
	r.POST("/api/wechat/phone", h.User.CompleteWechatPhoneLogin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/wechat/phone", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":1001`) {
		t.Fatalf("expected bad-request business code: %s", w.Body.String())
	}
}
