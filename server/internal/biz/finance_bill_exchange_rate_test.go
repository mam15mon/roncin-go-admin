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
	protected    bool
}

func (s *financeBillTargetOrganizationRepoStub) HasProtectedFinanceFact(context.Context, uuid.UUID) (bool, error) {
	s.callOrder = append(s.callOrder, "protected")
	return s.protected, nil
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

func (s *financeBillTargetExchangeRateRepoStub) ResolveRate(_ context.Context, organizationID uuid.UUID, _ OrderFeeDirection, _, _, rateDate string) (ResolvedRate, error) {
	s.resolvedRateOrganizationID = organizationID
	s.resolveDates = append(s.resolveDates, rateDate)
	return ResolvedRate{Rate: s.resolvedRate, Source: ExchangeRateSourceSystem}, nil
}

func TestApplyBillExchangeRateUsesBillDateSnapshot(t *testing.T) {
	organizationID := uuid.New()
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.20")},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)
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
	if bill.ExchangeRate.StringFixed(8) != "7.20000000" || bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" || bill.ExchangeRateSource != ExchangeRateSourceWeekly || bill.ExchangeRateDate != "2026-08-26" || bill.ExchangeRateSettingID != nil {
		t.Fatalf("账单汇率快照不完整: %#v", bill)
	}
	if len(exchangeRepo.resolveDates) != 1 || exchangeRepo.resolveDates[0] != "2026-08-26" {
		t.Fatalf("账单汇率必须按账单日解析: %v", exchangeRepo.resolveDates)
	}
}

func TestApplyBillExchangeRateUsesRoundedRateForBaseAmount(t *testing.T) {
	organizationID := uuid.New()
	// 超过 4 位小数（业务口径）的汇率必须先固化到 4 位，头本位币金额按已固化汇率计算。
	exchangeRepo := &exchangeRateRepoStub{
		rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.1234567891")},
	}
	usecase := NewFinanceBillUsecase(nil, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)
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
	if bill.ExchangeRate.StringFixed(4) != "7.1235" {
		t.Fatalf("账单汇率未按 4 位固化: %s", bill.ExchangeRate.StringFixed(4))
	}
	// 100 × 7.1235 = 712.35；不得使用未舍入汇率得出 712.34567891。
	if bill.BaseCurrencyAmount.StringFixed(8) != "712.35000000" {
		t.Fatalf("头本位币金额必须等于账单金额 × 已固化汇率: %s", bill.BaseCurrencyAmount.StringFixed(8))
	}
	if bill.Lines[0].BaseCurrencyAmount.StringFixed(8) != "712.35000000" {
		t.Fatalf("明细本位币金额未与头金额保持一致: %s", bill.Lines[0].BaseCurrencyAmount.StringFixed(8))
	}
}

