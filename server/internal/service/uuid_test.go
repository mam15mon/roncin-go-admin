package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestParseUUIDValues(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	values, err := parseUUIDValues([]string{first.String(), second.String(), first.String()}, errors.New("参数错误"))
	if err != nil {
		t.Fatalf("解析合法 UUID 失败: %v", err)
	}
	if len(values) != 3 || values[0] != first || values[1] != second || values[2] != first {
		t.Fatalf("解析结果未保留输入顺序和重复值: %v", values)
	}
}

func TestParseUUIDValuesReturnsCallerError(t *testing.T) {
	invalidErr := errors.New("所属领域参数错误")
	if _, err := parseUUIDValues([]string{"invalid"}, invalidErr); err != invalidErr {
		t.Fatalf("解析错误 = %v，期望调用方领域错误", err)
	}
}

func TestParseUniqueUUIDValues(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	values, err := parseUniqueUUIDValues([]string{first.String(), second.String(), first.String()}, errors.New("参数错误"))
	if err != nil {
		t.Fatalf("解析合法 UUID 失败: %v", err)
	}
	if len(values) != 2 || values[0] != first || values[1] != second {
		t.Fatalf("去重结果 = %v，期望按首次出现顺序保留", values)
	}
}

func TestParseTrimmedUUIDValues(t *testing.T) {
	value := uuid.New()
	values, err := parseTrimmedUUIDValues([]string{"  " + value.String() + "\n"}, errors.New("参数错误"))
	if err != nil {
		t.Fatalf("解析带空白 UUID 失败: %v", err)
	}
	if len(values) != 1 || values[0] != value {
		t.Fatalf("解析结果 = %v，期望 %s", values, value)
	}
}

func TestUUIDStringPtr(t *testing.T) {
	if uuidStringPtr(nil) != nil {
		t.Fatal("空 UUID 指针应保持为空")
	}
	zero := uuid.Nil
	if uuidStringPtr(&zero) != nil {
		t.Fatal("全零 UUID 应按未设置处理")
	}
	value := uuid.New()
	formatted := uuidStringPtr(&value)
	if formatted == nil || *formatted != value.String() {
		t.Fatalf("UUID 字符串指针 = %v，期望 %s", formatted, value)
	}
}
