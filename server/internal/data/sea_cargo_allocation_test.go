package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// sharedContainerFixture 构造同一运输执行下两张 HOUSE 订单与共享箱所需的最小数据
type sharedContainerFixture struct {
	data         *Data
	orgID        uuid.UUID
	userID       uuid.UUID
	partnerID    uuid.UUID
	specID       uuid.UUID
	teID         uuid.UUID
	mblID        uuid.UUID
	order1       *ent.Order
	order2       *ent.Order
	link1        *ent.SeaMasterBillOrderLink
	link2        *ent.SeaMasterBillOrderLink
	hbl1         *ent.SeaHouseBill
	hbl2         *ent.SeaHouseBill
	cargo1ID     uuid.UUID
	cargo2ID     uuid.UUID
	uc           *biz.SeaSharedContainerUsecase
	containerIDs []uuid.UUID
}

func newSharedContainerFixture(t *testing.T) *sharedContainerFixture {
	t.Helper()
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	org, err := data.db.Organization.Create().
		SetCode("SHARED-" + uuid.New().String()[:8]).
		SetName("共享箱集成测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织失败: %v", err)
	}
	user, err := data.db.User.Create().
		SetUsername("shared_user_" + uuid.New().String()[:8]).
		SetDisplayName("共享箱测试操作员").
		SetEmail("shared@example.com").
		SetPasswordHash("dummyhash").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("SHARED-P-" + uuid.New().String()[:8]).
		SetLegalName("共享箱测试客户").
		SetNormalizedName("共享箱测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户失败: %v", err)
	}
	line, err := data.db.ShippingLine.Create().
		SetOrganizationID(org.ID).
		SetScacCode("SHRD").
		SetNameZh("共享箱测试船公司").
		SetNameEn("Shared Test Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建船公司失败: %v", err)
	}
	spec, err := data.db.MasterDataItem.Create().
		SetOrganizationID(org.ID).
		SetKind(masterdataitement.KindContainerSpec).
		SetCode("40HC").
		SetName("40HC高箱").
		SetSortOrder(1).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱型失败: %v", err)
	}
	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(line.ID).
		SetVesselName("SHARED VESSEL").
		SetVoyageNo("001W").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行失败: %v", err)
	}
	mblNo := "SHRD" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(line.ID).
		SetMasterNo(mblNo).
		SetNormalizedMasterNo(mblNo).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 MBL 失败: %v", err)
	}

	f := &sharedContainerFixture{
		data: data, orgID: org.ID, userID: user.ID, partnerID: partner.ID,
		specID: spec.ID, teID: te.ID, mblID: mbl.ID,
		uc: biz.NewSeaSharedContainerUsecase(NewSeaSharedContainerRepo(data)),
	}
	for i := 1; i <= 2; i++ {
		order, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo("SHRD-ORD-" + uuid.New().String()[:8]).
			SetCustomerID(partner.ID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetShipmentType(orderent.ShipmentTypeFCL).
			SetFlowStatus(orderent.FlowStatusDRAFT).
			SetTerminationStatus(orderent.TerminationStatusACTIVE).
			SetClosureStatus(orderent.ClosureStatusOPEN).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建订单 %d 失败: %v", i, err)
		}
		link, err := data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(org.ID).
			SetOrderID(order.ID).
			SetMasterBillID(mbl.ID).
			SetTransportExecutionID(te.ID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 Link %d 失败: %v", i, err)
		}
		hblNo := "SHBL-" + uuid.New().String()[:8]
		hbl, err := data.db.SeaHouseBill.Create().
			SetOrganizationID(org.ID).
			SetOrderID(order.ID).
			SetMasterBillID(mbl.ID).
			SetHouseNo(hblNo).
			SetNormalizedHouseNo(hblNo).
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(org.ID).
			SetStatus(seahousebillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 HBL %d 失败: %v", i, err)
		}
		cargo, err := data.db.OrderCargoItem.Create().
			SetOrganizationID(org.ID).
			SetOrderID(order.ID).
			SetCargoName("共享箱货物").
			SetPackageCount(100).
			SetGrossWeightKg(1000.0).
			SetVolumeCbm(10.0).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建货物 %d 失败: %v", i, err)
		}
		if i == 1 {
			f.order1, f.link1, f.hbl1, f.cargo1ID = order, link, hbl, cargo.ID
		} else {
			f.order2, f.link2, f.hbl2, f.cargo2ID = order, link, hbl, cargo.ID
		}
	}
	return f
}

