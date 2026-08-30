package service

import "github.com/google/uuid"

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

func uuidStringPtr(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	result := value.String()
	return &result
}
