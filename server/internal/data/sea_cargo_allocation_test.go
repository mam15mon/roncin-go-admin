package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func TestSeaCargoAllocationDataIntegration(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaCargoAllocationRepo(data)
	cargoItemRepo := NewOrderCargoItemRepo(data)

	// 1. 基础主数据
	org, err := data.db.Organization.Create().
		SetCode("ALLOC-ORG-" + uuid.New().String()[:8]).
		SetName("箱货分配测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织失败: %v", err)
	}

	user, err := data.db.User.Create().
		SetUsername("alloc_user_" + uuid.New().String()[:8]).
		SetDisplayName("分配测试操作员").
		SetEmail("alloc@example.com").
		SetPasswordHash("dummyhash").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("PARTNER-" + uuid.New().String()[:8]).
		SetLegalName("分配测试客户").
		SetNormalizedName("分配测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户失败: %v", err)
	}

	spec, err := data.db.MasterDataItem.Create().
		SetOrganizationID(org.ID).
		SetKind(masterdataitement.KindContainerSpec).
		SetCode("40GP").
		SetName("40GP普通干货箱").
		SetSortOrder(1).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱型失败: %v", err)
	}

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetVesselName("EVER GIVEN").
		SetVoyageNo("001W").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行失败: %v", err)
	}

	shipmentFCL := orderent.ShipmentTypeFCL
	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("ORD-ALLOC-" + uuid.New().String()[:8]).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetShipmentType(shipmentFCL).
		SetFlowStatus(orderent.FlowStatusDRAFT).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单失败: %v", err)
	}

	mblNo := "MBL-ALLOC-" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(mblNo).
		SetNormalizedMasterNo(mblNo).
		SetIssuerPartnerID(partner.ID).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试 MBL 失败: %v", err)
	}

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT).
		SetCargoAllocationVersion(1).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 link 失败: %v", err)
	}

	// 2. 货物明细
	ci1, err := data.db.OrderCargoItem.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetCargoName("货物 A").
		SetPackageCount(100).
		SetGrossWeightKg(1000.000).
		SetVolumeCbm(10.000000).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建货物 A 失败: %v", err)
	}

	ci2, err := data.db.OrderCargoItem.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetCargoName("货物 B").
		SetPackageCount(200).
		SetGrossWeightKg(2000.000).
		SetVolumeCbm(20.000000).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建货物 B 失败: %v", err)
	}

	// 3. 实际箱
	cntr, err := data.db.OrderContainer.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetContainerNo("MSCU9988776").
		SetContainerSpecID(spec.ID).
		SetPackageCount(300).
		SetGrossWeightKg(3000.000).
		SetVolumeCbm(30.000000).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建集装箱失败: %v", err)
	}

	// 4. HBL
	hb, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetHouseNo("HBL-001").
		SetNormalizedHouseNo("HBL-001").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(org.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HBL 失败: %v", err)
	}

	audit := &biz.AuditEvent{
		OrganizationID: &org.ID,
		UserID:         &user.ID,
		Result:         "success",
	}

	// 5. 查询初始聚合
	initAgg, err := repo.GetSeaCargoAllocation(ctx, org.ID, order.ID)
	if err != nil {
		t.Fatalf("查询初始分配失败: %v", err)
	}
	if initAgg.AllocationStatus != biz.SeaCargoAllocationStatusDraft || initAgg.AllocationVersion != 1 {
		t.Fatalf("初始状态期望 DRAFT v1, 实际 %s v%d", initAgg.AllocationStatus, initAgg.AllocationVersion)
	}
	if initAgg.Progress.OrderRemainingPackageCount != 300 {
		t.Fatalf("初始未分配件数期望 300, 实际 %d", initAgg.Progress.OrderRemainingPackageCount)
	}

	// 6. 保存草稿（全量分配）
	allocInputs := []*biz.SeaCargoAllocationInput{
		{
			CargoItemID:   ci1.ID,
			HouseBillID:   hb.ID,
			ContainerID:   &cntr.ID,
			PackageCount:  100,
			GrossWeightKg: decimal.NewFromInt(1000),
			VolumeCbm:     decimal.NewFromInt(10),
		},
		{
			CargoItemID:   ci2.ID,
			HouseBillID:   hb.ID,
			ContainerID:   &cntr.ID,
			PackageCount:  200,
			GrossWeightKg: decimal.NewFromInt(2000),
			VolumeCbm:     decimal.NewFromInt(20),
		},
	}

	draftAgg, err := repo.SaveDraft(ctx, org.ID, user.ID, order.ID, 1, allocInputs, audit)
	if err != nil {
		t.Fatalf("保存草稿失败: %v", err)
	}
	if draftAgg.AllocationVersion != 2 {
		t.Fatalf("草稿保存后版本期望 2, 实际 %d", draftAgg.AllocationVersion)
	}
	if draftAgg.Progress.OrderRemainingPackageCount != 0 {
		t.Fatalf("全量分配后未分配件数期望 0, 实际 %d", draftAgg.Progress.OrderRemainingPackageCount)
	}

	// 7. 版本冲突校验：用旧版本 1 保存草稿应拒绝
	_, err = repo.SaveDraft(ctx, org.ID, user.ID, order.ID, 1, allocInputs, audit)
	if err != biz.ErrSeaCargoAllocationConflict {
		t.Fatalf("旧版本应返回 ErrSeaCargoAllocationConflict, 实际: %v", err)
	}

	// 8. 确认分配
	confirmedAgg, err := repo.Confirm(ctx, org.ID, user.ID, order.ID, 2, audit)
	if err != nil {
		t.Fatalf("确认分配失败: %v", err)
	}
	if confirmedAgg.AllocationStatus != biz.SeaCargoAllocationStatusConfirmed || confirmedAgg.AllocationVersion != 3 {
		t.Fatalf("确认分配后期望 CONFIRMED v3, 实际 %s v%d", confirmedAgg.AllocationStatus, confirmedAgg.AllocationVersion)
	}
	if confirmedAgg.ConfirmedBy == nil || *confirmedAgg.ConfirmedBy != user.ID {
		t.Fatalf("确认人记录不符: %v", confirmedAgg.ConfirmedBy)
	}

	// 9. CONFIRMED 状态下的写保护门禁：禁止修改货物明细
	_, err = cargoItemRepo.Add(ctx, org.ID, order.ID, &biz.OrderCargoItem{
		CargoName:     "新货物",
		PackageCount:  10,
		GrossWeightKg: 100,
		VolumeCbm:     1,
	}, audit)
	if err != biz.ErrSeaCargoAllocationStatusConflict {
		t.Fatalf("CONFIRMED 状态添加货物明细应返回 ErrSeaCargoAllocationStatusConflict, 实际: %v", err)
	}

	// 禁止在 CONFIRMED 下保存草稿
	_, err = repo.SaveDraft(ctx, org.ID, user.ID, order.ID, 3, allocInputs, audit)
	if err != biz.ErrSeaCargoAllocationStatusConflict {
		t.Fatalf("CONFIRMED 状态保存草稿应返回 ErrSeaCargoAllocationStatusConflict, 实际: %v", err)
	}

	// 10. 显式填入 HBL 汇总
	updatedHB, err := repo.ApplyHouseBillSummary(ctx, org.ID, user.ID, order.ID, hb.ID, 3, 1, audit)
	if err != nil {
		t.Fatalf("填入 HBL 汇总失败: %v", err)
	}
	if updatedHB.Content.PackageCount == nil || *updatedHB.Content.PackageCount != 300 {
		t.Fatalf("HBL 件数期望 300, 实际 %v", updatedHB.Content.PackageCount)
	}
	if updatedHB.Content.GrossWeightKg == nil || *updatedHB.Content.GrossWeightKg != 3000.000 {
		t.Fatalf("HBL 毛重期望 3000, 实际 %v", updatedHB.Content.GrossWeightKg)
	}
	if updatedHB.Content.VolumeCbm == nil || *updatedHB.Content.VolumeCbm != 30.000000 {
		t.Fatalf("HBL 体积期望 30, 实际 %v", updatedHB.Content.VolumeCbm)
	}
	if updatedHB.Version != 2 {
		t.Fatalf("HBL 版本期望递增为 2, 实际 %d", updatedHB.Version)
	}

	// 11. 撤回确认
	withdrawnAgg, err := repo.Withdraw(ctx, org.ID, user.ID, order.ID, 3, audit)
	if err != nil {
		t.Fatalf("撤回确认失败: %v", err)
	}
	if withdrawnAgg.AllocationStatus != biz.SeaCargoAllocationStatusDraft || withdrawnAgg.AllocationVersion != 4 {
		t.Fatalf("撤回后期望 DRAFT v4, 实际 %s v%d", withdrawnAgg.AllocationStatus, withdrawnAgg.AllocationVersion)
	}

	// 12. 未分配完毕时确认分配门禁拦截
	partialInputs := []*biz.SeaCargoAllocationInput{
		{
			CargoItemID:   ci1.ID,
			HouseBillID:   hb.ID,
			ContainerID:   &cntr.ID,
			PackageCount:  100,
			GrossWeightKg: decimal.NewFromInt(1000),
			VolumeCbm:     decimal.NewFromInt(10),
		},
	}
	_, err = repo.SaveDraft(ctx, org.ID, user.ID, order.ID, 4, partialInputs, audit)
	if err != nil {
		t.Fatalf("保存部分草稿失败: %v", err)
	}
	_, err = repo.Confirm(ctx, org.ID, user.ID, order.ID, 5, audit)
	if !biz.IsSeaCargoAllocationIncomplete(err) {
		t.Fatalf("未分配完毕确认应返回 ErrSeaCargoAllocationIncomplete, 实际: %v", err)
	}

	// 13. DIRECT 模式测试
	directOrder, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("ORD-DIRECT-" + uuid.New().String()[:8]).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetShipmentType(shipmentFCL).
		SetFlowStatus(orderent.FlowStatusDRAFT).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT 订单失败: %v", err)
	}

	directMblNo := "MBL-DIR-" + uuid.New().String()[:8]
	directMbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(directMblNo).
		SetNormalizedMasterNo(directMblNo).
		SetIssuerPartnerID(partner.ID).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT MBL 失败: %v", err)
	}

	_, err = data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(directOrder.ID).
		SetMasterBillID(directMbl.ID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
		SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT).
		SetCargoAllocationVersion(1).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT link 失败: %v", err)
	}

	_, err = data.db.OrderCargoItem.Create().
		SetOrganizationID(org.ID).
		SetOrderID(directOrder.ID).
		SetCargoName("DIRECT 货物").
		SetPackageCount(50).
		SetGrossWeightKg(500.000).
		SetVolumeCbm(5.000000).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT 货物失败: %v", err)
	}

	appliedMbl, err := repo.ApplyMasterBillSummary(ctx, org.ID, user.ID, directOrder.ID, 1, audit)
	if err != nil {
		t.Fatalf("填入 MBL 汇总失败: %v", err)
	}
	if appliedMbl.Content.PackageCount == nil || *appliedMbl.Content.PackageCount != 50 {
		t.Fatalf("MBL 件数期望 50, 实际 %v", appliedMbl.Content.PackageCount)
	}
	if appliedMbl.Content.GrossWeightKg == nil || *appliedMbl.Content.GrossWeightKg != 500.000 {
		t.Fatalf("MBL 毛重期望 500, 实际 %v", appliedMbl.Content.GrossWeightKg)
	}
	if appliedMbl.Version != 2 {
		t.Fatalf("MBL 版本期望 2, 实际 %d", appliedMbl.Version)
	}
	_ = link
}

