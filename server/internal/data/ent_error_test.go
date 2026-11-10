package data

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

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

// entBuilderError 经真实生成 builder 产生错误链，避免伪造 Ent 私有 wrap 字段。
func entBuilderError(t *testing.T, driverErr error) error {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	mock.ExpectExec(`INSERT INTO "currencies"`).WillReturnError(driverErr)
	_, err = client.Currency.Create().SetCode("USD").SetName("测试币种").SetSearchKeywords("测试币种").Save(context.Background())
	if err == nil || !errors.Is(err, driverErr) {
		t.Fatalf("生成 builder 应保留驱动错误: %v", err)
	}
	if mockErr := mock.ExpectationsWereMet(); mockErr != nil {
		t.Fatal(mockErr)
	}
	return err
}

func TestMapEntConstraint(t *testing.T) {
	const name = "financebill_organization_id_idempotency_key"
	domainErr := errors.New("领域约束冲突")
	pgErr := &pgconn.PgError{Code: "23505", ConstraintName: name, Message: "duplicate key value violates unique constraint"}
	constraintErr := entBuilderError(t, pgErr)
	if !ent.IsConstraintError(constraintErr) {
		t.Fatalf("真实 builder 未包装唯一约束: %v", constraintErr)
	}
	wrongNameErr := entBuilderError(t, &pgconn.PgError{
		Code: "23505", ConstraintName: name + "_other", Message: "duplicate key value violates unique constraint " + name,
	})
	emptyNameErr := entBuilderError(t, &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"})
	unknownDriverErr := entBuilderError(t, errors.New("UNIQUE constraint failed: "+name))
	nonConstraintStateErr := entBuilderError(t, &pgconn.PgError{
		Code: "08006", ConstraintName: name, Message: "violates unique constraint " + name,
	})
	notNullErr := entBuilderError(t, &pgconn.PgError{
		Code: "23502", ConstraintName: name, Message: "null value in column violates not-null constraint",
	})
	wrappedErr := fmt.Errorf("仓储外层: %w", constraintErr)
	rawErr := errors.New("connection failure mentioning " + name)
	tests := []struct {
		name       string
		err        error
		constraint string
		domain     error
		want       error
	}{
		{name: "无错误", constraint: name, domain: domainErr},
		{name: "完整名称", err: constraintErr, constraint: name, domain: domainErr, want: domainErr},
		{name: "外层包装", err: wrappedErr, constraint: name, domain: domainErr, want: domainErr},
		{name: "名字仅子串", err: constraintErr, constraint: "idempotency", domain: domainErr, want: constraintErr},
		{name: "不同约束文案含目标名", err: wrongNameErr, constraint: name, domain: domainErr, want: wrongNameErr},
		{name: "空配置名称", err: constraintErr, domain: domainErr, want: constraintErr},
		{name: "空领域错误", err: constraintErr, constraint: name, want: constraintErr},
		{name: "驱动约束名为空", err: emptyNameErr, constraint: name, domain: domainErr, want: emptyNameErr},
		{name: "驱动错误没有Ent包装", err: pgErr, constraint: name, domain: domainErr, want: pgErr},
		{name: "Ent错误没有驱动信息", err: &ent.ConstraintError{}, constraint: name, domain: domainErr},
		{name: "未知驱动不解析文本", err: unknownDriverErr, constraint: name, domain: domainErr, want: unknownDriverErr},
		{name: "非约束SQLSTATE不受文案欺骗", err: nonConstraintStateErr, constraint: name, domain: domainErr, want: nonConstraintStateErr},
		{name: "非空约束不在当前Ent包装边界", err: notNullErr, constraint: name, domain: domainErr, want: notNullErr},
		{name: "其他错误文案含目标名", err: rawErr, constraint: name, domain: domainErr, want: rawErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := test.want
			if want == nil && test.err != nil {
				want = test.err
			}
			if got := mapEntConstraint(test.err, test.constraint, test.domain); got != want {
				t.Fatalf("mapEntConstraint() = %v，期望 %v", got, want)
			}
		})
	}
}

func TestMapEntConstraintSupportedSQLStates(t *testing.T) {
	const name = "known_constraint"
	domainErr := errors.New("领域约束冲突")
	for _, test := range []struct{ code, message string }{
		{code: "23505", message: "duplicate key value violates unique constraint"},
		{code: "23503", message: "insert or update violates foreign key constraint"},
		{code: "23514", message: "new row violates check constraint"},
	} {
		t.Run(test.code, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: test.code, ConstraintName: name, Message: test.message}
			err := entBuilderError(t, pgErr)
			if !ent.IsConstraintError(err) {
				t.Fatalf("当前 Ent 应包装该 SQLSTATE: %v", err)
			}
			if got := mapEntConstraint(err, name, domainErr); got != domainErr {
				t.Fatalf("mapEntConstraint() = %v，期望领域错误", got)
			}
		})
	}
}

func TestMapEntConstraints(t *testing.T) {
	const name = "financebill_organization_id_idempotency_key"
	constraintErr := entBuilderError(t, &pgconn.PgError{
		Code: "23505", ConstraintName: name, Message: "duplicate key value violates unique constraint",
	})
	domainErr := errors.New("领域约束冲突")
	got := mapEntConstraints(constraintErr,
		entConstraintMapping{name: "idempotency", domainErr: errors.New("不应匹配子串")},
		entConstraintMapping{name: name, domainErr: nil},
		entConstraintMapping{name: name, domainErr: domainErr},
		entConstraintMapping{name: name, domainErr: errors.New("不应覆盖第一个有效映射")},
	)
	if got != domainErr {
		t.Fatalf("多约束映射结果 = %v，期望 %v", got, domainErr)
	}
	for _, mappings := range [][]entConstraintMapping{
		nil,
		{{name: "", domainErr: domainErr}},
		{{name: "unknown", domainErr: domainErr}},
	} {
		if got := mapEntConstraints(constraintErr, mappings...); got != constraintErr {
			t.Fatalf("未命中配置时不应改写错误: %v", got)
		}
	}
	rawErr := errors.New("数据库连接失败")
	if got := mapEntConstraints(rawErr, entConstraintMapping{name: name, domainErr: domainErr}); got != rawErr {
		t.Fatalf("非约束错误不应改写: %v", got)
	}
}
