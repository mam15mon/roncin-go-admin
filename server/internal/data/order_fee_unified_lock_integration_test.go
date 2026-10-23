package data

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	adjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	"github.com/shopspring/decimal"
)

func unlockFeeTestOrder(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order) {
	t.Helper()
	if _, err := f.data.db.OrderLockRecord.Update().Where(lockent.OrderIDEQ(o.ID), lockent.UnlockedAtIsNil()).SetUnlockedAt(time.Now().UTC()).SetUnlockedBy(f.approverID).SetUnlockMode(lockent.UnlockModeROLE_DIRECT).SetUnlockReason("测试直接解锁").Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.data.db.Order.UpdateOneID(o.ID).ClearLockedAt().ClearLockedBy().ClearLockSource().AddVersion(1).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
}

func addOrdinaryTestFee(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order, amount int64) *biz.OrderFee {
	t.Helper()
	input := correctionTestInput(f, "ADD", uuid.NewString(), nil, amount).After
	input.IdempotencyKey = uuid.NewString()
	input.TaxRateOverride = nil
	value, err := normalFeeUsecaseForRollback(f).Add(f.ctx, f.organizationID, f.approverID, o.ID, input, false)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func createOccupiedTestFeeBill(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order, fee *biz.OrderFee) *ent.FinanceBill {
	t.Helper()
	key := uuid.NewString()
	b, err := withTestFinanceBillSettlementAccountSnapshot(f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("B-"+key).SetIdempotencyKey(key).SetDirection(billent.Direction(fee.Direction)).SetStatus(billent.StatusDRAFT).SetSettlementPartyID(fee.SettlementPartyID).SetSettlementPartyName(fee.SettlementPartyName).SetCurrency(fee.Currency).SetBaseCurrency(fee.BaseCurrency).SetExchangeRate(fee.ExchangeRate.String()).SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-01").SetTotalAmount(fee.TotalAmount.String()).SetNetAmount(fee.NetAmount.String()).SetTaxAmount(fee.TaxAmount.String()).SetBaseCurrencyAmount(fee.BaseCurrencyAmount.String()).SetFeeCount(1).SetBillDate("2026-09-01"), uuid.New(), fee.Currency).Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().SetBillID(b.ID).SetOrderFeeID(fee.ID).SetOrderID(o.ID).SetOrderNo(o.OrderNo).SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetQuantity(fee.Quantity.String()).SetUnitPrice(fee.UnitPrice.String()).SetTotalAmount(fee.TotalAmount.String()).SetNetAmount(fee.NetAmount.String()).SetTaxAmount(fee.TaxAmount.String()).SetCurrency(fee.Currency).SetExchangeRate(fee.ExchangeRate.String()).SetBaseCurrency(fee.BaseCurrency).SetBaseCurrencyAmount(fee.BaseCurrencyAmount.String()).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestUnifiedFeeBillingAndCorrectionAdmissionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	o := f.createOrder("UNIFIED-" + f.suffix)
	first := addOrdinaryTestFee(t, f, o, 100)
	// 未人工审核也能选中第一家供应商费用建账，随后仍可录入其他费用。
	candidates, err := NewFinanceBillRepo(f.data).LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{first.ID})
	if err != nil || len(candidates) != 1 || !candidates[0].CanBill {
		t.Fatalf("未审核费用应可建账: %+v %v", candidates, err)
	}
	bill := createOccupiedTestFeeBill(t, f, o, first)
	second := addOrdinaryTestFee(t, f, o, 200)
	update := *first
	update.Note = new(string)
	if _, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, first.ID, &update, false); err != biz.ErrOrderFeeBillOccupied {
		t.Fatalf("草稿建账也应冻结原费: %v", err)
	}
	seedManualOrderReviewForTest(t, f.data, o.ID, f.approverID)
	if _, err = uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "REPLACE", "occupied", first, 150), false); err != biz.ErrOrderFeeBillOccupied {
		t.Fatalf("已建账源费不可提交补审: %v", err)
	}
	if _, err = NewFinanceBillRepo(f.data).Cancel(f.ctx, []uuid.UUID{f.organizationID}, bill.ID, f.approverID, bill.Version, "回退至未建账", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: "finance.bill.cancel", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	first, err = NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	r, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "REPLACE", "unbilled", first, 150), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewFinanceBillRepo(f.data).LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{first.ID}); err != biz.ErrFeeCorrectionPending {
		t.Fatalf("申请应只占用源费: %v", err)
	}
	if _, err = NewFinanceBillRepo(f.data).LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{second.ID}); err != nil {
		t.Fatalf("同票其他费用仍可建账: %v", err)
	}
	approved := approveCorrectionForTest(t, f, uc, o, r)
	unlockFeeTestOrder(t, f, o)
	current, err := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, approved.After.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.Note = new(string)
	current, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, current.ID, current, false)
	if err != nil {
		t.Fatalf("解锁后替代费应可编辑: %v", err)
	}
	if err = normalFeeUsecaseForRollback(f).Remove(f.ctx, f.organizationID, f.approverID, o.ID, current.ID, current.Version, "解锁后删费"); err != nil {
		t.Fatalf("解锁后当前未建账替代费应可删除: %v", err)
	}
	history, err := NewOrderFeeCorrectionRepo(f.data).Get(f.ctx, f.organizationID, r.ID)
	if err != nil || history.After == nil {
		t.Fatalf("删除当前费用须保留申请快照: %v", err)
	}
}

func TestOrdinaryFeeMutationReconcilesPaidCommissionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("MUTATION-" + f.suffix)
	f.approveInitialFees(o)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	uc := biz.NewCommissionUsecase(NewCommissionRepo(f.data), nil, f.data)
	// 使用正式发放契约，已付事实不得被费用重算覆盖。
	paid, err := markCommissionPaidForTest(f.ctx, uc, f.organizationID, f.approverID, parent.ID, parent.Version)
	if err != nil {
		t.Fatal(err)
	}
	unlockFeeTestOrder(t, f, o)
	fee := addOrdinaryTestFee(t, f, o, 100)
	assertNet := func(want string) {
		t.Helper()
		rows, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.CommissionIDEQ(parent.ID), adjustmentent.StatusIn(adjustmentent.StatusCONFIRMED, adjustmentent.StatusPAID)).All(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		net := decimal.RequireFromString("60")
		for _, row := range rows {
			amount := decimal.RequireFromString(row.Amount)
			if row.Direction == adjustmentent.DirectionDECREASE {
				net = net.Sub(amount)
			} else {
				net = net.Add(amount)
			}
		}
		if !net.Equal(decimal.RequireFromString(want)) {
			t.Fatalf("提成净额=%s，预期%s", net, want)
		}
	}
	assertNet("50")
	fee.UnitPrice = decimal.NewFromInt(200)
	fee.QuantitySource = nil
	fee, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee, false)
	if err != nil {
		t.Fatal(err)
	}
	assertNet("40")
	if err = normalFeeUsecaseForRollback(f).Remove(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee.Version, "撤销未建账成本"); err != nil {
		t.Fatal(err)
	}
	assertNet("60")
	latest, err := f.data.db.FinanceCommission.Get(f.ctx, paid.ID)
	if err != nil || latest.PaidAt == nil || latest.PaidAmount == nil || *latest.PaidAmount != "60.00000000" {
		t.Fatalf("费用重算必须保留已付事实: %+v %v", latest, err)
	}
}
