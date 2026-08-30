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

func TestMapEntConstraint(t *testing.T) {
	constraintErr := &ent.ConstraintError{}
	domainErr := errors.New("领域约束冲突")
	if got := mapEntConstraint(constraintErr, "", domainErr); got != domainErr {
		t.Fatalf("命中约束名时错误 = %v，期望领域错误", got)
	}
	if got := mapEntConstraint(constraintErr, "other_constraint", domainErr); got != constraintErr {
		t.Fatalf("未命中约束名时不应改写错误: %v", got)
	}
	rawErr := errors.New("数据库连接失败")
	if got := mapEntConstraint(rawErr, "", domainErr); got != rawErr {
		t.Fatalf("非约束错误不应改写: %v", got)
	}
}

func TestMapEntConstraints(t *testing.T) {
	constraintErr := &ent.ConstraintError{}
	domainErr := errors.New("领域约束冲突")
	got := mapEntConstraints(constraintErr,
		entConstraintMapping{name: "未命中的约束", domainErr: errors.New("错误映射")},
		entConstraintMapping{name: "", domainErr: domainErr},
	)
	if got != domainErr {
		t.Fatalf("多约束映射结果 = %v，期望 %v", got, domainErr)
	}

	rawErr := errors.New("数据库连接失败")
	if got := mapEntConstraints(rawErr, entConstraintMapping{name: "", domainErr: domainErr}); got != rawErr {
		t.Fatalf("非约束错误不应改写: %v", got)
	}
	if got := mapEntConstraints(constraintErr); got != constraintErr {
		t.Fatalf("无映射配置时不应改写错误: %v", got)
	}
}
