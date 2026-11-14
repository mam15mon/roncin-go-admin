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

func TestNormalizeInvoiceRegistrationsManualTaxAndHierarchy(t *testing.T) {
	valid := func() RegisterFinanceInvoicesInput {
		return RegisterFinanceInvoicesInput{IdempotencyKey: "manual-tax", Records: []FinanceInvoiceRegistrationInput{{InvoiceProfileID: uuid.New(), InvoiceType: FinanceInvoiceNormal, TaxInvoiceNo: " 原票 ", InvoiceDate: "2026-11-14", TotalAmount: decimal.RequireFromString("100.00000001"), Allocations: []FinanceInvoiceAllocationInput{{BillID: uuid.New(), ExpectedVersion: 1, Amount: decimal.RequireFromString("100.00000001"), Fees: []FinanceInvoiceFeeInput{{OrderFeeID: uuid.New(), Amount: decimal.RequireFromString("100.00000001"), TaxAmount: decimal.RequireFromString("7.00000001")}}}}}}}
	}
	input := valid()
	if err := normalizeInvoiceRegistrations(&input); err != nil {
		t.Fatal(err)
	}
	if input.Records[0].TaxInvoiceNo != "原票" || !input.Records[0].Allocations[0].Fees[0].TaxAmount.Equal(decimal.RequireFromString("7.00000001")) {
		t.Fatal("规范化不得改写手填税额")
	}
	for _, tc := range []struct {
		name string
		edit func(*FinanceInvoiceRegistrationInput)
	}{
		{"发票合计不等", func(r *FinanceInvoiceRegistrationInput) { r.TotalAmount = decimal.NewFromInt(99) }},
		{"费用合计不等", func(r *FinanceInvoiceRegistrationInput) { r.Allocations[0].Fees[0].Amount = decimal.NewFromInt(99) }},
		{"税额超额", func(r *FinanceInvoiceRegistrationInput) { r.Allocations[0].Fees[0].TaxAmount = decimal.NewFromInt(101) }},
		{"税额负数", func(r *FinanceInvoiceRegistrationInput) { r.Allocations[0].Fees[0].TaxAmount = decimal.NewFromInt(-1) }},
		{"九位精度", func(r *FinanceInvoiceRegistrationInput) {
			r.Allocations[0].Fees[0].TaxAmount = decimal.RequireFromString("0.000000001")
		}},
		{"重复来源", func(r *FinanceInvoiceRegistrationInput) {
			r.Allocations[0].Fees = append(r.Allocations[0].Fees, r.Allocations[0].Fees[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid()
			tc.edit(&input.Records[0])
			if err := normalizeInvoiceRegistrations(&input); !errors.Is(err, ErrFinanceInvoiceInvalidArgument) {
				t.Fatalf("必须拒绝: %v", err)
			}
		})
	}
}

func TestFinanceInvoiceCreationCandidatesValidateAndForwardScopedInputs(t *testing.T) {
	repo := &financeInvoiceCandidateRepoStub{}
	usecase := NewFinanceInvoiceUsecase(repo, nil, nil, nil)
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
	if _, err := usecase.ListCreationBills(context.Background(), uuid.Nil, FinanceInvoiceCreationBillFilter{Page: 1, PageSize: 20}); !errors.Is(err, ErrFinanceInvoiceInvalidArgument) {
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
