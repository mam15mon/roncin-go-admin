package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type invoiceRegistrationProfileStub struct {
	FinanceInvoiceRepo
	profile *PartnerInvoiceProfile
}

func (s *invoiceRegistrationProfileStub) LoadInvoiceProfile(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*PartnerInvoiceProfile, error) {
	return s.profile, nil
}

type invoiceDirectionRateRepoStub struct {
	exchangeRateRepoStub
	direction OrderFeeDirection
}

func (s *invoiceDirectionRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, direction OrderFeeDirection, _, _, _ string) (ResolvedRate, error) {
	s.direction = direction
	if s.resolveErr != nil {
		return ResolvedRate{}, s.resolveErr
	}
	rate := decimal.RequireFromString("7.2")
	if direction == OrderFeePayable {
		rate = decimal.NewFromInt(7)
	}
	return ResolvedRate{Rate: rate, Source: ExchangeRateSourceWeekly}, nil
}

func TestInvoiceRegistrationResolvesRateByBillDirection(t *testing.T) {
	for _, direction := range []OrderFeeDirection{OrderFeeReceivable, OrderFeePayable} {
		for _, missing := range []bool{false, true} {
			t.Run(string(direction)+map[bool]string{true: "缺汇率"}[missing], func(t *testing.T) {
				org, party, profileID, feeID, lineID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
				repo := &invoiceRegistrationProfileStub{profile: &PartnerInvoiceProfile{ID: profileID, InvoiceTitle: "测试抬头", TaxpayerIdentificationNo: "TAX", Enabled: true}}
				rates := &invoiceDirectionRateRepoStub{exchangeRateRepoStub: exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}}
				if missing {
					rates.resolveErr = ErrExchangeRateMissing
				}
				uc := NewFinanceInvoiceUsecase(repo, NewOrderConfigUsecase(orderConfigNumberStub{}), NewExchangeRateUsecase(rates, nil), nil)
				bill := &FinanceBill{ID: uuid.New(), OrganizationID: org, Status: FinanceBillConfirmed, Purpose: FinanceBillNormal, Direction: direction, SettlementPartyID: party, Currency: "USD", BaseCurrency: "CNY", Lines: []*FinanceBillLine{{ID: lineID, OrderFeeID: feeID, Active: true, TotalAmount: decimal.NewFromInt(100)}}}
				input := FinanceInvoiceRegistrationInput{InvoiceProfileID: profileID, InvoiceType: FinanceInvoiceDigitalNormal, TaxInvoiceNo: "INV-001", InvoiceDate: "2026-11-04", TotalAmount: decimal.NewFromInt(100), Allocations: []FinanceInvoiceAllocationInput{{BillID: bill.ID, ExpectedVersion: 1, Amount: decimal.NewFromInt(100), Fees: []FinanceInvoiceFeeInput{{OrderFeeID: feeID, Amount: decimal.NewFromInt(100), TaxAmount: decimal.NewFromInt(9)}}}}}
				invoices, err := uc.buildRegistrationInvoices(context.Background(), org, []FinanceInvoiceRegistrationInput{input}, []*FinanceBill{bill}, nil)
				if missing {
					if !errors.Is(err, ErrExchangeRateMissing) {
						t.Fatalf("缺少方向汇率应拒绝: %v", err)
					}
					return
				}
				amount := decimal.NewFromInt(720)
				if direction == OrderFeePayable {
					amount = decimal.NewFromInt(700)
				}
				if err != nil || rates.direction != direction || len(invoices) != 1 || !invoices[0].BaseCurrencyAmount.Equal(amount) || !invoices[0].TaxAmount.Equal(decimal.NewFromInt(9)) {
					t.Fatalf("登记未保留方向汇率/手填税额: %v %+v", err, invoices)
				}
			})
		}
	}
}

