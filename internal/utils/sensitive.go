package utils

import (
	"errors"
	"strings"

	"github.com/importcjj/sensitive"
	"go.uber.org/zap"
)

var filter *sensitive.Filter

// InitSensitive 初始化敏感词库
func InitSensitive(dictPath string) {
	filter = sensitive.New()
	if err := filter.LoadWordDict(dictPath); err != nil {
		zap.L().Error("加载敏感词词典失败: %w" + err.Error())
	}
}

// Check 检查一个字符串
func Check(text string) (bool, error) {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return false, errors.New("敏感词过滤器未初始化")
	}

	words := filter.FindAll(text)
	if len(words) == 0 {
		return false, nil
	}

	return true, errors.New("包含敏感词：" + strings.Join(words, "、"))
}

// CheckMulti 检查多个字符串
func CheckMulti(texts ...string) error {
	for _, text := range texts {
		if re, err := Check(text); re == true && err != nil {
			return err
		}
	}
	return nil
}

// Has 是否包含敏感词
func Has(text string) bool {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return false
	}

	return len(filter.FindAll(text)) > 0
}

// Replace 替换敏感词
func Replace(text string) string {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return text
	}

	return filter.Replace(text, '*')
}

// FindAll 返回所有命中的敏感词
func FindAll(text string) []string {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return nil
	}

	return filter.FindAll(text)
}

// AddWord添加初始词
func AddWord(word string) {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return
	}
	filter.AddWord(word)
}

// DelWord删除初始词
func DelWord(word string) {
	if filter == nil {
		zap.L().Error("敏感词过滤器未初始化")
		return
	}
	filter.DelWord(word)
}
