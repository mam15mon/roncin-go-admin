package data

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

func setupTestSettlementRepo(t *testing.T) (*settlementRepo, sqlmock.Sqlmock) {
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
	return &settlementRepo{data: &Data{db: client, sqlDB: db}}, mock
}

func TestFeeLedgerFinancialProgressPredicate(t *testing.T) {
	tests := []struct {
		name     string
		progress biz.FeeLedgerFinancialProgress
		query    string
	}{
		{
			name:     "未出账",
			progress: biz.FeeLedgerUnbilled,
			query:    `SELECT COUNT.*FROM "order_fees".*NOT EXISTS.*finance_bill_lines`,
		},
		{
			name:     "未开票部分核销",
			progress: biz.FeeLedgerPartiallyVerifiedUninvoiced,
			query:    `SELECT COUNT.*FROM "order_fees".*NOT EXISTS.*finance_invoice_bills.*COALESCE.*> 0 AND COALESCE.*< fb.total_amount`,
		},
		{
			name:     "已完成",
			progress: biz.FeeLedgerCompleted,
			query:    `SELECT COUNT.*FROM "order_fees".*EXISTS.*finance_invoice_bills.*COALESCE.*>= fb.total_amount`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := setupTestSettlementRepo(t)
			mock.ExpectQuery(tt.query).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			_, err := repo.data.db.OrderFee.Query().
				Where(feeLedgerFinancialProgressPredicate(tt.progress)).
				Count(context.Background())
			if err != nil {
				t.Fatalf("生成财务进度筛选查询失败: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("财务进度筛选未下推到数据库: %v", err)
			}
		})
	}
}

func TestSettlementRepoListFeeLedgerUsesDatabasePaginationAndSummary(t *testing.T) {
	repo, mock := setupTestSettlementRepo(t)
	locked := true

	mock.ExpectQuery(`SELECT COUNT.*FROM "order_fees".*finance_commission_lines.*finance_bill_lines.*finance_invoice_bills.*finance_verification_allocations`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT .*"active_count".*"base_amount".*FROM "order_fees".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "active_count", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", 2, "100.25").
			AddRow("PAYABLE", "CNY", 1, "30"))
	mock.ExpectQuery(`SELECT "order_fees"\..*FROM "order_fees".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(orderfee.Columns))

	result, err := repo.ListFeeLedger(context.Background(), uuid.New(), biz.FeeLedgerFilter{
		Page:              2,
		PageSize:          20,
		FinancialProgress: biz.FeeLedgerCompleted,
		FinanceLocked:     &locked,
	})
	if err != nil {
		t.Fatalf("查询费用台账失败: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 || result.Summary.ActiveCount != 3 {
		t.Fatalf("空费用台账结果不符合预期: total=%d items=%d active=%d", result.Total, len(result.Items), result.Summary.ActiveCount)
	}
	if result.Summary.BaseCurrency != "CNY" || !result.Summary.ReceivableBaseAmount.Equal(decimal.RequireFromString("100.25")) || !result.Summary.PayableBaseAmount.Equal(decimal.NewFromInt(30)) || !result.Summary.ProfitBaseAmount.Equal(decimal.RequireFromString("70.25")) {
		t.Fatalf("费用台账汇总不符合预期: %+v", result.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("费用台账未使用数据库分页或汇总: %v", err)
	}
}
