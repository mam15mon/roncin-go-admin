package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeBillTargetOrganizationRepoStub struct {
	FinanceBillRepo
	bill         *FinanceBill
	updateScopes []uuid.UUID
	updateInput  UpdateFinanceBillInput
	updateAudit  *AuditEvent
	fees         []*FinanceBillableFee
	callOrder    []string
}

func (s *financeBillTargetOrganizationRepoStub) Get(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (*FinanceBill, error) {
	s.callOrder = append(s.callOrder, "get")
	return s.bill, nil
}

func (s *financeBillTargetOrganizationRepoStub) Update(_ context.Context, organizationIDs []uuid.UUID, input UpdateFinanceBillInput, audit *AuditEvent) (*FinanceBill, error) {
	s.updateScopes = organizationIDs
	s.updateInput = input
	s.updateAudit = audit
	return s.bill, nil
}

func (s *financeBillTargetOrganizationRepoStub) LoadBillableFees(context.Context, uuid.UUID, []uuid.UUID) ([]*FinanceBillableFee, error) {
	s.callOrder = append(s.callOrder, "fees")
	return s.fees, nil
}

func (*financeBillTargetOrganizationRepoStub) ValidateBillCurrencies(context.Context, []string) error {
	return nil
}

func (s *financeBillTargetOrganizationRepoStub) HydrateBillSettlementAccounts(_ context.Context, bills []*FinanceBill) error {
	s.callOrder = append(s.callOrder, "account")
	for _, bill := range bills {
		bill.SettlementAccountName = "测试账户"
		bill.SettlementAccountHolder = "测试客户"
		bill.SettlementBankName = "测试银行"
		bill.SettlementBankAccount = "001"
		bill.SettlementAccountCurrency = bill.Currency
	}
	return nil
}

type financeBillTargetExchangeRateRepoStub struct {
	ExchangeRateRepo
	resolvedContextOrganizationID uuid.UUID
	resolvedRateOrganizationID    uuid.UUID
	resolvedRate                  decimal.Decimal
	resolveDates                  []string
}

func (s *financeBillTargetExchangeRateRepoStub) ResolveContext(_ context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error) {
	s.resolvedContextOrganizationID = organizationID
	return &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"}, nil
}

func (s *financeBillTargetExchangeRateRepoStub) ResolveRate(_ context.Context, organizationID uuid.UUID, _ OrderFeeDirection, _, _, _, rateDate string) (ResolvedRate, error) {
	s.resolvedRateOrganizationID = organizationID
	s.resolveDates = append(s.resolveDates, rateDate)
	return ResolvedRate{Rate: s.resolvedRate, Source: ExchangeRateSourceSystem}, nil
}

func TestApplyBillExchangeRateUsesBillDateSnapshot(t *testing.T) {
	organizationID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY", PivotCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.20")},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil, nil)
	bill := &FinanceBill{
		Direction:    OrderFeeReceivable,
		Currency:     "USD",
		BaseCurrency: "CNY",
		TotalAmount:  decimal.RequireFromString("100"),
		BillDate:     "2026-08-26",
	}

	if err := usecase.applyBillExchangeRate(context.Background(), organizationID, bill); err != nil {
		t.Fatalf("应用账单日汇率失败: %v", err)
	}
	if bill.ExchangeRate.StringFixed(8) != "7.20000000" || bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" || bill.ExchangeRateSource != "SYSTEM" || bill.ExchangeRateDate != "2026-08-26" || bill.ExchangeRateSettingID != nil {
		t.Fatalf("账单汇率快照不完整: %#v", bill)
	}
	if len(exchangeRepo.resolveDates) != 1 || exchangeRepo.resolveDates[0] != "2026-08-26" {
		t.Fatalf("账单汇率必须按账单日解析: %v", exchangeRepo.resolveDates)
	}
}

func TestApplyBillExchangeRateUsesRoundedRateForBaseAmount(t *testing.T) {
	organizationID := uuid.New()
	// 超过 8 位小数的汇率必须先固化到 8 位，头本位币金额按已固化汇率计算。
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY", PivotCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.1234567891")},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil, nil)
	taxRate := decimal.Zero
	bill := &FinanceBill{
		Direction:    OrderFeeReceivable,
		Currency:     "USD",
		BaseCurrency: "CNY",
		TotalAmount:  decimal.NewFromInt(100),
		BillDate:     "2026-08-26",
		Lines: []*FinanceBillLine{{
			OrderFeeID: uuid.New(), Currency: "USD", BaseCurrency: "CNY",
			TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), TaxRate: &taxRate,
		}},
	}

	if err := usecase.applyBillExchangeRate(context.Background(), organizationID, bill); err != nil {
		t.Fatalf("应用超长小数汇率失败: %v", err)
	}
	if bill.ExchangeRate.StringFixed(8) != "7.12345679" {
		t.Fatalf("账单汇率未按 8 位固化: %s", bill.ExchangeRate.StringFixed(8))
	}
	// 100 × 7.12345679 = 712.34567900；不得使用未舍入汇率得出 712.34567891。
	if bill.BaseCurrencyAmount.StringFixed(8) != "712.34567900" {
		t.Fatalf("头本位币金额必须等于账单金额 × 已固化汇率: %s", bill.BaseCurrencyAmount.StringFixed(8))
	}
	if bill.Lines[0].BaseCurrencyAmount.StringFixed(8) != "712.34567900" {
		t.Fatalf("明细本位币金额未与头金额保持一致: %s", bill.Lines[0].BaseCurrencyAmount.StringFixed(8))
	}
}

