package data

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func setupTransactionData(t *testing.T) (*Data, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})
	return &Data{db: client, sqlDB: db}, mock
}

func TestWithTxCommitsSuccessfulOperation(t *testing.T) {
	data, mock := setupTransactionData(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := data.WithTx(context.Background(), func(*ent.Tx) error { return nil }); err != nil {
		t.Fatalf("提交事务失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("成功事务未提交: %v", err)
	}
}

func TestWithTxRollsBackFailedOperation(t *testing.T) {
	data, mock := setupTransactionData(t)
	wantErr := errors.New("业务失败")
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := data.WithTx(context.Background(), func(*ent.Tx) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("事务错误 = %v，期望 %v", err, wantErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("失败事务未回滚: %v", err)
	}
}

func TestWithTxRollsBackPanicAndRethrows(t *testing.T) {
	data, mock := setupTransactionData(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	defer func() {
		if recovered := recover(); recovered != "测试 panic" {
			t.Fatalf("恢复的 panic = %v", recovered)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("panic 事务未回滚: %v", err)
		}
	}()
	_ = data.WithTx(context.Background(), func(*ent.Tx) error {
		panic("测试 panic")
	})
}

func TestWithinTransactionReusesTransactionForNestedRepository(t *testing.T) {
	data, mock := setupTransactionData(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var outerTx *ent.Tx
	err := data.WithinTransaction(context.Background(), func(ctx context.Context) error {
		return data.WithTx(ctx, func(tx *ent.Tx) error {
			outerTx = tx
			return data.WithTx(ctx, func(nestedTx *ent.Tx) error {
				if nestedTx != outerTx {
					t.Fatal("嵌套仓储没有复用外层事务")
				}
				return nil
			})
		})
	})
	if err != nil {
		t.Fatalf("共享事务执行失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("共享事务不应重复开启或提交: %v", err)
	}
}

func TestWithinTransactionRollsBackNestedRepositoryFailure(t *testing.T) {
	data, mock := setupTransactionData(t)
	wantErr := errors.New("第二个仓储失败")
	mock.ExpectBegin()
	mock.ExpectRollback()

	err := data.WithinTransaction(context.Background(), func(ctx context.Context) error {
		if err := data.WithTx(ctx, func(*ent.Tx) error { return nil }); err != nil {
			return err
		}
		return data.WithTx(ctx, func(*ent.Tx) error { return wantErr })
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("共享事务错误 = %v，期望 %v", err, wantErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("仓储失败后共享事务未整体回滚: %v", err)
	}
}

func TestWithinTransactionRejectsExpiredContext(t *testing.T) {
	data, mock := setupTransactionData(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var transactionCtx context.Context
	if err := data.WithinTransaction(context.Background(), func(ctx context.Context) error {
		transactionCtx = ctx
		return nil
	}); err != nil {
		t.Fatalf("共享事务执行失败: %v", err)
	}
	if err := data.WithTx(transactionCtx, func(*ent.Tx) error { return nil }); !errors.Is(err, errTransactionContextClosed) {
		t.Fatalf("已结束上下文错误 = %v，期望 %v", err, errTransactionContextClosed)
	}
	if _, err := data.client(transactionCtx); !errors.Is(err, errTransactionContextClosed) {
		t.Fatalf("已结束上下文读取错误 = %v，期望 %v", err, errTransactionContextClosed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("共享事务提交不符合预期: %v", err)
	}
}
