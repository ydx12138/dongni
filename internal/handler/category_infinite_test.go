package handler

import (
	"blog/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetCategoryArticlesPageRejectsMissingCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(service.New(nil, nil, nil))
	r := gin.New()
	r.GET("/api/categories/articles/page", h.User.GetCategoryArticlesPage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/categories/articles/page?page=1&page_size=10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":1001`) {
		t.Fatalf("expected bad-request business code: %s", w.Body.String())
	}
}
