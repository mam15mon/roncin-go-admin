package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
)

func TestOrderCommissionSnapshotRejectsMissingRoles(t *testing.T) {
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
	mock.ExpectBegin()
	tx, err := client.Tx(t.Context())
	if err != nil {
		t.Fatalf("开启事务失败: %v", err)
	}
	mock.ExpectQuery(`SELECT .* FROM "partner_assignments"`).
		WillReturnRows(sqlmock.NewRows(partnerassignment.Columns))
	err = snapshotOrderCommissionAttributions(t.Context(), tx, uuid.New(), uuid.New(), uuid.New(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "销售、操作、客服") {
		t.Fatalf("缺失全部提成岗位时应显式拒绝并列出岗位，实际: %v", err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("事务回滚失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("快照查询及回滚未符合预期: %v", err)
	}
}

func TestOrderCreateRollsBackAllocatedNumberWhenValidationFails(t *testing.T) {
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
	repo := &orderRepo{data: &Data{db: client, sqlDB: db}}
	organizationID, actorID, customerID := uuid.New(), uuid.New(), uuid.New()
	ruleID, sequenceID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "organizations"\."id" FROM "organizations".*"kind".*"enabled"`).
		WillReturnRows(sqlmock.NewRows([]string{organization.FieldID}).AddRow(organizationID))
	mock.ExpectQuery(`SELECT "number_rules"\..*FROM "number_rules".*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows(numberrule.Columns).AddRow(
			ruleID, now, now, organizationID, "order", "", "yyyyMMdd", 5, "daily", true,
		))
	mock.ExpectQuery(`SELECT "number_sequences"\..*FROM "number_sequences"`).
		WillReturnRows(sqlmock.NewRows(numbersequence.Columns).AddRow(
			sequenceID, now, now, ruleID, now.Format("20060102"), 1,
		))
	mock.ExpectExec(`UPDATE "number_sequences" SET .*"current_value".*WHERE "id" =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM "number_sequences" WHERE "id" =`).
		WillReturnRows(sqlmock.NewRows(numbersequence.Columns).AddRow(
			sequenceID, now, now, ruleID, now.Format("20060102"), 2,
		))
	mock.ExpectQuery(`SELECT .* FROM "partner_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	_, err = repo.Create(context.Background(), organizationID, actorID, &biz.Order{
		CustomerID:     customerID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: biz.OrderTradeExport,
		TradeTerm:      biz.OrderTradeFOB,
		PaymentTerm:    biz.OrderPaymentPrepaid,
	}, &biz.AuditEvent{Details: map[string]string{}})
	if err != biz.ErrOrderCustomerInvalid {
		t.Fatalf("创建订单错误 = %v，期望 ErrOrderCustomerInvalid", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("号码分配与订单校验未处于同一回滚事务: %v", err)
	}
}

func TestValidateOrderReferencesPreservesPartnerQueryError(t *testing.T) {
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

	mock.ExpectBegin()
	tx, err := client.Tx(t.Context())
	if err != nil {
		t.Fatalf("开启测试事务失败: %v", err)
	}
	databaseErr := errors.New("partner query failed")
	mock.ExpectQuery(`SELECT .* FROM "partner_roles"`).WillReturnError(databaseErr)

	err = validateOrderReferences(t.Context(), tx, uuid.New(), &biz.Order{CustomerID: uuid.New()}, nil)
	if !errors.Is(err, databaseErr) {
		t.Fatalf("合作方查询错误被改写: got %v, want %v", err, databaseErr)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("回滚测试事务失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
