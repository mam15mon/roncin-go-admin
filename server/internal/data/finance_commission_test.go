package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

func setupTestCommissionRepo(t *testing.T) (*commissionRepo, sqlmock.Sqlmock) {
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
	return &commissionRepo{data: &Data{db: client, sqlDB: db}}, mock
}

func TestCommissionCandidateEmployeePredicatesUseDatabaseFilteringAndPagination(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	source := &commissionCalculationSource{
		rule:     &ent.FinanceCommissionRule{PersonnelRole: financecommissionrule.PersonnelRoleSALES},
		orderIDs: []uuid.UUID{uuid.New()},
	}
	query := repo.data.db.User.Query().Where(commissionCandidateEmployeePredicates(uuid.New(), source, "zhangsan")...)
	mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*order_commission_attributions.*orders.*order_fees.*"username".*"display_name".*"search_keywords"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "users"\..*FROM "users".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(user.Columns))

	if _, err := query.Clone().Count(context.Background()); err != nil {
		t.Fatalf("统计提成候选失败: %v", err)
	}
	if _, err := query.Order(user.ByDisplayName(), user.ByUsername(), user.ByID()).Offset(20).Limit(20).All(context.Background()); err != nil {
		t.Fatalf("分页查询提成候选失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("提成候选未使用数据库筛选和分页: %v", err)
	}
}

func TestCalculateCommissionFromSourceUsesSharedBatchData(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	employeeID := uuid.New()
	customer := &ent.Partner{ID: uuid.New(), Code: "CUS001", LegalName: "测试客户"}
	settlementParty := &ent.Partner{ID: uuid.New(), LegalName: "测试结算单位"}
	orderItem := &ent.Order{
		ID: orderID, OrderNo: "SE20260001", OrderDate: "2026-08-28", CustomerID: customer.ID,
		Edges: ent.OrderEdges{Customer: customer},
	}
	receivable := &ent.OrderFee{
		ID: uuid.New(), OrderID: orderID, SettlementPartyID: settlementParty.ID,
		Direction: orderfee.DirectionRECEIVABLE, Status: orderfee.StatusCONFIRMED,
		FeeCode: "FREIGHT", FeeName: "海运费", Currency: "CNY", TotalAmount: "1000", ExchangeRate: "1",
		BaseCurrency: "CNY", BaseCurrencyAmount: "1000", ExpenseDate: "2026-08-28", Version: 1,
		Edges: ent.OrderFeeEdges{SettlementParty: settlementParty},
	}
	payable := &ent.OrderFee{
		ID: uuid.New(), OrderID: orderID, SettlementPartyID: settlementParty.ID,
		Direction: orderfee.DirectionPAYABLE, Status: orderfee.StatusCONFIRMED,
		FeeCode: "COST", FeeName: "成本", Currency: "CNY", TotalAmount: "400", ExchangeRate: "1",
		BaseCurrency: "CNY", BaseCurrencyAmount: "400", ExpenseDate: "2026-08-28", Version: 1,
		Edges: ent.OrderFeeEdges{SettlementParty: settlementParty},
	}
	source := &commissionCalculationSource{
		organizationID: organizationID,
		verification:   &ent.FinanceVerification{ID: uuid.New(), VerificationNo: "VR20260001"},
		rule: &ent.FinanceCommissionRule{
			ID: uuid.New(), Name: "销售提成", PersonnelRole: financecommissionrule.PersonnelRoleSALES,
			CalculationBasis: financecommissionrule.CalculationBasisREALIZED_PROFIT, Version: 1,
		},
		rate:          decimal.NewFromInt(10),
		baseCurrency:  "CNY",
		orderIDs:      []uuid.UUID{orderID},
		orderRealized: map[uuid.UUID]decimal.Decimal{orderID: decimal.NewFromInt(500)},
		orderByID:     map[uuid.UUID]*ent.Order{orderID: orderItem},
		feesByOrder:   map[uuid.UUID][]*ent.OrderFee{orderID: {receivable, payable}},
	}
	employee := &ent.User{ID: employeeID, DisplayName: "张三", Enabled: true}
	attributions := []*ent.OrderCommissionAttribution{{
		ID: uuid.New(), OrganizationID: organizationID, OrderID: orderID, CustomerID: customer.ID,
		SourceAssignmentID: uuid.New(), EmployeeID: employeeID, EmployeeName: "张三",
		PersonnelRole: ordercommissionattribution.PersonnelRoleSALES, AttributedAt: time.Now(),
	}}

	calculation, err := calculateCommissionFromSource(source, employee, attributions)
	if err != nil {
		t.Fatalf("使用批量上下文计算提成失败: %v", err)
	}
	if calculation.RealizedRevenue.StringFixed(8) != "500.00000000" || calculation.AllocatedCost.StringFixed(8) != "200.00000000" || calculation.RealizedProfit.StringFixed(8) != "300.00000000" || calculation.CommissionAmount.StringFixed(8) != "30.00000000" {
		t.Fatalf("批量提成计算结果不符合预期: %#v", calculation)
	}
	if calculation.CustomerCount != 1 || calculation.OrderCount != 1 || calculation.FeeCount != 2 || calculation.SourceFingerprint == "" {
		t.Fatalf("批量提成计算摘要不符合预期: %#v", calculation)
	}
}

