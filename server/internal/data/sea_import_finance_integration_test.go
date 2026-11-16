package data

import (
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	"testing"
	"time"
)

func TestSeaImportFeesUseAuthoritativeETAAndRollbackPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("SI-FEE-" + uuid.NewString())
	e := &rateRefreshEnv{t: t, data: f.data, orgID: f.organizationID, partnerID: f.customerID, actorID: f.approverID, suffix: f.suffix}
	e.seedUSDRates()
	row := saveRow(f, biz.OrderFeeReceivable, "USD", nil, 100)
	row.Fee.Currency = "USD"
	row.Fee.IdempotencyKey = uuid.NewString()
	fee, err := normalFeeUsecaseForRollback(f).Add(f.ctx, f.organizationID, f.approverID, o.ID, row.Fee, false, true)
	if err != nil {
		t.Fatal(err)
	}
	e.expectFee("进口新增费用按ETA周", fee.ID, "7.0", feeent.ExchangeRateSourceWEEKLY, "2026-09-16", "700", 1)
	uc := biz.NewSeaOrderChangeUsecase(NewSeaOrderChangeRepo(f.data), f.data)
	link := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(o.ID), linkent.StatusEQ(linkent.StatusACTIVE)).OnlyX(f.ctx)
	transport := f.data.db.SeaTransportExecution.GetX(f.ctx, link.TransportExecutionID)
	etd, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00Z")
	eta, _ := time.Parse(time.RFC3339, o.Eta)
	command := func(etd, eta *time.Time) *biz.SeaTransportExecutionUpdateCommand {
		return &biz.SeaTransportExecutionUpdateCommand{OrderID: o.ID, ExpectedTransportExecutionVersion: transport.Version, Reason: "进口经营日期核对", IdempotencyKey: uuid.NewString(), Input: &biz.SeaTransportExecutionUpdateInput{ETD: etd, ETA: eta}}
	}
	updated, err := uc.ExecuteTransportExecutionUpdate(f.ctx, f.organizationID, f.approverID, command(&etd, &eta))
	if err != nil {
		t.Fatal(err)
	}
	e.expectFee("进口只改ETD不改费用", fee.ID, "7.0", feeent.ExchangeRateSourceWEEKLY, "2026-09-16", "700", 1)
	transport = f.data.db.SeaTransportExecution.GetX(f.ctx, updated.TransportExecution.ID)
	eta, _ = time.Parse(time.RFC3339, rateRefreshW1)
	updated, err = uc.ExecuteTransportExecutionUpdate(f.ctx, f.organizationID, f.approverID, command(&etd, &eta))
	if err != nil {
		t.Fatal(err)
	}
	e.expectFee("进口共享ETA变更联动费用", fee.ID, "6.8", feeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 2)
	transport = f.data.db.SeaTransportExecution.GetX(f.ctx, updated.TransportExecution.ID)
	before := f.data.db.Order.GetX(f.ctx, o.ID)
	versions := f.data.db.SeaTransportExecutionVersion.Query().CountX(f.ctx)
	eta, _ = time.Parse(time.RFC3339, rateRefreshW3)
	if _, err = uc.ExecuteTransportExecutionUpdate(f.ctx, f.organizationID, f.approverID, command(&etd, &eta)); err == nil {
		t.Fatal("缺进口ETA当周汇率必须拒绝整个改期")
	}
	actual := f.data.db.Order.GetX(f.ctx, o.ID)
	actualTransport := f.data.db.SeaTransportExecution.GetX(f.ctx, transport.ID)
	if actual.Eta != before.Eta || actual.Version != before.Version || actualTransport.Version != transport.Version || f.data.db.SeaTransportExecutionVersion.Query().CountX(f.ctx) != versions {
		t.Fatal("缺汇率后订单/航程/历史未整体回滚")
	}
	e.expectFee("缺汇率保留旧费用", fee.ID, "6.8", feeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 2)
	// 取消未承接不会阻断费用和完结；完成服务不自动建账或核销。
	orderUC := biz.NewOrderUsecase(NewOrderRepo(f.data), nil, nil, nil, f.data)
	closed, err := orderUC.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, actual.Version, biz.OrderClosureClosed, "")
	if err != nil || closed.FlowStatus != biz.OrderFlowImportNotApplicable {
		t.Fatalf("无承接完结不伪造操作完成: %+v %v", closed, err)
	}
	stored := f.data.db.OrderFee.GetX(f.ctx, fee.ID)
	if stored.Version != 2 || f.data.db.FinanceBill.Query().CountX(f.ctx) != 0 {
		t.Fatal("完结不应改变费用或自动建账")
	}
}
