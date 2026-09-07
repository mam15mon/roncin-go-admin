package data

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func ptr[T any](v T) *T {
	return &v
}

func TestSeaDocumentPostgresIntegration(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)
	orderRepo := NewOrderRepo(data)

	// 1. 创建测试总部组织与下属部门组织
	hqOrg, err := data.db.Organization.Create().
		SetCode("TEST-HQ-" + uuid.New().String()[:8]).
		SetName("测试总部").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试总部组织失败: %v", err)
	}

	deptOrg, err := data.db.Organization.Create().
		SetCode("TEST-DEPT-" + uuid.New().String()[:8]).
		SetName("测试业务部").
		SetKind("department").
		SetParentID(hqOrg.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试部门组织失败: %v", err)
	}

	// 2. 创建客户 Partner
	customerPartner, err := data.db.Partner.Create().
		SetOrganizationID(deptOrg.ID).
		SetCode("CUST-" + uuid.New().String()[:8]).
		SetLegalName("测试客户").
		SetNormalizedName("测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户失败: %v", err)
	}
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(customerPartner.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试客户角色失败: %v", err)
	}

	// 3. 创建船公司主数据
	shippingLine, err := data.db.ShippingLine.Create().
		SetOrganizationID(deptOrg.ID).
		SetScacCode("TSTL").
		SetNameZh("测试船公司").
		SetNameEn("Test Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试船公司失败: %v", err)
	}

	actorID := uuid.New()
	makeAudit := func() *biz.AuditEvent {
		return &biz.AuditEvent{
			OrganizationID: &deptOrg.ID,
			UserID:         &actorID,
			Result:         "success",
		}
	}

	// 4. 创建 DIRECT 订单与关联验证
	directOrder, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-DIR-" + uuid.New().String()[:8]).
		SetCustomerID(customerPartner.ID).
		SetShippingLineID(shippingLine.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT 订单失败: %v", err)
	}
	teDirect, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(deptOrg.ID).
		SetShippingLineID(shippingLine.ID).
		SetVesselName("EVER GIVEN").
		SetVoyageNo("001W").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT 运输执行失败: %v", err)
	}
	mblDirectNo := "TESTDIRMBL" + uuid.New().String()[:6]
	mblDirect, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetMasterNo(mblDirectNo).
		SetNormalizedMasterNo(mblDirectNo).
		SetShippingLineID(shippingLine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT MBL 失败: %v", err)
	}
	linkDirect, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(directOrder.ID).
		SetMasterBillID(mblDirect.ID).
		SetTransportExecutionID(teDirect.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 DIRECT link 失败: %v", err)
	}

	// 校验 1：DIRECT 单证查询验证（HouseBill 为空，允许操作不包含 UPDATE_HOUSE_BILL）
	docAggDirect, err := repo.GetSeaOrderDocuments(ctx, deptOrg.ID, directOrder.ID)
	if err != nil {
		t.Fatalf("GetSeaOrderDocuments direct failed: %v", err)
	}
	if docAggDirect.DocumentStructure != biz.SeaDocumentStructureDirect {
		t.Fatalf("expected DIRECT, got %s", docAggDirect.DocumentStructure)
	}
	if docAggDirect.HouseBill != nil {
		t.Fatalf("expected nil HouseBill for DIRECT, got %+v", docAggDirect.HouseBill)
	}
	if docAggDirect.LinkVersion != linkDirect.Version {
		t.Fatalf("expected link version %d, got %d", linkDirect.Version, docAggDirect.LinkVersion)
	}
	hasUpdateHB := false
	for _, action := range docAggDirect.AllowedActions {
		if action == biz.SeaDocumentActionUpdateHouseBill {
			hasUpdateHB = true
		}
	}
	if hasUpdateHB {
		t.Fatalf("DIRECT 模式下允许动作不应包含 UPDATE_HOUSE_BILL")
	}

	// 5. 创建 HOUSE 订单及唯一分单
	houseOrder, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-HSE-" + uuid.New().String()[:8]).
		SetCustomerID(customerPartner.ID).
		SetShippingLineID(shippingLine.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HOUSE 订单失败: %v", err)
	}
	teHouse, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(deptOrg.ID).
		SetShippingLineID(shippingLine.ID).
		SetVesselName("EVER SMART").
		SetVoyageNo("002E").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HOUSE 运输执行失败: %v", err)
	}
	mblHouseNo := "TESTHSEMBL" + uuid.New().String()[:6]
	mblHouse, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetMasterNo(mblHouseNo).
		SetNormalizedMasterNo(mblHouseNo).
		SetShippingLineID(shippingLine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HOUSE MBL 失败: %v", err)
	}
	linkHouse, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(houseOrder.ID).
		SetMasterBillID(mblHouse.ID).
		SetTransportExecutionID(teHouse.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HOUSE link 失败: %v", err)
	}
	_ = linkHouse

	rawHouseNo := "  COSU 000123 / 2026.B  "
	hbHouse, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(houseOrder.ID).
		SetMasterBillID(mblHouse.ID).
		SetHouseNo(rawHouseNo).
		SetNormalizedHouseNo("COSU 000123 / 2026.B").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(hqOrg.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HOUSE HBL 失败: %v", err)
	}

	// 校验 2：HOUSE 单证查询验证（单值 HBL、IssuerOrg 向上解析）
	docAggHouse, err := repo.GetSeaOrderDocuments(ctx, deptOrg.ID, houseOrder.ID)
	if err != nil {
		t.Fatalf("GetSeaOrderDocuments house failed: %v", err)
	}
	if docAggHouse.DocumentStructure != biz.SeaDocumentStructureHouse {
		t.Fatalf("expected HOUSE, got %s", docAggHouse.DocumentStructure)
	}
	if docAggHouse.HouseBill == nil {
		t.Fatal("expected non-nil HouseBill for HOUSE")
	}
	if docAggHouse.HouseBill.HouseNo != rawHouseNo {
		t.Fatalf("expected raw houseNo preserved verbatim %q, got %q", rawHouseNo, docAggHouse.HouseBill.HouseNo)
	}
	if docAggHouse.HouseBill.IssuerOrganizationID == nil || *docAggHouse.HouseBill.IssuerOrganizationID != hqOrg.ID {
		t.Fatalf("expected IssuerOrganizationID = %s (hqOrg), got %v", hqOrg.ID, docAggHouse.HouseBill.IssuerOrganizationID)
	}

	// 校验 3：条件唯一索引（idx_sea_house_bills_current_order_unique）
	// 同一订单创建第二张非 VOIDED 分单必须触发唯一性冲突
	_, err = data.db.SeaHouseBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(houseOrder.ID).
		SetMasterBillID(mblHouse.ID).
		SetHouseNo("HBL-DUPLICATE-FAIL").
		SetNormalizedHouseNo("HBL-DUPLICATE-FAIL").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(hqOrg.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err == nil {
		t.Fatal("同一订单创建第二张活动分单必须触发唯一约束冲突，实际成功")
	}

	// 校验 4：UpdateSeaHouseBill 校验、版本递增与版本冲突
	pkgCount := int32(100)
	updatedHB, err := repo.UpdateSeaHouseBill(ctx, deptOrg.ID, actorID, houseOrder.ID, hbHouse.ID, hbHouse.Version, docAggHouse.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      "COSU 000123 / 2026.B",
		IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
		Content: &biz.SeaBillContent{
			ShipperText:  ptr("  UPDATED SHIPPER  "),
			PackageCount: &pkgCount,
		},
	}, makeAudit())
	if err != nil {
		t.Fatalf("UpdateSeaHouseBill failed: %v", err)
	}
	if updatedHB.Version != hbHouse.Version+1 {
		t.Fatalf("expected HBL version %d, got %d", hbHouse.Version+1, updatedHB.Version)
	}
	if updatedHB.Content.ShipperText == nil || *updatedHB.Content.ShipperText != "UPDATED SHIPPER" {
		t.Fatalf("expected ShipperText trimmed to 'UPDATED SHIPPER', got %v", updatedHB.Content.ShipperText)
	}

	// 旧 HBL 版本更新必须触发冲突
	_, err = repo.UpdateSeaHouseBill(ctx, deptOrg.ID, actorID, houseOrder.ID, hbHouse.ID, hbHouse.Version, docAggHouse.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      "COSU 000123 / 2026.B",
		IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
	}, makeAudit())
	if err != biz.ErrSeaHouseBillConflict {
		t.Fatalf("expected ErrSeaHouseBillConflict on stale HBL version, got: %v", err)
	}

	// 旧 Link 版本更新必须触发结构冲突
	_, err = repo.UpdateSeaHouseBill(ctx, deptOrg.ID, actorID, houseOrder.ID, hbHouse.ID, updatedHB.Version, docAggHouse.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo:      "COSU 000123 / 2026.B",
		IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
	}, makeAudit())
	if err != biz.ErrSeaDocumentStructureConflict {
		t.Fatalf("expected ErrSeaDocumentStructureConflict on stale Link version, got: %v", err)
	}

	// 校验 5：UpdateSeaMasterBillContent 校验与版本冲突
	mblPkgCount := int32(50)
	gw := 1200.0
	cbm := 15.5
	updatedMbl, err := repo.UpdateSeaMasterBillContent(ctx, deptOrg.ID, actorID, houseOrder.ID, docAggHouse.MasterBill.Version, &biz.SeaBillContent{
		ShipperText:   ptr("  MBL SHIPPER  "),
		PackageCount:  &mblPkgCount,
		GrossWeightKg: &gw,
		VolumeCbm:     &cbm,
	}, makeAudit())
	if err != nil {
		t.Fatalf("UpdateSeaMasterBillContent failed: %v", err)
	}
	if updatedMbl.Version != docAggHouse.MasterBill.Version+1 {
		t.Fatalf("expected MBL version %d, got %d", docAggHouse.MasterBill.Version+1, updatedMbl.Version)
	}
	if updatedMbl.Content.ShipperText == nil || *updatedMbl.Content.ShipperText != "MBL SHIPPER" {
		t.Fatalf("expected MBL ShipperText to be trimmed to 'MBL SHIPPER', got %v", *updatedMbl.Content.ShipperText)
	}

	// 旧版本更新应触发 409 Conflict
	_, err = repo.UpdateSeaMasterBillContent(ctx, deptOrg.ID, actorID, houseOrder.ID, docAggHouse.MasterBill.Version, &biz.SeaBillContent{
		ShipperText: ptr("STALE UPDATE"),
	}, makeAudit())
	if err != biz.ErrSeaMasterBillConflict {
		t.Fatalf("expected ErrSeaMasterBillConflict on stale MBL version, got: %v", err)
	}

	// 校验 6：当存在 CUSTOMER_PARTNER 签发的分单时，修改订单客户必须被阻断
	// 先将旧 HBL 置为 VOIDED，以允许为同一个订单创建新的 CUSTOMER_PARTNER HBL
	_, err = data.db.SeaHouseBill.UpdateOneID(hbHouse.ID).SetStatus(seahousebillent.StatusVOIDED).Save(ctx)
	if err != nil {
		t.Fatalf("置废原分单失败: %v", err)
	}

	custHB, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(houseOrder.ID).
		SetMasterBillID(mblHouse.ID).
		SetHouseNo("HBL-CUST-ACTIVE").
		SetNormalizedHouseNo("HBL-CUST-ACTIVE").
		SetIssuerSource(seahousebillent.IssuerSourceCUSTOMER_PARTNER).
		SetIssuerPartnerID(customerPartner.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 CUSTOMER_PARTNER 分单失败: %v", err)
	}
	_ = custHB

	otherCustomer, err := data.db.Partner.Create().
		SetOrganizationID(deptOrg.ID).
		SetCode("CUST-OTHER-" + uuid.New().String()[:8]).
		SetLegalName("另一个测试客户").
		SetNormalizedName("另一个测试客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建另一个测试客户失败: %v", err)
	}
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(otherCustomer.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建另一个测试客户角色失败: %v", err)
	}

	// 尝试修改订单客户为 otherCustomer
	freshHouseOrder, err := data.db.Order.Get(ctx, houseOrder.ID)
	if err != nil {
		t.Fatalf("获取最新订单失败: %v", err)
	}
	_, err = orderRepo.UpdateDraft(ctx, deptOrg.ID, houseOrder.ID, freshHouseOrder.Version, &biz.Order{
		CustomerID:     otherCustomer.ID,
		ShippingLineID: &shippingLine.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		VesselVoyage:   "EVER SMART / 002E",
		SeaMasterBillInput: &biz.SeaMasterBillInput{
			MasterNo: mblHouseNo,
		},
	}, makeAudit())
	if err != biz.ErrOrderCustomerChangeWithHouseBillBlocked {
		t.Fatalf("expected ErrOrderCustomerChangeWithHouseBillBlocked when Customer HBL exists, got %v", err)
	}

	// 校验 7：同批订单查询（ListSameBatchOrders）
	otherOrg, err := data.db.Organization.Create().
		SetCode("TEST-OTHER-" + uuid.New().String()[:8]).
		SetName("其他测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建其他组织失败: %v", err)
	}

	baseOrder, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-BASE-" + uuid.New().String()[:8]).
		SetCustomerID(customerPartner.ID).
		SetCustomerReferenceNo("BATCH-REF-1").
		SetBookingNo("BKG-BATCH-1").
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 baseOrder 失败: %v", err)
	}
	_, err = data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(baseOrder.ID).
		SetMasterBillID(mblHouse.ID).
		SetTransportExecutionID(teHouse.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 baseOrder link 失败: %v", err)
	}

	// Order 2: 命中 CUSTOMER_REFERENCE
	orderCustRef, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-REF-" + uuid.New().String()[:8]).
		SetCustomerID(customerPartner.ID).
		SetCustomerReferenceNo("BATCH-REF-1").
		SetBookingNo("BKG-OTHER-99").
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 orderCustRef 失败: %v", err)
	}

	// Order 3: 命中 BOOKING
	orderBooking, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-BKG-" + uuid.New().String()[:8]).
		SetCustomerID(otherCustomer.ID).
		SetBookingNo("BKG-BATCH-1").
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 orderBooking 失败: %v", err)
	}

	// Order 4: 命中 MASTER
	orderMBL, err := data.db.Order.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderNo("SE-MBL-" + uuid.New().String()[:8]).
		SetCustomerID(otherCustomer.ID).
		SetBookingNo("BKG-OTHER-88").
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 orderMBL 失败: %v", err)
	}
	_, err = data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(deptOrg.ID).
		SetOrderID(orderMBL.ID).
		SetMasterBillID(mblHouse.ID).
		SetTransportExecutionID(teHouse.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 orderMBL link 失败: %v", err)
	}

	// Order 5: 属于其他组织，即便 bookingNo 相同也不应出现在当前组织的结果中
	otherOrgCust, err := data.db.Partner.Create().
		SetOrganizationID(otherOrg.ID).
		SetCode("CUST-OTHERORG-" + uuid.New().String()[:8]).
		SetLegalName("其他组织客户").
		SetNormalizedName("其他组织客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建其他组织客户失败: %v", err)
	}
	_, err = data.db.Order.Create().
		SetOrganizationID(otherOrg.ID).
		SetOrderNo("SE-OTHERORG-" + uuid.New().String()[:8]).
		SetCustomerID(otherOrgCust.ID).
		SetBookingNo("BKG-BATCH-1").
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 otherOrgOrder 失败: %v", err)
	}

	sameBatchList, err := orderRepo.ListSameBatchOrders(ctx, deptOrg.ID, baseOrder.ID)
	if err != nil {
		t.Fatalf("ListSameBatchOrders failed: %v", err)
	}
	if len(sameBatchList) < 3 {
		t.Fatalf("expected at least 3 same batch orders, got %d", len(sameBatchList))
	}
	foundRef, foundBkg, foundMbl := false, false, false
	for _, item := range sameBatchList {
		if item.OrderID == baseOrder.ID {
			t.Fatal("ListSameBatchOrders 不应包含当前订单自身")
		}
		if item.OrderID == orderCustRef.ID {
			foundRef = true
			matched := false
			for _, s := range item.MatchSources {
				if s == "CUSTOMER_REFERENCE" {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("expected match sources to contain CUSTOMER_REFERENCE, got %v", item.MatchSources)
			}
		}
		if item.OrderID == orderBooking.ID {
			foundBkg = true
			matched := false
			for _, s := range item.MatchSources {
				if s == "BOOKING" {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("expected match sources to contain BOOKING, got %v", item.MatchSources)
			}
		}
		if item.OrderID == orderMBL.ID {
			foundMbl = true
			matched := false
			for _, s := range item.MatchSources {
				if s == "MASTER" {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("expected match sources to contain MASTER, got %v", item.MatchSources)
			}
		}
	}
	if !foundRef || !foundBkg || !foundMbl {
		t.Fatalf("同批订单未能全部匹配: foundRef=%v, foundBkg=%v, foundMbl=%v", foundRef, foundBkg, foundMbl)
	}
}

func TestSeaDocument_AuditEnforcementAndRollback(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)

	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()

	t.Run("nil audit is rejected", func(t *testing.T) {
		_, err := repo.UpdateSeaMasterBillContent(ctx, orgID, actorID, orderID, 1, &biz.SeaBillContent{ShipperText: ptr("test")}, nil)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for nil audit, got %v", err)
		}
		_, err = repo.UpdateSeaHouseBill(ctx, orgID, actorID, orderID, uuid.New(), 1, 1, &biz.SeaHouseBillInput{HouseNo: "HBL-1", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization}, nil)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for nil audit, got %v", err)
		}
	})

	t.Run("mismatched audit is rejected", func(t *testing.T) {
		diffOrg := uuid.New()
		mismatchedAudit := &biz.AuditEvent{
			OrganizationID: &diffOrg,
			UserID:         &actorID,
			Result:         "success",
		}
		_, err := repo.UpdateSeaMasterBillContent(ctx, orgID, actorID, orderID, 1, &biz.SeaBillContent{ShipperText: ptr("test")}, mismatchedAudit)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for mismatched audit, got %v", err)
		}
		_, err = repo.UpdateSeaHouseBill(ctx, orgID, actorID, orderID, uuid.New(), 1, 1, &biz.SeaHouseBillInput{HouseNo: "HBL-1", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization}, mismatchedAudit)
		if err != biz.ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for mismatched audit, got %v", err)
		}
	})
}