func TestCommissionRepoListEmployeesUsesKeywordAndDatabasePagination(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*EXISTS.*FROM "memberships".*"username".*"display_name".*"search_keywords"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "users"\..*FROM "users".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(user.Columns))

	result, err := repo.ListEmployees(context.Background(), uuid.New(), biz.SelectorListOptions{
		Page: 2, PageSize: 20, Keyword: "zhangsan",
	})
	if err != nil {
		t.Fatalf("查询提成员工失败: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 || result.Page != 2 || result.PageSize != 20 {
		t.Fatalf("提成员工分页结果不符合预期: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("提成员工未使用数据库搜索和分页: %v", err)
	}
}

func TestCommissionWithLinesToBizDerivesDynamicCNYAmounts(t *testing.T) {
	commission := &ent.FinanceCommission{
		CommissionAmount:      "100.00000000",
		RealizedRevenue:       "500.00000000",
		AllocatedCost:         "200.00000000",
		RealizedProfit:        "300.00000000",
		CommissionBaseAmount:  "300.00000000",
		RatePercent:           "10.0000",
		CommissionDate:        "2026-08-15",
		CnyExchangeRate:       "0.50000000",
		CnyExchangeRateSource: commission.CnyExchangeRateSourceDERIVED,
		CnyExchangeRateDate:   "2026-08-14",
		CnyCommissionAmount:   "50.00000000",
		Edges: ent.FinanceCommissionEdges{Adjustments: []*ent.FinanceCommissionAdjustment{
			{Status: adjustment.StatusCONFIRMED, Direction: adjustment.DirectionINCREASE, Amount: "10.00000000"},
			{Status: adjustment.StatusPAID, Direction: adjustment.DirectionDECREASE, Amount: "4.00000000"},
			{Status: adjustment.StatusDRAFT, Direction: adjustment.DirectionINCREASE, Amount: "999.00000000"},
			{Status: adjustment.StatusCANCELLED, Direction: adjustment.DirectionINCREASE, Amount: "888.00000000"},
		}},
	}

	result, err := commissionWithLinesToBiz(commission)
	if err != nil {
		t.Fatalf("commissionWithLinesToBiz() error = %v", err)
	}
	if result.CommissionDate != "2026-08-15" || result.CNYExchangeRate.StringFixed(8) != "0.50000000" ||
		result.CNYExchangeRateSource != biz.CommissionCNYRateSourceDerived || result.CNYExchangeRateDate != "2026-08-14" ||
		result.CNYCommissionAmount.StringFixed(8) != "50.00000000" {
		t.Fatalf("持久化 CNY 快照字段转换不符: %#v", result)
	}
	// 有效调整 = 10 - 4 = 6；草稿与已取消不计入；CNY 调整继承主单汇率。
	if result.AdjustmentAmount.StringFixed(8) != "6.00000000" {
		t.Fatalf("本位币调整金额不符: %s", result.AdjustmentAmount.StringFixed(8))
	}
	if result.CNYAdjustmentAmount.StringFixed(8) != "3.00000000" {
		t.Fatalf("动态 CNY 调整金额不符: %s", result.CNYAdjustmentAmount.StringFixed(8))
	}
	if result.CNYEffectiveCommissionAmount.StringFixed(8) != "53.00000000" {
		t.Fatalf("有效 CNY 提成金额不符: %s", result.CNYEffectiveCommissionAmount.StringFixed(8))
	}
}

func TestCommissionRepoListUsesCommissionDateFilterAndStableOrdering(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	mock.ExpectQuery(`SELECT COUNT\("finance_commissions"\."id"\) FROM "finance_commissions".*"commission_date" >= .*"commission_date" <=`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "finance_commissions".*ORDER BY "finance_commissions"\."commission_date" DESC, "finance_commissions"\."created_at" DESC, "finance_commissions"\."id" DESC LIMIT`).
		WillReturnRows(sqlmock.NewRows(commission.Columns))

	result, err := repo.List(context.Background(), uuid.New(), biz.CommissionFilter{
		Page: 1, PageSize: 20, CommissionDateFrom: "2026-07-01", CommissionDateTo: "2026-08-31",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("列表结果不符合预期: %#v", result)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("列表未使用归属日期筛选与稳定排序: %v", err)
	}
}

func TestCommissionRepoGetGenerationContextLocking(t *testing.T) {
	org := uuid.New()

	t.Run("事务内首次读取核销单即FOR UPDATE", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "finance_verifications" WHERE .*FOR UPDATE$`).
			WillReturnRows(sqlmock.NewRows(financeverification.Columns))
		mock.ExpectRollback()

		err := repo.data.WithinTransaction(context.Background(), func(ctx context.Context) error {
			_, queryErr := repo.GetGenerationContext(ctx, org, uuid.New())
			return queryErr
		})
		if !errors.Is(err, biz.ErrCommissionSource) {
			t.Fatalf("GetGenerationContext() error = %v, want %v", err, biz.ErrCommissionSource)
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("事务内生成上下文读取未对核销单加 FOR UPDATE: %v", err)
		}
	})

	t.Run("普通上下文读取不加锁", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		// 结尾锚定 LIMIT 2：若实际 SQL 追加 FOR UPDATE 将无法匹配，验证普通读取无锁。
		mock.ExpectQuery(`SELECT .* FROM "finance_verifications" WHERE .*LIMIT 2$`).
			WillReturnRows(sqlmock.NewRows(financeverification.Columns))

		if _, err := repo.GetGenerationContext(context.Background(), org, uuid.New()); !errors.Is(err, biz.ErrCommissionSource) {
			t.Fatalf("GetGenerationContext() error = %v, want %v", err, biz.ErrCommissionSource)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("普通上下文生成上下文读取不应加锁: %v", err)
		}
	})
}

