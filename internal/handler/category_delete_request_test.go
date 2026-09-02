package handler

import (
	"blog/models/dto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestDeleteCategoryRequestBindsSafeDeletionFields 验证删除分类请求可接收强制确认与迁移目标参数；无参数；返回绑定后的请求结构或测试失败。
func TestDeleteCategoryRequestBindsSafeDeletionFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodDelete, "/api/admin/categories/1", strings.NewReader(`{"force":true,"confirm_text":"确认删除","target_category_id":2}`))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	var req dto.DeleteCategoryReq
	if err := context.ShouldBindJSON(&req); err != nil {
		t.Fatalf("expected delete category payload to bind: %v", err)
	}
	if !req.Force || req.ConfirmText != "确认删除" || req.TargetCategoryID != 2 {
		t.Fatalf("unexpected delete category request: %#v", req)
	}
}
