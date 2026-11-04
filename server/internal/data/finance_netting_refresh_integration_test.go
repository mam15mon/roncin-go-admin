package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currency "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	netalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	number "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

func TestFinanceNettingRefreshDraftRatesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	for _, code := range []string{"CNY", "USD"} {
		exists, err := data.db.Currency.Query().Where(currency.CodeEQ(code)).Exist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			if _, err := data.db.Currency.Create().SetCode(code).SetName(code).SetEnabled(true).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	f := newFinanceBillBatchPostgresFixture(t, data)
	if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(number.DocumentTypeNetting).SetPrefix("NT-").SetDateFormat(number.DateFormatNone).SetSequenceLength(4).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.cleanupNettingRows)
	ar := f.createBatchUnbilledFee("refresh-ar")
	ap := f.createBatchUnbilledPayableFee("refresh-ap")
	if _, err := data.db.OrderFee.Update().Where(fee.IDIn(ar, ap)).SetCurrency("USD").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetExchangeRate("7").SetBaseCurrencyAmount("700").Save(ctx); err != nil {
		t.Fatal(err)
	}
	f.createExchangeRateSetting("7")
	if _, err := data.db.ExchangeRateSetting.Create().SetOrganizationID(f.organizationID).SetFromCurrency("USD").SetToCurrency("CNY").SetEffectiveFrom(time.Date(2026, 8, 31, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).SetRate("8").SetIsActive(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	uc := f.newUsecase(NewFinanceBillRepo(data))
	policy := biz.FinanceBillGroupingPolicy{Mode: "NETTING"}
	input := biz.PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{ar, ap}, GroupingPolicy: policy}
	pre, err := uc.PreviewBatch(ctx, f.organizationID, input)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range pre.Groups {
		input.GroupConfigs = append(input.GroupConfigs, biz.FinanceBillBatchPreviewGroupConfig{GroupKey: group.GroupKey, BillDate: financeBillIntegrationDate, SettlementAccountID: f.usdAccountID})
	}
	pre, err = uc.PreviewBatch(ctx, f.organizationID, input)
	if err != nil {
		t.Fatal(err)
	}
	create := biz.CreateFinanceBillBatchInput{FeeIDs: input.FeeIDs, GroupingPolicy: policy, PreviewToken: pre.PreviewToken, IdempotencyKey: "refresh-" + f.suffix}
	for _, group := range input.GroupConfigs {
		create.Groups = append(create.Groups, biz.CreateFinanceBillBatchGroupInput{GroupKey: group.GroupKey, StatementTitle: "测试账单", BillDate: group.BillDate, SettlementAccountID: group.SettlementAccountID})
	}
	batch, err := uc.CreateBatch(ctx, f.organizationID, f.actorUserID, create)
	if err != nil {
		t.Fatal(err)
	}
	netting := batch.Nettings[0]
	if !netting.BaseCurrencyAmount.Equal(decimal.NewFromInt(700)) || !netting.PayableBaseAmount.Equal(decimal.NewFromInt(700)) {
		t.Fatalf("草稿本币=%v", netting)
	}
	versions := make(map[uuid.UUID]uint64)
	for _, bill := range batch.Bills {
		versions[bill.ID] = bill.Version
		if bill.Direction == biz.OrderFeeReceivable {
			changed, err := uc.Update(ctx, []uuid.UUID{f.organizationID}, f.actorUserID, biz.UpdateFinanceBillInput{ID: bill.ID, ExpectedVersion: bill.Version, BillDate: "2026-09-01", SettlementAccountID: f.usdAccountID})
			if err != nil {
				t.Fatal(err)
			}
			versions[bill.ID] = changed.Version
		}
	}
	if _, err := uc.ConfirmBatch(ctx, []uuid.UUID{f.organizationID}, f.actorUserID, batch.ID, versions); err != nil {
		t.Fatal(err)
	}
	netUC := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil)
	audit := &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorUserID, Action: "finance.netting.confirm", Result: "invalid"}
	if _, err := NewFinanceNettingRepo(data).Confirm(ctx, f.organizationID, f.actorUserID, netting.ID, netting.Version, audit); err == nil {
		t.Fatal("失败审计应回滚确认和重算")
	}
	draft, err := netUC.Get(ctx, []uuid.UUID{f.organizationID}, netting.ID)
	if err != nil || draft.Status != biz.FinanceNettingDraft || !draft.BaseCurrencyAmount.Equal(decimal.NewFromInt(700)) {
		t.Fatalf("失败确认未回滚:%v,%v", draft, err)
	}
	active, err := data.db.FinanceNettingAllocation.Query().Where(netalloc.NettingIDEQ(netting.ID), netalloc.ActiveEQ(true)).Count(ctx)
	if err != nil || active != 0 {
		t.Fatalf("回滚后生效分摊=%d,%v", active, err)
	}
	confirmed, err := netUC.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorUserID, netting.ID, netting.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed.BaseCurrencyAmount.Equal(decimal.NewFromInt(800)) || !confirmed.PayableBaseAmount.Equal(decimal.NewFromInt(700)) || !confirmed.ExchangeGainLoss.Equal(decimal.NewFromInt(-100)) {
		t.Fatalf("确认仍使用旧汇率:%v", confirmed)
	}
	for _, allocation := range confirmed.Allocations {
		want := int64(700)
		if allocation.Direction == biz.OrderFeeReceivable {
			want = 800
		}
		if !allocation.BaseCurrencyAmount.Equal(decimal.NewFromInt(want)) || !allocation.Active {
			t.Fatalf("分摊快照未更新:%v", allocation)
		}
	}
	if _, err := netUC.Confirm(ctx, []uuid.UUID{f.organizationID}, f.actorUserID, netting.ID, confirmed.Version); !errors.Is(err, biz.ErrFinanceNettingTransition) {
		t.Fatalf("重复确认必须拒绝:%v", err)
	}
	stored, err := netUC.Get(ctx, []uuid.UUID{f.organizationID}, netting.ID)
	if err != nil || stored.Version != confirmed.Version || !stored.ExchangeGainLoss.Equal(confirmed.ExchangeGainLoss) {
		t.Fatalf("重复确认改变事实:%v,%v", stored, err)
	}
}
