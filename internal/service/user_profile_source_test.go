package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestCurrentUserProfileDoesNotDependOnPublicProfileSetting 确保登录用户资料查询不受公共“我的”页面开关影响。
func TestCurrentUserProfileDoesNotDependOnPublicProfileSetting(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "service.go", nil, 0)
	if err != nil {
		t.Fatalf("parse service.go: %v", err)
	}

	var target *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "CurrentUserProfile" {
			target = function
			break
		}
	}
	if target == nil {
		t.Fatal("CurrentUserProfile function not found")
	}

	ast.Inspect(target, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "RequireFeatureEnabled" {
			t.Fatal("CurrentUserProfile must not depend on the public profile page setting")
		}
		return true
	})
}
