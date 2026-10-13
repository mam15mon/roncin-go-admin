package data

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestAutoOrderLock_MergedBillSettlementPostgres(t *testing.T) {
	data, cleanup := newAutoLockTestContext(t)
	defer cleanup()

	for _, tc := range []struct {
		name    string
		amountB string
		shared  bool
	}{
		{name: "独立订单100加200收100不锁且收齐300后两票锁定", amountB: "200"},
		{name: "独立订单100加100收100不锁且收齐200后两票锁定", amountB: "100"},
		{name: "共享主单100加100收100不得整组误锁", amountB: "100", shared: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			createOrder := func(suffix string) *ent.Order {
				if tc.shared {
					return f.createSEOrder("SE-" + f.suffix + suffix)
				}
				return f.createPlainOrder("AI-"+f.suffix+suffix, orderent.BusinessTypeAI)
			}
			a, b := createOrder("-A"), createOrder("-B")
			billID := f.createReceivableBillAmounts([]*ent.Order{a, b}, []string{"100", tc.amountB}, "USD")
			partial := f.settleOnBill(billID, "100")
			if err := f.triggerVerification(partial); err != nil {
				t.Fatalf("部分结清触发: %v", err)
			}
			for _, o := range []*ent.Order{a, b} {
				if f.reload(o.ID).LockedAt != nil || len(f.lockRecords(o.ID)) != 0 {
					t.Fatalf("部分回款恰等于一票金额时误锁 %s", o.OrderNo)
				}
			}
			full := f.settleOnBill(billID, tc.amountB)
			if err := f.triggerVerification(full); err != nil {
				t.Fatalf("全额结清触发: %v", err)
			}
			if err := f.triggerVerification(full); err != nil {
				t.Fatalf("重复全额结清触发: %v", err)
			}
			for _, o := range []*ent.Order{a, b} {
				locked := f.reload(o.ID)
				if locked.LockedAt == nil || locked.LockGeneration != 1 || len(f.lockRecords(o.ID)) != 1 {
					t.Fatalf("全收后锁定或幂等异常 %s: locked=%v generation=%d", o.OrderNo, locked.LockedAt, locked.LockGeneration)
				}
			}
			if tc.shared {
				ra, rb := f.lockRecords(a.ID)[0], f.lockRecords(b.ID)[0]
				if ra.MasterBillVersionID == nil || rb.MasterBillVersionID == nil || *ra.MasterBillVersionID != *rb.MasterBillVersionID {
					t.Fatal("共享主单全收后未复用同一版本")
				}
			}
		})
	}
}

func TestAutoOrderLock_PerBillCurrencySettlementPostgres(t *testing.T) {
	data, cleanup := newAutoLockTestContext(t)
	defer cleanup()

	for _, currency := range []string{"USD", "EUR"} {
		t.Run("不同账单不得轧差-"+currency, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			a := f.createPlainOrder("AI-"+f.suffix+"-A", orderent.BusinessTypeAI)
			b := f.createPlainOrder("AI-"+f.suffix+"-B", orderent.BusinessTypeAI)
			// A 两个应收行共计200；合账150结清，加另一账单50结清，旧判定会算出零。
			merged := f.createReceivableBillAmounts([]*ent.Order{a, b}, []string{"100", "100"}, "USD")
			other := f.createReceivableBillAmounts([]*ent.Order{a}, []string{"100"}, currency)
			f.settleOnBill(merged, "150")
			trigger := f.settleOnBill(other, "50")
			if err := f.triggerVerification(trigger); err != nil {
				t.Fatalf("跨账单触发: %v", err)
			}
			if f.reload(a.ID).LockedAt != nil || f.auditReason(a) != biz.AutoLockReasonUnsettledReceivable {
				t.Fatal("不同账单或币种相抵造成误锁")
			}
			f.settleOnBill(merged, "50")
			full := f.settleOnBill(other, "50")
			if err := f.triggerVerification(full); err != nil {
				t.Fatalf("逐账单结清后触发: %v", err)
			}
			if f.reload(a.ID).LockedAt == nil {
				t.Fatal("每张关联账单全收后 A 未锁定")
			}
		})
	}

	t.Run("账单超额与另一张不足不得相抵", func(t *testing.T) {
		f := newAutoLockPostgresFixture(t, data)
		o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
		first := f.createReceivableBill([]*ent.Order{o}, "100")
		second := f.createReceivableBillAmounts([]*ent.Order{o}, []string{"100"}, "EUR")
		// 直接构造超额异常以验证判定拒绝；正常核销写入口仍拒绝超过账单余额。
		f.settleOnBill(first, "150")
		trigger := f.settleOnBill(second, "50")
		if err := f.triggerVerification(trigger); err != nil {
			t.Fatalf("异常结清触发: %v", err)
		}
		if f.reload(o.ID).LockedAt != nil || f.auditReason(o) != biz.AutoLockReasonUnsettledReceivable {
			t.Fatal("超额异常被另一张账单欠款抵消")
		}
	})

	t.Run("同票多个账单行不重复计算整张账单结清", func(t *testing.T) {
		f := newAutoLockPostgresFixture(t, data)
		o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
		billID := f.createReceivableBillAmounts([]*ent.Order{o, o}, []string{"100", "200"}, "USD")
		trigger := f.settleOnBill(billID, "300")
		if err := f.triggerVerification(trigger); err != nil {
			t.Fatalf("多行整单结清触发: %v", err)
		}
		if f.reload(o.ID).LockedAt == nil || len(f.lockRecords(o.ID)) != 1 {
			t.Fatal("同票多行账单去重或全收判定异常")
		}
	})
}

