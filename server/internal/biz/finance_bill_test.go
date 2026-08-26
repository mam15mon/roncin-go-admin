package biz

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestBuildFinanceBillAggregatesExactSnapshots(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(partyID, "100.00000000", "94.33962264", "5.66037736", "100.00000000"),
		financeBillableFeeForTest(partyID, "0.02000000", "0.01886792", "0.00113208", "0.02000000"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	bill, err := buildFinanceBill(organizationID, fees, input)
	if err != nil {
		t.Fatalf("构建账单失败: %v", err)
	}
	if bill.TotalAmount.StringFixed(8) != "100.02000000" || bill.NetAmount.StringFixed(8) != "94.35849056" || bill.TaxAmount.StringFixed(8) != "5.66150944" {
		t.Fatalf("账单精确汇总不正确: total=%s net=%s tax=%s", bill.TotalAmount, bill.NetAmount, bill.TaxAmount)
	}
	if bill.FeeCount != 2 || len(bill.Lines) != 2 {
		t.Fatalf("账单费用快照数量不正确: count=%d lines=%d", bill.FeeCount, len(bill.Lines))
	}
}

func TestBuildFinanceBillRejectsMixedSettlementScope(t *testing.T) {
	firstPartyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(firstPartyID, "100", "100", "0", "100"),
		financeBillableFeeForTest(uuid.Must(uuid.NewV7()), "100", "100", "0", "100"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	if _, err := buildFinanceBill(uuid.Must(uuid.NewV7()), fees, input); err != ErrFinanceBillFeeMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeCreateFinanceBillRejectsDuplicateFeesAndInvalidDueDate(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID, feeID}, BillDate: "2026-08-26", IdempotencyKey: "duplicate"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("重复费用应被拒绝，实际错误为 %v", err)
	}
	dueDate := "2026-08-25"
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", DueDate: &dueDate, IdempotencyKey: "due-date"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("早于账单日期的到期日应被拒绝，实际错误为 %v", err)
	}
}

func TestBuildFinanceBillBatchPreviewUsesFixedAndOptionalDimensions(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	first := financeBillableFeeForTest(partyID, "100", "94.33962264", "5.66037736", "100")
	second := financeBillableFeeForTest(partyID, "200", "188.67924528", "11.32075472", "200")
	third := financeBillableFeeForTest(partyID, "300", "283.01886792", "16.98113208", "300")
	second.Fee.OrderID = first.Fee.OrderID
	second.OrderNo = first.OrderNo
	otherRate := decimal.RequireFromString("9")
	third.Fee.TaxRate = &otherRate
	fees := []*FinanceBillableFee{third, first, second}

	basePreview, err := BuildFinanceBillBatchPreview(organizationID, fees, FinanceBillGroupingPolicy{})
	if err != nil {
		t.Fatalf("按固定维度预览失败: %v", err)
	}
	if len(basePreview.Groups) != 1 {
		t.Fatalf("固定维度应合并为 1 组，实际为 %d", len(basePreview.Groups))
	}
	if basePreview.Groups[0].OrderID != nil || basePreview.Groups[0].TaxRate != nil {
		t.Fatalf("未启用可选策略时不应返回订单或税率分组维度")
	}

	orderPreview, err := BuildFinanceBillBatchPreview(organizationID, fees, FinanceBillGroupingPolicy{SplitByOrder: true})
	if err != nil {
		t.Fatalf("按订单拆分预览失败: %v", err)
	}
	if len(orderPreview.Groups) != 2 {
		t.Fatalf("按订单拆分应得到 2 组，实际为 %d", len(orderPreview.Groups))
	}

	fullPreview, err := BuildFinanceBillBatchPreview(organizationID, fees, FinanceBillGroupingPolicy{SplitByOrder: true, SplitByTaxRate: true})
	if err != nil {
		t.Fatalf("按订单和税率拆分预览失败: %v", err)
	}
	if len(fullPreview.Groups) != 2 {
		t.Fatalf("按订单和税率拆分应得到 2 组，实际为 %d", len(fullPreview.Groups))
	}
	for _, group := range fullPreview.Groups {
		if group.OrderID == nil || group.OrderNo == nil || group.TaxRate == nil {
			t.Fatalf("启用可选策略后分组必须返回完整维度: %#v", group)
		}
	}
}

func TestBuildFinanceBillBatchPreviewIsDeterministicAndDetectsSnapshotChanges(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	first := financeBillableFeeForTest(partyID, "100", "94.33962264", "5.66037736", "100")
	second := financeBillableFeeForTest(partyID, "200", "188.67924528", "11.32075472", "200")
	fees := []*FinanceBillableFee{first, second}

	forward, err := BuildFinanceBillBatchPreview(organizationID, fees, FinanceBillGroupingPolicy{})
	if err != nil {
		t.Fatalf("首次预览失败: %v", err)
	}
	reverse, err := BuildFinanceBillBatchPreview(organizationID, slices.Clone([]*FinanceBillableFee{second, first}), FinanceBillGroupingPolicy{})
	if err != nil {
		t.Fatalf("倒序预览失败: %v", err)
	}
	if forward.PreviewToken != reverse.PreviewToken || forward.Groups[0].GroupKey != reverse.Groups[0].GroupKey {
		t.Fatalf("相同费用集合不应受输入顺序影响")
	}

	second.Fee.Version++
	changed, err := BuildFinanceBillBatchPreview(organizationID, fees, FinanceBillGroupingPolicy{})
	if err != nil {
		t.Fatalf("快照变化后预览失败: %v", err)
	}
	if changed.PreviewToken == forward.PreviewToken {
		t.Fatalf("费用版本变化后预览令牌必须变化")
	}
}

func TestNormalizeFinanceBillTermsDerivesDueDate(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	terms := 30
	normalized, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", PaymentTermsDays: &terms, IdempotencyKey: "terms"})
	if err != nil {
		t.Fatalf("账期归一化失败: %v", err)
	}
	if normalized.DueDate == nil || *normalized.DueDate != "2026-09-25" {
		t.Fatalf("30 天账期到期日错误: %v", normalized.DueDate)
	}

	inconsistent := "2026-09-24"
	if _, err = normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", DueDate: &inconsistent, PaymentTermsDays: &terms, IdempotencyKey: "bad-terms"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("账期与到期日不一致应被拒绝，实际错误为 %v", err)
	}
}

func financeBillableFeeForTest(partyID uuid.UUID, total, net, tax, base string) *FinanceBillableFee {
	feeID := uuid.Must(uuid.NewV7())
	taxRate := decimal.RequireFromString("6")
	return &FinanceBillableFee{
		OrderNo: "SE2026082600001", BusinessType: "SE",
		Fee: &OrderFee{
			ID: feeID, OrderID: uuid.Must(uuid.NewV7()), Direction: OrderFeeReceivable, Status: OrderFeeConfirmed,
			SettlementPartyID: partyID, SettlementPartyName: "验收客户", Currency: "CNY", BaseCurrency: "CNY",
			FeeCode: "OCEAN", FeeName: "海运费", TotalAmount: decimal.RequireFromString(total),
			NetAmount: decimal.RequireFromString(net), TaxAmount: decimal.RequireFromString(tax),
			TaxRate: &taxRate, ExchangeRate: decimal.NewFromInt(1), BaseCurrencyAmount: decimal.RequireFromString(base), Version: 1,
		},
	}
}