func TestSeaCargoAllocationConcurrentSaveDraft(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaCargoAllocationRepo(data)

	org, err := data.db.Organization.Create().
		SetCode("RACE-ORG-" + uuid.New().String()[:8]).
		SetName("并发测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织失败: %v", err)
	}

	user, err := data.db.User.Create().
		SetUsername("race_user_" + uuid.New().String()[:8]).
		SetDisplayName("并发操作员").
		SetEmail("race@example.com").
		SetPasswordHash("dummyhash").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("PARTNER-" + uuid.New().String()[:8]).
		SetLegalName("并发测试客户").
		SetNormalizedName("并发测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建客户失败: %v", err)
	}

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetVesselName("CONCURRENT SHIP").
		SetVoyageNo("999W").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建航次失败: %v", err)
	}

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("ORD-RACE-" + uuid.New().String()[:8]).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetShipmentType(orderent.ShipmentTypeLCL).
		SetFlowStatus(orderent.FlowStatusDRAFT).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}

	mblNo := "MBL-RACE-" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(mblNo).
		SetNormalizedMasterNo(mblNo).
		SetIssuerPartnerID(partner.ID).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 MBL 失败: %v", err)
	}

	_, err = data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT).
		SetCargoAllocationVersion(1).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 link 失败: %v", err)
	}

	ci, err := data.db.OrderCargoItem.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetCargoName("拼箱货物").
		SetPackageCount(100).
		SetGrossWeightKg(1000.000).
		SetVolumeCbm(10.000000).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建货物失败: %v", err)
	}

	hb, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetHouseNo("HBL-RACE-001").
		SetNormalizedHouseNo("HBL-RACE-001").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(org.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HBL 失败: %v", err)
	}

	allocInputs := []*biz.SeaCargoAllocationInput{
		{
			CargoItemID:   ci.ID,
			HouseBillID:   hb.ID,
			PackageCount:  100,
			GrossWeightKg: decimal.NewFromInt(1000),
			VolumeCbm:     decimal.NewFromInt(10),
		},
	}

	audit := &biz.AuditEvent{
		OrganizationID: &org.ID,
		UserID:         &user.ID,
		Action:         "order.sea_cargo_allocation.race",
		Result:         "success",
	}

	// 启动两个并发请求竞争同一初始版本 1
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, saveErr := repo.SaveDraft(ctx, org.ID, user.ID, order.ID, 1, allocInputs, audit)
			errCh <- saveErr
		}()
	}

	err1 := <-errCh
	err2 := <-errCh

	successCount := 0
	conflictCount := 0

	for _, e := range []error{err1, err2} {
		if e == nil {
			successCount++
		} else if e == biz.ErrSeaCargoAllocationConflict {
			conflictCount++
		}
	}

	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("并发 SaveDraft 期望恰好 1 个成功且 1 个冲突，实际: success=%d, conflict=%d (err1=%v, err2=%v)",
			successCount, conflictCount, err1, err2)
	}
}