// netOnBill 构造双边对冲事实，返回对冲单与应收分摊供有效性回归使用。
func (f *autoLockPostgresFixture) netOnBill(billID uuid.UUID, amount string, status financenettingent.Status, active bool) (*ent.FinanceNetting, *ent.FinanceNettingAllocation) {
	f.t.Helper()
	bill, err := f.data.db.FinanceBill.Get(f.ctx, billID)
	if err != nil {
		f.t.Fatalf("读取对冲账单: %v", err)
	}
	base := decimal.RequireFromString(amount).Mul(decimal.RequireFromString(bill.ExchangeRate)).StringFixed(8)
	key := uuid.NewString()
	create := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).SetBillNo("PAY-" + key).SetIdempotencyKey("pay-" + key).
		SetDirection(financebillent.DirectionPAYABLE).SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).SetSettlementPartyName("自动锁定测试客户").
		SetCurrency(bill.Currency).SetBaseCurrency("CNY").SetExchangeRate(bill.ExchangeRate).
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount(amount).SetNetAmount(amount).SetTaxAmount("0").SetBaseCurrencyAmount(base).
		SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1)
	payable, err := withTestFinanceBillSettlementAccountSnapshot(create, f.usdAccountOrCreate(), bill.Currency).Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建对冲应付账单: %v", err)
	}
	netting, err := f.data.db.FinanceNetting.Create().
		SetOrganizationID(f.organizationID).SetNettingNo("NT-" + key).
		SetIdempotencyKey("nt-" + key).SetRequestHash(key).SetStatus(status).
		SetSettlementPartyID(f.partnerID).SetSettlementPartyName("自动锁定测试客户").
		SetCurrency(bill.Currency).SetAmount(amount).SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(base).SetPayableBaseAmount(base).SetExchangeGainLoss("0").SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建对冲单: %v", err)
	}
	var receivable *ent.FinanceNettingAllocation
	for _, side := range []struct {
		bill      *ent.FinanceBill
		direction financenettingallocationent.Direction
	}{
		{bill: bill, direction: financenettingallocationent.DirectionRECEIVABLE},
		{bill: payable, direction: financenettingallocationent.DirectionPAYABLE},
	} {
		allocation, err := f.data.db.FinanceNettingAllocation.Create().
			SetNettingID(netting.ID).SetBillID(side.bill.ID).SetBillNo(side.bill.BillNo).
			SetDirection(side.direction).SetAmount(amount).SetBaseCurrencyAmount(base).SetActive(active).
			Save(f.ctx)
		if err != nil {
			f.t.Fatalf("创建对冲分摊: %v", err)
		}
		if side.direction == financenettingallocationent.DirectionRECEIVABLE {
			receivable = allocation
		}
	}
	return netting, receivable
}

