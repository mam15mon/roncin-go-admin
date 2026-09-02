package service

import (
	"strings"

	"github.com/google/uuid"
)

// parseUUIDValues 按输入顺序解析 UUID，解析失败时返回调用方所属领域错误。
func parseUUIDValues(values []string, invalidErr error) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, invalidErr
		}
		result = append(result, id)
	}
	return result, nil
}

// parseUniqueUUIDValues 按首次出现顺序返回去重后的 UUID。
func parseUniqueUUIDValues(values []string, invalidErr error) ([]uuid.UUID, error) {
	parsed, err := parseUUIDValues(values, invalidErr)
	if err != nil {
		return nil, err
	}
	result := make([]uuid.UUID, 0, len(parsed))
	seen := make(map[uuid.UUID]struct{}, len(parsed))
	for _, id := range parsed {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

// parseTrimmedUUIDValues 在解析前移除输入两端空白。
func parseTrimmedUUIDValues(values []string, invalidErr error) ([]uuid.UUID, error) {
	trimmed := make([]string, len(values))
	for index, value := range values {
		trimmed[index] = strings.TrimSpace(value)
	}
	return parseUUIDValues(trimmed, invalidErr)
}

func uuidStringPtr(value *uuid.UUID) *string {
	if value == nil || *value == uuid.Nil {
		return nil
	}
	result := value.String()
	return &result
}
