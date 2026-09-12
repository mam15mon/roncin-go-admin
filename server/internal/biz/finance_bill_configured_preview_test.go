package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type configuredFinanceBillRepoStub struct {
	FinanceBillRepo
	fees []*FinanceBillableFee
}

func (s *configuredFinanceBillRepoStub) LoadBillableFees(context.Context, uuid.UUID, []uuid.UUID) ([]*FinanceBillableFee, error) {
	return s.fees, nil
}
func (*configuredFinanceBillRepoStub) ValidateBillCurrencies(context.Context, []string) error {
	return nil
}
func (*configuredFinanceBillRepoStub) HydrateBillSettlementAccounts(_ context.Context, bills []*FinanceBill) error {
	for _, bill := range bills {
		bill.SettlementAccountName, bill.SettlementAccountHolder = "结算账户", "验收客户"
		bill.SettlementBankName, bill.SettlementBankAccount, bill.SettlementAccountCurrency = "验收银行", "001", bill.Currency
	}
	return nil
}

func (*configuredFinanceBillRepoStub) GetPartnerCreditSummaries(context.Context, uuid.UUID, []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error) {
	return map[uuid.UUID]*PartnerCreditSummary{}, nil
}

func TestConfiguredFinanceBillPreviewSplitsByFeeCurrencyAndUsesBillBaseRate(t *testing.T) {
	organizationID, partyID, accountID := uuid.New(), uuid.New(), uuid.New()
	cnyFee := financeBillableFeeForTest(partyID, "100", "94", "6", "100")
	usdFee := financeBillableFeeForTest(partyID, "10", "8", "2", "72")
	usdFee.Fee.Currency, usdFee.Fee.ExchangeRate, usdFee.Fee.BaseCurrencyAmount = "USD", decimal.RequireFromString("7.2"), decimal.RequireFromString("72")
	fees := []*FinanceBillableFee{usdFee, cnyFee}
	rates := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY", PivotCurrency: "CNY"}, rateByCurrency: map[string]decimal.Decimal{
		"USD": decimal.RequireFromString("7.2"),
	}})
	uc := NewFinanceBillUsecase(&configuredFinanceBillRepoStub{fees: fees}, rates, nil)
	input := PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{cnyFee.Fee.ID, usdFee.Fee.ID}, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}}
	initial, err := uc.PreviewBatch(t.Context(), organizationID, input)
	if err != nil || len(initial.Groups) != 2 || initial.PreviewToken != "" {
		t.Fatalf("初次预览应按费用币种分叶且未签发令牌: %#v, %v", initial, err)
	}
	for _, group := range initial.Groups {
		input.GroupConfigs = append(input.GroupConfigs, FinanceBillBatchPreviewGroupConfig{GroupKey: group.GroupKey, BillDate: "2026-09-10", SettlementAccountID: accountID})
	}
	preview, err := uc.PreviewBatch(t.Context(), organizationID, input)
	if err != nil || preview.PreviewToken == "" {
		t.Fatalf("完整预览失败: %#v, %v", preview, err)
	}
	for _, group := range preview.Groups {
		if !group.ConfigurationComplete || group.preparedBill == nil || group.Currency != group.Fees[0].Fee.Currency || group.preparedBill.ExchangeRateDate != "2026-09-10" {
			t.Fatalf("费用币种或账单日汇率未固化: %#v", group)
		}
		if err := validateFinanceBillAmountInvariants(group.preparedBill); err != nil {
			t.Fatal(err)
		}
	}
	fees[0].Fee.Version++
	changed, err := uc.PreviewBatch(t.Context(), organizationID, input)
	if err != nil || changed.PreviewToken == preview.PreviewToken {
		t.Fatalf("费用版本变化后预览令牌必须轮转: %#v, %v", changed, err)
	}
}

func TestConfiguredFinanceBillPreviewRejectsMixedDirection(t *testing.T) {
	organizationID, partyID := uuid.New(), uuid.New()
	receivable, payable := financeBillableFeeForTest(partyID, "100", "100", "0", "100"), financeBillableFeeForTest(partyID, "20", "20", "0", "20")
	payable.Fee.Direction = OrderFeePayable
	if _, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{receivable, payable}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}}); err != ErrFinanceBillMixedDirection {
		t.Fatalf("混方向错误=%v", err)
	}
}

func TestConfiguredFinanceBillInitialPreviewDoesNotRequireTemporaryDateRate(t *testing.T) {
	organizationID, partyID, accountID := uuid.New(), uuid.New(), uuid.New()
	fee := financeBillableFeeForTest(partyID, "100", "94", "6", "100")
	fee.Fee.Currency = "USD"
	fee.Fee.ExchangeRate = decimal.RequireFromString("7.2")
	fee.Fee.BaseCurrencyAmount = decimal.NewFromInt(720)
	rateRepo := &exchangeRateRepoStub{
		rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		resolveErr:  ErrExchangeRateMissing,
	}
	uc := NewFinanceBillUsecase(&configuredFinanceBillRepoStub{fees: []*FinanceBillableFee{fee}}, NewExchangeRateUsecase(rateRepo), nil)
	input := PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{fee.Fee.ID}, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}}
	initial, err := uc.PreviewBatch(t.Context(), organizationID, input)
	if err != nil || len(initial.Groups) != 1 || initial.PreviewToken != "" || !initial.Groups[0].TemporaryBillDate || len(rateRepo.resolveDates) != 0 {
		t.Fatalf("初次预览不应依赖临时日期汇率: preview=%#v resolveDates=%v err=%v", initial, rateRepo.resolveDates, err)
	}
	input.GroupConfigs = []FinanceBillBatchPreviewGroupConfig{{GroupKey: initial.Groups[0].GroupKey, BillDate: "2026-09-10", SettlementAccountID: accountID}}
	if _, err = uc.PreviewBatch(t.Context(), organizationID, input); err != ErrExchangeRateMissing {
		t.Fatalf("选定账单日期后缺汇率必须拒绝，错误=%v", err)
	}
}