func TestFinanceBillUpdateKeepsExchangeRateAndTargetOrganizationAudit(t *testing.T) {
	currentOrganizationID := uuid.New()
	targetOrganizationID := uuid.New()
	actorID := uuid.New()
	billID := uuid.New()
	feeID, orderID, partyID, accountID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	taxRate := decimal.Zero
	billRepo := &financeBillTargetOrganizationRepoStub{bill: &FinanceBill{
		ID: billID, OrganizationID: targetOrganizationID, Direction: OrderFeeReceivable,
		Status: FinanceBillConfirmed, SettlementPartyID: partyID, SettlementPartyName: "测试客户",
		Currency: "USD", BaseCurrency: "CNY", ExchangeRate: decimal.RequireFromString("7.2"), ExchangeRateSource: "WEEKLY", ExchangeRateDate: "2026-09-09", BillDate: "2026-09-09", TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), Version: 2,
		Lines: []*FinanceBillLine{{ID: uuid.New(), OrderFeeID: feeID, OrderID: orderID, OrderNo: "SE001", FeeCode: "FEE", FeeName: "运费", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TaxRate: &taxRate, Currency: "USD", TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), TaxAmount: decimal.Zero, BaseCurrency: "CNY", Active: true}},
	}}
	billRepo.fees = []*FinanceBillableFee{{OrderNo: "SE001", Fee: &OrderFee{ID: feeID, OrderID: orderID, Direction: OrderFeeReceivable, HasActiveBill: true, SettlementPartyID: partyID, SettlementPartyName: "测试客户", FeeCode: "FEE", FeeName: "运费", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), TaxAmount: decimal.Zero, TaxRate: &taxRate, Currency: "USD", ExchangeRate: decimal.RequireFromString("7.10"), BaseCurrency: "CNY", BaseCurrencyAmount: decimal.NewFromInt(710)}}}
	exchangeRepo := &financeBillTargetExchangeRateRepoStub{resolvedRate: decimal.RequireFromString("7.20")}
	usecase := NewFinanceBillUsecase(billRepo, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)

	_, err := usecase.Update(t.Context(), []uuid.UUID{currentOrganizationID, targetOrganizationID}, actorID, UpdateFinanceBillInput{
		ID: billID, BillDate: "2026-09-09", ExpectedVersion: 2, SettlementAccountID: accountID,
	})
	if err != nil {
		t.Fatalf("跨组织账单更新失败: %v", err)
	}
	if exchangeRepo.resolvedContextOrganizationID != uuid.Nil || exchangeRepo.resolvedRateOrganizationID != uuid.Nil || len(exchangeRepo.resolveDates) != 0 {
		t.Fatal("编辑账单不得查询汇率")
	}
	if billRepo.updateAudit == nil || billRepo.updateAudit.OrganizationID == nil || *billRepo.updateAudit.OrganizationID != targetOrganizationID {
		t.Fatalf("更新审计必须记录目标账单组织，audit=%#v target=%s", billRepo.updateAudit, targetOrganizationID)
	}
	if billRepo.updateInput.ExchangeRate.StringFixed(8) != "7.20000000" || billRepo.updateInput.BaseCurrencyAmount.StringFixed(8) != "720.00000000" {
		t.Fatalf("目标组织汇率快照未传入仓储: %#v", billRepo.updateInput)
	}
	if len(billRepo.callOrder) < 5 || billRepo.callOrder[0] != "get" || billRepo.callOrder[1] != "get" || billRepo.callOrder[2] != "protected" || billRepo.callOrder[3] != "fees" || billRepo.callOrder[4] != "account" {
		t.Fatalf("账单更新锁定顺序 = %v，期望预读后按账单、受保护复核、费用、账户处理", billRepo.callOrder)
	}
	if billRepo.updateInput.Protected {
		t.Fatal("未受保护账单不得走受限更新")
	}
	for _, tc := range []struct {
		name         string
		currency     *string
		rate         *decimal.Decimal
		wantCurrency string
		wantRate     string
		wantAmount   string
		wantErr      bool
	}{
		{name: "省略配置默认同币种", wantCurrency: "USD", wantRate: "1", wantAmount: "100"},
		{name: "跨币种预计开票", currency: stringPointer("CNY"), rate: decimalPointer(decimal.RequireFromString("7.5")), wantCurrency: "CNY", wantRate: "7.5", wantAmount: "750"},
		{name: "无币种不能指定汇率", rate: decimalPointer(decimal.NewFromInt(2)), wantErr: true},
		{name: "跨币种缺汇率", currency: stringPointer("CNY"), wantErr: true},
		{name: "同币种汇率必须为一", currency: stringPointer("USD"), rate: decimalPointer(decimal.NewFromInt(2)), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			billRepo.updateAudit = nil
			_, err := usecase.Update(t.Context(), []uuid.UUID{targetOrganizationID}, actorID, UpdateFinanceBillInput{ID: billID, ExpectedVersion: 2, SettlementAccountID: accountID, EstimatedInvoiceCurrency: tc.currency, EstimatedInvoiceRate: tc.rate})
			if tc.wantErr {
				if err != ErrFinanceBillInvalidArgument || billRepo.updateAudit != nil {
					t.Fatalf("非法预计开票必须零写入: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			input := billRepo.updateInput
			if input.EstimatedInvoiceCurrency == nil || *input.EstimatedInvoiceCurrency != tc.wantCurrency || input.EstimatedInvoiceRate == nil || !input.EstimatedInvoiceRate.Equal(decimal.RequireFromString(tc.wantRate)) || input.EstimatedInvoiceAmount == nil || !input.EstimatedInvoiceAmount.Equal(decimal.RequireFromString(tc.wantAmount)) || !input.ExchangeRate.Equal(decimal.RequireFromString("7.2")) || !input.BaseCurrencyAmount.Equal(decimal.NewFromInt(720)) || len(exchangeRepo.resolveDates) != 0 {
				t.Fatalf("预计开票须独立于账单固定汇率: %#v", input)
			}
		})
	}

}

// 受保护账单（已有发票、核销或已确认对冲）只能改备注、对账标题与到期日，且不重取汇率。
func TestFinanceBillUpdateProtectedOnlyAllowsNonFinancialFields(t *testing.T) {
	organizationID, actorID, billID, accountID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	terms := 30
	currency, rate, amount := "USD", decimal.NewFromInt(1), decimal.NewFromInt(100)
	newRepo := func() *financeBillTargetOrganizationRepoStub {
		return &financeBillTargetOrganizationRepoStub{protected: true, bill: &FinanceBill{
			ID: billID, OrganizationID: organizationID, Direction: OrderFeeReceivable, Status: FinanceBillConfirmed,
			Currency: "USD", BaseCurrency: "CNY", TotalAmount: amount, NetAmount: amount, BaseCurrencyAmount: decimal.NewFromInt(710),
			ExchangeRate: decimal.RequireFromString("7.1"), ExchangeRateSource: "WEEKLY", ExchangeRateDate: "2026-09-01",
			BillDate: "2026-09-01", PaymentTermsDays: &terms, SettlementAccountID: accountID,
			EstimatedInvoiceCurrency: &currency, EstimatedInvoiceRate: &rate, EstimatedInvoiceAmount: &amount, Version: 3,
		}}
	}
	exchangeRepo := &financeBillTargetExchangeRateRepoStub{resolvedRate: decimal.RequireFromString("7.20")}
	note := "仅改备注"
	dueDate := "2026-10-01"
	cases := []struct {
		name  string
		input UpdateFinanceBillInput
	}{
		{"改结算账户", UpdateFinanceBillInput{BillDate: "2026-09-01", PaymentTermsDays: &terms, SettlementAccountID: uuid.New()}},
		{"改账期", UpdateFinanceBillInput{BillDate: "2026-09-01", SettlementAccountID: accountID}},
		{"改预计开票币种", UpdateFinanceBillInput{BillDate: "2026-09-01", PaymentTermsDays: &terms, SettlementAccountID: accountID, EstimatedInvoiceCurrency: stringPointer("CNY"), EstimatedInvoiceRate: decimalPointer(decimal.RequireFromString("7.1"))}},
	}
	for _, item := range cases {
		repo := newRepo()
		usecase := NewFinanceBillUsecase(repo, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)
		item.input.ID, item.input.ExpectedVersion = billID, 3
		if _, err := usecase.Update(t.Context(), []uuid.UUID{organizationID}, actorID, item.input); err != ErrFinanceBillProtectedFields {
			t.Fatalf("%s: 受保护账单应拒绝，实际 %v", item.name, err)
		}
		if repo.updateAudit != nil {
			t.Fatalf("%s: 拒绝时不得写入", item.name)
		}
	}
	repo := newRepo()
	usecase := NewFinanceBillUsecase(repo, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)
	exchangeRepo.resolveDates = nil
	if _, err := usecase.Update(t.Context(), []uuid.UUID{organizationID}, actorID, UpdateFinanceBillInput{ID: billID, ExpectedVersion: 3, BillDate: "2099-01-01", PaymentTermsDays: &terms, SettlementAccountID: accountID, Note: &note, EstimatedInvoiceCurrency: &currency}); err != nil {
		t.Fatalf("受保护账单改备注应允许: %v", err)
	}
	if !repo.updateInput.Protected || len(exchangeRepo.resolveDates) != 0 || !repo.updateInput.ExchangeRate.Equal(decimal.RequireFromString("7.1")) || !repo.updateInput.BaseCurrencyAmount.Equal(decimal.NewFromInt(710)) {
		t.Fatalf("受保护更新不得重取汇率或改金额: %#v resolve=%v", repo.updateInput, exchangeRepo.resolveDates)
	}
	if repo.updateInput.Note == nil || *repo.updateInput.Note != note {
		t.Fatalf("备注未传入仓储: %#v", repo.updateInput.Note)
	}
	// 未设置账期时可单独改到期日。
	repo = newRepo()
	repo.bill.PaymentTermsDays = nil
	usecase = NewFinanceBillUsecase(repo, NewExchangeRateUsecase(exchangeRepo, nil), &financeBillTransactorStub{}, nil)
	if _, err := usecase.Update(t.Context(), []uuid.UUID{organizationID}, actorID, UpdateFinanceBillInput{ID: billID, ExpectedVersion: 3, BillDate: "2026-09-01", DueDate: &dueDate, SettlementAccountID: accountID}); err != nil {
		t.Fatalf("受保护账单改到期日应允许: %v", err)
	}
	if repo.updateInput.DueDate == nil || *repo.updateInput.DueDate != dueDate || !repo.updateInput.Protected {
		t.Fatalf("到期日未按受限更新传入: %#v", repo.updateInput)
	}
}

func decimalPointer(value decimal.Decimal) *decimal.Decimal { return &value }
