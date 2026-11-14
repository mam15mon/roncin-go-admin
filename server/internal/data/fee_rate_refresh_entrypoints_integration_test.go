package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

// TestFeeRateRefreshUpdateDraftPostgres 验证订单保存（含单票 SE 内联改 TE 日期）按取汇率日联动，
// 缺汇率时整个保存回滚。
func TestFeeRateRefreshUpdateDraftPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture := newOrderPostgresFixture(t, data)
	cleanupRateRefresh(t, data, fixture.organizationID)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = data.db.SeaTransportExecution.Update().Where(seatransportexecutionent.OrganizationIDEQ(fixture.organizationID)).ClearCurrentVersionID().Save(ctx)
		_, _ = data.db.SeaTransportExecutionVersion.Delete().Where(seatransportexecutionversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
	})
	e := &rateRefreshEnv{t: t, data: data, orgID: fixture.organizationID, partnerID: fixture.partnerID, actorID: fixture.actorID, suffix: fixture.suffix}
	e.seedUSDRates()
	ctx := context.Background()
	uc := fixture.newUsecase()
	input := fixture.validInput()
	input.SeaMasterBillInput.MasterNo = "COSCORR" + strings.ToUpper(fixture.suffix[:6])
	created, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	fee := e.addFee(created.ID, "ud", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
	manual := e.addFee(created.ID, "ud-m", orderfeeent.DirectionPAYABLE, "USD", "10", "6.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10")

	save := func(version uint64, etd, eta string) (*biz.Order, error) {
		current, getErr := uc.Get(ctx, fixture.organizationID, created.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		update := fixture.validUpdateInput()
		update.SeaMasterBillInput.MasterNo = input.SeaMasterBillInput.MasterNo
		update.SeaMasterBillInput.ExpectedCandidateVersion = &current.SeaMasterBill.Version
		update.ETD, update.ETA = etd, eta
		return uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, version, update)
	}

	// 缺目标周汇率：整个保存回滚，订单日期、TE 日期与费用都不变。
	_, err = save(created.Version, rateRefreshW3, "2026-09-25T00:00:00Z")
	if !errors.Is(err, biz.ErrExchangeRateMissing) {
		t.Fatalf("应返回缺汇率错误，实际 %v", err)
	}
	if got := e.order(created.ID); got.Etd != rateRefreshW1 || got.Version != created.Version {
		t.Fatalf("订单保存应回滚: etd=%s version=%d", got.Etd, got.Version)
	}
	e.expectFee("缺汇率费用不变", fee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)

	// 只改 ETA（出口非基准日期）不触发。
	saved, err := save(created.Version, rateRefreshW1, "2026-09-30T00:00:00Z")
	if err != nil {
		t.Fatalf("改 ETA 保存失败: %v", err)
	}
	e.expectFee("改ETA不触发", fee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)

	// 改 ETD 跨周：联动，手工汇率保留。
	if _, err = save(saved.Version, rateRefreshW2, "2026-09-30T00:00:00Z"); err != nil {
		t.Fatalf("改 ETD 保存失败: %v", err)
	}
	e.expectFee("改ETD联动", fee.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
	e.expectFee("手工保留", manual.ID, "6.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10", "65", 1)
	if got := e.order(created.ID); got.Etd != rateRefreshW2 {
		t.Fatalf("订单 ETD 应保存: %s", got.Etd)
	}
}

// TestFeeRateRefreshSharedSeaTransportPostgres 验证共享海运航程更新：新日期写回各成员订单列、
// 逐票联动，任一成员缺汇率时主单/TE 日期与成员订单列均不落库。
func TestFeeRateRefreshSharedSeaTransportPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture := newOrderPostgresFixture(t, data)
	cleanupRateRefresh(t, data, fixture.organizationID)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = data.db.SeaTransportExecution.Update().Where(seatransportexecutionent.OrganizationIDEQ(fixture.organizationID)).ClearCurrentVersionID().Save(ctx)
		_, _ = data.db.SeaTransportExecutionVersion.Delete().Where(seatransportexecutionversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
	})
	e := &rateRefreshEnv{t: t, data: data, orgID: fixture.organizationID, partnerID: fixture.partnerID, actorID: fixture.actorID, suffix: fixture.suffix}
	e.seedUSDRates()
	uc := fixture.newUsecase()
	masterNo := "SHAREDRR" + strings.ToUpper(fixture.suffix[:6])
	makeInput := func() *biz.Order {
		in := fixture.validInput()
		in.SeaMasterBillInput.MasterNo = masterNo
		return in
	}
	first, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, makeInput())
	if err != nil {
		t.Fatal(err)
	}
	secondInput := makeInput()
	secondInput.SeaMasterBillInput.CandidateID = &first.SeaMasterBill.MasterBillID
	secondInput.SeaMasterBillInput.ExpectedCandidateVersion = &first.SeaMasterBill.Version
	secondInput.SeaMasterBillInput.CandidateTEID = &first.SeaMasterBill.TransportExecutionID
	secondInput.SeaMasterBillInput.ExpectedCandidateTEVersion = &first.SeaMasterBill.TransportExecutionVersion
	second, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, secondInput)
	if err != nil {
		t.Fatal(err)
	}
	if second.SeaMasterBill.TransportExecutionID != first.SeaMasterBill.TransportExecutionID {
		t.Fatal("两票应共享同一航程")
	}
	feeA := e.addFee(first.ID, "sea-a", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
	feeB := e.addFee(second.ID, "sea-b", orderfeeent.DirectionPAYABLE, "EUR", "50", "8.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")

	teID := first.SeaMasterBill.TransportExecutionID
	te, err := data.db.SeaTransportExecution.Get(ctx, teID)
	if err != nil {
		t.Fatal(err)
	}
	newETD := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
	newETA := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	command := func(key string) *biz.SeaTransportExecutionUpdateCommand {
		return &biz.SeaTransportExecutionUpdateCommand{
			OrderID: first.ID, ExpectedTransportExecutionVersion: te.Version,
			Input:  &biz.SeaTransportExecutionUpdateInput{VesselName: te.VesselName, VoyageNo: te.VoyageNo, ETD: &newETD, ETA: &newETA},
			Reason: "共享改期联动", IdempotencyKey: key + fixture.suffix,
		}
	}
	changeUsecase := biz.NewSeaOrderChangeUsecase(NewSeaOrderChangeRepo(data), data)

	// B 票 EUR 系统费用缺目标周汇率：整次失败，A 票费用、成员订单列、TE 日期都不变。
	_, err = changeUsecase.ExecuteTransportExecutionUpdate(ctx, fixture.organizationID, fixture.actorID, command("fail-"))
	if !errors.Is(err, biz.ErrExchangeRateMissing) {
		t.Fatalf("应返回缺汇率错误，实际 %v", err)
	}
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		if got := e.order(id); got.Etd != rateRefreshW1 {
			t.Fatalf("成员订单 ETD 应回滚: %s", got.Etd)
		}
	}
	teAfterFail, err := data.db.SeaTransportExecution.Get(ctx, teID)
	if err != nil || teAfterFail.Version != te.Version || !optionalTimeEquals(teAfterFail.Etd, te.Etd) {
		t.Fatalf("TE 应回滚: %+v err=%v", teAfterFail, err)
	}
	e.expectFee("A 票费用不变", feeA.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)

	// B 票费用改为手工汇率后逐票判定：A 票联动，B 票保留；两票订单列都同步到新日期。
	data.db.OrderFee.UpdateOneID(feeB.ID).SetExchangeRateSource(orderfeeent.ExchangeRateSourceMANUAL).ExecX(ctx)
	orderBefore := e.order(second.ID)
	if _, err = changeUsecase.ExecuteTransportExecutionUpdate(ctx, fixture.organizationID, fixture.actorID, command("ok-")); err != nil {
		t.Fatalf("共享改期失败: %v", err)
	}
	e.expectFee("A 票联动", feeA.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
	e.expectFee("B 票手工保留", feeB.ID, "8.0", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10", "400", 1)
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		got := e.order(id)
		if got.Etd != newETD.Format(time.RFC3339) || got.Eta != newETA.Format(time.RFC3339) {
			t.Fatalf("成员订单列应同步新日期: etd=%s eta=%s", got.Etd, got.Eta)
		}
	}
	if got := e.order(second.ID); got.Version != orderBefore.Version+1 {
		t.Fatalf("成员订单版本应+1: %d→%d", orderBefore.Version, got.Version)
	}
	if e.refreshAuditCount(first.ID) != 1 || e.refreshAuditCount(second.ID) != 0 {
		t.Fatal("仅 A 票应有联动审计")
	}
}

