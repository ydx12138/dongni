package router

import (
	"blog/config"
	"blog/internal/app"
	"blog/internal/handler"
	"blog/internal/service"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestArticleRankingRoute 验证点赞排行榜公开接口已注册；无参数；返回接口方法和路径的断言结果。
func TestArticleRankingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalConfig := config.Cfg
	config.Cfg = &config.Config{CORS: config.CORSConfig{AllowOrigins: []string{"http://localhost"}}}
	t.Cleanup(func() { config.Cfg = originalConfig })

	router := Register(&app.Container{Handler: handler.New(service.New(nil, nil, nil))})
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/articles/ranking" {
			return
		}
	}
	t.Fatal("expected GET /api/articles/ranking to be registered")
}

func TestCategoryArticlesPageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalConfig := config.Cfg
	config.Cfg = &config.Config{CORS: config.CORSConfig{AllowOrigins: []string{"http://localhost"}}}
	t.Cleanup(func() { config.Cfg = originalConfig })

	router := Register(&app.Container{Handler: handler.New(service.New(nil, nil, nil))})
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/categories/articles/page" {
			return
		}
	}
	t.Fatal("expected GET /api/categories/articles/page to be registered")
}
