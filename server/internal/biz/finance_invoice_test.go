package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type financeInvoiceCandidateRepoStub struct {
	FinanceInvoiceRepo
	organizationID  uuid.UUID
	filter          FinanceInvoiceCreationBillFilter
	billID          uuid.UUID
	organizationIDs []uuid.UUID
	err             error
}

func (s *financeInvoiceCandidateRepoStub) ListCreationBills(_ context.Context, organizationID uuid.UUID, filter FinanceInvoiceCreationBillFilter) (*FinanceInvoiceCreationBillListResult, error) {
	s.organizationID = organizationID
	s.filter = filter
	return &FinanceInvoiceCreationBillListResult{}, s.err
}

func (s *financeInvoiceCandidateRepoStub) ListProfilesForBill(_ context.Context, organizationIDs []uuid.UUID, billID uuid.UUID) (*FinanceInvoiceProfilesForBill, error) {
	s.organizationIDs = append([]uuid.UUID(nil), organizationIDs...)
	s.billID = billID
	return &FinanceInvoiceProfilesForBill{}, s.err
}

func TestBuildFinanceInvoiceAggregatesConfirmedBills(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	taxRate := decimal.RequireFromString("6")
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI001", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", BaseCurrency: "CNY", TotalAmount: decimal.RequireFromString("100"), NetAmount: decimal.RequireFromString("94"), TaxAmount: decimal.RequireFromString("6"), Lines: []*FinanceBillLine{{FeeCode: "OCEAN", FeeName: "海运费", Currency: "CNY", TotalAmount: decimal.RequireFromString("100"), NetAmount: decimal.RequireFromString("94"), TaxAmount: decimal.RequireFromString("6"), TaxRate: &taxRate, Active: true}}},
		{ID: uuid.Must(uuid.NewV7()), BillNo: "BI002", Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: partyID, SettlementPartyName: "客户", Currency: "CNY", BaseCurrency: "CNY", TotalAmount: decimal.RequireFromString("0.02"), NetAmount: decimal.RequireFromString("0.0188"), TaxAmount: decimal.RequireFromString("0.0012"), Lines: []*FinanceBillLine{{FeeCode: "OCEAN", FeeName: "海运费", Currency: "CNY", TotalAmount: decimal.RequireFromString("0.02"), NetAmount: decimal.RequireFromString("0.0188"), TaxAmount: decimal.RequireFromString("0.0012"), TaxRate: &taxRate, Active: true}}},
	}
	input := CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceProfileID: uuid.Must(uuid.NewV7()), InvoiceType: FinanceInvoiceSpecial, IdempotencyKey: "invoice-test"}
	profile := &PartnerInvoiceProfile{ID: input.InvoiceProfileID, OrganizationID: organizationID, PartnerID: partyID, InvoiceTitle: "客户", TaxpayerIdentificationNo: "91310000TEST", RegisteredAddress: "上海市", RegisteredPhone: "021-12345678", BankName: "测试银行", BankAccount: "62220000", Enabled: true}
	invoice, err := buildFinanceInvoice(organizationID, bills, profile, input)
	if err != nil {
		t.Fatalf("构建开票记录失败: %v", err)
	}
	if invoice.TotalAmount.StringFixed(8) != "100.02000000" || invoice.NetAmount.StringFixed(8) != "94.01880000" || invoice.TaxAmount.StringFixed(8) != "6.00120000" || invoice.BillCount != 2 || len(invoice.Lines) != 1 || invoice.Lines[0].SourceLineCount != 2 {
		t.Fatalf("开票汇总不正确: total=%s tax=%s count=%d", invoice.TotalAmount, invoice.TaxAmount, invoice.BillCount)
	}
	if invoice.BaseCurrency != "CNY" {
		t.Fatalf("开票记录应继承账单本币，实际为 %q", invoice.BaseCurrency)
	}
}

func TestBuildFinanceInvoiceRejectsMixedParties(t *testing.T) {
	bills := []*FinanceBill{
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY", BaseCurrency: "CNY"},
		{ID: uuid.Must(uuid.NewV7()), Status: FinanceBillConfirmed, Direction: OrderFeeReceivable, SettlementPartyID: uuid.Must(uuid.NewV7()), Currency: "CNY", BaseCurrency: "CNY"},
	}
	organizationID := uuid.Must(uuid.NewV7())
	profile := &PartnerInvoiceProfile{ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, PartnerID: bills[0].SettlementPartyID, InvoiceTitle: "客户", TaxpayerIdentificationNo: "91310000TEST", Enabled: true}
	_, err := buildFinanceInvoice(organizationID, bills, profile, CreateFinanceInvoiceInput{BillIDs: []uuid.UUID{bills[0].ID, bills[1].ID}, InvoiceProfileID: profile.ID, InvoiceType: FinanceInvoiceNormal})
	if err != ErrFinanceInvoiceBillMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际 %v", err)
	}
}

func TestFinanceInvoiceCreationCandidatesValidateAndForwardScopedInputs(t *testing.T) {
	repo := &financeInvoiceCandidateRepoStub{}
	usecase := NewFinanceInvoiceUsecase(repo, nil, nil)
	organizationID := uuid.New()
	partyID := uuid.New()
	if _, err := usecase.ListCreationBills(context.Background(), organizationID, FinanceInvoiceCreationBillFilter{
		Page: 1, PageSize: 20, Keyword: "  客户  ", Direction: OrderFeeReceivable, SettlementPartyID: &partyID, Currency: "usd",
	}); err != nil {
		t.Fatalf("读取创建账单候选失败: %v", err)
	}
	if repo.organizationID != organizationID || repo.filter.Keyword != "客户" || repo.filter.Currency != "USD" || repo.filter.SettlementPartyID == nil || *repo.filter.SettlementPartyID != partyID {
		t.Fatalf("候选参数未规范化转发: org=%s filter=%+v", repo.organizationID, repo.filter)
	}
	if _, err := usecase.ListCreationBills(context.Background(), uuid.Nil, FinanceInvoiceCreationBillFilter{Page: 1, PageSize: 20}); err != ErrFinanceInvoiceInvalidArgument {
		t.Fatalf("空组织错误 = %v，期望 %v", err, ErrFinanceInvoiceInvalidArgument)
	}
	billID := uuid.New()
	if _, err := usecase.ListProfilesForBill(context.Background(), []uuid.UUID{organizationID}, billID); err != nil {
		t.Fatalf("读取源账单开票资料失败: %v", err)
	}
	if repo.billID != billID || len(repo.organizationIDs) != 1 || repo.organizationIDs[0] != organizationID {
		t.Fatalf("资料查询未保留源账单和范围: bill=%s scope=%v", repo.billID, repo.organizationIDs)
	}
	databaseErr := errors.New("资料查询数据库故障")
	repo.err = databaseErr
	if _, err := usecase.ListProfilesForBill(context.Background(), []uuid.UUID{organizationID}, billID); !errors.Is(err, databaseErr) {
		t.Fatalf("资料查询错误 = %v，期望原样返回 %v", err, databaseErr)
	}
}
