package biz

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func supplementFeeSnapshotFixture() OrderFeeSupplementFeeSnapshot {
	settingID := uuid.New()
	partyID := uuid.New()
	unitID := uuid.New()
	taxRate := decimal.RequireFromString("6")
	quantity := decimal.RequireFromString("1")
	unitPrice := decimal.RequireFromString("100")
	note := "漏录的拖车费"
	return OrderFeeSupplementFeeSnapshot{
		Direction:          OrderFeePayable,
		FeeSettingID:       &settingID,
		FeeCode:            "TRUCK",
		FeeName:            "拖车费",
		SettlementPartyID:  partyID,
		BillingUnitID:      &unitID,
		BillingUnit:        "票",
		TaxRate:            &taxRate,
		Quantity:           quantity,
		UnitPrice:          unitPrice,
		TotalAmount:        decimal.RequireFromString("100.00000000"),
		TaxInclusive:       true,
		NetAmount:          decimal.RequireFromString("94.33962263"),
		TaxAmount:          decimal.RequireFromString("5.66037737"),
		Currency:           "CNY",
		ExpenseDate:        "2026-09-01",
		BaseCurrency:       "CNY",
		BaseCurrencyAmount: decimal.RequireFromString("100.00000000"),
		Note:               &note,
	}
}

func TestBuildOrderFeeSupplementFingerprint(t *testing.T) {
	orderID := uuid.New()
	snapshot := supplementFeeSnapshotFixture()
	base := BuildOrderFeeSupplementFingerprint(orderID, snapshot, "漏录成本")
	if base == "" || len(base) != 64 {
		t.Fatalf("指纹应为 64 位哈希: %q", base)
	}
	if again := BuildOrderFeeSupplementFingerprint(orderID, snapshot, "漏录成本"); again != base {
		t.Fatalf("同内容指纹必须一致: %s vs %s", base, again)
	}
	if changedReason := BuildOrderFeeSupplementFingerprint(orderID, snapshot, "原因变化"); changedReason == base {
		t.Fatalf("原因变化必须改变指纹")
	}
	changed := snapshot
	changed.TotalAmount = decimal.RequireFromString("120.00000000")
	if changedAmount := BuildOrderFeeSupplementFingerprint(orderID, changed, "漏录成本"); changedAmount == base {
		t.Fatalf("金额变化必须改变指纹")
	}
	if otherOrder := BuildOrderFeeSupplementFingerprint(uuid.New(), snapshot, "漏录成本"); otherOrder == base {
		t.Fatalf("订单变化必须改变指纹")
	}
}

func TestClassifySupplementLockBasis(t *testing.T) {
	evidence := &OrderFeeSupplementLockEvidence{
		BusinessLocked:           true,
		BusinessLockGeneration:   4,
		FinancialLocked:          true,
		FinancialEvidenceVersion: financialEvidenceVersionForTest(),
		FinancialEvidenceHash:    "hash-1",
		FinancialNetAmount:       decimal.RequireFromString("80.00000000"),
	}
	basis, generation, version, hash, net := classifySupplementLockBasis(evidence)
	if basis != SupplementLockBasisBoth || generation == nil || *generation != 4 || version == nil || hash == nil || net == nil || net.Sign() <= 0 {
		t.Fatalf("双锁应固化为 BOTH 且携带两组依据: %s", basis)
	}
	businessOnly := &OrderFeeSupplementLockEvidence{BusinessLocked: true, BusinessLockGeneration: 2, OrderNo: "SO-1"}
	basis, generation, version, hash, net = classifySupplementLockBasis(businessOnly)
	if basis != SupplementLockBasisBusiness || generation == nil || *generation != 2 || version != nil || hash != nil || net != nil {
		t.Fatalf("仅业务锁应固化为 BUSINESS 且财务字段为空: %s", basis)
	}
	financialOnly := &OrderFeeSupplementLockEvidence{FinancialLocked: true, FinancialEvidenceVersion: financialEvidenceVersionForTest(), FinancialEvidenceHash: "hash-9", FinancialNetAmount: decimal.RequireFromString("1.00000000")}
	basis, generation, version, hash, net = classifySupplementLockBasis(financialOnly)
	if basis != SupplementLockBasisFinancial || generation != nil || version == nil || hash == nil || net == nil {
		t.Fatalf("仅财务锁应固化为 FINANCIAL 且业务字段为空: %s", basis)
	}
	if none, _, _, _, _ := classifySupplementLockBasis(&OrderFeeSupplementLockEvidence{}); none != "" {
		t.Fatalf("无锁应返回空 basis 引导普通新增: %s", none)
	}
}

func financialEvidenceVersionForTest() string {
	return "FIN_LOCK_EVIDENCE_V1"
}

