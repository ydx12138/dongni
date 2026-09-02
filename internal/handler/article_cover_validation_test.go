package handler

import (
	"blog/models/dto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCreateArticleRejectsMissingCover 验证创建文章请求缺少封面时不能通过参数绑定。
// 参数：无；返回值：无，通过测试失败报告校验结果。
func TestCreateArticleRejectsMissingCover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	emptyRequest := httptest.NewRequest(http.MethodPost, "/api/admin/articles", strings.NewReader(`{"title":"test","cover":""}`))
	emptyRequest.Header.Set("Content-Type", "application/json")
	emptyContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	emptyContext.Request = emptyRequest
	var emptyCover dto.CreateArticleReq
	if err := emptyContext.ShouldBindJSON(&emptyCover); err == nil {
		t.Fatal("expected empty cover to fail request binding")
	}

	validRequest := httptest.NewRequest(http.MethodPost, "/api/admin/articles", strings.NewReader(`{"title":"test","cover":"https://example.com/cover.jpg"}`))
	validRequest.Header.Set("Content-Type", "application/json")
	validContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	validContext.Request = validRequest
	var validCover dto.CreateArticleReq
	if err := validContext.ShouldBindJSON(&validCover); err != nil {
		t.Fatalf("expected cover to pass request binding: %v", err)
	}
}
