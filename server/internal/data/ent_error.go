package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

type entConstraintMapping struct {
	name      string
	domainErr error
}

// mapEntError 将 Ent 标准查询和约束错误映射为调用方指定的领域错误。
func mapEntError(err, notFoundErr, constraintErr error) error {
	if err == nil {
		return nil
	}
	if notFoundErr != nil && ent.IsNotFound(err) {
		return notFoundErr
	}
	if constraintErr != nil && ent.IsConstraintError(err) {
		return constraintErr
	}
	return err
}

// mapEntConstraint 仅在命中指定数据库约束名时返回对应领域错误。
func mapEntConstraint(err error, constraintName string, constraintErr error) error {
	if constraintName == "" || constraintErr == nil || !ent.IsConstraintError(err) {
		return err
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != constraintName {
		return err
	}
	// 当前 Ent 生成器包装唯一、外键和 CHECK 约束；其他 SQLSTATE 不按命名约束映射。
	switch pgErr.Code {
	case "23505", "23503", "23514":
		return constraintErr
	default:
		return err
	}
}

// mapEntConstraints 按顺序尝试多个命名约束映射，未命中时保留原错误。
func mapEntConstraints(err error, mappings ...entConstraintMapping) error {
	for _, mapping := range mappings {
		err = mapEntConstraint(err, mapping.name, mapping.domainErr)
	}
	return err
}