func TestCommissionRepoReadMethodsRejectClosedTransactionContext(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var transactionCtx context.Context
	if err := repo.data.WithinTransaction(context.Background(), func(ctx context.Context) error {
		transactionCtx = ctx
		return nil
	}); err != nil {
		t.Fatalf("创建已结束事务上下文失败: %v", err)
	}

	org := uuid.New()
	verificationID := uuid.New()
	ruleID := uuid.New()
	checks := []struct {
		name string
		call func() error
	}{
		{
			name: "员工列表",
			call: func() error {
				_, err := repo.ListEmployees(transactionCtx, org, biz.SelectorListOptions{Page: 1, PageSize: 20})
				return err
			},
		},
		{
			name: "提成候选",
			call: func() error {
				_, err := repo.ListCandidates(transactionCtx, org, biz.CommissionCandidateFilter{VerificationID: verificationID, RuleID: ruleID, Page: 1, PageSize: 20})
				return err
			},
		},
		{
			name: "规则列表",
			call: func() error {
				_, err := repo.ListRules(transactionCtx, org, biz.CommissionRuleFilter{Page: 1, PageSize: 20})
				return err
			},
		},
		{
			name: "调整幂等查询",
			call: func() error {
				_, err := repo.GetAdjustmentByKey(transactionCtx, org, "adjustment-key")
				return err
			},
		},
		{
			name: "导出计数",
			call: func() error {
				_, err := repo.Count(transactionCtx, org, biz.CommissionFilter{})
				return err
			},
		},
		{
			name: "导出批量读取",
			call: func() error {
				_, err := repo.ExportBatch(transactionCtx, org, biz.CommissionFilter{Page: 1, PageSize: 200})
				return err
			},
		},
		{
			name: "导出审计",
			call: func() error {
				return repo.SaveExportAudit(transactionCtx, &biz.AuditEvent{Action: "finance.commission.export", Result: "success"})
			},
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, errTransactionContextClosed) {
				t.Fatalf("读取错误 = %v，期望 %v", err, errTransactionContextClosed)
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("已结束事务上下文不应发起查询: %v", err)
	}
}

func TestCommissionRepoCountUsesListPredicates(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	mock.ExpectQuery(`SELECT COUNT\("finance_commissions"\."id"\) FROM "finance_commissions" WHERE .*"commission_no".*"employee_name".*"rule_name".*"status".*"commission_date" >= .*"commission_date" <=`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10001))

	total, err := repo.Count(context.Background(), uuid.New(), biz.CommissionFilter{
		Keyword: "TC2026", Status: biz.CommissionPaid, CommissionDateFrom: "2026-07-01", CommissionDateTo: "2026-08-31",
	})
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if total != 10001 {
		t.Fatalf("计数结果不符: %d", total)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("导出计数未复用列表谓词: %v", err)
	}
}

