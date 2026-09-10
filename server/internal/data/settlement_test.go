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
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflow "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
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
			name:     "未开票部分结清",
			progress: biz.FeeLedgerPartiallyVerifiedUninvoiced,
			query:    `SELECT COUNT.*FROM "order_fees".*NOT EXISTS.*finance_invoice_bills.*COALESCE.*\+ COALESCE.*\) > 0 AND \(COALESCE.*\+ COALESCE.*\) < fb.total_amount`,
		},
		{
			name:     "已完成",
			progress: biz.FeeLedgerCompleted,
			query:    `SELECT COUNT.*FROM "order_fees".*EXISTS.*finance_invoice_bills.*COALESCE.*\+ COALESCE.*\) >= fb.total_amount`,
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

	organizationID := uuid.New()
	mock.ExpectQuery(`SELECT COUNT.*FROM "order_fees".*organization_id.*finance_commission_lines.*finance_bill_lines.*finance_invoice_bills.*finance_verification_allocations`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT .*"active_count".*"base_amount".*FROM "order_fees".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "active_count", "base_amount"}).
			AddRow("RECEIVABLE", "CNY", 2, "100.25").
			AddRow("PAYABLE", "CNY", 1, "30").
			AddRow("RECEIVABLE", "USD", 1, "20"))
	mock.ExpectQuery(`SELECT "order_fees"\..*FROM "order_fees".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(orderfee.Columns))

	result, err := repo.ListFeeLedger(context.Background(), []uuid.UUID{organizationID}, biz.FeeLedgerFilter{
		Page:              2,
		PageSize:          20,
		FinancialProgress: biz.FeeLedgerCompleted,
		FinanceLocked:     &locked,
	})
	if err != nil {
		t.Fatalf("查询费用台账失败: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 || result.Summary.ActiveCount != 4 {
		t.Fatalf("空费用台账结果不符合预期: total=%d items=%d active=%d", result.Total, len(result.Items), result.Summary.ActiveCount)
	}
	if len(result.Summary.AmountsByBaseCurrency) != 2 ||
		result.Summary.AmountsByBaseCurrency[0].BaseCurrency != "CNY" ||
		result.Summary.AmountsByBaseCurrency[0].ReceivableBaseAmount.String() != "100.25" ||
		result.Summary.AmountsByBaseCurrency[0].PayableBaseAmount.String() != "30" ||
		result.Summary.AmountsByBaseCurrency[0].ProfitBaseAmount.String() != "70.25" ||
		result.Summary.AmountsByBaseCurrency[1].BaseCurrency != "USD" ||
		result.Summary.AmountsByBaseCurrency[1].ReceivableBaseAmount.String() != "20" ||
		result.Summary.AmountsByBaseCurrency[1].PayableBaseAmount.String() != "0" ||
		result.Summary.AmountsByBaseCurrency[1].ProfitBaseAmount.String() != "20" {
		t.Fatalf("费用台账汇总不符合预期: %+v", result.Summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("费用台账未使用数据库分页或汇总: %v", err)
	}
}

func TestVerificationCreationCandidatesKeepOrganizationPredicateInBothSourceQueries(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &verificationRepo{data: settlementRepo.data}
	organizationID := uuid.New()
	partyID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows".*organization_id.*finance_verification_allocations.*finance_verifications`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations".*finance_cashflows.*organization_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*organization_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills".*organization_id.*finance_verification_allocations.*finance_netting_allocations`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations".*finance_bills.*organization_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations".*finance_bills.*organization_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*organization_id`).
		WillReturnRows(sqlmock.NewRows(financebill.Columns))

	result, err := repo.ListCreationCandidates(context.Background(), organizationID, biz.VerificationCreationCandidateFilter{
		Direction:         biz.OrderFeeReceivable,
		SettlementPartyID: partyID,
		Currency:          "USD",
	})
	if err != nil {
		t.Fatalf("查询核销创建候选失败: %v", err)
	}
	if len(result.Cashflows) != 0 || len(result.Bills) != 0 {
		t.Fatalf("空数据源不应返回跨组织候选: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("核销候选未在账单和资金查询中下推组织范围与未结清余额过滤: %v", err)
	}
}