func TestAutoOrderLock_SettlementSourcesPostgres(t *testing.T) {
	data, cleanup := newAutoLockTestContext(t)
	defer cleanup()

	for _, tc := range []struct {
		name         string
		verification string
		netting      string
	}{
		{name: "纯对冲结清", netting: "100"},
		{name: "核销与多笔对冲分别聚合不倍增", verification: "40", netting: "30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
			billID := f.createReceivableBill([]*ent.Order{o}, "100")
			if tc.verification != "" {
				f.settleOnBill(billID, "20")
				f.settleOnBill(billID, "20")
				f.netOnBill(billID, tc.netting, financenettingent.StatusCONFIRMED, true)
			}
			netting, _ := f.netOnBill(billID, tc.netting, financenettingent.StatusCONFIRMED, true)
			if err := f.repo.RunAutoSettlementLockCheck(f.ctx, biz.AutoOrderLockTrigger{
				Type: biz.AutoLockTriggerNetting, ResourceID: netting.ID,
				OrganizationID: f.organizationID, TriggeredBy: f.triggerUserID,
			}); err != nil {
				t.Fatalf("对冲触发: %v", err)
			}
			if f.reload(o.ID).LockedAt == nil {
				t.Fatal("有效对冲或混合全额结清未锁定")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		status financenettingent.Status
		active bool
	}{
		{name: "草稿对冲不计入", status: financenettingent.StatusDRAFT, active: true},
		{name: "取消对冲不计入", status: financenettingent.StatusCANCELLED, active: true},
		{name: "反转对冲不计入", status: financenettingent.StatusREVERSED, active: true},
		{name: "失效对冲分摊不计入", status: financenettingent.StatusCONFIRMED, active: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
			billID := f.createReceivableBill([]*ent.Order{o}, "100")
			trigger := f.settleOnBill(billID, "50")
			f.netOnBill(billID, "50", tc.status, tc.active)
			if err := f.triggerVerification(trigger); err != nil {
				t.Fatalf("无效对冲触发: %v", err)
			}
			if f.reload(o.ID).LockedAt != nil || f.auditReason(o) != biz.AutoLockReasonUnsettledReceivable {
				t.Fatal("无效对冲造成误锁")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		status financeverificationent.Status
		active bool
	}{
		{name: "反转核销父单仍有活跃分摊不计入", status: financeverificationent.StatusREVERSED, active: true},
		{name: "活跃核销父单的失效分摊不计入", status: financeverificationent.StatusACTIVE, active: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
			billID := f.createReceivableBill([]*ent.Order{o}, "100")
			trigger := f.settleOnBill(billID, "100")
			update := f.data.db.FinanceVerification.UpdateOneID(trigger).SetStatus(tc.status)
			if tc.status == financeverificationent.StatusREVERSED {
				update.SetReversedAt(time.Now()).SetReversedBy(f.triggerUserID).SetReversalReason("测试反转核销")
			}
			if _, err := update.Save(f.ctx); err != nil {
				t.Fatalf("设置核销状态: %v", err)
			}
			if _, err := f.data.db.FinanceVerificationAllocation.Update().Where(financeverificationallocationent.VerificationIDEQ(trigger)).SetActive(tc.active).Save(f.ctx); err != nil {
				t.Fatalf("设置核销分摊状态: %v", err)
			}
			if err := f.triggerVerification(trigger); err != nil {
				t.Fatalf("无效核销触发: %v", err)
			}
			if f.reload(o.ID).LockedAt != nil || f.auditReason(o) != biz.AutoLockReasonNoSettlement {
				t.Fatal("无效核销被作为有效结清事实")
			}
		})
	}
}

func TestAutoOrderLock_EligibilityGuardsPostgres(t *testing.T) {
	data, cleanup := newAutoLockTestContext(t)
	defer cleanup()

	t.Run("草稿应收即使异常存在足额分摊仍阻断", func(t *testing.T) {
		f := newAutoLockPostgresFixture(t, data)
		o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
		billID := f.createReceivableBill([]*ent.Order{o}, "100")
		trigger := f.settleOnBill(billID, "100")
		if _, err := f.data.db.FinanceBill.UpdateOneID(billID).SetStatus(financebillent.StatusDRAFT).Save(f.ctx); err != nil {
			t.Fatalf("设置账单草稿: %v", err)
		}
		if err := f.triggerVerification(trigger); err != nil {
			t.Fatalf("草稿触发: %v", err)
		}
		if f.reload(o.ID).LockedAt != nil || f.auditReason(o) != biz.AutoLockReasonUnsettledReceivable {
			t.Fatal("草稿账单异常足额分摊造成误锁")
		}
	})

	for _, tc := range []struct {
		name        string
		termination orderent.TerminationStatus
		closure     orderent.ClosureStatus
	}{
		{name: "已终止保持不锁", termination: orderent.TerminationStatusTERMINATED, closure: orderent.ClosureStatusOPEN},
		{name: "结案保持不锁", termination: orderent.TerminationStatusACTIVE, closure: orderent.ClosureStatusCLOSED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutoLockPostgresFixture(t, data)
			o := f.createPlainOrder("AI-"+f.suffix, orderent.BusinessTypeAI)
			billID := f.createReceivableBill([]*ent.Order{o}, "100")
			trigger := f.settleOnBill(billID, "100")
			update := f.data.db.Order.UpdateOneID(o.ID).SetTerminationStatus(tc.termination).SetClosureStatus(tc.closure)
			if tc.termination == orderent.TerminationStatusTERMINATED {
				update.SetTerminationType(orderent.TerminationTypeOTHER).SetTerminatedAt(time.Now()).SetTerminatedBy(f.triggerUserID).SetTerminationReason("测试终止")
			}
			if tc.closure == orderent.ClosureStatusCLOSED {
				update.SetClosedAt(time.Now()).SetClosedBy(f.triggerUserID).SetClosureReason("测试结案")
			}
			if _, err := update.Save(f.ctx); err != nil {
				t.Fatalf("设置生命周期: %v", err)
			}
			if err := f.triggerVerification(trigger); err != nil {
				t.Fatalf("生命周期触发: %v", err)
			}
			if f.reload(o.ID).LockedAt != nil || f.auditReason(o) != biz.AutoLockReasonLifecycle {
				t.Fatal("已终止或结案订单被自动锁定")
			}
		})
	}
}