func TestSeaDocument_ConcurrentOperationsNoDeadlock(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaDocumentRepo(data)

	org, err := data.db.Organization.Create().
		SetCode("TEST-CONC-" + uuid.New().String()[:8]).
		SetName("并发测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org failed: %v", err)
	}

	cust, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-CONC-" + uuid.New().String()[:8]).
		SetLegalName("并发客户").
		SetNormalizedName("并发客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("create customer failed: %v", err)
	}
	if _, err := data.db.PartnerRole.Create().SetPartnerID(cust.ID).SetRoleType(partnerroleent.RoleTypeCustomer).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("create customer role failed: %v", err)
	}

	shippingLine, err := data.db.ShippingLine.Create().
		SetOrganizationID(org.ID).
		SetScacCode("CNCL").
		SetNameZh("并发船公司").
		SetNameEn("Concurrent Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create shipping line failed: %v", err)
	}

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-CONC-" + uuid.New().String()[:8]).
		SetCustomerID(cust.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(shippingLine.ID).
		SetVesselName("CONC SHIP").
		SetVoyageNo("888").
		Save(ctx)
	if err != nil {
		t.Fatalf("create te failed: %v", err)
	}

	masterNo := "CONCMBL" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetMasterNo(masterNo).
		SetNormalizedMasterNo(masterNo).
		SetShippingLineID(shippingLine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create mbl failed: %v", err)
	}

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetTransportExecutionID(te.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		Save(ctx)
	if err != nil {
		t.Fatalf("create link failed: %v", err)
	}

	hb, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetHouseNo("CONC-HBL-001").
		SetNormalizedHouseNo("CONC-HBL-001").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(org.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("create HBL failed: %v", err)
	}

	actorID := uuid.New()

	// 并发执行多次更新分单（严格锁序保证无死锁，且仅能有一个成功更新）
	var wg sync.WaitGroup
	workers := 5
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			audit := &biz.AuditEvent{
				OrganizationID: &org.ID,
				UserID:         &actorID,
				Result:         "success",
			}
			shipper := fmt.Sprintf("SHIPPER-%d", idx)
			_, updateErr := repo.UpdateSeaHouseBill(ctx, org.ID, actorID, order.ID, hb.ID, hb.Version, link.Version, &biz.SeaHouseBillInput{
				HouseNo:      "CONC-HBL-001",
				IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
				Content:      &biz.SeaBillContent{ShipperText: &shipper},
			}, audit)

			mu.Lock()
			defer mu.Unlock()
			if updateErr == nil {
				successCount++
			} else if updateErr == biz.ErrSeaHouseBillConflict || updateErr == biz.ErrSeaDocumentStructureConflict {
				conflictCount++
			}
		}(i)
	}
	wg.Wait()

	if successCount != 1 || conflictCount != workers-1 {
		t.Fatalf("并发更新分单期望 1 个成功且 %d 个冲突，实际: success=%d, conflict=%d", workers-1, successCount, conflictCount)
	}
}

