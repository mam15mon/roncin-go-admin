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
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/shopspring/decimal"
)

func setupFinanceSummaryData(t *testing.T) (*Data, sqlmock.Sqlmock) {
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

func TestFinanceBillListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeBillRepo{data: data}

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "150.25").
			AddRow("PAYABLE", "CNY", "40"))
	mock.ExpectQuery(`SELECT .*"verified_base_amount".*FROM "finance_verification_allocations".*finance_bills.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"active", "verified_base_amount"}).AddRow(true, "60.25"))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.FinanceBillFilter{Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单列表失败: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 {
		t.Fatalf("账单分页结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Summary.BaseCurrency != "CNY" || !result.Summary.ReceivableBaseAmount.Equal(decimal.RequireFromString("150.25")) || !result.Summary.PayableBaseAmount.Equal(decimal.NewFromInt(40)) || !result.Summary.UnverifiedBaseAmount.Equal(decimal.NewFromInt(130)) {
		t.Fatalf("账单汇总不符合预期: %+v", result.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("账单列表未使用数据库分页或汇总: %v", err)
	}
}

func TestFinanceInvoiceListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeInvoiceRepo{data: data}

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_invoices"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_invoices".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "80").
			AddRow("PAYABLE", "CNY", "25.5"))
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_invoices".*status`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT "finance_invoices"\..*FROM "finance_invoices".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financeinvoiceent.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.FinanceInvoiceFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询发票列表失败: %v", err)
	}
	if result.Total != 4 || result.Summary.IssuedCount != 2 || result.Summary.BaseCurrency != "CNY" || !result.Summary.ReceivableBaseAmount.Equal(decimal.NewFromInt(80)) || !result.Summary.PayableBaseAmount.Equal(decimal.RequireFromString("25.5")) {
		t.Fatalf("发票汇总不符合预期: total=%d summary=%+v", result.Total, result.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("发票列表未使用数据库分页或汇总: %v", err)
	}
}

func TestVerificationListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &verificationRepo{data: data}

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_verifications"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_verifications".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "60").
			AddRow("PAYABLE", "CNY", "10"))
	mock.ExpectQuery(`SELECT "finance_verifications"\..*FROM "finance_verifications".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financeverificationent.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.VerificationFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询核销列表失败: %v", err)
	}
	if result.Total != 2 || result.Summary.BaseCurrency != "CNY" || !result.Summary.ReceivableBaseAmount.Equal(decimal.NewFromInt(60)) || !result.Summary.PayableBaseAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("核销汇总不符合预期: total=%d summary=%+v", result.Total, result.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("核销列表未使用数据库分页或汇总: %v", err)
	}
}