// TestFeeRateRefreshAirMasterPostgres 验证空运主单更新同步成员订单列并逐票联动。
func TestFeeRateRefreshAirMasterPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, data)
	cleanupRateRefresh(t, data, f.organizationID)
	e := &rateRefreshEnv{t: t, data: data, orgID: f.organizationID, partnerID: f.partnerID, actorID: f.actorID, suffix: f.suffix}
	createWeeklyRate(t, data, &e.orgID, biz.ExchangeRateSettingSourceManual, "USD", time.Date(2026, 10, 19, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation()), "6.8000", "6.7000")
	createWeeklyRate(t, data, &e.orgID, biz.ExchangeRateSettingSourceManual, "USD", time.Date(2026, 10, 26, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation()), "7.0000", "6.9000")
	ctx := context.Background()
	air := biz.NewAirExportUsecase(NewAirExportRepo(data))
	a := f.create(t, f.input("78112345675", "H-RR-1"))
	masterID := a.AirExport.Master.ID
	secondInput := f.input("78112345675", "H-RR-2")
	secondInput.AirExportInput.MasterWaybillID = &masterID
	secondInput.AirExportInput.ExpectedMasterVersion = a.AirExport.Master.Version
	b := f.create(t, secondInput)
	const oldETD = "2026-10-25T08:00:00Z"
	const newETD = "2026-10-27T08:00:00Z"
	for _, id := range []uuid.UUID{a.ID, b.ID} {
		if got := e.order(id); got.Etd != oldETD {
			t.Fatalf("建票后成员订单 ETD 应与主单一致: %s", got.Etd)
		}
	}
	feeA := e.addFee(a.ID, "air-a", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25")
	feeB := e.addFee(b.ID, "air-b", orderfeeent.DirectionPAYABLE, "EUR", "50", "8.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25")

	run := func() error {
		detail, err := air.Get(ctx, f.organizationID, masterID, biz.SelectorListOptions{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatal(err)
		}
		next := detail.Master.Content
		next.ETD, next.ETA = newETD, "2026-10-29T08:00:00Z"
		preview, err := air.Preview(ctx, f.organizationID, masterID, detail.Master.Version, &next)
		if err != nil || !preview.CanExecute {
			t.Fatalf("预览: %+v %v", preview, err)
		}
		return air.Update(ctx, f.organizationID, f.actorID, masterID, detail.Master.Version, &next, preview.InputFingerprint, "空运改期联动")
	}
	if err := run(); !errors.Is(err, biz.ErrExchangeRateMissing) {
		t.Fatalf("B 票缺汇率应整体失败，实际 %v", err)
	}
	master, err := data.db.AirMasterWaybill.Get(ctx, masterID)
	if err != nil || master.Etd != oldETD {
		t.Fatalf("主单日期应回滚: %+v %v", master, err)
	}
	for _, id := range []uuid.UUID{a.ID, b.ID} {
		if got := e.order(id); got.Etd != oldETD {
			t.Fatalf("成员订单列应回滚: %s", got.Etd)
		}
	}
	e.expectFee("A 票不变", feeA.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-25", "680", 1)

	data.db.OrderFee.UpdateOneID(feeB.ID).SetExchangeRateSource(orderfeeent.ExchangeRateSourceMANUAL).ExecX(ctx)
	if err := run(); err != nil {
		t.Fatalf("空运改期失败: %v", err)
	}
	e.expectFee("A 票联动", feeA.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-10-27", "700", 2)
	e.expectFee("B 票手工保留", feeB.ID, "8.0", orderfeeent.ExchangeRateSourceMANUAL, "2026-10-25", "400", 1)
	for _, id := range []uuid.UUID{a.ID, b.ID} {
		if got := e.order(id); got.Etd != newETD {
			t.Fatalf("成员订单列应同步: %s", got.Etd)
		}
	}
}

// TestFeeRateRefreshCommissionPaidFactsPostgres 验证联动后经济重算保留已付提成事实：
// 已付父单金额与状态不变，费用汇率变化引起的差额以系统调整记录。
func TestFeeRateRefreshCommissionPaidFactsPostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	e := &rateRefreshEnv{t: t, data: f.data, orgID: f.organizationID, partnerID: f.customerID, actorID: f.actorID, suffix: f.suffix}
	e.seedUSDRates()
	f.data.db.Order.UpdateOneID(f.orderID).SetEtd(rateRefreshW1).ExecX(ctx)

	uc := f.newUsecase(NewCommissionRepo(f.data))
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("rate-refresh"))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	parent, err = markCommissionPaidForTest(ctx, uc, f.organizationID, f.actorID, parent.ID, parent.Version)
	if err != nil {
		t.Fatal(err)
	}
	if parent.PaidAmount == nil || parent.PaidAmount.StringFixed(2) != "60.00" {
		t.Fatalf("原提成应实付 60: %+v", parent)
	}
	// 提成发放后新增的未建账系统汇率成本：10 USD，W1 汇率 6.8（本币 68）。
	cost := e.addFee(f.orderID, "commission-cost", orderfeeent.DirectionPAYABLE, "USD", "10", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
	if err := e.changeDates(f.orderID, rateRefreshW2, ""); err != nil {
		t.Fatalf("联动失败: %v", err)
	}
	e.expectFee("成本费用汇率更新", cost.ID, "6.9", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "69", 2)

	after, err := uc.Get(ctx, f.organizationID, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != biz.CommissionPaid || after.PaidAmount == nil || after.PaidAmount.StringFixed(2) != "60.00" {
		t.Fatalf("已付事实必须保持: %+v", after)
	}
	var system *biz.FinanceCommissionAdjustment
	for _, adjustment := range after.Adjustments {
		if adjustment.SourceType == biz.CommissionAdjustmentSourceFeeMutation {
			system = adjustment
		}
	}
	// 目标提成 = (1000 - 400 - 69) × 10% = 53.1，已付 60，系统冲减 6.9。
	if system == nil || system.Direction != biz.CommissionAdjustmentDecrease || system.Amount.StringFixed(2) != "6.90" {
		t.Fatalf("应生成 6.90 的系统冲减调整: %+v", after.Adjustments)
	}
}