func TestSeaDocument_UpdateOrderValidation(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	org, err := data.db.Organization.Create().
		SetCode("TEST-UO-" + uuid.New().String()[:8]).
		SetName("UpdateOrder测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org failed: %v", err)
	}

	cust, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-UO-" + uuid.New().String()[:8]).
		SetLegalName("UO客户").
		SetNormalizedName("UO客户").
		Save(ctx)
	if err != nil {
		t.Fatalf("create customer failed: %v", err)
	}
	if _, err := data.db.PartnerRole.Create().SetPartnerID(cust.ID).SetRoleType(partnerroleent.RoleTypeCustomer).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("create customer role failed: %v", err)
	}

	shippingLine, err := data.db.ShippingLine.Create().
		SetOrganizationID(org.ID).
		SetScacCode("UOTL").
		SetNameZh("UO船公司").
		SetNameEn("Update Order Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create shipping line failed: %v", err)
	}

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-UO-" + uuid.New().String()[:8]).
		SetCustomerID(cust.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		Save(ctx)
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(shippingLine.ID).
		SetVesselName("UO SHIP").
		SetVoyageNo("101").
		Save(ctx)
	if err != nil {
		t.Fatalf("create te failed: %v", err)
	}

	masterNo := "UOMBL" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetMasterNo(masterNo).
		SetNormalizedMasterNo(masterNo).
		SetShippingLineID(shippingLine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create mbl failed: %v", err)
	}

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetTransportExecutionID(te.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
		Save(ctx)
	if err != nil {
		t.Fatalf("create link failed: %v", err)
	}

	orderRepo := NewOrderRepo(data)
	actorID := uuid.New()
	makeAudit := func() *biz.AuditEvent {
		return &biz.AuditEvent{
			OrganizationID: &org.ID,
			UserID:         &actorID,
			Result:         "success",
		}
	}

	strDirect := biz.SeaDocumentStructureDirect

	// 1. UpdateDraft 缺少 expected_link_version 应被拒绝
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			DocumentStructure: &strDirect,
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for missing expected_link_version in UpdateDraft, got nil")
	}

	// 2. UpdateDraft 缺少 expected_mbl_version 应被拒绝
	s := "NEW SHIPPER"
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			MasterBillContent: &biz.SeaBillContent{ShipperText: &s},
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for missing expected_mbl_version in UpdateDraft, got nil")
	}

	// 3. UpdateDraft 携带 HouseBill 应被拒绝（必须走专用命令）
	_, err = orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			HouseBill: &biz.SeaHouseBillInput{
				HouseNo: "HBL1", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
			},
		},
	}, makeAudit())
	if err == nil {
		t.Fatalf("expected error for HouseBill in UpdateDraft, got nil")
	}

	// 4. UpdateDraft 正确携带版本号成功更新
	ver := uint64(1)
	updatedOrder, err := orderRepo.UpdateDraft(ctx, org.ID, order.ID, order.Version, &biz.Order{
		CustomerID:     cust.ID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: "export",
		TradeTerm:      "FOB",
		PaymentTerm:    "PREPAID",
		SeaDocumentInput: &biz.SeaOrderDocumentInput{
			DocumentStructure:   &strDirect,
			ExpectedLinkVersion: &ver,
			MasterBillContent:   &biz.SeaBillContent{ShipperText: &s},
			ExpectedMblVersion:  &ver,
		},
	}, makeAudit())
	if err != nil {
		t.Fatalf("UpdateDraft with valid versions failed: %v", err)
	}
	if updatedOrder == nil {
		t.Fatalf("expected non-nil updated order")
	}
	_ = link
}