func TestFinanceBillUpdateUsesTargetOrganizationForExchangeRateAndAudit(t *testing.T) {
	currentOrganizationID := uuid.New()
	targetOrganizationID := uuid.New()
	actorID := uuid.New()
	billID := uuid.New()
	feeID, orderID, partyID, accountID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	taxRate := decimal.Zero
	billRepo := &financeBillTargetOrganizationRepoStub{bill: &FinanceBill{
		ID: billID, OrganizationID: targetOrganizationID, Direction: OrderFeeReceivable,
		Status: FinanceBillDraft, SettlementPartyID: partyID, SettlementPartyName: "测试客户",
		Currency: "USD", BaseCurrency: "CNY", TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), Version: 2,
		Lines: []*FinanceBillLine{{ID: uuid.New(), OrderFeeID: feeID, OrderID: orderID, OrderNo: "SE001", FeeCode: "FEE", FeeName: "运费", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: &taxRate, Currency: "USD", TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), TaxAmount: decimal.Zero}},
	}}
	billRepo.fees = []*FinanceBillableFee{{OrderNo: "SE001", Fee: &OrderFee{ID: feeID, OrderID: orderID, Direction: OrderFeeReceivable, Status: OrderFeeBilled, SettlementPartyID: partyID, SettlementPartyName: "测试客户", FeeCode: "FEE", FeeName: "运费", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), TaxAmount: decimal.Zero, TaxRate: &taxRate, Currency: "USD", ExchangeRate: decimal.RequireFromString("7.10"), BaseCurrency: "CNY", BaseCurrencyAmount: decimal.NewFromInt(710)}}}
	exchangeRepo := &financeBillTargetExchangeRateRepoStub{resolvedRate: decimal.RequireFromString("7.20")}
	usecase := NewFinanceBillUsecase(billRepo, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil, nil)

	_, err := usecase.Update(t.Context(), []uuid.UUID{currentOrganizationID, targetOrganizationID}, actorID, UpdateFinanceBillInput{
		ID: billID, BillDate: "2026-09-09", ExpectedVersion: 2, SettlementAccountID: accountID,
	})
	if err != nil {
		t.Fatalf("跨组织账单更新失败: %v", err)
	}
	if exchangeRepo.resolvedContextOrganizationID != targetOrganizationID || exchangeRepo.resolvedRateOrganizationID != targetOrganizationID {
		t.Fatalf("汇率必须按目标账单组织解析，context=%s rate=%s target=%s", exchangeRepo.resolvedContextOrganizationID, exchangeRepo.resolvedRateOrganizationID, targetOrganizationID)
	}
	if len(exchangeRepo.resolveDates) != 1 || exchangeRepo.resolveDates[0] != "2026-09-09" {
		t.Fatalf("账单汇率必须按更新后的账单日解析: %v", exchangeRepo.resolveDates)
	}
	if billRepo.updateAudit == nil || billRepo.updateAudit.OrganizationID == nil || *billRepo.updateAudit.OrganizationID != targetOrganizationID {
		t.Fatalf("更新审计必须记录目标账单组织，audit=%#v target=%s", billRepo.updateAudit, targetOrganizationID)
	}
	if billRepo.updateInput.ExchangeRate.StringFixed(8) != "7.20000000" || billRepo.updateInput.BaseCurrencyAmount.StringFixed(8) != "720.00000000" {
		t.Fatalf("目标组织汇率快照未传入仓储: %#v", billRepo.updateInput)
	}
	if len(billRepo.callOrder) < 4 || billRepo.callOrder[0] != "get" || billRepo.callOrder[1] != "get" || billRepo.callOrder[2] != "fees" || billRepo.callOrder[3] != "account" {
		t.Fatalf("草稿更新锁定顺序 = %v，期望预读后按账单、费用、账户处理", billRepo.callOrder)
	}
}
