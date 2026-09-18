package data

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// bfBackfillTime 是回填场景统一的「提成行首次写入时点」（唯一截止时点），
// 固定取过去时刻，保证测试内构造的截止时点前后事实稳定可判。
var bfBackfillTime = time.Now().Add(-1 * time.Hour).UTC()

// bfCreateFee 创建一条 CNY 已确认订单费用并返回配套的费用快照条目（快照与
// 生成路径同构：成员与金额以提成行不可变费用快照为准）。
func bfCreateFee(f *feeSupplementPostgresFixture, order *ent.Order, direction string, baseAmount, key string) (*ent.OrderFee, biz.CommissionFeeDetail) {
	f.t.Helper()
	detail := biz.CommissionFeeDetail{
		FeeID:               uuid.New(),
		Direction:           direction,
		FeeCode:             "BF_" + direction,
		FeeName:             "回填测试费用-" + key,
		SettlementPartyID:   f.customerID,
		SettlementPartyName: "回填测试客户",
		Currency:            "CNY",
		TotalAmount:         decimal.RequireFromString(baseAmount),
		ExchangeRate:        decimal.NewFromInt(1),
		BaseCurrency:        "CNY",
		BaseCurrencyAmount:  decimal.RequireFromString(baseAmount),
		ExpenseDate:         "2026-09-01",
		Status:              "CONFIRMED",
	}
	created, err := f.data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("bf-fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.Direction(direction)).
		SetStatus(orderfeeent.StatusCONFIRMED).
		SetFeeCode(detail.FeeCode).
		SetFeeName(detail.FeeName).
		SetSettlementPartyID(f.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice(decimal.RequireFromString(baseAmount).StringFixed(4)).
		SetTotalAmount(decimal.RequireFromString(baseAmount).StringFixed(8)).
		SetNetAmount(decimal.RequireFromString(baseAmount).StringFixed(8)).
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate("2026-09-01").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(decimal.RequireFromString(baseAmount).StringFixed(8)).
		SetExpenseDate("2026-09-01").
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建回填测试费用 %s: %v", key, err)
	}
	detail.FeeID = created.ID
	return created, detail
}

// bfCreateBillWithLine 为费用创建账单行，支持注入截止时点前后事实：
// 账单行创建时间、活动状态与账单停用时间均显式给定。
func bfCreateBillWithLine(f *feeSupplementPostgresFixture, order *ent.Order, fee *ent.OrderFee, lineBase string, lineCreatedAt time.Time, active bool, billCancelledAt *time.Time) *ent.FinanceBill {
	f.t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("BILL-BF-" + suffix).
		SetIdempotencyKey("bill-bf-" + suffix).
		SetDirection(financebillent.DirectionPAYABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("回填测试客户").
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate("2026-09-01").
		SetTotalAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetNetAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetFeeCount(1).
		SetBillDate("2026-09-01").
		SetVersion(1)
	if billCancelledAt != nil {
		// 数据库 CHECK 要求 CANCELLED 账单同时携带停用人与原因，与真实停用一致。
		billCreate = billCreate.
			SetStatus(financebillent.StatusCANCELLED).
			SetCancelledAt(*billCancelledAt).
			SetCancelledBy(f.approverID).
			SetCancellationReason("回填测试停用账单")
	}
	bill, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建回填测试账单: %v", err)
	}
	if _, err := f.data.db.FinanceBillLine.Create().
		SetBillID(bill.ID).
		SetOrderID(order.ID).
		SetOrderFeeID(fee.ID).
		SetOrderNo(order.OrderNo).
		SetFeeCode(fee.FeeCode).
		SetFeeName(fee.FeeName).
		SetQuantity("1.0000").
		SetUnitPrice("1.0000").
		SetTotalAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetNetAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrencyAmount(decimal.RequireFromString(lineBase).StringFixed(8)).
		SetBaseCurrency("CNY").
		SetActive(active).
		SetCreatedAt(lineCreatedAt).
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建回填测试账单行: %v", err)
	}
	return bill
}

// bfLegacyCommissionSpec 是存量提成行回填场景的插入参数。
type bfLegacyCommissionSpec struct {
	order              *ent.Order
	createdAt          time.Time
	realizedRevenue    string
	ratePercent        string
	allocatedCost      string
	realizedProfit     string
	commissionAmount   string
	calculationVersion string
	basis              biz.CommissionCalculationBasis
	fees               []biz.CommissionFeeDetail
}