func (f *sharedContainerFixture) createContainer(t *testing.T, suffix string) *biz.SeaSharedContainer {
	t.Helper()
	created, err := f.uc.Create(context.Background(), f.orgID, f.userID, &biz.SeaSharedContainer{
		TransportExecutionID: f.teID,
		ContainerNo:          "SHCU" + suffix + uuid.New().String()[:6],
		ContainerSpecID:      f.specID,
		PackageCount:         200,
		GrossWeightKg:        decimal.NewFromInt(2000),
		VolumeCbm:            decimal.NewFromInt(20),
	})
	if err != nil {
		t.Fatalf("创建共享箱失败: %v", err)
	}
	f.containerIDs = append(f.containerIDs, created.ID)
	return created
}

func (f *sharedContainerFixture) fullAllocationInputs() []*biz.SeaSharedContainerAllocationInput {
	return []*biz.SeaSharedContainerAllocationInput{
		{OrderID: f.order1.ID, HouseBillID: f.hbl1.ID, CargoItemID: f.cargo1ID, PackageCount: 100, GrossWeightKg: decimal.NewFromInt(1000), VolumeCbm: decimal.NewFromInt(10), ExpectedOrderVersion: 1, ExpectedLinkVersion: 1, ExpectedHouseBillVersion: 1, ExpectedCargoItemVersion: 1},
		{OrderID: f.order2.ID, HouseBillID: f.hbl2.ID, CargoItemID: f.cargo2ID, PackageCount: 100, GrossWeightKg: decimal.NewFromInt(1000), VolumeCbm: decimal.NewFromInt(10), ExpectedOrderVersion: 1, ExpectedLinkVersion: 1, ExpectedHouseBillVersion: 1, ExpectedCargoItemVersion: 1},
	}
}