// commissionExportScanRow 按列顺序构造一行可扫描的提成记录。
func commissionExportScanRow(t *testing.T, org uuid.UUID) []driver.Value {
	t.Helper()
	now := time.Now()
	return []driver.Value{
		uuid.New(), now, now, org, "TC20260815000001", "idempotency-export", uuid.New(), "VR20260815000001",
		uuid.New(), "张三", 1, 2, 5, nil, "销售提成", "SALES", "REALIZED_PROFIT", 1, biz.CommissionCalculationVersion,
		"fingerprint", "CONFIRMED", "USD", "500.00000000", "200.00000000", "300.00000000", "300.00000000",
		"10.0000", "30.00000000", "2026-08-15", "0.50000000", "DERIVED", "2026-08-14", nil, "15.00000000",
		0, nil, 1, nil, nil, nil, nil, nil, nil, nil,
	}
}

func TestCommissionRepoExportBatchUsesStableOrderingAndDynamicCNY(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	org := uuid.New()
	// 第二批读取：固定排序 + LIMIT 200 OFFSET 200，末尾锚定防止误加锁。
	mock.ExpectQuery(`SELECT "finance_commissions".*ORDER BY "finance_commissions"\."commission_date" DESC, "finance_commissions"\."created_at" DESC, "finance_commissions"\."id" DESC LIMIT 200 OFFSET 200$`).
		WillReturnRows(sqlmock.NewRows(commission.Columns).AddRow(commissionExportScanRow(t, org)...))
	mock.ExpectQuery(`SELECT "finance_commission_adjustments".*FROM "finance_commission_adjustments".*ORDER BY "finance_commission_adjustments"\."created_at"`).
		WillReturnRows(sqlmock.NewRows(adjustment.Columns))

	items, err := repo.ExportBatch(context.Background(), org, biz.CommissionFilter{Page: 2, PageSize: biz.MaxListPageSize})
	if err != nil {
		t.Fatalf("ExportBatch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("批量读取结果不符: %d", len(items))
	}
	item := items[0]
	if item.CommissionNo != "TC20260815000001" || item.Status != biz.CommissionConfirmed || item.CommissionDate != "2026-08-15" {
		t.Fatalf("批量读取字段转换不符: %#v", item)
	}
	// 无确认调整时 CNY 有效提成等于原始 CNY 快照，动态口径与列表一致。
	if item.CNYCommissionAmount.StringFixed(8) != "15.00000000" || item.CNYAdjustmentAmount.StringFixed(8) != "0.00000000" ||
		item.CNYEffectiveCommissionAmount.StringFixed(8) != "15.00000000" {
		t.Fatalf("动态 CNY 金额不符: %#v", item)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("导出批量读取未复用列表排序、分页或未加载调整单: %v", err)
	}
}

func TestCommissionRepoSaveExportAuditWritesOutsideTransaction(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	org, actor := uuid.New(), uuid.New()
	mock.ExpectExec(`INSERT INTO "audit_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	event := biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "finance.commission.export", Result: "success", ResourceType: "finance_commission", Details: map[string]string{"row_count": "3"}}
	if err := repo.SaveExportAudit(context.Background(), &event); err != nil {
		t.Fatalf("SaveExportAudit() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("导出审计应在无事务上下文中直接写入: %v", err)
	}
}

func TestCommissionCalculationBillsQueryOrderingAndLocking(t *testing.T) {
	org := uuid.New()
	billID1 := uuid.New()
	billID2 := uuid.New()

	t.Run("锁定路径严格要求主键升序排序与FOR UPDATE", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		mock.ExpectQuery(`SELECT .* FROM "finance_bills" WHERE .* ORDER BY "finance_bills"\."id" FOR UPDATE$`).
			WillReturnError(errors.New("stop_after_bill_query"))

		bq := commissionCalculationBillsQuery(commissionStoreFromClient(repo.data.db), org, []uuid.UUID{billID2, billID1}, true)
		_, err := bq.All(context.Background())
		if err == nil || err.Error() != "stop_after_bill_query" {
			t.Fatalf("查询执行错误 = %v, 期望 stop_after_bill_query", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("账单锁定查询未按主键升序排序或缺少 FOR UPDATE: %v", err)
		}
	})

	t.Run("非锁定路径包含主键升序排序但不含FOR UPDATE", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		mock.ExpectQuery(`SELECT .* FROM "finance_bills" WHERE .* ORDER BY "finance_bills"\."id"$`).
			WillReturnError(errors.New("stop_after_bill_query"))

		bq := commissionCalculationBillsQuery(commissionStoreFromClient(repo.data.db), org, []uuid.UUID{billID2, billID1}, false)
		_, err := bq.All(context.Background())
		if err == nil || err.Error() != "stop_after_bill_query" {
			t.Fatalf("查询执行错误 = %v, 期望 stop_after_bill_query", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("账单非锁定查询未按主键升序排序或误加 FOR UPDATE: %v", err)
		}
	})

	t.Run("生产计算加载路径贯穿调用账单加锁与排序", func(t *testing.T) {
		repo, mock := setupTestCommissionRepo(t)
		verificationID := uuid.New()
		ruleID := uuid.New()
		cashflowID := uuid.New()
		now := time.Now()

		// 1. 核销单查询 (lock=true -> FOR UPDATE)
		mock.ExpectQuery(`SELECT .* FROM "finance_verifications" WHERE .* FOR UPDATE$`).
			WillReturnRows(sqlmock.NewRows(financeverification.Columns).AddRow(
				verificationID, now, now, org, "VR202608300001", "key1", "ACTIVE", "RECEIVABLE",
				uuid.New(), "客户A", "USD", "100.00", "CNY", "7.20000000", "SETTING", "2026-08-30",
				nil, "720.00", "720.00", "720.00", "0.00", "2026-08-30", nil, 1, nil, nil, nil,
			))
		// 2. 分摊边查询
		mock.ExpectQuery(`SELECT .* FROM "finance_verification_allocations" WHERE .*`).
			WillReturnRows(sqlmock.NewRows(financeverificationallocation.Columns).
				AddRow(uuid.New(), now, now, verificationID, cashflowID, billID1, "CF1", "BILL1", "50.00", "360.00", "360.00", "360.00", "0.00", true).
				AddRow(uuid.New(), now, now, verificationID, cashflowID, billID2, "CF1", "BILL2", "50.00", "360.00", "360.00", "360.00", "0.00", true),
			)
		// 3. 提成规则查询 (lock=true -> FOR UPDATE)
		mock.ExpectQuery(`SELECT .* FROM "finance_commission_rules" WHERE .* FOR UPDATE$`).
			WillReturnRows(sqlmock.NewRows(financecommissionrule.Columns).AddRow(
				ruleID, now, now, org, "销售提成", "SALES", "REALIZED_PROFIT", "10.0000", nil, nil, true, nil, 1,
			))
		// 4. 账单批量查询 (要求 ORDER BY "finance_bills"."id" FOR UPDATE)
		mock.ExpectQuery(`SELECT .* FROM "finance_bills" WHERE .* ORDER BY "finance_bills"\."id" FOR UPDATE$`).
			WillReturnError(errors.New("stop_after_bills_query"))

		_, err := loadCommissionCalculationSource(context.Background(), commissionStoreFromClient(repo.data.db), org, verificationID, ruleID, true)
		if err == nil || err.Error() != "stop_after_bills_query" {
			t.Fatalf("loadCommissionCalculationSource() error = %v, 期望 stop_after_bills_query", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("生产计算加载路径未在账单查询中输出 ORDER BY finance_bills.id FOR UPDATE: %v", err)
		}
	})
}
