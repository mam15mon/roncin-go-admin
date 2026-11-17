package biz

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFinanceBillCreateAutomaticDateAndReplayAfterBusinessChanges(t *testing.T) {
	org, party, account := uuid.New(), uuid.New(), uuid.New()
	terms := 30
	repo := &defaultTermsWriteRepoStub{fees: defaultTermsWriteFees(party, false), summaries: map[uuid.UUID]*PartnerCreditSummary{party: {DefaultPaymentTermsDays: &terms}}, rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}
	uc := newDefaultTermsWriteUsecase(repo)
	now := time.Date(2026, 11, 16, 16, 5, 0, 0, time.UTC)
	uc.now = func() time.Time { return now }
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{repo.fees[0].Fee.ID}, SettlementAccountID: account, IdempotencyKey: "automatic-date", BillDate: "1900-01-01"}
	created, err := uc.Create(t.Context(), org, uuid.New(), input)
	if err != nil {
		t.Fatal(err)
	}
	if created.BillDate != "2026-11-17" || created.DueDate == nil || *created.DueDate != "2026-12-17" || created.CreateRequestHash == "" {
		t.Fatalf("必须使用北京时间并联动默认账期: %#v", created)
	}
	// 后续增减、编辑和客户默认账期变化不改变原始创建意图；跨日重试也不重新读取费用或汇率。
	created.BillDate = "2026-11-18"
	created.Lines = []*FinanceBillLine{{OrderFeeID: uuid.New(), Active: true}}
	changedTerms := 90
	created.PaymentTermsDays = &changedTerms
	repo.existingBill, repo.createdBill, repo.fees = created, nil, nil
	repo.summaries[party].DefaultPaymentTermsDays = &changedTerms
	now = now.AddDate(0, 0, 3)
	replay, err := uc.Create(t.Context(), org, uuid.New(), input)
	if err != nil || replay != created || repo.createdBill != nil {
		t.Fatalf("原请求须回放当前原单且不重建: %#v %v", replay, err)
	}
	input.SettlementAccountID = uuid.New()
	if _, err := uc.Create(t.Context(), org, uuid.New(), input); !errors.Is(err, ErrFinanceBillIdempotencyConflict) {
		t.Fatalf("改动原意图须冲突: %v", err)
	}
}

func TestFinanceBillBatchMidnightTokenAndReplay(t *testing.T) {
	for _, mode := range []string{"NORMAL", "NETTING"} {
		t.Run(mode, func(t *testing.T) {
			org, party, account := uuid.New(), uuid.New(), uuid.New()
			fees := defaultTermsWriteFees(party, false)
			if mode == "NETTING" {
				ap := financeBillableFeeForTest(party, "50", "50", "0", "50")
				ap.Fee.Direction = OrderFeePayable
				fees = append(fees, ap)
			}
			repo := &defaultTermsWriteRepoStub{fees: fees, rateContext: &ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}}
			for _, fee := range fees {
				fee.Fee.Currency = "USD"
			}
			rates := &exchangeRateRepoStub{rateContext: repo.rateContext, rateByCurrency: map[string]decimal.Decimal{"USD": decimal.RequireFromString("7.2")}}
			uc := NewFinanceBillUsecase(repo, NewExchangeRateUsecase(rates, nil), &financeBillTransactorStub{}, nil)
			now := time.Date(2026, 11, 16, 15, 59, 0, 0, time.UTC)
			uc.now = func() time.Time { return now }
			input := PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: mode}}
			for _, f := range fees {
				input.FeeIDs = append(input.FeeIDs, f.Fee.ID)
			}
			initial, err := uc.PreviewBatch(t.Context(), org, input)
			if err != nil {
				t.Fatal(err)
			}
			for _, g := range initial.Groups {
				input.GroupConfigs = append(input.GroupConfigs, FinanceBillBatchPreviewGroupConfig{GroupKey: g.GroupKey, SettlementAccountID: account})
			}
			preview, err := uc.PreviewBatch(t.Context(), org, input)
			if err != nil {
				t.Fatal(err)
			}
			request := CreateFinanceBillBatchInput{GroupingPolicy: input.GroupingPolicy, IdempotencyKey: "midnight-" + mode}
			for _, g := range preview.Groups {
				if g.BillDate != "2026-11-16" || !strings.HasPrefix(g.GroupToken, "20261116") {
					t.Fatalf("预览须固化自动日期: %#v", g)
				}
				group := CreateFinanceBillBatchGroupInput{GroupKey: g.GroupKey, GroupToken: g.GroupToken, StatementTitle: "测试客户", SettlementAccountID: account}
				for _, f := range g.Fees {
					group.FeeIDs = append(group.FeeIDs, f.Fee.ID)
				}
				request.Groups = append(request.Groups, group)
			}
			// 不成功的新请求跨上海午夜，必须在今日汇率解析前拒绝旧令牌。
			now = now.Add(2 * time.Minute)
			beforeLookups := len(rates.resolveDates)
			rates.resolveErr = ErrExchangeRateMissing
			if mode == "NORMAL" {
				result, err := uc.CreateBatch(context.Background(), org, uuid.New(), request)
				if err != nil || len(result.Groups) != 1 || !errors.Is(result.Groups[0].Err, ErrFinanceBillPreviewStale) {
					t.Fatalf("跨日旧预览须拒绝: %#v %v", result, err)
				}
			} else if _, err := uc.CreateBatch(context.Background(), org, uuid.New(), request); !errors.Is(err, ErrFinanceBillPreviewStale) {
				t.Fatalf("对冲跨日旧预览须原子拒绝: %v", err)
			}
			if len(rates.resolveDates) != beforeLookups {
				t.Fatal("跨日旧令牌必须先拒绝，不能查询今日缺失的汇率")
			}
			rates.resolveErr = nil
			// 新的当天预览创建成功后，跨日原键仍回放已创建账单，不要求重新预览。
			preview, err = uc.PreviewBatch(t.Context(), org, input)
			if err != nil {
				t.Fatal(err)
			}
			request.IdempotencyKey = "created-" + mode
			for i, g := range preview.Groups {
				for j := range request.Groups {
					if request.Groups[j].GroupKey == g.GroupKey {
						request.Groups[j].GroupToken = preview.Groups[i].GroupToken
					}
				}
			}
			created, err := uc.CreateBatch(t.Context(), org, uuid.New(), request)
			if err != nil {
				t.Fatal(err)
			}
			for _, g := range created.Groups {
				if g.Err != nil {
					t.Fatal(g.Err)
				}
			}
			now = now.AddDate(0, 0, 1)
			rates.resolveErr = ErrExchangeRateMissing
			beforeLookups = len(rates.resolveDates)
			replay, err := uc.CreateBatch(t.Context(), org, uuid.New(), request)
			if err != nil {
				t.Fatal(err)
			}
			if len(rates.resolveDates) != beforeLookups {
				t.Fatal("成功原键跨日回放不应再次解析汇率")
			}
			if len(replay.Groups) != len(created.Groups) {
				t.Fatal("重放组数变化")
			}
			for i, g := range replay.Groups {
				if g.Err != nil || g.Bill.ID != created.Groups[i].Bill.ID || g.Bill.BillDate != "2026-11-17" {
					t.Fatalf("跨日回放必须保留原单日期: %#v", g)
				}
			}
		})
	}
}
