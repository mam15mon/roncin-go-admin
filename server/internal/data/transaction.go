package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

type transaction interface {
	Commit() error
	Rollback() error
}

func runTransaction[T transaction](begin func() (T, error), operation func(T) error) (err error) {
	tx, err := begin()
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()
	if err := operation(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	return tx.Commit()
}

// WithTx 在统一的 Ent 事务生命周期内执行操作。
func (d *Data) WithTx(ctx context.Context, operation func(*ent.Tx) error) error {
	return runTransaction(func() (*ent.Tx, error) {
		return d.db.Tx(ctx)
	}, operation)
}

func (d *Data) withSQLTx(ctx context.Context, operation func(*sql.Tx) error) error {
	return runTransaction(func() (*sql.Tx, error) {
		return d.sqlDB.BeginTx(ctx, nil)
	}, operation)
}