// bfInsertLegacyCommission 插入一张 CONFIRMED 提成父单与「存量行」：复算快照
// 全空（迁移前状态），费用快照与首次写入时间显式给定。
func bfInsertLegacyCommission(f *feeSupplementPostgresFixture, spec bfLegacyCommissionSpec) *ent.FinanceCommission {
	f.t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	rule, ruleErr := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("回填测试规则-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(f.ctx)
	if ruleErr != nil {
		f.t.Fatalf("插入回填测试规则: %v", ruleErr)
	}
	calculationVersion := spec.calculationVersion
	if calculationVersion == "" {
		calculationVersion = biz.CommissionCalculationVersion
	}
	feeSnapshot, marshalErr := json.Marshal(spec.fees)
	if marshalErr != nil {
		f.t.Fatalf("序列化回填费用快照: %v", marshalErr)
	}
	parent, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("FC-BF-" + suffix).
		SetIdempotencyKey("bf-commission-" + suffix).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("回填测试员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(len(spec.fees)).
		SetRuleID(rule.ID).
		SetRuleName("回填测试规则-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis(string(spec.basis)).
		SetCalculationVersion(calculationVersion).
		SetSourceFingerprint(strings.Repeat("c", 64)).
		SetStatus(commissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue(spec.realizedRevenue).
		SetAllocatedCost(spec.allocatedCost).
		SetRealizedProfit(spec.realizedProfit).
		SetCommissionBaseAmount(spec.realizedProfit).
		SetRatePercent(spec.ratePercent).
		SetCommissionAmount(spec.commissionAmount).
		SetCommissionDate("2026-09-01").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate("2026-09-01").
		SetCnyCommissionAmount(spec.commissionAmount).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入回填测试提成父单: %v", err)
	}
	if _, lineErr := f.data.db.FinanceCommissionLine.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(spec.order.ID).
		SetOrderNo(spec.order.OrderNo).
		SetOrderDate("2026-08-01").
		SetCustomerID(f.customerID).
		SetCustomerCode("CUST").
		SetCustomerName("回填测试客户").
		SetPersonnelAssignmentID(uuid.New()).
		SetPersonnelOrganizationID(f.organizationID).
		SetPersonnelAssignedAt(time.Now().UTC()).
		SetFeeCount(len(spec.fees)).
		SetFeeSnapshot(string(feeSnapshot)).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("回填测试员工-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis(string(spec.basis)).
		SetBaseCurrency("CNY").
		SetRealizedRevenue(spec.realizedRevenue).
		SetAllocatedCost(spec.allocatedCost).
		SetRealizedProfit(spec.realizedProfit).
		SetCommissionBaseAmount(spec.realizedProfit).
		SetRatePercent(spec.ratePercent).
		SetCommissionAmount(spec.commissionAmount).
		SetCreatedAt(spec.createdAt).
		Save(f.ctx); lineErr != nil {
		f.t.Fatalf("插入回填测试提成订单行: %v", lineErr)
	}
	return parent
}

// bfRequireLineSnapshot 断言提成行复算快照的落库结果。
func bfRequireLineSnapshot(f *feeSupplementPostgresFixture, commissionID uuid.UUID, wantStatus, wantSource, wantReceivable, wantPayable, wantReason string) {
	f.t.Helper()
	line, err := f.data.db.FinanceCommissionLine.Query().
		Where(commissionlineent.CommissionIDEQ(commissionID)).
		Only(f.ctx)
	if err != nil {
		f.t.Fatalf("读取回填提成行: %v", err)
	}
	status, source := "", ""
	if line.SnapshotStatus != nil {
		status = string(*line.SnapshotStatus)
	}
	if line.SnapshotSource != nil {
		source = string(*line.SnapshotSource)
	}
	if status != wantStatus || source != wantSource {
		f.t.Fatalf("快照状态或来源不符: status=%q source=%q want=%q/%q", status, source, wantStatus, wantSource)
	}
	if wantReceivable != "" && (line.TotalReceivableSnapshot == nil || *line.TotalReceivableSnapshot != wantReceivable) {
		f.t.Fatalf("历史总应收分母不符: got=%v want=%s", line.TotalReceivableSnapshot, wantReceivable)
	}
	if wantPayable != "" && (line.TotalPayableSnapshot == nil || *line.TotalPayableSnapshot != wantPayable) {
		f.t.Fatalf("历史总应付分母不符: got=%v want=%s", line.TotalPayableSnapshot, wantPayable)
	}
	if wantStatus == "READY" {
		if line.SnapshotBackfillVersion == nil || *line.SnapshotBackfillVersion != CommissionSnapshotBackfillVersion {
			f.t.Fatalf("回填算法版本不符: %v", line.SnapshotBackfillVersion)
		}
		if line.SnapshotEvidenceHash == nil || len(*line.SnapshotEvidenceHash) != 64 {
			f.t.Fatalf("证据集合哈希缺失或长度不符: %v", line.SnapshotEvidenceHash)
		}
		if line.SnapshotUnavailableReasonCode != nil {
			f.t.Fatalf("READY 行不应携带不可用原因码: %v", *line.SnapshotUnavailableReasonCode)
		}
	}
	if wantReason != "" && (line.SnapshotUnavailableReasonCode == nil || *line.SnapshotUnavailableReasonCode != wantReason) {
		f.t.Fatalf("不可用原因码不符: got=%v want=%s", line.SnapshotUnavailableReasonCode, wantReason)
	}
}

// TestFeeSupplementCommissionSnapshotBackfillPostgres 验证存量提成行复算快照
// 回填迁移：可还原行按「计算快照形成时点」确定性写 READY+MIGRATED，截止时点
// 前的不可变费用/账单行事实唯一决定分母；曾作废或后补费用不改变结果；不可
// 还原行标记 UNAVAILABLE + 稳定原因码且不改写原提成金额；回填后审批链路可
// 正常消费 READY 行。
func TestFeeSupplementCommissionSnapshotBackfillPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	f := newFeeSupplementFixture(t)
	ctx := context.Background()
	// REALIZED_PROFIT 下分母与已存结果的一致性锚点：realized=1000、receivable=1000、
	// rate=10 时 cost=payable、profit=1000−payable、amount=100−payable/10。
	runBackfill := func() *CommissionSnapshotBackfillReport {
		f.t.Helper()
		report, err := BackfillCommissionLineSnapshots(ctx, f.data.sqlDB)
		if err != nil {
			f.t.Fatalf("执行存量提成行快照回填: %v", err)
		}
		return report
	}

	// 场景 1：未建账费用按费用自身快照重建分母，复算一致写 READY+MIGRATED。
	order1 := f.createOrder("SE-BF-1-" + f.suffix)
	_, recFee1 := bfCreateFee(f, order1, "RECEIVABLE", "1000.00000000", "s1-rec")
	_, payFee1 := bfCreateFee(f, order1, "PAYABLE", "400.00000000", "s1-pay")
	parent1 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order1, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "60.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee1, payFee1},
	})

	// 场景 2：已建账费用取截止时点前账单行本位币（500 而非费用自身 400）。
	order2 := f.createOrder("SE-BF-2-" + f.suffix)
	_, recFee2 := bfCreateFee(f, order2, "RECEIVABLE", "1000.00000000", "s2-rec")
	payFee2, payDetail2 := bfCreateFee(f, order2, "PAYABLE", "400.00000000", "s2-pay")
	bfCreateBillWithLine(f, order2, payFee2, "500.00000000", bfBackfillTime.Add(-30*time.Minute), true, nil)
	parent2 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order2, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "500.00000000", realizedProfit: "500.00000000", commissionAmount: "50.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee2, payDetail2},
	})

	// 场景 3：截止时点活动、截止时点后停用的账单行仍参与分母（停用发生在后）。
	order3 := f.createOrder("SE-BF-3-" + f.suffix)
	_, recFee3 := bfCreateFee(f, order3, "RECEIVABLE", "1000.00000000", "s3-rec")
	payFee3, payDetail3 := bfCreateFee(f, order3, "PAYABLE", "400.00000000", "s3-pay")
	cancelledAfter := time.Now().UTC()
	bfCreateBillWithLine(f, order3, payFee3, "500.00000000", bfBackfillTime.Add(-30*time.Minute), false, &cancelledAfter)
	parent3 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order3, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "500.00000000", realizedProfit: "500.00000000", commissionAmount: "50.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee3, payDetail3},
	})

	// 场景 4：后补费用与截止时点后新建的账单行不改变截止时点前分母。
	order4 := f.createOrder("SE-BF-4-" + f.suffix)
	_, recFee4 := bfCreateFee(f, order4, "RECEIVABLE", "1000.00000000", "s4-rec")
	payFee4, payDetail4 := bfCreateFee(f, order4, "PAYABLE", "400.00000000", "s4-pay")
	parent4 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order4, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "60.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee4, payDetail4},
	})
	// 截止时点之后才发生的事实：后补应付费用 999，以及为快照内应付费用新
	// 建的账单行 999。二者都不得进入历史分母。
	bfCreateFee(f, order4, "PAYABLE", "999.00000000", "s4-post-fee")
	bfCreateBillWithLine(f, order4, payFee4, "999.00000000", time.Now().UTC(), true, nil)

	// 场景 5：账单在截止时点前已停用，费用回落自身快照（400 而非 500）。
	order5 := f.createOrder("SE-BF-5-" + f.suffix)
	_, recFee5 := bfCreateFee(f, order5, "RECEIVABLE", "1000.00000000", "s5-rec")
	payFee5, payDetail5 := bfCreateFee(f, order5, "PAYABLE", "400.00000000", "s5-pay")
	cancelledBefore := bfBackfillTime.Add(-30 * time.Minute)
	bfCreateBillWithLine(f, order5, payFee5, "500.00000000", bfBackfillTime.Add(-2*time.Hour), false, &cancelledBefore)
	parent5 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order5, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "60.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee5, payDetail5},
	})

	// 场景 6：复算与已存值不一致 → RECOMPUTE_MISMATCH，不改写原提成金额。
	order6 := f.createOrder("SE-BF-6-" + f.suffix)
	_, recFee6 := bfCreateFee(f, order6, "RECEIVABLE", "1000.00000000", "s6-rec")
	_, payFee6 := bfCreateFee(f, order6, "PAYABLE", "400.00000000", "s6-pay")
	parent6 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order6, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "99.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee6, payFee6},
	})

	// 场景 7：未知 calculation_version → CALCULATION_VERSION_UNSUPPORTED，失败关闭。
	order7 := f.createOrder("SE-BF-7-" + f.suffix)
	_, recFee7 := bfCreateFee(f, order7, "RECEIVABLE", "1000.00000000", "s7-rec")
	_, payFee7 := bfCreateFee(f, order7, "PAYABLE", "400.00000000", "s7-pay")
	parent7 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order7, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "60.00000000",
		calculationVersion: "CUSTOMER_REALIZED_PROFIT_V2",
		basis:              biz.CommissionBasisRealizedProfit,
		fees:               []biz.CommissionFeeDetail{recFee7, payFee7},
	})

	// 场景 8：已停用账单行缺少停用时间证据 → BILL_LINE_NOT_TRACEABLE。
	order8 := f.createOrder("SE-BF-8-" + f.suffix)
	_, recFee8 := bfCreateFee(f, order8, "RECEIVABLE", "1000.00000000", "s8-rec")
	payFee8, payDetail8 := bfCreateFee(f, order8, "PAYABLE", "400.00000000", "s8-pay")
	bfCreateBillWithLine(f, order8, payFee8, "500.00000000", bfBackfillTime.Add(-30*time.Minute), false, nil)
	parent8 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order8, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "500.00000000", realizedProfit: "500.00000000", commissionAmount: "50.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee8, payDetail8},
	})

	report := runBackfill()
	if report.TotalLines != 8 || report.ReadyCount != 5 || report.UnavailableCount != 3 {
		f.t.Fatalf("回填报告统计不符: total=%d ready=%d unavailable=%d (%+v)", report.TotalLines, report.ReadyCount, report.UnavailableCount, report.UnavailableByReason)
	}
	for _, reason := range []string{"RECOMPUTE_MISMATCH", "CALCULATION_VERSION_UNSUPPORTED", "BILL_LINE_NOT_TRACEABLE"} {
		if report.UnavailableByReason[reason] != 1 {
			f.t.Fatalf("原因码 %s 计数不符: %d", reason, report.UnavailableByReason[reason])
		}
	}
	bfRequireLineSnapshot(f, parent1.ID, "READY", "MIGRATED", "1000.00000000", "400.00000000", "")
	bfRequireLineSnapshot(f, parent2.ID, "READY", "MIGRATED", "1000.00000000", "500.00000000", "")
	bfRequireLineSnapshot(f, parent3.ID, "READY", "MIGRATED", "1000.00000000", "500.00000000", "")
	bfRequireLineSnapshot(f, parent4.ID, "READY", "MIGRATED", "1000.00000000", "400.00000000", "")
	bfRequireLineSnapshot(f, parent5.ID, "READY", "MIGRATED", "1000.00000000", "400.00000000", "")
	bfRequireLineSnapshot(f, parent6.ID, "UNAVAILABLE", "", "", "", "RECOMPUTE_MISMATCH")
	bfRequireLineSnapshot(f, parent7.ID, "UNAVAILABLE", "", "", "", "CALCULATION_VERSION_UNSUPPORTED")
	bfRequireLineSnapshot(f, parent8.ID, "UNAVAILABLE", "", "", "", "BILL_LINE_NOT_TRACEABLE")

	// 不可还原行不覆盖原提成金额、不填写猜测分母。
	tampered, err := f.data.db.FinanceCommissionLine.Query().Where(commissionlineent.CommissionIDEQ(parent6.ID)).Only(ctx)
	if err != nil {
		f.t.Fatalf("读取篡改行: %v", err)
	}
	if tampered.CommissionAmount != "99.00000000" || tampered.TotalReceivableSnapshot != nil || tampered.TotalPayableSnapshot != nil {
		f.t.Fatalf("不可还原行被改写: amount=%s receivable=%v payable=%v", tampered.CommissionAmount, tampered.TotalReceivableSnapshot, tampered.TotalPayableSnapshot)
	}

	// 重入幂等：再次执行不处理任何行。
	repeat := runBackfill()
	if repeat.TotalLines != 0 {
		f.t.Fatalf("重入回填不应处理已回填行: total=%d", repeat.TotalLines)
	}

	// 场景 9：回填后审批链路可正常消费 READY 行——补录审批基于回填分母生成建议。
	order9 := f.createOrder("SE-BF-9-" + f.suffix)
	f.lockOrder(order9)
	_, recFee9 := bfCreateFee(f, order9, "RECEIVABLE", "1000.00000000", "s9-rec")
	_, payFee9 := bfCreateFee(f, order9, "PAYABLE", "400.00000000", "s9-pay")
	parent9 := bfInsertLegacyCommission(f, bfLegacyCommissionSpec{
		order: order9, createdAt: bfBackfillTime,
		realizedRevenue: "1000.00000000", ratePercent: "10.0000",
		allocatedCost: "400.00000000", realizedProfit: "600.00000000", commissionAmount: "60.00000000",
		basis: biz.CommissionBasisRealizedProfit,
		fees:  []biz.CommissionFeeDetail{recFee9, payFee9},
	})
	second := runBackfill()
	if second.TotalLines != 1 || second.ReadyCount != 1 {
		f.t.Fatalf("第二批回填统计不符: total=%d ready=%d", second.TotalLines, second.ReadyCount)
	}
	bfRequireLineSnapshot(f, parent9.ID, "READY", "MIGRATED", "1000.00000000", "400.00000000", "")

	request := f.createSupplement(order9, "bf-approve-"+f.suffix)
	result := f.approveSupplement(order9, request.ID, 1)
	if len(result.Suggestions) != 1 {
		f.t.Fatalf("回填行应生成一条冲减建议: got=%d", len(result.Suggestions))
	}
	expected := biz.ComputeSupplementMarginalImpact(
		decimal.RequireFromString("1000.00000000"),
		decimal.RequireFromString("1000.00000000"),
		decimal.RequireFromString("400.00000000"),
		decimal.Zero, result.Fee.BaseCurrencyAmount,
		decimal.RequireFromString("10.0000"), biz.CommissionBasisRealizedProfit,
	)
	if result.Suggestions[0].Amount.StringFixed(8) != expected.Delta.StringFixed(8) || expected.Delta.Sign() <= 0 {
		f.t.Fatalf("回填分母生成的建议金额不符: got=%s want=%s", result.Suggestions[0].Amount, expected.Delta)
	}
	suggestions := f.adjustmentsForRequest(request.ID)
	if len(suggestions) != 1 || suggestions[0].SourceType != commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT {
		f.t.Fatalf("回填行审批生成的调整不符: %+v", suggestions)
	}
}
