package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/shopspring/decimal"
)

func assertFinanceBaseCurrencyAmount(t *testing.T, items []biz.FinanceBaseCurrencyAmount, currency, receivable, payable, unverified string) {
	t.Helper()
	for _, item := range items {
		if item.BaseCurrency != currency {
			continue
		}
		if !item.ReceivableBaseAmount.Equal(decimal.RequireFromString(receivable)) ||
			!item.PayableBaseAmount.Equal(decimal.RequireFromString(payable)) ||
			!item.UnverifiedBaseAmount.Equal(decimal.RequireFromString(unverified)) {
			t.Fatalf("本位币 %s 汇总不符合预期: %+v", currency, item)
		}
		return
	}
	t.Fatalf("未找到本位币 %s 的汇总，实际=%+v", currency, items)
}

func TestFinanceBillOrganizationScopePredicateUsesExplicitOrganizationIDs(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	table := entsql.Table(financebillent.Table)
	selector := entsql.Dialect(dialect.Postgres).Select(table.C(financebillent.FieldID)).From(table)

	financeBillOrganizationScopePredicate([]uuid.UUID{tianjinID, beijingID})(selector)

	query, args := selector.Query()
	if !strings.Contains(query, `"finance_bills"."organization_id" IN ($1, $2)`) {
		t.Fatalf("账单组织谓词应由数据库显式过滤，query=%s", query)
	}
	if len(args) != 2 || args[0] != tianjinID || args[1] != beijingID {
		t.Fatalf("账单组织谓词参数=%v，期望天津和北京组织", args)
	}
}

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

func TestFinanceCashflowListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeCashflowRepo{data: data}

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_cashflows".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "200.5").
			AddRow("PAYABLE", "CNY", "70"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT "finance_cashflows"\..*FROM "finance_cashflows".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(financecashflowent.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.FinanceCashflowFilter{Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("查询资金流水列表失败: %v", err)
	}
	if result.Total != 5 || len(result.Items) != 0 {
		t.Fatalf("资金流水分页结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "200.5", "70", "270.5")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("资金流水列表未使用数据库分页或汇总: %v", err)
	}
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
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{uuid.New()}, biz.FinanceBillFilter{Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单列表失败: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 {
		t.Fatalf("账单分页结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "150.25", "40", "190.25")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("账单列表未使用数据库分页或汇总: %v", err)
	}
}

func TestFinanceCashflowListDeductsActiveAllocationsByBaseCurrency(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeCashflowRepo{data: data}
	organizationID := uuid.New()
	usdID := uuid.New()
	cnyID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_cashflows".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "USD", "120").
			AddRow("PAYABLE", "USD", "10").
			AddRow("RECEIVABLE", "CNY", "200"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(financeSummaryAllocationRows(
			financeSummaryAllocation(usdID, uuid.Nil, "30", "0", "30"),
			financeSummaryAllocation(cnyID, uuid.Nil, "50", "0", "50"),
		))
	mock.ExpectQuery(`SELECT "finance_cashflows"\."id", "finance_cashflows"\."base_currency" FROM "finance_cashflows"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(usdID, "USD").
			AddRow(cnyID, "CNY"))
	mock.ExpectQuery(`SELECT "finance_cashflows"\..*FROM "finance_cashflows".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financecashflowent.Columns))

	result, err := repo.List(context.Background(), organizationID, biz.FinanceCashflowFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询资金流水多本位币汇总失败: %v", err)
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "USD", "120", "10", "100")
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "200", "0", "150")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("资金流水未按本位币精确扣减有效核销: %v", err)
	}
}

func TestFinanceBillListDeductsActiveAllocationsByBaseCurrency(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeBillRepo{data: data}
	organizationID := uuid.New()
	usdID := uuid.New()
	cnyID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "USD", "120").
			AddRow("PAYABLE", "USD", "10").
			AddRow("RECEIVABLE", "CNY", "200"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(financeSummaryAllocationRows(
			financeSummaryAllocation(uuid.Nil, usdID, "30", "30", "0"),
			financeSummaryAllocation(uuid.Nil, cnyID, "50", "50", "0"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(usdID, "USD").
			AddRow(cnyID, "CNY"))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(financeNettingSummaryAllocationRows(
			financeNettingSummaryAllocation(usdID, "20", "20"),
			financeNettingSummaryAllocation(cnyID, "10", "10"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(usdID, "USD").
			AddRow(cnyID, "CNY"))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单多本位币汇总失败: %v", err)
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "USD", "120", "10", "80")
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "200", "0", "140")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("账单未按本位币精确扣减有效核销: %v", err)
	}
}

type financeSummaryAllocationInput struct {
	cashflowID     uuid.UUID
	billID         uuid.UUID
	amount         string
	billBaseAmount string
	cashflowAmount string
}

func financeSummaryAllocation(cashflowID, billID uuid.UUID, amount, billBaseAmount, cashflowAmount string) financeSummaryAllocationInput {
	return financeSummaryAllocationInput{cashflowID: cashflowID, billID: billID, amount: amount, billBaseAmount: billBaseAmount, cashflowAmount: cashflowAmount}
}

func financeSummaryAllocationRows(items ...financeSummaryAllocationInput) *sqlmock.Rows {
	rows := sqlmock.NewRows(financeverificationallocationent.Columns)
	for _, item := range items {
		rows.AddRow(uuid.New(), time.Now(), time.Now(), uuid.New(), item.cashflowID, item.billID, "FLOW", "BILL", item.amount, item.billBaseAmount, item.cashflowAmount, item.amount, "0", true)
	}
	return rows
}

type financeNettingSummaryAllocationInput struct {
	billID             uuid.UUID
	amount             string
	baseCurrencyAmount string
}

func financeNettingSummaryAllocation(billID uuid.UUID, amount, baseCurrencyAmount string) financeNettingSummaryAllocationInput {
	return financeNettingSummaryAllocationInput{billID: billID, amount: amount, baseCurrencyAmount: baseCurrencyAmount}
}

func financeNettingSummaryAllocationRows(items ...financeNettingSummaryAllocationInput) *sqlmock.Rows {
	rows := sqlmock.NewRows(financenettingallocationent.Columns)
	for _, item := range items {
		rows.AddRow(uuid.New(), time.Now(), time.Now(), uuid.New(), item.billID, "BILL", "RECEIVABLE", item.amount, item.baseCurrencyAmount, true)
	}
	return rows
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
	if result.Total != 4 || result.Summary.IssuedCount != 2 {
		t.Fatalf("发票汇总不符合预期: total=%d summary=%+v", result.Total, result.Summary)
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "80", "25.5", "0")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("发票列表未使用数据库分页或汇总: %v", err)
	}
}

func TestVerificationListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &verificationRepo{data: data}

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_verifications".*"organization_id".*"status".*"direction"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_verifications".*"organization_id".*"status".*"direction".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "60").
			AddRow("PAYABLE", "CNY", "10"))
	mock.ExpectQuery(`SELECT "finance_verifications"\..*FROM "finance_verifications".*"organization_id".*"status".*"direction".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financeverificationent.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.VerificationFilter{Page: 1, PageSize: 20, Status: biz.VerificationActive, Direction: biz.OrderFeeReceivable})
	if err != nil {
		t.Fatalf("查询核销列表失败: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("核销汇总不符合预期: total=%d summary=%+v", result.Total, result.Summary)
	}
	assertFinanceBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "60", "10", "0")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("核销列表未使用数据库分页或汇总: %v", err)
	}
}
