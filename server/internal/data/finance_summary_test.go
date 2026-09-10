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
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/shopspring/decimal"
)

func assertFinanceBaseCurrencyAmount(t *testing.T, items []biz.FinanceBaseCurrencyAmount, currency, receivable, payable, unverified string) {
	assertFinanceBillBaseCurrencyAmount(t, items, currency, receivable, payable, unverified, "0")
}

func assertFinanceBillBaseCurrencyAmount(t *testing.T, items []biz.FinanceBaseCurrencyAmount, currency, receivable, payable, unverified, overdueReceivable string) {
	t.Helper()
	for _, item := range items {
		if item.BaseCurrency != currency {
			continue
		}
		if !item.ReceivableBaseAmount.Equal(decimal.RequireFromString(receivable)) ||
			!item.PayableBaseAmount.Equal(decimal.RequireFromString(payable)) ||
			!item.UnverifiedBaseAmount.Equal(decimal.RequireFromString(unverified)) ||
			!item.OverdueReceivableBaseAmount.Equal(decimal.RequireFromString(overdueReceivable)) {
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
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{uuid.New()}, biz.FinanceBillFilter{Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单列表失败: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 {
		t.Fatalf("账单分页结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
	}
	assertFinanceBillBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "150.25", "40", "190.25", "0")
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
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单多本位币汇总失败: %v", err)
	}
	assertFinanceBillBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "USD", "120", "10", "80", "0")
	assertFinanceBillBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "200", "0", "140", "0")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("账单未按本位币精确扣减有效核销: %v", err)
	}
}

func TestFinanceBillListOverdueSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeBillRepo{data: data}
	organizationID := uuid.New()
	overdueBillID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*status.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "100"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(financeSummaryAllocationRows(
			financeSummaryAllocation(uuid.Nil, overdueBillID, "20", "20", "0"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(overdueBillID, "CNY"))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(financeNettingSummaryAllocationRows(
			financeNettingSummaryAllocation(overdueBillID, "10", "10"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(overdueBillID, "CNY"))

	// 逾期应收聚合
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}).
			AddRow("CNY", "100"))
	// 逾期核销扣减
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(financeSummaryAllocationRows(
			financeSummaryAllocation(uuid.Nil, overdueBillID, "20", "20", "0"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(overdueBillID, "CNY"))
	// 逾期对冲扣减
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(financeNettingSummaryAllocationRows(
			financeNettingSummaryAllocation(overdueBillID, "10", "10"),
		))
	mock.ExpectQuery(`SELECT "finance_bills"\."id", "finance_bills"\."base_currency" FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_currency"}).
			AddRow(overdueBillID, "CNY"))

	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询账单列表失败: %v", err)
	}
	// 应收总额 100，核销 20，对冲 10，未核销 70，逾期未结清 70
	assertFinanceBillBaseCurrencyAmount(t, result.Summary.AmountsByBaseCurrency, "CNY", "100", "0", "70", "70")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("逾期应收汇总未按有效核销和对冲精确扣减: %v", err)
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

func TestFinanceNettingListUsesFilteredDatabaseSummary(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := NewFinanceNettingRepo(data)
	orgID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_nettings".*"organization_id".*("netting_no" ILIKE|"settlement_party_name" ILIKE)`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_nettings".*"organization_id".*"status".*("netting_no" ILIKE|"settlement_party_name" ILIKE)`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_nettings".*"organization_id".*"status".*("netting_no" ILIKE|"settlement_party_name" ILIKE).*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}).AddRow("CNY", "150.00000000"))

	mock.ExpectQuery(`SELECT "finance_nettings"\..*FROM "finance_nettings".*"organization_id".*("netting_no" ILIKE|"settlement_party_name" ILIKE).*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financenettingent.Columns))

	result, err := repo.List(context.Background(), []uuid.UUID{orgID}, biz.FinanceNettingFilter{Page: 1, PageSize: 20, Keyword: "NT-001"})
	if err != nil {
		t.Fatalf("查询对冲列表失败: %v", err)
	}
	if result.Total != 1 || result.Summary.ConfirmedCount != 1 {
		t.Fatalf("对冲列表与汇总统计不符合预期: total=%d confirmedCount=%d", result.Total, result.Summary.ConfirmedCount)
	}
	if len(result.Summary.AmountsByBaseCurrency) != 1 || result.Summary.AmountsByBaseCurrency[0].BaseCurrency != "CNY" || !result.Summary.AmountsByBaseCurrency[0].NettingBaseAmount.Equal(decimal.RequireFromString("150")) {
		t.Fatalf("对冲本币汇总不符合预期: %+v", result.Summary.AmountsByBaseCurrency)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("对冲列表 summary 查询未完整传递 keyword 过滤: %v", err)
	}
}

func TestFinanceBillDueDateAndOverduePredicates(t *testing.T) {
	table := entsql.Table(financebillent.Table)
	selector := entsql.Dialect(dialect.Postgres).Select(table.C(financebillent.FieldID)).From(table)

	// 测试未结清谓词
	billUnsettledPredicate()(selector)
	query, _ := selector.Query()
	if !strings.Contains(query, "SELECT SUM(fva.amount) FROM finance_verification_allocations AS fva WHERE fva.bill_id = \"finance_bills\".\"id\" AND fva.active = TRUE") ||
		!strings.Contains(query, "SELECT SUM(fna.amount) FROM finance_netting_allocations AS fna WHERE fna.bill_id = \"finance_bills\".\"id\" AND fna.active = TRUE") {
		t.Fatalf("未结清谓词 SQL 不符合预期: %s", query)
	}

	// 测试逾期谓词
	overdueSelector := entsql.Dialect(dialect.Postgres).Select(table.C(financebillent.FieldID)).From(table)
	billOverduePredicate("2026-09-10")(overdueSelector)
	overdueQuery, overdueArgs := overdueSelector.Query()
	if !strings.Contains(overdueQuery, `"finance_bills"."direction" = $1`) ||
		!strings.Contains(overdueQuery, `"finance_bills"."status" = $2`) ||
		!strings.Contains(overdueQuery, `"finance_bills"."due_date" IS NOT NULL`) ||
		!strings.Contains(overdueQuery, `"finance_bills"."due_date" < $4`) {
		t.Fatalf("逾期谓词 SQL 不符合预期: %s", overdueQuery)
	}
	if len(overdueArgs) < 4 || overdueArgs[0] != financebillent.DirectionRECEIVABLE || overdueArgs[1] != financebillent.StatusCONFIRMED || overdueArgs[3] != "2026-09-10" {
		t.Fatalf("逾期谓词参数不符合预期: %v", overdueArgs)
	}
}

func TestFinanceBillOnlyUnsettledDefaultsToConfirmed(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	repo := &financeBillRepo{data: data}
	organizationID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills".*WHERE.*"organization_id" IN \(\$1\).*AND.*"status" = \$2.*fva.*fna`).
		WithArgs(organizationID, financebillent.StatusCONFIRMED).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(sqlmock.NewRows(financeverificationallocationent.Columns))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(sqlmock.NewRows(financenettingallocationent.Columns))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*WHERE.*"organization_id" IN \(\$1\).*AND.*"status" = \$2.*ORDER BY.*LIMIT 20`).
		WithArgs(organizationID, financebillent.StatusCONFIRMED).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	_, err := repo.List(context.Background(), []uuid.UUID{organizationID}, biz.FinanceBillFilter{Page: 1, PageSize: 20, OnlyUnsettled: true})
	if err != nil {
		t.Fatalf("查询未结清账单列表失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未结清账单列表未默认追加已确认状态过滤: %v", err)
	}
}

func TestFinanceSummaryExcludesInvalidStatuses(t *testing.T) {
	data, mock := setupFinanceSummaryData(t)
	orgID := uuid.New()

	// 1. 验证账单汇总严格过滤已确认状态（即便列表未传状态筛选）
	billRepo := &financeBillRepo{data: data}
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// 汇总查询必须包含 "finance_bills"."status" = $2 (CONFIRMED)
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*"status".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "100"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT "finance_bills"\..*FROM "finance_bills".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financebillent.Columns))

	_, err := billRepo.List(context.Background(), []uuid.UUID{orgID}, biz.FinanceBillFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("账单列表查询失败: %v", err)
	}

	// 2. 验证发票汇总严格过滤已开具状态
	invoiceRepo := &financeInvoiceRepo{data: data}
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_invoices"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_invoices".*"status".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "100"))
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_invoices".*"status"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT "finance_invoices"\..*FROM "finance_invoices".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financeinvoiceent.Columns))

	_, err = invoiceRepo.List(context.Background(), orgID, biz.FinanceInvoiceFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("发票列表查询失败: %v", err)
	}

	// 3. 验证资金流水汇总严格过滤已确认状态
	cashflowRepo := &financeCashflowRepo{data: data}
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_cashflows".*"status".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "100"))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT "finance_cashflows"\..*FROM "finance_cashflows".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financecashflowent.Columns))

	_, err = cashflowRepo.List(context.Background(), orgID, biz.FinanceCashflowFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("资金流水列表查询失败: %v", err)
	}

	// 4. 验证核销汇总严格过滤有效状态
	verRepo := &verificationRepo{data: data}
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_verifications"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .*"base_amount".*FROM "finance_verifications".*"status".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", "100"))
	mock.ExpectQuery(`SELECT "finance_verifications"\..*FROM "finance_verifications".*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financeverificationent.Columns))

	_, err = verRepo.List(context.Background(), orgID, biz.VerificationFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("核销列表查询失败: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("财务汇总未排除无效状态（草稿/取消/红冲/反核销）: %v", err)
	}
}
