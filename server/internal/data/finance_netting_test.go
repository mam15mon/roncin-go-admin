package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func TestFinanceNettingLoadPreviewRejectsTooManyBills(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeNettingRepo{data: data}
	organizationID, partyID := uuid.New(), uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT .*FROM "organizations" WHERE "organizations"\."id" = \$1`).
		WithArgs(organizationID).
		WillReturnRows(sqlmock.NewRows(organizationent.Columns).
			AddRow(organizationID, now, now, "HQ", "总部", "headquarters", nil, true, "CNY", "总部"))
	mock.ExpectQuery(`SELECT .*FROM "partners" WHERE`).
		WithArgs(partyID, organizationID).
		WillReturnRows(sqlmock.NewRows(partnerent.Columns).
			AddRow(partyID, now, now, organizationID, "P001", "测试伙伴", "测试伙伴", nil, "", true, false, ""))

	// 按 financebillent.Columns 顺序构造一行可完整扫描的已确认账单，
	// 重复 biz.MaxFinanceNettingBills+1 行触发预览上限。
	row := []driver.Value{
		uuid.New(), now, now, organizationID, "B0001", "idem-1", nil,
		"RECEIVABLE", "CONFIRMED", partyID, "测试伙伴",
		uuid.New(), "测试账户", "测试持有人", "测试银行", "6222000000000000", "CNY", nil,
		nil, nil, nil,
		"CNY", "CNY", "1", "BASE_CURRENCY", "2026-09-10", nil,
		"100", "100", "0", "100", 1, "2026-09-10",
		nil, nil, nil, nil, 1, nil, nil, nil, nil, nil,
	}
	if len(row) != len(financebillent.Columns) {
		t.Fatalf("账单 fixture 列数=%d，期望=%d", len(row), len(financebillent.Columns))
	}
	billRows := sqlmock.NewRows(financebillent.Columns)
	for i := 0; i <= biz.MaxFinanceNettingBills; i++ {
		billRows = billRows.AddRow(row...)
	}
	mock.ExpectQuery(`SELECT .*FROM "finance_bills" WHERE`).WillReturnRows(billRows)

	if _, err := repo.LoadPreview(context.Background(), organizationID, partyID, "CNY"); !errors.Is(err, biz.ErrFinanceNettingTooManyBills) {
		t.Fatalf("错误=%v，期望 ErrFinanceNettingTooManyBills", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}
}
