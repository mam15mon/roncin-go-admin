package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type billFeeCandidateRepoStub struct {
	FinanceBillRepo
	bill   *FinanceBill
	items  []*FinanceBillableFee
	filter FinanceBillFeeCandidateFilter
	calls  int
}

func (s *billFeeCandidateRepoStub) Get(context.Context, []uuid.UUID, uuid.UUID) (*FinanceBill, error) {
	return s.bill, nil
}

func (s *billFeeCandidateRepoStub) ListBillFeeCandidates(_ context.Context, _ *FinanceBill, filter FinanceBillFeeCandidateFilter) (*FinanceBillCreationCandidateResult, error) {
	s.calls++
	s.filter = filter
	return &FinanceBillCreationCandidateResult{Items: s.items, Total: int64(len(s.items))}, nil
}

func billFeeTestBill() *FinanceBill {
	return &FinanceBill{ID: uuid.New(), OrganizationID: uuid.New(), Status: FinanceBillConfirmed, Purpose: FinanceBillNormal, Direction: OrderFeeReceivable, SettlementPartyID: uuid.New(), Currency: "USD", BaseCurrency: "CNY", ExchangeRate: decimal.RequireFromString("7.1234")}
}

func billFeeTestCandidate(bill *FinanceBill) *FinanceBillableFee {
	return &FinanceBillableFee{OrganizationID: bill.OrganizationID, Fee: &OrderFee{ID: uuid.New(), OrderID: uuid.New(), EconomicEffect: OrderFeeEconomicIncrease, AccountingRequired: true, Direction: bill.Direction, SettlementPartyID: bill.SettlementPartyID, Currency: bill.Currency, BaseCurrency: bill.BaseCurrency, TotalAmount: decimal.RequireFromString("10"), NetAmount: decimal.RequireFromString("10")}}
}

// TestListFeeCandidatesPaginationBoundary 复用 biz 公共分页校验：page>=1、1<=pageSize<=200，关键字最多 100 字。
func TestListFeeCandidatesPaginationBoundary(t *testing.T) {
	bill := billFeeTestBill()
	cases := []struct {
		name     string
		filter   FinanceBillFeeCandidateFilter
		accepted bool
	}{
		{"最小页容量", FinanceBillFeeCandidateFilter{Page: 1, PageSize: 1}, true},
		{"最大页容量", FinanceBillFeeCandidateFilter{Page: 2, PageSize: MaxListPageSize}, true},
		{"页容量超过上限", FinanceBillFeeCandidateFilter{Page: 1, PageSize: MaxListPageSize + 1}, false},
		{"页容量为零", FinanceBillFeeCandidateFilter{Page: 1, PageSize: 0}, false},
		{"页码为零", FinanceBillFeeCandidateFilter{Page: 0, PageSize: 20}, false},
		{"关键字超长", FinanceBillFeeCandidateFilter{Page: 1, PageSize: 20, Keyword: string(make([]rune, 101))}, false},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			repo := &billFeeCandidateRepoStub{bill: bill}
			usecase := NewFinanceBillUsecase(repo, nil, nil, nil)
			_, err := usecase.ListFeeCandidates(context.Background(), []uuid.UUID{bill.OrganizationID}, bill.ID, item.filter)
			if item.accepted && (err != nil || repo.calls != 1 || repo.filter.PageSize != item.filter.PageSize) {
				t.Fatalf("应接受: err=%v calls=%d filter=%#v", err, repo.calls, repo.filter)
			}
			if !item.accepted && (!errors.Is(err, ErrFinanceBillInvalidArgument) || repo.calls != 0) {
				t.Fatalf("应拒绝且不查询: err=%v calls=%d", err, repo.calls)
			}
		})
	}
}