func TestVerifyLockBasis(t *testing.T) {
	generation := uint64(7)
	version := financialEvidenceVersionForTest()
	hash := "hash-same"
	netAmount := decimal.RequireFromString("66.00000000")
	request := &OrderFeeSupplementRequest{
		LockBasis:                    SupplementLockBasisBusiness,
		BusinessLockGeneration:       &generation,
		FinancialLockEvidenceVersion: &version,
		FinancialLockEvidenceHash:    &hash,
		FinancialLockNetAmount:       &netAmount,
	}

	sameGeneration := &OrderFeeSupplementLockEvidence{BusinessLocked: true, BusinessLockGeneration: generation}
	if err := verifyLockBasis(request, sameGeneration); err != nil {
		t.Fatalf("同一业务锁代次应通过复核: %v", err)
	}
	newGeneration := &OrderFeeSupplementLockEvidence{BusinessLocked: true, BusinessLockGeneration: generation + 1}
	err := verifyLockBasis(request, newGeneration)
	if err == nil || !strings.Contains(err.Error(), "重新发起补录") {
		t.Fatalf("代次失效且仍有新锁应提示重新发起: %v", err)
	}
	noLock := &OrderFeeSupplementLockEvidence{}
	err = verifyLockBasis(request, noLock)
	if err == nil || !strings.Contains(err.Error(), "普通费用新增") {
		t.Fatalf("代次失效且已无锁应提示普通新增: %v", err)
	}

	request.LockBasis = SupplementLockBasisFinancial
	if err := verifyLockBasis(request, &OrderFeeSupplementLockEvidence{FinancialLocked: true, FinancialEvidenceVersion: version, FinancialEvidenceHash: hash, FinancialNetAmount: netAmount}); err != nil {
		t.Fatalf("同版本同哈希且净额为正应通过复核: %v", err)
	}
	// CONFIRMED↔PAID 流转不改变证据：净额与哈希不变即通过。
	if err := verifyLockBasis(request, &OrderFeeSupplementLockEvidence{FinancialLocked: true, FinancialEvidenceVersion: version, FinancialEvidenceHash: hash, FinancialNetAmount: decimal.RequireFromString("66.00000000")}); err != nil {
		t.Fatalf("净额不变时复核必须通过: %v", err)
	}
	changedSet := &OrderFeeSupplementLockEvidence{FinancialLocked: true, FinancialEvidenceVersion: version, FinancialEvidenceHash: "hash-changed", FinancialNetAmount: netAmount}
	if err := verifyLockBasis(request, changedSet); err == nil {
		t.Fatalf("证据集合变化必须拒绝审批")
	}
	released := &OrderFeeSupplementLockEvidence{FinancialLocked: false, FinancialEvidenceVersion: version, FinancialEvidenceHash: hash, FinancialNetAmount: decimal.Zero}
	if err := verifyLockBasis(request, released); err == nil {
		t.Fatalf("净额释放后必须拒绝审批")
	}

	// BOTH：任一依据匹配即可。
	request.LockBasis = SupplementLockBasisBoth
	if err := verifyLockBasis(request, sameGeneration); err != nil {
		t.Fatalf("BOTH 依据业务代次匹配应通过: %v", err)
	}
	if err := verifyLockBasis(request, &OrderFeeSupplementLockEvidence{FinancialLocked: true, FinancialEvidenceVersion: version, FinancialEvidenceHash: hash, FinancialNetAmount: netAmount}); err != nil {
		t.Fatalf("BOTH 依据财务证据匹配应通过: %v", err)
	}
	if err := verifyLockBasis(request, noLock); err == nil {
		t.Fatalf("BOTH 双依据全部失效必须拒绝")
	}
}

func TestCommissionSupplementCostSensitivity(t *testing.T) {
	sensitive, err := CommissionSupplementCostSensitivity(CommissionCalculationVersion, CommissionBasisRealizedProfit)
	if err != nil || !sensitive {
		t.Fatalf("REALIZED_PROFIT 应为成本敏感: %v %v", sensitive, err)
	}
	insensitive, err := CommissionSupplementCostSensitivity(CommissionCalculationVersion, CommissionBasisRealizedRevenue)
	if err != nil || insensitive {
		t.Fatalf("REALIZED_REVENUE 不受应付成本影响且不应阻断: %v %v", insensitive, err)
	}
	if _, err := CommissionSupplementCostSensitivity("UNKNOWN_VERSION_V9", CommissionBasisRealizedProfit); err != ErrCommissionCalculationVersionUnsupported {
		t.Fatalf("未知计算版本必须失败关闭: %v", err)
	}
	if _, err := CommissionSupplementCostSensitivity(CommissionCalculationVersion, CommissionCalculationBasis("OTHER")); err != ErrCommissionCalculationVersionUnsupported {
		t.Fatalf("未知计提口径必须失败关闭: %v", err)
	}
}

