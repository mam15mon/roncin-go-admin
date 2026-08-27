// Package searchtext 生成供服务端联想查询使用的拼音检索文本。
package searchtext

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// Build 为一组名称生成无声调全拼、分词全拼与拼音首字母。
// 原始名称仍由各仓储的结构化字段查询，不重复写入检索键。
func Build(values ...string) string {
	seen := make(map[string]struct{})
	parts := make([]string, 0, len(values)*3)
	for _, value := range values {
		normalized := strings.ToUpper(strings.Join(strings.Fields(value), " "))
		if normalized != "" {
			if _, exists := seen[normalized]; !exists {
				seen[normalized] = struct{}{}
				parts = append(parts, normalized)
			}
		}
		syllables := pinyin.LazyPinyin(value, pinyin.NewArgs())
		if len(syllables) == 0 {
			continue
		}
		full := strings.ToUpper(strings.Join(syllables, ""))
		spaced := strings.ToUpper(strings.Join(syllables, " "))
		initials := strings.Builder{}
		for _, syllable := range syllables {
			for _, letter := range syllable {
				initials.WriteRune(unicode.ToUpper(letter))
				break
			}
		}
		for _, candidate := range []string{full, spaced, initials.String()} {
			if candidate == "" {
				continue
			}
			if _, exists := seen[candidate]; exists {
				continue
			}
			seen[candidate] = struct{}{}
			parts = append(parts, candidate)
		}
	}
	return strings.Join(parts, " ")
}