// TestVerificationCreationCandidatesPushDownBalancePredicates 断言候选的未结清过滤发生在
// 数据库 Count 内（即分页 LIMIT 之前），而不是取回 200 条后在 Go 层过滤。
func TestVerificationCreationCandidatesPushDownBalancePredicates(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &verificationRepo{data: settlementRepo.data}
	cashflowBalance := `finance_cashflows"\."amount" > COALESCE\(\(SELECT SUM\(fva.amount\) FROM finance_verification_allocations AS fva JOIN finance_verifications AS fv`
	billBalance := `finance_bills"\."total_amount" > \(COALESCE\(\(SELECT SUM\(fva.amount\) FROM finance_verification_allocations AS fva WHERE fva.bill_id = "finance_bills"\."id" AND fva.active = TRUE\), 0\) \+ COALESCE\(\(SELECT SUM\(fna.amount\) FROM finance_netting_allocations AS fna`

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows".*` + cashflowBalance).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations".*finance_cashflows`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*` + cashflowBalance + `.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows(financecashflow.Columns))
	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills".*` + billBalance).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations".*finance_bills`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_netting_allocations".*finance_bills`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*due_date.*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*` + billBalance + `.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows(financebill.Columns))

	result, err := repo.ListCreationCandidates(context.Background(), uuid.New(), biz.VerificationCreationCandidateFilter{
		Direction:         biz.OrderFeeReceivable,
		SettlementPartyID: uuid.New(),
		Currency:          "CNY",
	})
	if err != nil {
		t.Fatalf("查询核销创建候选失败: %v", err)
	}
	if len(result.Cashflows) != 0 || len(result.Bills) != 0 {
		t.Fatalf("空数据源候选结果错误: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未结清过滤未在账单和资金流水的 Count 与分页 Query 中同时下推: %v", err)
	}
}

func TestFinanceCashflowOnlyUnverifiedEntersCountAndPageQuery(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &financeCashflowRepo{data: settlementRepo.data}
	balance := `finance_cashflows"\."amount" > COALESCE\(\(SELECT SUM\(fva.amount\) FROM finance_verification_allocations AS fva JOIN finance_verifications AS fv`

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_cashflows".*` + balance).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"direction", "base_currency", "base_amount"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_verification_allocations".*finance_cashflows.*` + balance).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT .*FROM "finance_cashflows".*` + balance + `.*ORDER BY`).
		WillReturnRows(sqlmock.NewRows(financecashflow.Columns))

	result, err := repo.ListScoped(context.Background(), []uuid.UUID{uuid.New()}, biz.FinanceCashflowFilter{Page: 1, PageSize: 20, OnlyUnverified: true})
	if err != nil {
		t.Fatalf("查询未核销资金流水失败: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("空结果错误: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("OnlyUnverified 余额谓词未进入 Count 与分页 Query 的共用条件: %v", err)
	}
}

// TestFeeLedgerFinancialProgressPredicateCountsConfirmedNetting 断言有效结清金额 SQL
// 同时累计有效核销分摊与有效对冲分摊（父单据状态分别为 ACTIVE / CONFIRMED）。
func TestFeeLedgerFinancialProgressPredicateCountsConfirmedNetting(t *testing.T) {
	for _, progress := range []biz.FeeLedgerFinancialProgress{
		biz.FeeLedgerUnverifiedUninvoiced,
		biz.FeeLedgerInvoicedUnverified,
		biz.FeeLedgerPartiallyVerifiedUninvoiced,
		biz.FeeLedgerInvoicedPartiallyVerified,
		biz.FeeLedgerVerifiedUninvoiced,
		biz.FeeLedgerCompleted,
	} {
		t.Run(string(progress), func(t *testing.T) {
			repo, mock := setupTestSettlementRepo(t)
			mock.ExpectQuery(`SELECT COUNT.*FROM "order_fees".*finance_verification_allocations.*finance_verifications.*finance_netting_allocations.*finance_nettings`).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			if _, err := repo.data.db.OrderFee.Query().
				Where(feeLedgerFinancialProgressPredicate(progress)).
				Count(context.Background()); err != nil {
				t.Fatalf("生成含有效对冲的进度筛选查询失败: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("进度筛选 SQL 的有效结清金额缺少有效对冲分摊: %v", err)
			}
		})
	}
}

func TestInvoiceCreationCandidatesUseOrganizationAndActiveInvoicePredicates(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &financeInvoiceRepo{data: settlementRepo.data}
	organizationID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT.*FROM "finance_bills".*organization_id.*status.*NOT.*finance_invoice_bills`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "finance_bills".*FROM "finance_bills".*organization_id.*status.*NOT.*finance_invoice_bills.*ORDER BY.*LIMIT 20`).
		WillReturnRows(sqlmock.NewRows(financebill.Columns))

	result, err := repo.ListCreationBills(context.Background(), organizationID, biz.FinanceInvoiceCreationBillFilter{
		Page: 1, PageSize: 20, Direction: biz.OrderFeeReceivable, Currency: "USD",
	})
	if err != nil {
		t.Fatalf("查询发票创建账单候选失败: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("空候选结果错误: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("发票创建候选未在数据库下推组织和有效开票资格: %v", err)
	}
}

func TestInvoiceProfilesForBillRequireEligibleScopedBill(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &financeInvoiceRepo{data: settlementRepo.data}

	mock.ExpectQuery(`SELECT .*FROM "finance_bills".*id.*organization_id.*status.*NOT.*finance_invoice_bills`).
		WillReturnRows(sqlmock.NewRows(financebill.Columns))
	_, err := repo.ListProfilesForBill(context.Background(), []uuid.UUID{uuid.New()}, uuid.New())
	if err != biz.ErrFinanceInvoiceBillInvalid {
		t.Fatalf("不可用或越权源账单错误 = %v，期望 %v", err, biz.ErrFinanceInvoiceBillInvalid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("开票资料源账单未绑定组织和创建资格: %v", err)
	}
}

func TestBillCreationCandidatesKeepOrganizationEligibilityAndPartnerSearchInSQL(t *testing.T) {
	settlementRepo, mock := setupTestSettlementRepo(t)
	repo := &financeBillRepo{data: settlementRepo.data}
	mock.ExpectQuery(`SELECT COUNT.*FROM "order_fees".*status.*orders.*organization_id.*NOT.*finance_bill_lines.*partners.*partner_alias`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "order_fees".*FROM "order_fees".*status.*orders.*organization_id.*NOT.*finance_bill_lines.*partners.*partner_alias`).WillReturnRows(sqlmock.NewRows(orderfee.Columns))
	result, err := repo.ListCreationCandidates(context.Background(), uuid.New(), biz.FinanceBillCreationCandidateFilter{Page: 1, PageSize: 20, Keyword: "客户"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("结果=%+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 未下推组织、状态、未建账及往来单位搜索:%v", err)
	}
}
