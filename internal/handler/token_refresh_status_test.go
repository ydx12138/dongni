package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestTokenRefreshChecksUserStatus 确保刷新令牌前校验用户是否仍可登录。
func TestTokenRefreshChecksUserStatus(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "handler.go", nil, 0)
	if err != nil {
		t.Fatalf("parse handler.go: %v", err)
	}
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "TokenRefresh" {
			target = function
			break
		}
	}
	if target == nil {
		t.Fatal("TokenRefresh function not found")
	}
	found := false
	ast.Inspect(target, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "IsUserActive" {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("TokenRefresh must call IsUserActive before issuing a new access token")
	}
}

// TestTokenRefreshRequiresSessionValidation 确保刷新令牌必须校验 PC 会话，旧令牌不能绕过单点登录。
func TestTokenRefreshRequiresSessionValidation(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "handler.go", nil, 0)
	if err != nil {
		t.Fatalf("parse handler.go: %v", err)
	}
	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "TokenRefresh" {
			target = function
			break
		}
	}
	if target == nil {
		t.Fatal("TokenRefresh function not found")
	}
	found := false
	ast.Inspect(target, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "ValidateSession" {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("TokenRefresh must validate the PC session before issuing a new access token")
	}
}