func TestListFeeCandidatesMarksScopeMismatchAndRejectsCancelledBill(t *testing.T) {
	bill := billFeeTestBill()
	valid := billFeeTestCandidate(bill)
	valid.CanBill = true
	foreign := billFeeTestCandidate(bill)
	foreign.CanBill = true
	foreign.Fee.Currency = "EUR"
	repo := &billFeeCandidateRepoStub{bill: bill, items: []*FinanceBillableFee{valid, foreign}}
	usecase := NewFinanceBillUsecase(repo, nil, nil, nil)
	result, err := usecase.ListFeeCandidates(context.Background(), []uuid.UUID{bill.OrganizationID}, bill.ID, FinanceBillFeeCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Items[0].CanBill || result.Items[1].CanBill || len(result.Items[1].BlockedReasons) != 1 {
		t.Fatalf("候选须与提交同口径复核: %#v %#v", result.Items[0], result.Items[1])
	}
	bill.Status = FinanceBillCancelled
	if _, err := usecase.ListFeeCandidates(context.Background(), []uuid.UUID{bill.OrganizationID}, bill.ID, FinanceBillFeeCandidateFilter{Page: 1, PageSize: 20}); !errors.Is(err, ErrFinanceBillInvalidTransition) {
		t.Fatalf("已取消账单不应返回候选: %v", err)
	}
}

func TestValidateBillFeeForAttachMatrix(t *testing.T) {
	bill := billFeeTestBill()
	cases := []struct {
		name   string
		mutate func(item *FinanceBillableFee)
		want   error
	}{
		{"同口径费用", func(*FinanceBillableFee) {}, nil},
		{"已建账", func(item *FinanceBillableFee) { item.Fee.HasActiveBill = true }, ErrFinanceBillFeeInvalid},
		{"无需建账", func(item *FinanceBillableFee) { item.Fee.AccountingRequired = false }, ErrFinanceBillFeeInvalid},
		{"已被替代", func(item *FinanceBillableFee) { id := uuid.New(); item.Fee.SupersededByRequestID = &id }, ErrFinanceBillFeeInvalid},
		{"内部冲回", func(item *FinanceBillableFee) { item.Fee.EconomicEffect = OrderFeeEconomicDecrease }, ErrFinanceBillFeeInvalid},
		{"跨公司", func(item *FinanceBillableFee) { item.OrganizationID = uuid.New() }, ErrFinanceBillFeeInvalid},
		{"不同方向", func(item *FinanceBillableFee) { item.Fee.Direction = OrderFeePayable }, ErrFinanceBillFeeMismatch},
		{"不同结算单位", func(item *FinanceBillableFee) { item.Fee.SettlementPartyID = uuid.New() }, ErrFinanceBillFeeMismatch},
		{"不同币种", func(item *FinanceBillableFee) { item.Fee.Currency = "EUR" }, ErrFinanceBillFeeMismatch},
		{"不同本币", func(item *FinanceBillableFee) { item.Fee.BaseCurrency = "HKD" }, ErrFinanceBillFeeMismatch},
		{"更正来源费用加入普通账单", func(item *FinanceBillableFee) { id := uuid.New(); item.Fee.SourceFeeID = &id }, ErrFinanceBillFeeMismatch},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			candidate := billFeeTestCandidate(bill)
			item.mutate(candidate)
			err := ValidateBillFeeForAttach(bill, candidate)
			if (item.want == nil && err != nil) || (item.want != nil && !errors.Is(err, item.want)) {
				t.Fatalf("校验结果=%v，期望 %v", err, item.want)
			}
		})
	}
}

