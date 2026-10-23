package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// 跨票对冲仅应收订单贡献计提收入；应付账单属于其他订单时，不应扩大预锁集合。
func newCrossOrderNettingFixture(t *testing.T) *commissionNettingPostgresFixture {
	t.Helper()
	f := newCommissionNettingPostgresFixture(t)
	ctx := context.Background()
	payableOrder, err := f.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).
		SetOrderNo("CROSS-AP-" + f.suffix).SetCustomerID(f.customerID).
		SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOrderDate(financeCommissionIntegrationDate).Save(ctx)
	if err != nil {
		t.Fatalf("创建独立应付订单: %v", err)
	}
	lines, err := f.data.db.FinanceBillLine.Query().Where(billline.BillIDEQ(f.payableBillID)).All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		if err := f.data.db.OrderFee.UpdateOneID(line.OrderFeeID).SetOrderID(payableOrder.ID).Exec(ctx); err != nil {
			t.Fatal(err)
		}
		// 账单行的订单引用不可变，夹具重建完整应付事实。
		if err := f.data.db.FinanceBillLine.DeleteOne(line).Exec(ctx); err != nil {
			t.Fatal(err)
		}
		if err := f.data.db.FinanceBillLine.Create().
			SetBillID(line.BillID).SetOrderID(payableOrder.ID).SetOrderFeeID(line.OrderFeeID).
			SetOrderNo(payableOrder.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).
			SetQuantity(line.Quantity).SetUnitPrice(line.UnitPrice).SetTotalAmount(line.TotalAmount).
			SetNetAmount(line.NetAmount).SetTaxAmount(line.TaxAmount).SetCurrency(line.Currency).
			SetExchangeRate(line.ExchangeRate).SetBaseCurrency(line.BaseCurrency).
			SetBaseCurrencyAmount(line.BaseCurrencyAmount).SetActive(true).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}

	return f
}

func TestCommissionCrossOrderNettingPostgres(t *testing.T) {
	t.Run("预览创建及确认采用相同应收订单集合", func(t *testing.T) {
		f := newCrossOrderNettingFixture(t)
		ctx := context.Background()
		u := f.newUsecase()
		preview, err := u.Preview(ctx, f.organizationID, uuid.Nil, f.nettingID, f.employeeID, biz.CommissionRoleSales)
		if err != nil {
			t.Fatal(err)
		}
		if len(preview.Lines) != 2 || preview.RealizedRevenue.StringFixed(8) != "800.00000000" || preview.CommissionAmount.StringFixed(8) != "80.00000000" {
			t.Fatalf("预览应只计入两票应收: %+v", preview)
		}
		created, err := u.Create(ctx, f.organizationID, f.actorID, f.input("cross-create"))
		if err != nil || created == nil || !created.CommissionAmount.Equal(preview.CommissionAmount) || created.SourceFingerprint != preview.SourceFingerprint {
			t.Fatalf("跨票对冲创建应与预览一致: %+v %v", created, err)
		}
		confirmed, err := u.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
		if err != nil || confirmed == nil || confirmed.Status != biz.CommissionConfirmed {
			t.Fatalf("跨票对冲确认应复核成功: %+v %v", confirmed, err)
		}
	})

	t.Run("月度申请受控创建及批准", func(t *testing.T) {
		f := newCrossOrderNettingFixture(t)
		ctx := context.Background()
		// 对冲确认日落在本次申请覆盖的上月，方案分配区间仍覆盖来源日。
		today := biz.FinanceBusinessDate(time.Now())
		_, coverageTo, ok := biz.CommissionApplicationPeriod(today)
		if !ok {
			t.Fatal("申请期间无效")
		}
		confirmedAt, err := time.Parse("2006-01-02", coverageTo)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.data.db.FinanceNetting.UpdateOneID(f.nettingID).SetConfirmedAt(confirmedAt).Exec(ctx); err != nil {
			t.Fatal(err)
		}
		u := biz.NewFinanceCommissionApplicationUsecase(NewFinanceCommissionApplicationRepo(f.data))
		scope := biz.WorkbenchScope{OrganizationID: f.organizationID, UserID: f.employeeID, Today: today, Now: time.Now(), CanOperateBusiness: true}
		candidates, err := u.ListMyCandidates(ctx, scope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
		if err != nil || candidates == nil || candidates.Total != 1 || len(candidates.Items) != 1 || candidates.Items[0].CommissionAmount.StringFixed(8) != "80.00000000" {
			t.Fatalf("月度候选应包含跨票对冲: %+v %v", candidates, err)
		}
		application, err := u.Submit(ctx, scope)
		if err != nil || application == nil || application.CommissionCount != 1 || application.TotalCommissionAmount.StringFixed(8) != "80.00000000" {
			t.Fatalf("月度提交应受控创建跨票对冲提成: %+v %v", application, err)
		}
		approved, err := u.Approve(ctx, []uuid.UUID{f.organizationID}, f.actorID, application.ID, application.Version)
		if err != nil || approved == nil || approved.Status != biz.CommissionApplicationApproved {
			t.Fatalf("月度批准应在锁内复核相同应收订单集合: %+v %v", approved, err)
		}
	})
}