func TestSeaSharedContainerDataIntegration(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx := context.Background()

	// 1. 创建共享箱并读取初始状态
	container := f.createContainer(t, "01")
	if container.Status != biz.SeaSharedContainerStatusDraft || container.Version != 1 {
		t.Fatalf("初始状态期望 DRAFT v1, 实际 %s v%d", container.Status, container.Version)
	}

	// 2. 同一执行下重复箱号被唯一索引兜底
	_, err := f.uc.Create(ctx, f.orgID, f.userID, &biz.SeaSharedContainer{
		TransportExecutionID: f.teID,
		ContainerNo:          container.ContainerNo,
		ContainerSpecID:      f.specID,
		PackageCount:         10,
		GrossWeightKg:        decimal.NewFromInt(100),
		VolumeCbm:            decimal.NewFromInt(1),
	})
	if err != biz.ErrSeaSharedContainerExists {
		t.Fatalf("重复箱号应返回 ErrSeaSharedContainerExists, 实际: %v", err)
	}

	// 3. 候选订单：仅同执行 HOUSE 订单可见，支持 keyword 过滤
	candidates, total, err := f.uc.ListCandidates(ctx, f.orgID, f.teID, "", 1, 50)
	if err != nil || total != 2 || len(candidates) != 2 {
		t.Fatalf("候选订单期望 2 条, got total=%d len=%d err=%v", total, len(candidates), err)
	}
	keywordHit, keywordTotal, err := f.uc.ListCandidates(ctx, f.orgID, f.teID, f.hbl1.HouseNo, 1, 50)
	if err != nil || keywordTotal != 1 || len(keywordHit) != 1 || keywordHit[0].HouseBillID != f.hbl1.ID {
		t.Fatalf("候选订单 keyword 过滤异常: total=%d hit=%+v err=%v", keywordTotal, keywordHit, err)
	}

	// 4. 保存草稿：跨订单完整分配
	draft, err := f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, container.Version, f.fullAllocationInputs())
	if err != nil {
		t.Fatalf("保存草稿失败: %v", err)
	}
	if draft.Version != 2 || len(draft.Allocations) != 2 {
		t.Fatalf("草稿保存后期望 v2 两行分配, 实际 v%d %d 行", draft.Version, len(draft.Allocations))
	}
	if draft.Progress == nil || !draft.Progress.CargoBalanced || !draft.Progress.ContainerBalanced {
		t.Fatalf("完整分配后守恒进度异常: %+v", draft.Progress)
	}

	// 5. 旧版本保存草稿必须冲突
	_, err = f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, 1, f.fullAllocationInputs())
	if err != biz.ErrSeaSharedContainerConflict {
		t.Fatalf("旧版本保存应返回 ErrSeaSharedContainerConflict, 实际: %v", err)
	}

	// 6. 超分：单票分配超出货物或箱总量时拒绝
	overInputs := []*biz.SeaSharedContainerAllocationInput{
		{OrderID: f.order1.ID, HouseBillID: f.hbl1.ID, CargoItemID: f.cargo1ID, PackageCount: 150, GrossWeightKg: decimal.NewFromInt(1000), VolumeCbm: decimal.NewFromInt(10), ExpectedOrderVersion: 1, ExpectedLinkVersion: 1, ExpectedHouseBillVersion: 1, ExpectedCargoItemVersion: 1},
	}
	if _, err = f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, draft.Version, overInputs); err != biz.ErrSeaSharedContainerExceeded {
		t.Fatalf("货物超分应返回 ErrSeaSharedContainerExceeded, 实际: %v", err)
	}

	// 7. 未完整分配时草稿允许、确认拒绝
	partial := []*biz.SeaSharedContainerAllocationInput{f.fullAllocationInputs()[0]}
	partialDraft, err := f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, draft.Version, partial)
	if err != nil {
		t.Fatalf("部分草稿保存失败: %v", err)
	}
	if _, err = f.uc.Confirm(ctx, f.orgID, f.userID, container.ID, partialDraft.Version, nil); err != biz.ErrSeaSharedContainerIncomplete {
		t.Fatalf("未完整分配确认应返回 ErrSeaSharedContainerIncomplete, 实际: %v", err)
	}

	// 8. 恢复完整分配并确认：携带 allocations 输入在单事务内保存并严格确认
	restored, err := f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, partialDraft.Version, partial)
	if err != nil {
		t.Fatalf("恢复部分分配失败: %v", err)
	}
	confirmed, err := f.uc.Confirm(ctx, f.orgID, f.userID, container.ID, restored.Version, f.fullAllocationInputs())
	if err != nil {
		t.Fatalf("确认共享箱失败: %v", err)
	}
	if confirmed.Status != biz.SeaSharedContainerStatusConfirmed || confirmed.ConfirmedBy == nil || *confirmed.ConfirmedBy != f.userID {
		t.Fatalf("确认后状态或确认人异常: %+v", confirmed)
	}
	if len(confirmed.Allocations) != 2 || confirmed.Version != restored.Version+1 {
		t.Fatalf("单事务确认应写入两行分配并递增一次版本: allocs=%d version=%d", len(confirmed.Allocations), confirmed.Version)
	}

	// 9. 确认态禁止保存草稿；撤回后回到草稿态
	if _, err = f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, confirmed.Version, f.fullAllocationInputs()); err != biz.ErrSeaSharedContainerStatusConflict {
		t.Fatalf("确认态保存草稿应返回 ErrSeaSharedContainerStatusConflict, 实际: %v", err)
	}
	withdrawn, err := f.uc.Withdraw(ctx, f.orgID, f.userID, container.ID, confirmed.Version)
	if err != nil || withdrawn.Status != biz.SeaSharedContainerStatusDraft {
		t.Fatalf("撤回确认失败: status=%s err=%v", withdrawn.Status, err)
	}

	// 10. 活动 Link 指向其他运输执行的订单不得加入共享箱
	// 先结束 order2 当前活动 Link，再挂到其他执行上，避免违反一订单一活动 Link 约束
	if err := f.data.db.SeaMasterBillOrderLink.DeleteOneID(f.link2.ID).Exec(ctx); err != nil {
		t.Fatalf("移除 order2 原活动 Link 失败: %v", err)
	}
	otherTE, err := f.data.db.SeaTransportExecution.Create().
		SetOrganizationID(f.orgID).
		SetShippingLineID(f.mblShippingLineID(t)).
		SetVesselName("OTHER VESSEL").
		SetVoyageNo("002E").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建其他运输执行失败: %v", err)
	}
	otherLink, err := f.data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(f.orgID).
		SetOrderID(f.order2.ID).
		SetMasterBillID(f.mblID).
		SetTransportExecutionID(otherTE.ID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(2).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建其他执行 Link 失败: %v", err)
	}
	t.Cleanup(func() {
		_ = f.data.db.SeaMasterBillOrderLink.DeleteOneID(otherLink.ID).Exec(ctx)
	})
	crossInputs := []*biz.SeaSharedContainerAllocationInput{
		{OrderID: f.order2.ID, HouseBillID: f.hbl2.ID, CargoItemID: f.cargo2ID, PackageCount: 100, GrossWeightKg: decimal.NewFromInt(1000), VolumeCbm: decimal.NewFromInt(10), ExpectedOrderVersion: f.order2.Version, ExpectedLinkVersion: otherLink.Version, ExpectedHouseBillVersion: 1, ExpectedCargoItemVersion: 1},
	}
	if _, err = f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, withdrawn.Version, crossInputs); err != biz.ErrSeaSharedContainerInvalidReference {
		t.Fatalf("活动 Link 指向其他运输执行的订单不得加入共享箱, 实际: %v", err)
	}

	// 11. 有分配的共享箱禁止删除；清空分配后可删除
	if err = f.uc.Delete(ctx, f.orgID, f.userID, container.ID, withdrawn.Version); err != biz.ErrSeaSharedContainerStatusConflict {
		t.Fatalf("存在分配时删除应返回 ErrSeaSharedContainerStatusConflict, 实际: %v", err)
	}
	emptied, err := f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, withdrawn.Version, nil)
	if err != nil {
		t.Fatalf("清空分配失败: %v", err)
	}
	if err = f.uc.Delete(ctx, f.orgID, f.userID, container.ID, emptied.Version); err != nil {
		t.Fatalf("清空后删除共享箱失败: %v", err)
	}
}

