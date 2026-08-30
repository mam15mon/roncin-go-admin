package data

import "github.com/roncin/roncin-go-admin/server/internal/data/ent"

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
