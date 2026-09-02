package utils

import (
	"html"
	"regexp"
	"strings"
)

var searchHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)

// BuildArticleSearchExcerpt 按正文、摘要、标题顺序生成命中片段；参数为文章字段、搜索词，返回适合列表展示的纯文本片段。
func BuildArticleSearchExcerpt(title, summary, content, keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}

	for _, source := range []string{content, summary, title} {
		if excerpt := searchTextExcerpt(source, keyword, 120); excerpt != "" {
			return excerpt
		}
	}
	return truncateSearchText(cleanSearchText(summary), 120)
}

// searchTextExcerpt 截取关键词附近的文本；参数为原文、关键词和最大字符数，返回带省略号的片段或空字符串。
func searchTextExcerpt(source, keyword string, maxRunes int) string {
	cleanText := cleanSearchText(source)
	textRunes := []rune(strings.ToLower(cleanText))
	keywordRunes := []rune(strings.ToLower(keyword))
	if len(textRunes) == 0 || len(keywordRunes) == 0 || len(keywordRunes) > len(textRunes) {
		return ""
	}

	matchStart := -1
	for index := 0; index <= len(textRunes)-len(keywordRunes); index++ {
		matched := true
		for offset := range keywordRunes {
			if textRunes[index+offset] != keywordRunes[offset] {
				matched = false
				break
			}
		}
		if matched {
			matchStart = index
			break
		}
	}
	if matchStart < 0 {
		return ""
	}

	if maxRunes < len(keywordRunes) {
		maxRunes = len(keywordRunes)
	}
	windowStart := maxInt(matchStart-(maxRunes-len(keywordRunes))/2, 0)
	windowEnd := minInt(windowStart+maxRunes, len([]rune(cleanText)))
	if windowEnd-windowStart < maxRunes {
		windowStart = maxInt(windowEnd-maxRunes, 0)
	}

	excerpt := string([]rune(cleanText)[windowStart:windowEnd])
	if windowStart > 0 {
		excerpt = "..." + excerpt
	}
	if windowEnd < len([]rune(cleanText)) {
		excerpt += "..."
	}
	return excerpt
}

// cleanSearchText 去除富文本标签并规范空白；参数为文章原文，返回可直接展示的纯文本。
func cleanSearchText(source string) string {
	plainText := searchHTMLTagPattern.ReplaceAllString(source, " ")
	plainText = html.UnescapeString(plainText)
	return strings.Join(strings.Fields(plainText), " ")
}

// truncateSearchText 截取文本前 maxRunes 个字符；参数为纯文本和长度上限，返回截断后的文本。
func truncateSearchText(source string, maxRunes int) string {
	textRunes := []rune(source)
	if len(textRunes) <= maxRunes {
		return source
	}
	return string(textRunes[:maxRunes]) + "..."
}

// maxInt 返回两个整数中的较大值；参数为两个整数，返回较大整数。
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

// minInt 返回两个整数中的较小值；参数为两个整数，返回较小整数。
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
