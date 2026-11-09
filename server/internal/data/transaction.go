package data

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

var errTransactionContextClosed = errors.New("事务上下文已经结束")
var errReadSnapshotRequired = errors.New("一致性快照查询必须在写事务提交后执行")

type transactionContextKey struct{}

type transactionContext struct {
	tx           *ent.Tx
	active       atomic.Bool
	readSnapshot bool
}

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
	if transaction, ok := transactionFromContext(ctx); ok {
		if !transaction.active.Load() {
			return errTransactionContextClosed
		}
		return operation(transaction.tx)
	}
	return runTransaction(func() (*ent.Tx, error) {
		return d.db.Tx(ctx)
	}, operation)
}

// WithinTransaction 为 Biz 用例创建不暴露 Ent 类型的共享事务上下文。
func (d *Data) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	if transaction, ok := transactionFromContext(ctx); ok {
		if !transaction.active.Load() {
			return errTransactionContextClosed
		}
		return operation(ctx)
	}
	return d.WithTx(ctx, func(tx *ent.Tx) error {
		transaction := &transactionContext{tx: tx}
		transaction.active.Store(true)
		defer transaction.active.Store(false)
		return operation(context.WithValue(ctx, transactionContextKey{}, transaction))
	})
}

// withReadSnapshot 让一次财务查询的多条 SELECT 使用同一只读快照。
// 普通写事务不提供该隔离保证，完整响应应在提交后用普通上下文重读。
func (d *Data) withReadSnapshot(ctx context.Context, operation func(context.Context) error) error {
	if current, ok := transactionFromContext(ctx); ok {
		if !current.active.Load() {
			return errTransactionContextClosed
		}
		if !current.readSnapshot {
			return errReadSnapshotRequired
		}
		return operation(ctx)
	}
	return runTransaction(func() (*ent.Tx, error) {
		return d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	}, func(tx *ent.Tx) error {
		current := &transactionContext{tx: tx, readSnapshot: true}
		current.active.Store(true)
		defer current.active.Store(false)
		return operation(context.WithValue(ctx, transactionContextKey{}, current))
	})
}

func (d *Data) client(ctx context.Context) (*ent.Client, error) {
	transaction, ok := transactionFromContext(ctx)
	if !ok {
		return d.db, nil
	}
	if !transaction.active.Load() {
		return nil, errTransactionContextClosed
	}
	return transaction.tx.Client(), nil
}

func transactionFromContext(ctx context.Context) (*transactionContext, bool) {
	transaction, ok := ctx.Value(transactionContextKey{}).(*transactionContext)
	return transaction, ok
}

func (d *Data) withSQLTx(ctx context.Context, operation func(*sql.Tx) error) error {
	return runTransaction(func() (*sql.Tx, error) {
		return d.sqlDB.BeginTx(ctx, nil)
	}, operation)
}

var _ biz.Transactor = (*Data)(nil)

// WithClientTx 供命令行种子复用统一事务生命周期，不接管客户端所有权。
func WithClientTx(ctx context.Context, client *ent.Client, operation func(*ent.Tx) error) error {
	return (&Data{db: client}).WithTx(ctx, operation)
}
