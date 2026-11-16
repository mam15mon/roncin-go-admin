// Package searchtext 生成供服务端联想查询使用的拼音检索文本。
package searchtext

import (
	"strings"
	"unicode"

	"github.com/go-ego/gpy/phrase"
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
		syllables := phrasePinyin(value)
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

// gpy v0.50.0 的词组最长为 10 个字；只读取词组表，不加载可变分词器。
// 逐段最长匹配使检索键生成保持线性，并保留中文连续片段中的上下文读音。
func phrasePinyin(value string) []string {
	const maxPhraseRunes = 10
	runes := []rune(value)
	result := make([]string, 0, len(runes))
	for i := 0; i < len(runes); {
		if !unicode.Is(unicode.Han, runes[i]) {
			i++
			continue
		}
		end := i + 1
		for end < len(runes) && end-i < maxPhraseRunes && unicode.Is(unicode.Han, runes[end]) {
			end++
		}
		matched := false
		for end > i+1 {
			word := string(runes[i:end])
			reading := phrase.Match(word)
			// 与前端 utils/search.ts 的业务词组读音保持一致。
			switch word {
			case "闵行":
				reading = "min hang"
			case "还箱":
				reading = "huan xiang"
			case "还柜":
				reading = "huan gui"
			}
			if reading != "" {
				result = append(result, strings.Fields(reading)...)
				i = end
				matched = true
				break
			}
			end--
		}
		if matched {
			continue
		}
		result = append(result, pinyin.LazyPinyin(string(runes[i]), pinyin.NewArgs())...)
		i++
	}
	return result
}