// TestRecomputeFinanceBillAmountsTailAndEstimate 验证按账单汇率重算：头本位币 = 总额 × 汇率，末行吸收尾差，
// fee_count 等于有效行数，预计开票金额按已存预计币种与汇率重算；行汇率统一为账单汇率。
func TestRecomputeFinanceBillAmountsTailAndEstimate(t *testing.T) {
	bill := billFeeTestBill()
	bill.ExchangeRate = decimal.RequireFromString("7.1235")
	estimatedCurrency, estimatedRate := "CNY", decimal.RequireFromString("7.2")
	bill.EstimatedInvoiceCurrency, bill.EstimatedInvoiceRate = &estimatedCurrency, &estimatedRate
	amounts := []string{"0.01000001", "0.01000001", "0.01000001"}
	lines := make([]*FinanceBillLine, 0, len(amounts))
	for _, amount := range amounts {
		value := decimal.RequireFromString(amount)
		lines = append(lines, &FinanceBillLine{ID: uuid.New(), Currency: "USD", BaseCurrency: "CNY", TotalAmount: value, NetAmount: value, ExchangeRate: decimal.RequireFromString("6.9")})
	}
	if err := RecomputeFinanceBillAmounts(bill, lines); err != nil {
		t.Fatal(err)
	}
	total := decimal.RequireFromString("0.03000003")
	wantBase := total.Mul(bill.ExchangeRate).RoundBank(8)
	if !bill.TotalAmount.Equal(total) || !bill.BaseCurrencyAmount.Equal(wantBase) || bill.FeeCount != 3 {
		t.Fatalf("账单头=%s/%s/%d，期望 %s/%s/3", bill.TotalAmount, bill.BaseCurrencyAmount, bill.FeeCount, total, wantBase)
	}
	sum := decimal.Zero
	for _, line := range lines {
		if !line.ExchangeRate.Equal(bill.ExchangeRate) {
			t.Fatalf("行汇率应为账单汇率: %s", line.ExchangeRate)
		}
		sum = sum.Add(line.BaseCurrencyAmount)
	}
	perLine := decimal.RequireFromString("0.01000001").Mul(bill.ExchangeRate).RoundBank(8)
	if !sum.Equal(wantBase) || !lines[0].BaseCurrencyAmount.Equal(perLine) || !lines[2].BaseCurrencyAmount.Equal(wantBase.Sub(perLine.Mul(decimal.NewFromInt(2)))) {
		t.Fatalf("行本位币合计 %s 期望 %s，末行=%s", sum, wantBase, lines[2].BaseCurrencyAmount)
	}
	if bill.EstimatedInvoiceAmount == nil || !bill.EstimatedInvoiceAmount.Equal(total.Mul(estimatedRate).RoundBank(8)) {
		t.Fatalf("预计开票金额=%v", bill.EstimatedInvoiceAmount)
	}
	if err := RecomputeFinanceBillAmounts(bill, nil); !errors.Is(err, ErrFinanceBillLastFee) {
		t.Fatalf("移空应拒绝: %v", err)
	}
}

func TestNormalizeUpdateFinanceBillFees(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	base := UpdateFinanceBillFeesInput{ID: uuid.New(), ExpectedVersion: 1}
	cases := []struct {
		name  string
		input func() UpdateFinanceBillFeesInput
		ok    bool
	}{
		{"同时增删", func() UpdateFinanceBillFeesInput {
			v := base
			v.AddFeeIDs, v.RemoveFeeIDs = []uuid.UUID{a}, []uuid.UUID{b}
			return v
		}, true},
		{"增删均为空", func() UpdateFinanceBillFeesInput { return base }, false},
		{"缺少版本", func() UpdateFinanceBillFeesInput {
			v := base
			v.ExpectedVersion = 0
			v.AddFeeIDs = []uuid.UUID{a}
			return v
		}, false},
		{"同一费用既增又删", func() UpdateFinanceBillFeesInput {
			v := base
			v.AddFeeIDs, v.RemoveFeeIDs = []uuid.UUID{a}, []uuid.UUID{a}
			return v
		}, false},
		{"重复费用", func() UpdateFinanceBillFeesInput { v := base; v.AddFeeIDs = []uuid.UUID{a, a}; return v }, false},
		{"空 ID", func() UpdateFinanceBillFeesInput { v := base; v.RemoveFeeIDs = []uuid.UUID{uuid.Nil}; return v }, false},
		{"增加超过上限", func() UpdateFinanceBillFeesInput {
			v := base
			for range MaxListPageSize + 1 {
				v.AddFeeIDs = append(v.AddFeeIDs, uuid.New())
			}
			return v
		}, false},
		{"增加恰为上限", func() UpdateFinanceBillFeesInput {
			v := base
			for range MaxListPageSize {
				v.AddFeeIDs = append(v.AddFeeIDs, uuid.New())
			}
			return v
		}, true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			_, err := normalizeUpdateFinanceBillFees(item.input())
			if item.ok != (err == nil) {
				t.Fatalf("结果=%v，期望通过=%v", err, item.ok)
			}
		})
	}
}

func TestFinanceBillFeesLockedErrorKeepsReason(t *testing.T) {
	err := FinanceBillFeesLockedError([]FinanceBillFeeLockFact{FinanceBillFeeLockInvoice, FinanceBillFeeLockNettingDraft})
	if !errors.Is(err, ErrFinanceBillFeesLocked) {
		t.Fatalf("应保持 FEES_LOCKED 原因: %v", err)
	}
}