func TestConfiguredFinanceBillPreviewRejectsUnsupportedGroupingMode(t *testing.T) {
	organizationID, partyID := uuid.New(), uuid.New()
	fee := financeBillableFeeForTest(partyID, "100", "100", "0", "100")
	if _, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{fee}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{}}); err != ErrFinanceBillGroupingModeUnsupported {
		t.Fatalf("未指定模式必须 fail-closed，错误=%v", err)
	}
}

func TestConfiguredFinanceBillNettingPreviewSplitsDirectionsIntoLeaves(t *testing.T) {
	organizationID, partyID := uuid.New(), uuid.New()
	receivable := financeBillableFeeForTest(partyID, "100", "100", "0", "100")
	payable := financeBillableFeeForTest(partyID, "20", "20", "0", "20")
	payable.Fee.Direction = OrderFeePayable
	preview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{receivable, payable}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NETTING"}})
	if err != nil {
		t.Fatalf("对冲模式应允许混合方向并按方向分叶: %v", err)
	}
	if len(preview.Groups) != 2 {
		t.Fatalf("应收与应付费用必须分别形成叶子: %#v", preview.Groups)
	}
	directions := map[OrderFeeDirection]bool{}
	for _, group := range preview.Groups {
		directions[group.Direction] = true
	}
	if !directions[OrderFeeReceivable] || !directions[OrderFeePayable] {
		t.Fatalf("对冲叶子应同时包含应收与应付: %#v", preview.Groups)
	}
}

func TestConfiguredFinanceBillNettingPreviewRejectsSingleDirection(t *testing.T) {
	organizationID, partyID := uuid.New(), uuid.New()
	receivable := financeBillableFeeForTest(partyID, "100", "100", "0", "100")
	payableOtherParty := financeBillableFeeForTest(uuid.New(), "20", "20", "0", "20")
	payableOtherParty.Fee.Direction = OrderFeePayable
	if _, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{receivable, payableOtherParty}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NETTING"}}); err != ErrFinanceNettingSingleDirection {
		t.Fatalf("对冲模式下同一结算单位缺少反方向费用必须拒绝，错误=%v", err)
	}
}

func TestConfiguredFinanceBillPreviewRejectsEmptyFeesAsInvalidArgument(t *testing.T) {
	if _, err := BuildConfiguredFinanceBillBatchPreview(
		uuid.New(),
		nil,
		PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}},
	); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("空费用错误=%v，期望参数错误", err)
	}
}

func TestConfiguredFinanceBillPreviewRejectsIncompleteOrSameCurrencyEstimatedRate(t *testing.T) {
	organizationID, partyID, accountID := uuid.New(), uuid.New(), uuid.New()
	fee := financeBillableFeeForTest(partyID, "100", "94", "6", "100")
	rate := decimal.RequireFromString("1.2")
	if _, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{fee}, PreviewFinanceBillBatchInput{
		GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"},
		GroupConfigs:   []FinanceBillBatchPreviewGroupConfig{{GroupKey: "group", EstimatedInvoiceRate: &rate}},
	}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("预计开票汇率缺币种时错误=%v", err)
	}

	rates := NewExchangeRateUsecase(&exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"}})
	uc := NewFinanceBillUsecase(&configuredFinanceBillRepoStub{fees: []*FinanceBillableFee{fee}}, rates, nil)
	initial, err := uc.PreviewBatch(t.Context(), organizationID, PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{fee.Fee.ID}, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil || len(initial.Groups) != 1 {
		t.Fatalf("初次预览失败: %#v, %v", initial, err)
	}
	currency := "CNY"
	_, err = uc.PreviewBatch(t.Context(), organizationID, PreviewFinanceBillBatchInput{
		FeeIDs: []uuid.UUID{fee.Fee.ID}, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"},
		GroupConfigs: []FinanceBillBatchPreviewGroupConfig{{GroupKey: initial.Groups[0].GroupKey, BillDate: "2026-09-10", SettlementAccountID: accountID, EstimatedInvoiceCurrency: &currency, EstimatedInvoiceRate: &rate}},
	})
	if err != ErrFinanceBillInvalidArgument {
		t.Fatalf("同币种预计开票非 1 汇率错误=%v", err)
	}
}

func TestNormalizeFinanceBillBatchGroupsRejectsEstimatedRateWithoutCurrency(t *testing.T) {
	rate := decimal.RequireFromString("1.2")
	if _, err := normalizeFinanceBillBatchGroups([]CreateFinanceBillBatchGroupInput{{
		GroupKey:             strings.Repeat("a", 64),
		StatementTitle:       "测试结算单位",
		BillDate:             "2026-09-10",
		SettlementAccountID:  uuid.New(),
		EstimatedInvoiceRate: &rate,
	}}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("批量创建预计开票汇率缺币种错误=%v", err)
	}
}