func TestInvoiceRevisionUsesEachOriginalProfileSnapshot(t *testing.T) {
	org, party, profileID := uuid.New(), uuid.New(), uuid.New()
	rate := decimal.Zero
	makeBill := func() *FinanceBill {
		return &FinanceBill{ID: uuid.New(), OrganizationID: org, Status: FinanceBillConfirmed, Purpose: FinanceBillNormal, Direction: OrderFeeReceivable, SettlementPartyID: party, Currency: "CNY", BaseCurrency: "CNY", Lines: []*FinanceBillLine{{ID: uuid.New(), OrderFeeID: uuid.New(), Active: true, TotalAmount: decimal.NewFromInt(100)}}}
	}
	first, second, third := makeBill(), makeBill(), makeBill()
	second.Lines[0].TaxRate = &rate
	makeRecord := func(b *FinanceBill, no string) FinanceInvoiceRegistrationInput {
		return FinanceInvoiceRegistrationInput{InvoiceProfileID: profileID, InvoiceType: FinanceInvoiceDigitalNormal, TaxInvoiceNo: no, InvoiceDate: "2026-11-14", TotalAmount: decimal.NewFromInt(10), Allocations: []FinanceInvoiceAllocationInput{{BillID: b.ID, ExpectedVersion: 1, Amount: decimal.NewFromInt(10), Fees: []FinanceInvoiceFeeInput{{OrderFeeID: b.Lines[0].OrderFeeID, Amount: decimal.NewFromInt(10), TaxAmount: decimal.NewFromInt(2)}}}}}
	}
	old1 := &FinanceInvoice{InvoiceProfileID: &profileID, InvoiceTitle: "早期原抬头", TaxpayerIdentificationNo: "OLD-1", RecordNo: "OLD-REC-1"}
	old2 := &FinanceInvoice{InvoiceProfileID: &profileID, InvoiceTitle: "后期原抬头", TaxpayerIdentificationNo: "OLD-2", RegisteredAddress: "原地址", RegisteredPhone: "原电话", BankName: "原银行", BankAccount: "原账号", RecordNo: "OLD-REC-2"}
	repo := &invoiceRegistrationProfileStub{profile: &PartnerInvoiceProfile{ID: profileID, InvoiceTitle: "当前主档抬头", TaxpayerIdentificationNo: "NEW-TAX", Enabled: true}}
	rates := &invoiceDirectionRateRepoStub{exchangeRateRepoStub: exchangeRateRepoStub{rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}}
	uc := NewFinanceInvoiceUsecase(repo, NewOrderConfigUsecase(orderConfigNumberStub{}), NewExchangeRateUsecase(rates, nil), nil)
	records := []FinanceInvoiceRegistrationInput{makeRecord(first, "N1"), makeRecord(second, "N2"), makeRecord(third, "N3")}
	invoices, err := uc.buildRegistrationInvoices(context.Background(), org, records, []*FinanceBill{first, second, third}, []*FinanceInvoice{old1, old2, nil})
	if err != nil {
		t.Fatal(err)
	}
	if invoices[0].InvoiceTitle != "早期原抬头" || invoices[1].InvoiceTitle != "后期原抬头" || invoices[2].InvoiceTitle != "当前主档抬头" || invoices[0].RecordNo != old1.RecordNo || invoices[1].RecordNo != old2.RecordNo {
		t.Fatalf("同profile各票原快照/新增主档不可相互替代: %+v", invoices)
	}
	if invoices[0].Links[0].Fees[0].SourceTaxRate != nil || invoices[0].Lines[0].TaxRate != nil || invoices[1].Links[0].Fees[0].SourceTaxRate == nil || !invoices[1].Links[0].Fees[0].SourceTaxRate.IsZero() {
		t.Fatal("来源未设与显式零税率应分别保留")
	}
	records[0].InvoiceType = FinanceInvoiceDigitalSpecial
	if _, err = uc.buildRegistrationInvoices(context.Background(), org, records, []*FinanceBill{first, second, third}, []*FinanceInvoice{old1, old2, nil}); !errors.Is(err, ErrFinanceInvoiceProfileRequired) {
		t.Fatalf("普通改专票须按该票原快照检查完整性，不得使用另一票快照: %v", err)
	}
	for _, special := range []FinanceInvoiceType{FinanceInvoicePaperSpecial, FinanceInvoiceElectronicSpecial} {
		records[0].InvoiceType = special
		if _, err = uc.buildRegistrationInvoices(context.Background(), org, records, []*FinanceBill{first, second, third}, []*FinanceInvoice{old1, old2, nil}); !errors.Is(err, ErrFinanceInvoiceProfileRequired) {
			t.Fatalf("%s 属专票类，须要求抬头开户资料齐全: %v", special, err)
		}
	}
	records[0].InvoiceType = FinanceInvoicePaperNormal
	if _, err = uc.buildRegistrationInvoices(context.Background(), org, records, []*FinanceBill{first, second, third}, []*FinanceInvoice{old1, old2, nil}); err != nil {
		t.Fatalf("纸质普票不要求开户资料: %v", err)
	}
}
