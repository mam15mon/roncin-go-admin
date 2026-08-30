package data

import (
	"errors"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestMapEntError(t *testing.T) {
	notFoundErr := errors.New("领域对象不存在")
	constraintErr := errors.New("领域约束冲突")
	rawErr := errors.New("数据库连接失败")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "无错误", err: nil, want: nil},
		{name: "对象不存在", err: &ent.NotFoundError{}, want: notFoundErr},
		{name: "约束冲突", err: &ent.ConstraintError{}, want: constraintErr},
		{name: "其他错误保持原样", err: rawErr, want: rawErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mapEntError(test.err, notFoundErr, constraintErr); !errors.Is(got, test.want) {
				t.Fatalf("mapEntError() = %v，期望 %v", got, test.want)
			}
		})
	}
}

func TestMapEntErrorIgnoresUnconfiguredKinds(t *testing.T) {
	notFoundErr := &ent.NotFoundError{}
	constraintErr := &ent.ConstraintError{}
	if got := mapEntError(notFoundErr, nil, errors.New("约束冲突")); got != notFoundErr {
		t.Fatalf("未配置 NotFound 映射时不应改写错误: %v", got)
	}
	if got := mapEntError(constraintErr, errors.New("不存在"), nil); got != constraintErr {
		t.Fatalf("未配置 Constraint 映射时不应改写错误: %v", got)
	}
}