func (f *sharedContainerFixture) mblShippingLineID(t *testing.T) uuid.UUID {
	t.Helper()
	mbl, err := f.data.db.SeaMasterBill.Get(context.Background(), f.mblID)
	if err != nil {
		t.Fatalf("读取 MBL 失败: %v", err)
	}
	return mbl.ShippingLineID
}

func TestSeaSharedContainerConcurrentSaveDraft(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx := context.Background()
	container := f.createContainer(t, "02")

	inputs := f.fullAllocationInputs()
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, saveErr := f.uc.SaveDraft(ctx, f.orgID, f.userID, container.ID, container.Version, inputs)
			errCh <- saveErr
		}()
	}
	wg.Wait()
	close(errCh)

	successCount, conflictCount := 0, 0
	for saveErr := range errCh {
		if saveErr == nil {
			successCount++
		} else if saveErr == biz.ErrSeaSharedContainerConflict {
			conflictCount++
		} else {
			t.Errorf("并发 SaveDraft 返回非预期错误: %v", saveErr)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("并发 SaveDraft 期望恰好 1 个成功且 1 个冲突，实际: success=%d conflict=%d", successCount, conflictCount)
	}

	// 竞争后共享箱内只保留一份完整分配，且版本只推进一次
	final, err := f.uc.Get(ctx, f.orgID, container.ID)
	if err != nil {
		t.Fatalf("读取最终共享箱失败: %v", err)
	}
	if len(final.Allocations) != 2 || final.Version != container.Version+1 {
		t.Fatalf("竞争后分配或版本异常: allocations=%d version=%d", len(final.Allocations), final.Version)
	}
	if final.UpdatedAt.IsZero() || time.Now().UTC().Before(final.UpdatedAt.Add(-time.Minute)) {
		t.Fatalf("共享箱更新时间异常: %v", final.UpdatedAt)
	}
}
