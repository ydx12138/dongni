package utils

import (
	"strings"
	"testing"
)

// TestBuildArticleSearchExcerptPrefersContent 验证正文命中时优先返回正文附近片段；参数为固定文章文本，返回测试断言结果。
func TestBuildArticleSearchExcerptPrefersContent(t *testing.T) {
	excerpt := BuildArticleSearchExcerpt(
		"标题没有关键词",
		"摘要中的关键词位置",
		"正文开头内容。这里是 Redis 缓存实践的命中位置，后面还有更多正文。",
		"Redis",
	)

	if !strings.Contains(excerpt, "Redis") {
		t.Fatalf("expected excerpt to contain keyword, got %q", excerpt)
	}
	if strings.Contains(excerpt, "摘要中的") {
		t.Fatalf("expected content excerpt to take priority, got %q", excerpt)
	}
}

// TestBuildArticleSearchExcerptCleansRichText 验证富文本标签不会出现在搜索片段中；参数为带标签正文，返回测试断言结果。
func TestBuildArticleSearchExcerptCleansRichText(t *testing.T) {
	excerpt := BuildArticleSearchExcerpt(
		"",
		"",
		"<p>这是 <strong>Docker</strong> 部署博客的正文。</p>",
		"Docker",
	)

	if strings.Contains(excerpt, "<strong>") || !strings.Contains(excerpt, "Docker") {
		t.Fatalf("expected clean rich-text excerpt, got %q", excerpt)
	}
}

// TestBuildArticleSearchExcerptFallsBackToTitle 验证正文和摘要均未命中时回退到标题；参数为固定文章文本，返回测试断言结果。
func TestBuildArticleSearchExcerptFallsBackToTitle(t *testing.T) {
	excerpt := BuildArticleSearchExcerpt("Vue 组件设计", "普通摘要", "普通正文", "Vue")
	if !strings.Contains(excerpt, "Vue") {
		t.Fatalf("expected title fallback excerpt, got %q", excerpt)
	}
}