func TestComputeSupplementMarginalImpact(t *testing.T) {
	// 历史快照：收入 1000、总应收 1000、总应付 400、比例 10%。
	realized := decimal.RequireFromString("1000")
	receivable := decimal.RequireFromString("1000")
	payable := decimal.RequireFromString("400")
	rate := decimal.RequireFromString("10")
	// before: 利润 600 → 提成 60。
	// after（补录 100）: 应付 500 → 利润 500 → 提成 50；边际差 10。
	impact := ComputeSupplementMarginalImpact(realized, receivable, payable, decimal.Zero, decimal.RequireFromString("100"), rate, CommissionBasisRealizedProfit)
	if impact.AmountBefore.StringFixed(8) != "60.00000000" || impact.AmountAfter.StringFixed(8) != "50.00000000" || impact.Delta.StringFixed(8) != "10.00000000" {
		t.Fatalf("边际影响计算不符: %+v", impact)
	}
	// 此前未作废补录进入 before 基线：再次补录 100 后 after 应付 600、提成 40、差 20。
	impact = ComputeSupplementMarginalImpact(realized, receivable, payable, decimal.RequireFromString("100"), decimal.RequireFromString("100"), rate, CommissionBasisRealizedProfit)
	if impact.AmountAfter.StringFixed(8) != "40.00000000" || impact.Delta.StringFixed(8) != "20.00000000" {
		t.Fatalf("历史补录应进入 before 基线: %+v", impact)
	}
	// 亏损时 before 提成为零则不生成负差。
	impact = ComputeSupplementMarginalImpact(realized, receivable, decimal.RequireFromString("1200"), decimal.Zero, decimal.RequireFromString("100"), rate, CommissionBasisRealizedProfit)
	if impact.Delta.Sign() != 0 {
		t.Fatalf("提成已为零时不应生成冲减: %+v", impact)
	}
	// REALIZED_REVENUE 口径不随应付变化。
	impact = ComputeSupplementMarginalImpact(realized, receivable, payable, decimal.Zero, decimal.RequireFromString("100"), rate, CommissionBasisRealizedRevenue)
	if impact.AmountBefore.StringFixed(8) != impact.AmountAfter.StringFixed(8) || impact.Delta.Sign() != 0 {
		t.Fatalf("收入口径不应生成冲减: %+v", impact)
	}
}

func TestCapSupplementSuggestion(t *testing.T) {
	marginal := decimal.RequireFromString("30")
	suggested, excess := CapSupplementSuggestion(marginal, decimal.RequireFromString("25"), decimal.RequireFromString("100"))
	if suggested.StringFixed(8) != "25.00000000" || excess.StringFixed(8) != "5.00000000" {
		t.Fatalf("订单行余额封顶不符: %s %s", suggested, excess)
	}
	suggested, excess = CapSupplementSuggestion(marginal, decimal.RequireFromString("100"), decimal.RequireFromString("12.5"))
	if suggested.StringFixed(8) != "12.50000000" || excess.StringFixed(8) != "17.50000000" {
		t.Fatalf("父单余额封顶不符: %s %s", suggested, excess)
	}
	suggested, excess = CapSupplementSuggestion(marginal, decimal.RequireFromString("-5"), decimal.RequireFromString("100"))
	if suggested.Sign() != 0 || excess.StringFixed(8) != marginal.StringFixed(8) {
		t.Fatalf("负余额不应生成建议且超出额应完整入审计: %s %s", suggested, excess)
	}
}

func TestFeeSupplementNotificationRendering(t *testing.T) {
	approval := &NotificationDelivery{
		Channel:        NotificationChannelDingTalk,
		Template:       NotificationTemplateFeeSupplementApprovalPending,
		ResourceType:   "FEE_SUPPLEMENT_REQUEST",
		ResourceID:     uuid.New(),
		ReferenceCode:  "SO-100",
		Parameter:      "订单 SO-100 补录应付 100.00000000 CNY",
		DingTalkUserID: "dt-1",
	}
	content, err := renderNotification(approval)
	if err != nil || !strings.Contains(content, "费用补录待审批") || !strings.Contains(content, "SO-100") {
		t.Fatalf("待审批通知渲染不符: %q %v", content, err)
	}
	wrongResource := *approval
	wrongResource.ResourceType = "ORDER"
	if _, err := renderNotification(&wrongResource); err == nil {
		t.Fatalf("资源类型不符的待审批通知必须拒绝渲染")
	}
	decrease := &NotificationDelivery{
		Channel:        NotificationChannelDingTalk,
		Template:       NotificationTemplateCommissionDecreaseSuggested,
		ResourceType:   "COMMISSION_ADJUSTMENT",
		ResourceID:     uuid.New(),
		ReferenceCode:  "SO-100",
		Parameter:      "冲减建议 10.00000000 CNY",
		DingTalkUserID: "dt-2",
	}
	content, err = renderNotification(decrease)
	if err != nil || !strings.Contains(content, "冲减建议") || !strings.Contains(content, "SO-100") {
		t.Fatalf("冲减知情通知渲染不符: %q %v", content, err)
	}
	wrongResource = *decrease
	wrongResource.ResourceType = "FEE_SUPPLEMENT_REQUEST"
	if _, err := renderNotification(&wrongResource); err == nil {
		t.Fatalf("资源类型不符的知情通知必须拒绝渲染")
	}
}
