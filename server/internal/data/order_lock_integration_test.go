package data

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	dingtalkapprovalinboxeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovalinboxevent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
)

func TestOrderLock_PostgresFlows(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		source = "postgresql://roncin:roncin_local_dev@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable"
		t.Setenv("RONCIN_INTEGRATION_DATABASE_SOURCE", source)
	}
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]

	// 1. 创建测试组织
	org, err := data.db.Organization.Create().
		SetCode("ORG-" + suffix).
		SetName("测试海运组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织失败: %v", err)
	}

	// 2. 确保权限记录存在
	lockPerm, err := data.db.Permission.Query().Where(permissionent.KeyEQ("business.order.se.lock")).First(ctx)
	if err != nil || lockPerm == nil {
		lockPerm, err = data.db.Permission.Create().
			SetKey("business.order.se.lock").
			SetName("海运出口订单锁定").
			SetDescription("锁定海运出口订单与单证").
			SetGroup("order").
			Save(ctx)
		if err != nil {
			t.Fatalf("创建锁定权限记录失败: %v", err)
		}
	}

	updatePerm, err := data.db.Permission.Query().Where(permissionent.KeyEQ("business.order.se.update")).First(ctx)
	if err != nil || updatePerm == nil {
		updatePerm, err = data.db.Permission.Create().
			SetKey("business.order.se.update").
			SetName("海运出口订单修改").
			SetDescription("修改海运出口订单").
			SetGroup("order").
			Save(ctx)
		if err != nil {
			t.Fatalf("创建修改权限记录失败: %v", err)
		}
	}

	// 3. 创建业务锁定角色 (data_scope: organization, permissions: lock, update)
	seLockRole, err := data.db.Role.Create().
		SetOrganizationID(org.ID).
		SetCode("se_locker_"+suffix).
		SetName("海运出口锁定专员-"+suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(lockPerm, updatePerm).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建业务锁定角色失败: %v", err)
	}

	// 4. 创建普通业务角色 (无 lock 权限)
	normalRole, err := data.db.Role.Create().
		SetOrganizationID(org.ID).
		SetCode("se_operator_" + suffix).
		SetName("普通海运操作员-" + suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(updatePerm).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建普通角色失败: %v", err)
	}

	// 5. 创建用户与成员关系
	dtRoleUserID := "dt-role-" + suffix
	roleUser, err := data.db.User.Create().
		SetDisplayName("业务锁定员-" + suffix).
		SetDingtalkUserid(dtRoleUserID).
		SetEnabled(true).
		SetIsBootstrapAdmin(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建角色用户失败: %v", err)
	}
	roleMembership, err := data.db.Membership.Create().
		SetUserID(roleUser.ID).
		SetOrganizationID(org.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建角色用户组织关系失败: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().
		SetMembershipID(roleMembership.ID).
		SetRoleID(seLockRole.ID).
		Save(ctx); err != nil {
		t.Fatalf("分配锁定角色失败: %v", err)
	}

	// 普通用户
	dtNormalUserID := "dt-normal-" + suffix
	normalUser, err := data.db.User.Create().
		SetDisplayName("普通业务员-" + suffix).
		SetDingtalkUserid(dtNormalUserID).
		SetEnabled(true).
		SetIsBootstrapAdmin(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建普通用户失败: %v", err)
	}
	normalMembership, err := data.db.Membership.Create().
		SetUserID(normalUser.ID).
		SetOrganizationID(org.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建普通用户组织关系失败: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().
		SetMembershipID(normalMembership.ID).
		SetRoleID(normalRole.ID).
		Save(ctx); err != nil {
		t.Fatalf("分配普通角色失败: %v", err)
	}

	// 超管用户 (bootstrap admin)
	adminUser, err := data.db.User.Create().
		SetDisplayName("系统管理员-" + suffix).
		SetEnabled(true).
		SetIsBootstrapAdmin(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建超管用户失败: %v", err)
	}

	t.Cleanup(func() {
		_, _ = data.sqlDB.ExecContext(ctx, `
			DELETE FROM ding_talk_approval_inbox_events WHERE organization_id = $1;
			DELETE FROM order_unlock_approver_candidates WHERE request_id IN (SELECT id FROM order_unlock_requests WHERE organization_id = $1);
			DELETE FROM ding_talk_approval_dispatches WHERE organization_id = $1;
			DELETE FROM background_tasks WHERE organization_id = $1;
			DELETE FROM order_lock_house_bill_snapshots WHERE organization_id = $1;
			DELETE FROM order_unlock_requests WHERE organization_id = $1;
			DELETE FROM order_lock_records WHERE organization_id = $1;
			UPDATE sea_master_bills SET current_version_id = NULL WHERE organization_id = $1;
			UPDATE sea_house_bills SET current_version_id = NULL WHERE organization_id = $1;
			DELETE FROM sea_house_bill_versions WHERE organization_id = $1;
			DELETE FROM sea_master_bill_versions WHERE organization_id = $1;
			DELETE FROM sea_house_bills WHERE organization_id = $1;
			DELETE FROM sea_master_bill_order_links WHERE organization_id = $1;
			DELETE FROM sea_master_bills WHERE organization_id = $1;
			DELETE FROM sea_transport_executions WHERE organization_id = $1;
			DELETE FROM orders WHERE organization_id = $1;
			DELETE FROM role_assignments WHERE membership_id IN (SELECT id FROM memberships WHERE organization_id = $1);
			DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE organization_id = $1);
			DELETE FROM roles WHERE organization_id = $1;
			DELETE FROM memberships WHERE organization_id = $1;
			DELETE FROM partner_roles WHERE partner_id IN (SELECT id FROM partners WHERE organization_id = $1);
			DELETE FROM partners WHERE organization_id = $1;
			DELETE FROM users WHERE id IN ($2, $3, $4);
			DELETE FROM organizations WHERE id = $1;
		`, org.ID, roleUser.ID, normalUser.ID, adminUser.ID)
	})

	// 6. 创建客户与船东往来单位
	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-" + suffix).
		SetLegalName("测试发货人客户-" + suffix).
		SetNormalizedName("测试发货人客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建客户失败: %v", err)
	}
	if _, err = data.db.PartnerRole.Create().
		SetPartnerID(customer.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建客户角色失败: %v", err)
	}

	carrier, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CARR-" + suffix).
		SetLegalName("马士基航运-" + suffix).
		SetNormalizedName("马士基航运-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建船东失败: %v", err)
	}
	if _, err = data.db.PartnerRole.Create().
		SetPartnerID(carrier.ID).
		SetRoleType(partnerroleent.RoleTypeCarrier).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建船东角色失败: %v", err)
	}

	// 7. 创建海运运输执行与主单 MBL
	etd := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	eta := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	routeCarrierID := carrier.ID
	routeOriginID := uuid.New()
	routeDischargeID := uuid.New()
	routeTransitID := uuid.New()
	for _, routePort := range []struct {
		id       uuid.UUID
		unlocode string
		name     string
	}{
		{id: routeOriginID, unlocode: "CNORG", name: "起运港"},
		{id: routeDischargeID, unlocode: "CNDIS", name: "卸货港"},
		{id: routeTransitID, unlocode: "CNTRA", name: "中转港"},
	} {
		if _, err := data.db.Port.Create().
			SetID(routePort.id).
			SetOrganizationID(org.ID).
			SetUnLocode(routePort.unlocode).
			SetNameZh(routePort.name).
			SetNameEn(routePort.name).
			SetCountryCode("CN").
			SetTransportModes([]string{"SEA"}).
			SetSource("system").
			SetSortOrder(1).
			SetEnabled(true).
			Save(ctx); err != nil {
			t.Fatalf("创建航程港口 %s 失败: %v", routePort.unlocode, err)
		}
	}
	exec, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetCarrierID(routeCarrierID).
		SetOriginLocationID(routeOriginID).
		SetDischargeLocationID(routeDischargeID).
		SetTransitLocationID(routeTransitID).
		SetVesselName("MAERSK MC-KINNEY MOLLER").
		SetVoyageNo("2609W").
		SetEtd(etd).
		SetEta(eta).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行记录失败: %v", err)
	}

	pkgCount := 500
	gw := 12500.5
	vol := 45.2
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetMasterNo("MBL-" + suffix).
		SetNormalizedMasterNo("MBL-" + suffix).
		SetIssuerPartnerID(carrier.ID).
		SetTransportExecutionID(exec.ID).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetPackageCount(pkgCount).
		SetGrossWeightKg(gw).
		SetVolumeCbm(vol).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 MBL 失败: %v", err)
	}

	// 8. 创建主订单 Order A
	orderA, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-" + suffix + "-A").
		SetCustomerID(customer.ID).
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
		t.Fatalf("创建订单 A 失败: %v", err)
	}

	// 关联 MBL 到 Order A
	_, err = data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(orderA.ID).
		SetMasterBillID(mbl.ID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
		SetCargoAllocationVersion(1).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("关联 MBL 到订单 A 失败: %v", err)
	}

	// 创建 HBL
	hblA, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(org.ID).
		SetOrderID(orderA.ID).
		SetMasterBillID(mbl.ID).
		SetHouseNo("HBL-" + suffix + "-A").
		SetNormalizedHouseNo("HBL-" + suffix + "-A").
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(org.ID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 HBL 失败: %v", err)
	}

	// 构造 Principals
	rolePrincipal := &biz.Principal{
		UserID:           roleUser.ID,
		DisplayName:      roleUser.DisplayName,
		IsBootstrapAdmin: false,
		Organization:     biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name},
		Permissions:      []string{"business.order.se.lock", "business.order.se.update"},
	}

	normalPrincipal := &biz.Principal{
		UserID:           normalUser.ID,
		DisplayName:      normalUser.DisplayName,
		IsBootstrapAdmin: false,
		Organization:     biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name},
		Permissions:      []string{"business.order.se.update"},
		RoleScopes:       []biz.RoleScope{{RoleCode: normalRole.Code, DataScope: biz.DataScopeOrganization}},
		RolePermissions: map[string]map[string]struct{}{
			normalRole.Code: {"business.order.se.update": {}},
		},
	}

	adminPrincipal := &biz.Principal{
		UserID:           adminUser.ID,
		DisplayName:      adminUser.DisplayName,
		IsBootstrapAdmin: true,
		Organization:     biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name},
		Permissions:      []string{"*"},
	}

	orderLockRepo := NewOrderLockRepo(data, &conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:             true,
		CorpId:              "CORP-ORDER-UNLOCK-TEST",
		ApprovalProcessCode: "PROC-ORDER-UNLOCK-TEST",
		EventToken:          "EVENT-TOKEN-TEST",
		EventAesKey:         "EVENT-AES-KEY-TEST",
	}})
	orderWriteRepo := NewOrderRepo(data)
	seaDocRepo := NewSeaDocumentRepo(data)

	createSEOrder := func(orderNo string) *ent.Order {
		o, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo(orderNo).
			SetCustomerID(customer.ID).
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
			t.Fatalf("创建订单 %s 失败: %v", orderNo, err)
		}
		return o
	}

	linkMBL := func(orderID uuid.UUID, mblID uuid.UUID, docStruct seamasterbillorderlinkent.DocumentStructure) *ent.SeaMasterBillOrderLink {
		link, err := data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(org.ID).
			SetOrderID(orderID).
			SetMasterBillID(mblID).
			SetDocumentStructure(docStruct).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
			SetCargoAllocationVersion(1).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("关联 MBL 到订单 %s 失败: %v", orderID, err)
		}
		return link
	}

	createHBL := func(orderID uuid.UUID, mblID uuid.UUID, houseNo string) *ent.SeaHouseBill {
		hbl, err := data.db.SeaHouseBill.Create().
			SetOrganizationID(org.ID).
			SetOrderID(orderID).
			SetMasterBillID(mblID).
			SetHouseNo(houseNo).
			SetNormalizedHouseNo(houseNo).
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(org.ID).
			SetStatus(seahousebillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 HBL %s 失败: %v", houseNo, err)
		}
		return hbl
	}

	// --- 阶段 1：普通用户尝试锁定订单应被业务角色资格校验拦截 ---
	t.Run("非海运出口业务角色成员锁定订单应被拦截", func(t *testing.T) {
		audit := &biz.AuditEvent{Action: "order.lock", UserID: &normalUser.ID, Details: map[string]string{}}
		_, err := orderLockRepo.LockOrder(ctx, normalPrincipal, orderA.ID, 1, "idem-test-unauthorized", audit)
		if err == nil {
			t.Fatal("期望普通用户锁定返回错误，但成功了")
		}
		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "ORDER_LOCK_ROLE_REQUIRED" {
			t.Fatalf("期望错误 ORDER_LOCK_ROLE_REQUIRED，得到: %v", err)
		}
	})

	// --- 阶段 2：具备业务角色的成员成功锁定订单 ---
	t.Run("具备角色的成员锁定订单并创建单证不可变版本快照", func(t *testing.T) {
		audit := &biz.AuditEvent{Action: "order.lock", UserID: &roleUser.ID, Details: map[string]string{}}
		lockRes, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 1, "idem-lock-key-1", audit)
		if err != nil {
			t.Fatalf("角色用户锁定订单失败: %v", err)
		}

		if lockRes == nil || lockRes.State == nil || !lockRes.State.IsLocked {
			t.Fatalf("锁定结果状态异常: %#v", lockRes)
		}
		if lockRes.State.LockGeneration != 1 {
			t.Errorf("期望锁定代数 1，得到: %d", lockRes.State.LockGeneration)
		}

		// 检查数据库中 Order 实体状态
		dbOrder, err := data.db.Order.Get(ctx, orderA.ID)
		if err != nil {
			t.Fatalf("读取订单失败: %v", err)
		}
		if dbOrder.LockedAt == nil || dbOrder.LockedBy == nil || *dbOrder.LockedBy != roleUser.ID {
			t.Fatalf("数据库订单锁定字段不符合预期: locked_at=%v, locked_by=%v", dbOrder.LockedAt, dbOrder.LockedBy)
		}
		if dbOrder.LockGeneration != 1 {
			t.Errorf("订单锁定代数不符合预期: %d", dbOrder.LockGeneration)
		}

		// 检查 MBL 版本与快照
		dbMbl, err := data.db.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			t.Fatalf("读取 MBL 失败: %v", err)
		}
		if dbMbl.CurrentVersionID == nil {
			t.Fatal("MBL 当前版本 ID 未设置")
		}
		mblVer, err := data.db.SeaMasterBillVersion.Get(ctx, *dbMbl.CurrentVersionID)
		if err != nil {
			t.Fatalf("读取 MBL 版本失败: %v", err)
		}
		if mblVer.VersionNo != 1 || mblVer.ContentHash == "" {
			t.Fatalf("MBL 版本字段异常: version_no=%d, hash=%s", mblVer.VersionNo, mblVer.ContentHash)
		}
		if mblVer.VesselVoyageSnapshot == nil || *mblVer.VesselVoyageSnapshot != "MAERSK MC-KINNEY MOLLER 2609W" {
			t.Errorf("MBL 船名航次快照异常: %v", mblVer.VesselVoyageSnapshot)
		}
		if mblVer.CarrierID == nil || *mblVer.CarrierID != routeCarrierID ||
			mblVer.OriginLocationID == nil || *mblVer.OriginLocationID != routeOriginID ||
			mblVer.DischargeLocationID == nil || *mblVer.DischargeLocationID != routeDischargeID ||
			mblVer.TransitLocationID == nil || *mblVer.TransitLocationID != routeTransitID ||
			mblVer.Etd == nil || !mblVer.Etd.Equal(etd) || mblVer.Eta == nil || !mblVer.Eta.Equal(eta) {
			t.Fatalf("MBL 权威航程快照不完整: %#v", mblVer)
		}

		// 检查 HBL 版本与快照
		dbHbl, err := data.db.SeaHouseBill.Get(ctx, hblA.ID)
		if err != nil {
			t.Fatalf("读取 HBL 失败: %v", err)
		}
		if dbHbl.CurrentVersionID == nil {
			t.Fatal("HBL 当前版本 ID 未设置")
		}
		hblVer, err := data.db.SeaHouseBillVersion.Get(ctx, *dbHbl.CurrentVersionID)
		if err != nil {
			t.Fatalf("读取 HBL 版本失败: %v", err)
		}
		if hblVer.VersionNo != 1 || hblVer.ContentHash == "" {
			t.Fatalf("HBL 版本字段异常: version_no=%d, hash=%s", hblVer.VersionNo, hblVer.ContentHash)
		}

		// 检查 OrderLockRecord 事实表
		record, err := data.db.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(orderA.ID),
				orderlockrecordent.GenerationEQ(1),
			).
			WithHouseBillSnapshots().
			Only(ctx)
		if err != nil {
			t.Fatalf("读取锁定事实记录失败: %v", err)
		}
		if record.MasterBillVersionID != mblVer.ID {
			t.Errorf("锁定事实记录中的 MBL 版本 ID 不匹配")
		}
		if len(record.Edges.HouseBillSnapshots) != 1 {
			t.Fatalf("期望 1 条 HBL 快照关联，实际得到: %d", len(record.Edges.HouseBillSnapshots))
		}
		if record.Edges.HouseBillSnapshots[0].HouseBillVersionID != hblVer.ID {
			t.Errorf("HBL 快照版本 ID 不匹配")
		}
	})

	// --- 阶段 3：锁定幂等性验证 ---
	t.Run("锁定幂等性契约：同键同指纹重放返回原结果、同键异指纹409、不同键已锁拦截", func(t *testing.T) {
		audit := &biz.AuditEvent{Action: "order.lock", UserID: &roleUser.ID, Details: map[string]string{}}
		// 1. 同键同指纹重放
		res2, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 1, "idem-lock-key-1", audit)
		if err != nil {
			t.Fatalf("幂等锁定重放失败: %v", err)
		}
		if res2.State.LockGeneration != 1 {
			t.Errorf("期望仍为代数 1，得到: %d", res2.State.LockGeneration)
		}

		// 验证没有生成重复的 MBL 版本
		mblVerCount, err := data.db.SeaMasterBillVersion.Query().
			Where(seamasterbillversionent.MasterBillIDEQ(mbl.ID)).
			Count(ctx)
		if err != nil {
			t.Fatalf("查询 MBL 版本数失败: %v", err)
		}
		if mblVerCount != 1 {
			t.Fatalf("期望 MBL 只有 1 个版本，实际有: %d", mblVerCount)
		}

		// 2. 同键异指纹（版本不符或调用人不同）应返回 409 ORDER_STATUS_CONFLICT
		_, err = orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 2, "idem-lock-key-1", audit)
		if err == nil {
			t.Fatal("期望同键异指纹返回 409 错误，但成功了")
		}
		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "ORDER_STATUS_CONFLICT" {
			t.Fatalf("期望错误 ORDER_STATUS_CONFLICT，得到: %v", err)
		}

		// 3. 不同键对已锁定的订单再次锁定应返回 ORDER_ALREADY_LOCKED
		_, err = orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 1, "idem-lock-different-key", audit)
		if err == nil {
			t.Fatal("期望不同键锁定已锁订单返回错误，但成功了")
		}
		kErr2 := errors.FromError(err)
		if kErr2 == nil || kErr2.Reason != "ORDER_ALREADY_LOCKED" {
			t.Fatalf("期望错误 ORDER_ALREADY_LOCKED，得到: %v", err)
		}
	})

	// --- 阶段 4：写入门禁阻断验证 ---
	t.Run("已锁定订单阻止业务修改并返回结构化元数据", func(t *testing.T) {
		audit := &biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		newRef := "PO-MODIFIED-SHOULD-FAIL"
		_, err := orderWriteRepo.UpdateDraft(ctx, org.ID, orderA.ID, 1, &biz.Order{
			ID:                  orderA.ID,
			Version:             1,
			CustomerReferenceNo: newRef,
			BusinessType:        biz.OrderBusinessSE,
		}, audit)
		if err == nil {
			t.Fatal("期望阻止修改已锁定订单，但成功了")
		}

		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "ORDER_BUSINESS_LOCKED" {
			t.Fatalf("期望错误 ORDER_BUSINESS_LOCKED，得到: %v", err)
		}
		if kErr.Metadata["order_id"] != orderA.ID.String() {
			t.Errorf("元数据中缺少正确的 order_id: %v", kErr.Metadata)
		}
		if kErr.Metadata["lock_generation"] != "1" {
			t.Errorf("元数据中缺少正确的 lock_generation: %v", kErr.Metadata)
		}
	})

	// --- 阶段 5：共享 MBL 锁定门禁阻断验证 ---
	t.Run("共享 MBL 存在成员订单被锁定时阻止修改 MBL", func(t *testing.T) {
		// 创建第二个订单 Order B，并共享关联同一个 MBL
		orderB, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo("SE-" + suffix + "-B").
			SetCustomerID(customer.ID).
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
			t.Fatalf("创建订单 B 失败: %v", err)
		}
		_, err = data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(org.ID).
			SetOrderID(orderB.ID).
			SetMasterBillID(mbl.ID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
			SetCargoAllocationVersion(1).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("关联 MBL 到订单 B 失败: %v", err)
		}

		// 尝试修改共享 MBL 的描述（通过未锁定的 Order B 触发）
		newGoodsDesc := "MODIFIED GOODS DESC"
		_, err = seaDocRepo.UpdateSeaMasterBillContent(ctx, org.ID, roleUser.ID, orderB.ID, 1, &biz.SeaBillContent{
			GoodsDescriptionText: &newGoodsDesc,
		}, &biz.AuditEvent{Action: "sea_mbl.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}})
		if err == nil {
			t.Fatal("期望共享 MBL 修改被阻断，但成功了")
		}

		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "SEA_MASTER_BILL_MEMBER_ORDER_LOCKED" {
			t.Fatalf("期望错误 SEA_MASTER_BILL_MEMBER_ORDER_LOCKED，得到: %v", err)
		}
		if kErr.Metadata["locked_count"] != "1" {
			t.Errorf("元数据中 locked_count 期望 1，得到: %s", kErr.Metadata["locked_count"])
		}
		if !strings.Contains(kErr.Metadata["locked_order_nos"], orderA.OrderNo) {
			t.Errorf("元数据中 locked_order_nos 期望包含 %s，得到: %s", orderA.OrderNo, kErr.Metadata["locked_order_nos"])
		}

		shipmentType := biz.OrderShipmentFCL
		expectedMBLVersion := uint64(1)
		buildOrderUpdate := func(vesselVoyage, customerReferenceNo string) *biz.Order {
			return &biz.Order{
				ID:                  orderB.ID,
				CustomerID:          customer.ID,
				Version:             1,
				CustomerReferenceNo: customerReferenceNo,
				BusinessType:        biz.OrderBusinessSE,
				TradeDirection:      biz.OrderTradeExport,
				TradeTerm:           biz.OrderTradeFOB,
				PaymentTerm:         biz.OrderPaymentPrepaid,
				ShipmentType:        &shipmentType,
				CarrierID:           &routeCarrierID,
				OriginLocationID:    &routeOriginID,
				DischargeLocationID: &routeDischargeID,
				TransitLocationID:   &routeTransitID,
				VesselVoyage:        vesselVoyage,
				ETD:                 etd.Format(time.RFC3339Nano),
				ETA:                 eta.Format(time.RFC3339Nano),
				SeaMasterBillInput: &biz.SeaMasterBillInput{
					MasterNo:                 mbl.MasterNo,
					IssuerPartnerID:          carrier.ID,
					ExpectedCandidateVersion: &expectedMBLVersion,
				},
			}
		}

		mblBefore, err := data.db.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			t.Fatalf("读取共享 MBL 修改前状态失败: %v", err)
		}
		execBefore, err := data.db.SeaTransportExecution.Get(ctx, exec.ID)
		if err != nil {
			t.Fatalf("读取共享运输执行修改前状态失败: %v", err)
		}
		_, err = orderWriteRepo.UpdateDraft(
			ctx,
			org.ID,
			orderB.ID,
			1,
			buildOrderUpdate("EVER GIVEN / 999E", "SHOULD-ROLLBACK"),
			&biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}},
		)
		if err == nil {
			t.Fatal("期望从未锁定成员订单修改共享航程时被阻断，但成功了")
		}
		kErr = errors.FromError(err)
		if kErr == nil || kErr.Reason != "SEA_MASTER_BILL_MEMBER_ORDER_LOCKED" {
			t.Fatalf("期望 UpdateDraft 返回 SEA_MASTER_BILL_MEMBER_ORDER_LOCKED，得到: %v", err)
		}
		orderBAfterRejectedUpdate, err := data.db.Order.Get(ctx, orderB.ID)
		if err != nil {
			t.Fatalf("读取被拒绝更新后的订单 B 失败: %v", err)
		}
		mblAfterRejectedUpdate, err := data.db.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			t.Fatalf("读取被拒绝更新后的共享 MBL 失败: %v", err)
		}
		execAfterRejectedUpdate, err := data.db.SeaTransportExecution.Get(ctx, exec.ID)
		if err != nil {
			t.Fatalf("读取被拒绝更新后的共享运输执行失败: %v", err)
		}
		if orderBAfterRejectedUpdate.Version != 1 || orderBAfterRejectedUpdate.CustomerReferenceNo != "" {
			t.Fatalf("共享字段门禁失败后订单写入未回滚: version=%d customer_ref=%q", orderBAfterRejectedUpdate.Version, orderBAfterRejectedUpdate.CustomerReferenceNo)
		}
		if mblAfterRejectedUpdate.Version != mblBefore.Version || execAfterRejectedUpdate.Version != execBefore.Version || execAfterRejectedUpdate.VesselName != execBefore.VesselName || execAfterRejectedUpdate.VoyageNo != execBefore.VoyageNo {
			t.Fatalf("共享字段门禁失败后 MBL/运输执行发生变化: mbl_version=%d/%d exec_version=%d/%d vessel=%q/%q voyage=%q/%q",
				mblAfterRejectedUpdate.Version, mblBefore.Version,
				execAfterRejectedUpdate.Version, execBefore.Version,
				execAfterRejectedUpdate.VesselName, execBefore.VesselName,
				execAfterRejectedUpdate.VoyageNo, execBefore.VoyageNo,
			)
		}

		// 即使没有提交 SeaMasterBillInput，整单更新中的 MasterBillContent 也会修改共享
		// MBL，必须复用相同的全成员 Order → MBL → Link 锁序和成员锁定门禁。
		contentShipper := "CONTENT-SHOULD-ROLLBACK"
		contentUpdate := buildOrderUpdate("MAERSK MC-KINNEY MOLLER / 2609W", "CONTENT-SHOULD-ROLLBACK")
		contentUpdate.SeaMasterBillInput = nil
		contentUpdate.SeaDocumentInput = &biz.SeaOrderDocumentInput{
			ExpectedMblVersion: &expectedMBLVersion,
			MasterBillContent: &biz.SeaBillContent{
				ShipperText: &contentShipper,
			},
		}
		_, err = orderWriteRepo.UpdateDraft(
			ctx,
			org.ID,
			orderB.ID,
			1,
			contentUpdate,
			&biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}},
		)
		if err == nil {
			t.Fatal("期望通过整单 MasterBillContent 修改共享 MBL 时被阻断，但成功了")
		}
		kErr = errors.FromError(err)
		if kErr == nil || kErr.Reason != "SEA_MASTER_BILL_MEMBER_ORDER_LOCKED" {
			t.Fatalf("期望整单 MasterBillContent 更新返回 SEA_MASTER_BILL_MEMBER_ORDER_LOCKED，得到: %v", err)
		}
		orderBAfterContentRejected, err := data.db.Order.Get(ctx, orderB.ID)
		if err != nil {
			t.Fatalf("读取整单内容更新被拒后的订单 B 失败: %v", err)
		}
		mblAfterContentRejected, err := data.db.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			t.Fatalf("读取整单内容更新被拒后的共享 MBL 失败: %v", err)
		}
		if orderBAfterContentRejected.Version != 1 || orderBAfterContentRejected.CustomerReferenceNo != "" {
			t.Fatalf("整单内容门禁失败后订单写入未回滚: version=%d customer_ref=%q", orderBAfterContentRejected.Version, orderBAfterContentRejected.CustomerReferenceNo)
		}
		if mblAfterContentRejected.Version != mblBefore.Version || mblAfterContentRejected.ShipperText != nil {
			t.Fatalf("整单内容门禁失败后共享 MBL 未完整回滚: version=%d/%d shipper=%v",
				mblAfterContentRejected.Version, mblBefore.Version, mblAfterContentRejected.ShipperText,
			)
		}

		// 共享字段保持不变时，只修改未锁定成员自身字段仍应成功。
		updatedOrderB, err := orderWriteRepo.UpdateDraft(
			ctx,
			org.ID,
			orderB.ID,
			1,
			buildOrderUpdate("MAERSK MC-KINNEY MOLLER / 2609W", "ORDINARY-FIELD-UPDATED"),
			&biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}},
		)
		if err != nil {
			t.Fatalf("共享字段未变化时更新订单 B 普通字段失败: %v", err)
		}
		if updatedOrderB.Version != 2 || updatedOrderB.CustomerReferenceNo != "ORDINARY-FIELD-UPDATED" {
			t.Fatalf("订单 B 普通字段更新结果不正确: version=%d customer_ref=%q", updatedOrderB.Version, updatedOrderB.CustomerReferenceNo)
		}
		mblAfterOrdinaryUpdate, err := data.db.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			t.Fatalf("读取普通字段更新后的共享 MBL 失败: %v", err)
		}
		execAfterOrdinaryUpdate, err := data.db.SeaTransportExecution.Get(ctx, exec.ID)
		if err != nil {
			t.Fatalf("读取普通字段更新后的共享运输执行失败: %v", err)
		}
		if mblAfterOrdinaryUpdate.Version != mblBefore.Version || execAfterOrdinaryUpdate.Version != execBefore.Version {
			t.Fatalf("普通订单字段更新不应修改共享 MBL/运输执行版本: mbl=%d/%d exec=%d/%d",
				mblAfterOrdinaryUpdate.Version, mblBefore.Version,
				execAfterOrdinaryUpdate.Version, execBefore.Version,
			)
		}
	})

	// --- 阶段 6：业务角色直接解锁路径 (ROLE_DIRECT) ---
	t.Run("具备业务角色的用户直接解锁订单", func(t *testing.T) {
		reason := "业务需要调整装箱计划"
		audit := &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		unlockRes, err := orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderA.ID, 2, "idem-unlock-key-1", &reason, audit)
		if err != nil {
			t.Fatalf("业务角色直接解锁失败: %v", err)
		}

		if unlockRes == nil || unlockRes.Request == nil {
			t.Fatalf("解锁结果异常: %#v", unlockRes)
		}
		if unlockRes.Request.Route != biz.UnlockRouteRoleDirect {
			t.Errorf("期望解锁路由 ROLE_DIRECT，实际得到: %s", unlockRes.Request.Route)
		}
		if unlockRes.Request.Status != biz.UnlockStatusApproved {
			t.Errorf("期望解锁状态 APPROVED，实际得到: %s", unlockRes.Request.Status)
		}

		// 检查订单状态：已解锁且版本已自增
		dbOrder, err := data.db.Order.Get(ctx, orderA.ID)
		if err != nil {
			t.Fatalf("读取订单失败: %v", err)
		}
		if dbOrder.LockedAt != nil || dbOrder.LockedBy != nil {
			t.Fatalf("订单应已清除锁定标记: locked_at=%v, locked_by=%v", dbOrder.LockedAt, dbOrder.LockedBy)
		}
		if dbOrder.Version != 3 {
			t.Errorf("解锁后订单版本期望自增为 3，实际为: %d", dbOrder.Version)
		}

		// 检查锁定事实记录回填
		record, err := data.db.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(orderA.ID),
				orderlockrecordent.GenerationEQ(1),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取锁定事实记录失败: %v", err)
		}
		if record.UnlockedAt == nil || record.UnlockedBy == nil || *record.UnlockedBy != roleUser.ID {
			t.Errorf("锁定事实记录缺少解锁信息: unlocked_at=%v, unlocked_by=%v", record.UnlockedAt, record.UnlockedBy)
		}
		if record.UnlockMode == nil || *record.UnlockMode != biz.UnlockRouteRoleDirect {
			t.Errorf("解锁模式期望 ROLE_DIRECT，实际为: %v", record.UnlockMode)
		}

		// 解锁幂等性契约：订单已直接解锁后，同键同指纹重放仍返回原 APPROVED 请求，同键异指纹返回 409
		replayUnlock, err := orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderA.ID, 2, "idem-unlock-key-1", &reason, audit)
		if err != nil {
			t.Fatalf("已解锁后同键同指纹重放失败: %v", err)
		}
		if replayUnlock.Request == nil || replayUnlock.Request.Status != biz.UnlockStatusApproved {
			t.Fatalf("重放期望 APPROVED 请求，得到: %#v", replayUnlock)
		}

		diffReason := "不同原因"
		_, err = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderA.ID, 2, "idem-unlock-key-1", &diffReason, audit)
		if err == nil {
			t.Fatal("期望同键异指纹重放返回 409 冲突，但成功了")
		}
		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "ORDER_STATUS_CONFLICT" {
			t.Fatalf("期望错误 ORDER_STATUS_CONFLICT，得到: %v", err)
		}

		// 验证解锁后订单可正常写入
		newRef := "PO-AFTER-UNLOCK-SUCCESS"
		auditUpdate := &biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		shipType := biz.OrderShipmentFCL
		updated, err := orderWriteRepo.UpdateDraft(ctx, org.ID, orderA.ID, 3, &biz.Order{
			ID:                  orderA.ID,
			CustomerID:          customer.ID,
			Version:             3,
			CustomerReferenceNo: newRef,
			BusinessType:        biz.OrderBusinessSE,
			TradeDirection:      biz.OrderTradeExport,
			TradeTerm:           biz.OrderTradeFOB,
			PaymentTerm:         biz.OrderPaymentPrepaid,
			ShipmentType:        &shipType,
		}, auditUpdate)
		if err != nil {
			t.Fatalf("解锁后更新订单草稿失败: %v", err)
		}
		if updated.CustomerReferenceNo != newRef {
			t.Errorf("更新后的参考号未生效")
		}
	})

	// --- 阶段 7：系统管理员紧急解锁路径 (ADMIN_EMERGENCY) ---
	t.Run("超管紧急解锁代数2订单", func(t *testing.T) {
		// 重新锁定订单（当前版本 4，锁定后代数 2，版本 5）
		audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		lockRes, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 4, "idem-lock-key-2", audit)
		if err != nil {
			t.Fatalf("第二周期锁定失败: %v", err)
		}
		if lockRes.State.LockGeneration != 2 {
			t.Fatalf("期望锁定代数 2，实际为: %d", lockRes.State.LockGeneration)
		}

		// 超管发起紧急解锁（当前版本 5，解锁后版本 6）
		reason := "海关截关临界超管紧急干预"
		auditUnlock := &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &adminUser.ID, Result: "success", Details: map[string]string{}}
		unlockRes, err := orderLockRepo.RequestOrderUnlock(ctx, adminPrincipal, orderA.ID, 5, "idem-admin-unlock-1", &reason, auditUnlock)
		if err != nil {
			t.Fatalf("超管紧急解锁失败: %v", err)
		}

		if unlockRes.Request.Route != biz.UnlockRouteAdminEmergency {
			t.Errorf("期望解锁路由 ADMIN_EMERGENCY，实际为: %s", unlockRes.Request.Route)
		}
		if unlockRes.Request.Status != biz.UnlockStatusApproved {
			t.Errorf("期望解锁状态 APPROVED，实际为: %s", unlockRes.Request.Status)
		}

		// 检查订单已被解锁且版本升级为 6
		dbOrder, err := data.db.Order.Get(ctx, orderA.ID)
		if err != nil {
			t.Fatalf("读取订单失败: %v", err)
		}
		if dbOrder.LockedAt != nil {
			t.Fatalf("订单应已清除锁定标记")
		}
		if dbOrder.Version != 6 {
			t.Errorf("期望订单版本 6，实际为: %d", dbOrder.Version)
		}
	})

	// --- 阶段 8：普通用户申请解锁排队钉钉审批 (DINGTALK_APPROVAL) ---
	t.Run("普通用户申请解锁生成审批中请求并快照候选人与后台任务", func(t *testing.T) {
		// 重新锁定订单（当前版本 6，锁定后代数 3，版本 7）
		audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		lockRes, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderA.ID, 6, "idem-lock-key-3", audit)
		if err != nil {
			t.Fatalf("第三周期锁定失败: %v", err)
		}
		if lockRes.State.LockGeneration != 3 {
			t.Fatalf("期望锁定代数 3，实际为: %d", lockRes.State.LockGeneration)
		}

		// 普通用户发起解锁申请（当前版本 7）
		reason := "客户要求修改件毛体数据"
		auditUnlock := &biz.AuditEvent{Action: "order.unlock_request", OrganizationID: &org.ID, UserID: &normalUser.ID, Result: "success", Details: map[string]string{}}
		unlockRes, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, orderA.ID, 7, "idem-normal-request-1", &reason, auditUnlock)
		if err != nil {
			t.Fatalf("普通用户申请解锁失败: %v", err)
		}

		if unlockRes.Request.Route != biz.UnlockRouteDingTalkApproval {
			t.Errorf("期望解锁路由 DINGTALK_APPROVAL，实际为: %s", unlockRes.Request.Route)
		}
		if unlockRes.Request.Status != biz.UnlockStatusPendingDispatch {
			t.Errorf("期望解锁状态 PENDING_DISPATCH，实际为: %s", unlockRes.Request.Status)
		}

		// 检查订单仍处于锁定状态
		dbOrder, err := data.db.Order.Get(ctx, orderA.ID)
		if err != nil {
			t.Fatalf("读取订单失败: %v", err)
		}
		if dbOrder.LockedAt == nil {
			t.Fatalf("普通用户申请解锁后订单不应被立即解锁")
		}

		// 检查数据库中 OrderUnlockRequest 记录与候选人快照
		reqRecord, err := data.db.OrderUnlockRequest.Query().
			Where(
				orderunlockrequestent.OrderIDEQ(orderA.ID),
				orderunlockrequestent.LockGenerationEQ(3),
			).
			WithApproverCandidates().
			Only(ctx)
		if err != nil {
			t.Fatalf("读取解锁请求事实表记录失败: %v", err)
		}
		if reqRecord.Status != biz.UnlockStatusPendingDispatch {
			t.Errorf("请求状态不符合预期: %s", reqRecord.Status)
		}
		if len(reqRecord.Edges.ApproverCandidates) != 1 {
			t.Fatalf("期望 1 个审批候选人快照，实际为: %d", len(reqRecord.Edges.ApproverCandidates))
		}
		candidate := reqRecord.Edges.ApproverCandidates[0]
		if candidate.UserID != roleUser.ID {
			t.Errorf("候选人用户 ID 不匹配: %v vs %v", candidate.UserID, roleUser.ID)
		}
		if candidate.DingtalkUseridSnapshot != dtRoleUserID {
			t.Errorf("候选人钉钉 ID 快照不匹配: %s", candidate.DingtalkUseridSnapshot)
		}

		// 检查生成的后台任务记录
		tasks, err := data.db.BackgroundTask.Query().
			Where(
				backgroundtaskent.OrganizationIDEQ(org.ID),
				backgroundtaskent.KindEQ(backgroundtaskent.KindDINGTALK_APPROVAL_CREATE),
			).
			All(ctx)
		if err != nil {
			t.Fatalf("查询挂起后台任务失败: %v", err)
		}
		if len(tasks) == 0 {
			t.Errorf("未找到预期的 order.unlock.dingtalk.dispatch 后台调度任务")
		}
	})

	// --- 阶段 9：DIRECT 单证结构零 HBL 成功锁定与结构异常校验 ---
	t.Run("DIRECT 单证结构零 HBL 成功锁定与结构异常校验", func(t *testing.T) {
		orderC := createSEOrder("SE-" + suffix + "-C")
		linkMBL(orderC.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureDIRECT)

		audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		lockRes, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderC.ID, 1, "idem-direct-lock-1", audit)
		if err != nil {
			t.Fatalf("DIRECT 零 HBL 锁定失败: %v", err)
		}
		if !lockRes.State.IsLocked || lockRes.State.LockGeneration != 1 {
			t.Fatalf("DIRECT 订单锁定状态异常: %#v", lockRes.State)
		}

		rec, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderC.ID), orderlockrecordent.GenerationEQ(1)).
			WithHouseBillSnapshots().
			Only(ctx)
		if err != nil {
			t.Fatalf("读取 DIRECT 锁定记录失败: %v", err)
		}
		if len(rec.Edges.HouseBillSnapshots) != 0 {
			t.Errorf("DIRECT 订单期望 0 条 HBL 快照，实际得到: %d", len(rec.Edges.HouseBillSnapshots))
		}

		orderC2 := createSEOrder("SE-" + suffix + "-C2")
		linkMBL(orderC2.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureDIRECT)
		createHBL(orderC2.ID, mbl.ID, "HBL-INVALID-DIRECT-"+suffix)

		_, err = orderLockRepo.LockOrder(ctx, rolePrincipal, orderC2.ID, 1, "idem-direct-invalid-lock", audit)
		if err == nil {
			t.Fatal("DIRECT 单证结构含有 HBL 应被拦截，但成功了")
		}
		kErr := errors.FromError(err)
		if kErr == nil || kErr.Reason != "SEA_DOCUMENT_STRUCTURE_CONFLICT" {
			t.Fatalf("期望错误 SEA_DOCUMENT_STRUCTURE_CONFLICT，得到: %v", err)
		}
	})

	// --- 阶段 10：解锁后修改并第二次锁定正确复用/产生版本且第一代历史可重现 ---
	t.Run("解锁后修改并第二次锁定正确复用产生版本且第一代历史可重现", func(t *testing.T) {
		recGen1, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderA.ID), orderlockrecordent.GenerationEQ(1)).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取 Gen 1 锁定记录失败: %v", err)
		}
		recGen2, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderA.ID), orderlockrecordent.GenerationEQ(2)).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取 Gen 2 锁定记录失败: %v", err)
		}

		if recGen1.MasterBillVersionID != recGen2.MasterBillVersionID {
			t.Errorf("MBL 未修改时期望 Gen 1 与 Gen 2 指向同一 MBL 版本，但不同: %v vs %v", recGen1.MasterBillVersionID, recGen2.MasterBillVersionID)
		}

		execD, err := data.db.SeaTransportExecution.Create().
			SetOrganizationID(org.ID).
			SetVesselName("VESSEL D").
			SetVoyageNo("VOY D").
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 execD 失败: %v", err)
		}
		mblD, err := data.db.SeaMasterBill.Create().
			SetOrganizationID(org.ID).
			SetIssuerPartnerID(carrier.ID).
			SetTransportExecutionID(execD.ID).
			SetMasterNo("MBL-D-" + suffix).
			SetNormalizedMasterNo("MBL-D-" + suffix).
			SetStatus(seamasterbillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 mblD 失败: %v", err)
		}

		orderD := createSEOrder("SE-" + suffix + "-D")
		linkMBL(orderD.ID, mblD.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		hblD := createHBL(orderD.ID, mblD.ID, "HBL-D-"+suffix)

		auditD := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		lockDRes, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderD.ID, 1, "idem-order-d-lock-1", auditD)
		if err != nil {
			t.Fatalf("锁定 Order D 失败: %v", err)
		}
		mblVerD1ID := lockDRes.LockRecord.MasterBillVersionID

		reasonD := "调整货物"
		unlockDRes, err := orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderD.ID, 2, "idem-order-d-unlock-1", &reasonD, auditD)
		if err != nil {
			t.Fatalf("解锁 Order D 失败: %v", err)
		}
		if unlockDRes.Request.Status != biz.UnlockStatusApproved {
			t.Fatalf("Order D 解锁状态异常: %s", unlockDRes.Request.Status)
		}

		newDesc := "UPDATED GOODS DESCRIPTION FOR VERSION 2"
		_, err = seaDocRepo.UpdateSeaMasterBillContent(ctx, org.ID, roleUser.ID, orderD.ID, 1, &biz.SeaBillContent{
			GoodsDescriptionText: &newDesc,
		}, &biz.AuditEvent{Action: "mbl.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}})
		if err != nil {
			t.Fatalf("更新 MBL 内容失败: %v", err)
		}

		lockDRes2, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderD.ID, 3, "idem-order-d-lock-2", auditD)
		if err != nil {
			t.Fatalf("再次锁定 Order D 失败: %v", err)
		}
		if lockDRes2.LockRecord.MasterBillVersionID == mblVerD1ID {
			t.Fatalf("MBL 内容修改后再次锁定期望生成新版本，但仍为旧版本: %v", lockDRes2.LockRecord.MasterBillVersionID)
		}

		recDGen1, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderD.ID), orderlockrecordent.GenerationEQ(1)).
			Only(ctx)
		if err != nil {
			t.Fatalf("重新读取 Order D Gen 1 失败: %v", err)
		}
		if recDGen1.MasterBillVersionID != mblVerD1ID {
			t.Errorf("Order D Gen 1 历史记录被污染: %v vs %v", recDGen1.MasterBillVersionID, mblVerD1ID)
		}
		recDGen2, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderD.ID), orderlockrecordent.GenerationEQ(2)).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取 Order D Gen 2 失败: %v", err)
		}
		if recDGen2.MasterBillVersionID != lockDRes2.LockRecord.MasterBillVersionID {
			t.Errorf("Order D Gen 2 版本记录不匹配: %v vs %v", recDGen2.MasterBillVersionID, lockDRes2.LockRecord.MasterBillVersionID)
		}

		// 正式迁移的触发器必须阻止把其他单证的版本设为当前版本。
		if _, err := data.sqlDB.ExecContext(ctx,
			`UPDATE sea_master_bills SET current_version_id = $1 WHERE id = $2`,
			lockDRes2.LockRecord.MasterBillVersionID, mbl.ID,
		); err == nil {
			t.Fatal("MBL current_version_id 指向其他 MBL 版本应被正式迁移触发器拒绝")
		}
		hblDCurrent, err := data.db.SeaHouseBill.Get(ctx, hblD.ID)
		if err != nil || hblDCurrent.CurrentVersionID == nil {
			t.Fatalf("读取 HBL D 当前版本失败: hbl=%#v err=%v", hblDCurrent, err)
		}
		if _, err := data.sqlDB.ExecContext(ctx,
			`UPDATE sea_house_bills SET current_version_id = $1 WHERE id = $2`,
			*hblDCurrent.CurrentVersionID, hblA.ID,
		); err == nil {
			t.Fatal("HBL current_version_id 指向其他 HBL 版本应被正式迁移触发器拒绝")
		}
	})

	// --- 阶段 11：普通业务写与 LockOrder 并发只有两种原子结果 ---
	t.Run("普通业务写与 LockOrder 并发只有两种原子结果", func(t *testing.T) {
		orderE := createSEOrder("SE-" + suffix + "-E")
		linkMBL(orderE.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderE.ID, mbl.ID, "HBL-CONCURRENT-"+suffix)

		var wg sync.WaitGroup
		wg.Add(2)
		var writeErr, lockErr error

		go func() {
			defer wg.Done()
			audit := &biz.AuditEvent{Action: "order.update", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			shipType := biz.OrderShipmentFCL
			_, writeErr = orderWriteRepo.UpdateDraft(ctx, org.ID, orderE.ID, 1, &biz.Order{
				ID:                  orderE.ID,
				CustomerID:          customer.ID,
				Version:             1,
				CustomerReferenceNo: "CONCURRENT-REF",
				BusinessType:        biz.OrderBusinessSE,
				TradeDirection:      biz.OrderTradeExport,
				TradeTerm:           biz.OrderTradeFOB,
				PaymentTerm:         biz.OrderPaymentPrepaid,
				ShipmentType:        &shipType,
			}, audit)
		}()

		go func() {
			defer wg.Done()
			audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			_, lockErr = orderLockRepo.LockOrder(ctx, rolePrincipal, orderE.ID, 1, "idem-concurrent-lock-1", audit)
		}()

		wg.Wait()

		if writeErr == nil && lockErr != nil {
			kErr := errors.FromError(lockErr)
			if kErr == nil || kErr.Reason != "ORDER_STATUS_CONFLICT" {
				t.Fatalf("写先成功时锁期望返回 ORDER_STATUS_CONFLICT，得到: %v", lockErr)
			}
		} else if lockErr == nil && writeErr != nil {
			kErr := errors.FromError(writeErr)
			if kErr == nil || kErr.Reason != "ORDER_BUSINESS_LOCKED" {
				t.Fatalf("锁先成功时写期望返回 ORDER_BUSINESS_LOCKED，得到: %v", writeErr)
			}
		} else {
			t.Fatalf("并发冲突出现非预期组合: writeErr=%v, lockErr=%v", writeErr, lockErr)
		}
	})

	// --- 阶段 12：双锁与双直解竞争仅一条成功事实 ---
	t.Run("双锁与双直解竞争仅一条成功事实", func(t *testing.T) {
		orderF := createSEOrder("SE-" + suffix + "-F")
		linkMBL(orderF.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderF.ID, mbl.ID, "HBL-RACE-"+suffix)

		// 1. 双锁竞争
		var wgLock sync.WaitGroup
		wgLock.Add(2)
		var lockErr1, lockErr2 error

		go func() {
			defer wgLock.Done()
			audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			_, lockErr1 = orderLockRepo.LockOrder(ctx, rolePrincipal, orderF.ID, 1, "race-lock-key-1", audit)
		}()

		go func() {
			defer wgLock.Done()
			audit := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			_, lockErr2 = orderLockRepo.LockOrder(ctx, rolePrincipal, orderF.ID, 1, "race-lock-key-2", audit)
		}()

		wgLock.Wait()

		successCount := 0
		if lockErr1 == nil {
			successCount++
		}
		if lockErr2 == nil {
			successCount++
		}
		if successCount != 1 {
			t.Fatalf("双锁竞争期望恰好 1 个成功，实际成功 %d 个 (err1=%v, err2=%v)", successCount, lockErr1, lockErr2)
		}

		lockRecCount, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderF.ID), orderlockrecordent.GenerationEQ(1)).
			Count(ctx)
		if err != nil {
			t.Fatalf("查询锁记录失败: %v", err)
		}
		if lockRecCount != 1 {
			t.Fatalf("期望仅 1 条锁记录，实际得到: %d", lockRecCount)
		}

		// 2. 双直解竞争（当前版本应为 2）
		var wgUnlock sync.WaitGroup
		wgUnlock.Add(2)
		var unlockErr1, unlockErr2 error
		reason1 := "解1"
		reason2 := "解2"

		go func() {
			defer wgUnlock.Done()
			audit := &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			_, unlockErr1 = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderF.ID, 2, "race-unlock-key-1", &reason1, audit)
		}()

		go func() {
			defer wgUnlock.Done()
			audit := &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
			_, unlockErr2 = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderF.ID, 2, "race-unlock-key-2", &reason2, audit)
		}()

		wgUnlock.Wait()

		unlockSuccessCount := 0
		if unlockErr1 == nil {
			unlockSuccessCount++
		}
		if unlockErr2 == nil {
			unlockSuccessCount++
		}
		if unlockSuccessCount != 1 {
			t.Fatalf("双直解竞争期望恰好 1 个成功，实际成功 %d 个 (err1=%v, err2=%v)", unlockSuccessCount, unlockErr1, unlockErr2)
		}

		approvedCount, err := data.db.OrderUnlockRequest.Query().
			Where(orderunlockrequestent.OrderIDEQ(orderF.ID), orderunlockrequestent.StatusEQ(orderunlockrequestent.StatusAPPROVED)).
			Count(ctx)
		if err != nil {
			t.Fatalf("查询解锁记录失败: %v", err)
		}
		if approvedCount != 1 {
			t.Fatalf("期望仅 1 条 APPROVED 解锁请求，实际得到: %d", approvedCount)
		}
	})

	// --- 阶段 13：审计写失败让锁定与直解完整回滚 ---
	t.Run("同键并发锁定与直解重放返回同一事实", func(t *testing.T) {
		orderG := createSEOrder("SE-" + suffix + "-G")
		linkMBL(orderG.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderG.ID, mbl.ID, "HBL-IDEMPOTENT-RACE-"+suffix)

		startLock := make(chan struct{})
		var lockResult1, lockResult2 *biz.OrderLockResult
		var lockErr1, lockErr2 error
		var lockWG sync.WaitGroup
		lockWG.Add(2)
		go func() {
			defer lockWG.Done()
			<-startLock
			lockResult1, lockErr1 = orderLockRepo.LockOrder(ctx, rolePrincipal, orderG.ID, 1, "same-key-lock-race", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		go func() {
			defer lockWG.Done()
			<-startLock
			lockResult2, lockErr2 = orderLockRepo.LockOrder(ctx, rolePrincipal, orderG.ID, 1, "same-key-lock-race", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		close(startLock)
		lockWG.Wait()
		if lockErr1 != nil || lockErr2 != nil {
			t.Fatalf("同键并发锁定应均成功重放: err1=%v err2=%v", lockErr1, lockErr2)
		}
		if lockResult1 == nil || lockResult2 == nil || lockResult1.LockRecord == nil || lockResult2.LockRecord == nil || lockResult1.LockRecord.ID != lockResult2.LockRecord.ID {
			t.Fatalf("同键并发锁定未返回同一事实: result1=%#v result2=%#v", lockResult1, lockResult2)
		}
		recordCount, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(orderG.ID), orderlockrecordent.GenerationEQ(1)).
			Count(ctx)
		if err != nil || recordCount != 1 {
			t.Fatalf("同键并发锁定应仅产生一条事实: count=%d err=%v", recordCount, err)
		}

		reason := "同键并发直解"
		startUnlock := make(chan struct{})
		var unlockResult1, unlockResult2 *biz.OrderUnlockResult
		var unlockErr1, unlockErr2 error
		var unlockWG sync.WaitGroup
		unlockWG.Add(2)
		go func() {
			defer unlockWG.Done()
			<-startUnlock
			unlockResult1, unlockErr1 = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderG.ID, 2, "same-key-unlock-race", &reason, &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		go func() {
			defer unlockWG.Done()
			<-startUnlock
			unlockResult2, unlockErr2 = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderG.ID, 2, "same-key-unlock-race", &reason, &biz.AuditEvent{Action: "order.unlock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		close(startUnlock)
		unlockWG.Wait()
		if unlockErr1 != nil || unlockErr2 != nil {
			t.Fatalf("同键并发直解应均成功重放: err1=%v err2=%v", unlockErr1, unlockErr2)
		}
		if unlockResult1 == nil || unlockResult2 == nil || unlockResult1.Request == nil || unlockResult2.Request == nil || unlockResult1.Request.ID != unlockResult2.Request.ID {
			t.Fatalf("同键并发直解未返回同一请求: result1=%#v result2=%#v", unlockResult1, unlockResult2)
		}
		requestCount, err := data.db.OrderUnlockRequest.Query().
			Where(orderunlockrequestent.OrderIDEQ(orderG.ID), orderunlockrequestent.StatusEQ(orderunlockrequestent.StatusAPPROVED)).
			Count(ctx)
		if err != nil || requestCount != 1 {
			t.Fatalf("同键并发直解应仅产生一条事实: count=%d err=%v", requestCount, err)
		}
	})

	t.Run("不同订单并发复用组织级幂等键返回稳定冲突", func(t *testing.T) {
		orderI := createSEOrder("SE-" + suffix + "-I")
		orderJ := createSEOrder("SE-" + suffix + "-J")
		linkMBL(orderI.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		linkMBL(orderJ.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderI.ID, mbl.ID, "HBL-IDEMPOTENCY-I-"+suffix)
		createHBL(orderJ.ID, mbl.ID, "HBL-IDEMPOTENCY-J-"+suffix)

		start := make(chan struct{})
		var errI, errJ error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, errI = orderLockRepo.LockOrder(ctx, rolePrincipal, orderI.ID, 1, "cross-order-same-key", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		go func() {
			defer wg.Done()
			<-start
			_, errJ = orderLockRepo.LockOrder(ctx, rolePrincipal, orderJ.ID, 1, "cross-order-same-key", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"})
		}()
		close(start)
		wg.Wait()

		if (errI == nil) == (errJ == nil) {
			t.Fatalf("不同订单同键竞争应一成一败: errI=%v errJ=%v", errI, errJ)
		}
		loserErr := errI
		if loserErr == nil {
			loserErr = errJ
		}
		kErr := errors.FromError(loserErr)
		if kErr == nil || kErr.Reason != "ORDER_STATUS_CONFLICT" {
			t.Fatalf("不同订单同键失败应映射稳定 409: %v", loserErr)
		}
		keyCount, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrganizationIDEQ(org.ID), orderlockrecordent.IdempotencyKeyEQ("cross-order-same-key")).
			Count(ctx)
		if err != nil || keyCount != 1 {
			t.Fatalf("组织级幂等键应仅有一条事实: count=%d err=%v", keyCount, err)
		}
	})

	// --- 阶段 15：审计写失败让锁定与直解完整回滚 ---
	t.Run("审计写失败让锁定与直解完整回滚", func(t *testing.T) {
		orderH := createSEOrder("SE-" + suffix + "-H")
		linkMBL(orderH.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderH.ID, mbl.ID, "HBL-ROLLBACK-"+suffix)

		// 1. 锁定过程中审计失败回滚
		failingAudit := &biz.AuditEvent{
			Action:         strings.Repeat("TOOLONGACTION", 20), // 超过 160 字符触发 DB 校验错误
			OrganizationID: &org.ID,
			UserID:         &roleUser.ID,
			Result:         "success",
		}
		_, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderH.ID, 1, "failing-audit-lock", failingAudit)
		if err == nil {
			t.Fatal("期望审计超长导致锁定失败，但成功了")
		}

		// 验证回滚：OrderH 仍然未锁且版本为 1
		dbH, err := data.db.Order.Get(ctx, orderH.ID)
		if err != nil {
			t.Fatalf("读取 Order H 失败: %v", err)
		}
		if dbH.LockedAt != nil || dbH.Version != 1 || dbH.LockGeneration != 0 {
			t.Fatalf("Order H 锁定未完整回滚: locked_at=%v, ver=%d, gen=%d", dbH.LockedAt, dbH.Version, dbH.LockGeneration)
		}
		recCount, _ := data.db.OrderLockRecord.Query().Where(orderlockrecordent.OrderIDEQ(orderH.ID)).Count(ctx)
		if recCount != 0 {
			t.Fatalf("Order H 锁定记录未回滚，残留数: %d", recCount)
		}

		// 2. 直解过程中审计失败回滚
		auditOK := &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success", Details: map[string]string{}}
		_, err = orderLockRepo.LockOrder(ctx, rolePrincipal, orderH.ID, 1, "valid-lock-order-h", auditOK)
		if err != nil {
			t.Fatalf("正常锁定 Order H 失败: %v", err)
		}

		reason := "解锁原因"
		_, err = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderH.ID, 2, "failing-audit-unlock", &reason, failingAudit)
		if err == nil {
			t.Fatal("期望审计超长导致解锁失败，但成功了")
		}

		// 验证回滚：Order H 仍处于锁定状态且版本为 2
		dbHAfter, err := data.db.Order.Get(ctx, orderH.ID)
		if err != nil {
			t.Fatalf("读取 Order H 失败: %v", err)
		}
		if dbHAfter.LockedAt == nil || dbHAfter.Version != 2 || dbHAfter.LockGeneration != 1 {
			t.Fatalf("Order H 解锁未完整回滚: locked_at=%v, ver=%d, gen=%d", dbHAfter.LockedAt, dbHAfter.Version, dbHAfter.LockGeneration)
		}
		recH, err := data.db.OrderLockRecord.Query().Where(orderlockrecordent.OrderIDEQ(orderH.ID), orderlockrecordent.GenerationEQ(1)).Only(ctx)
		if err != nil {
			t.Fatalf("读取锁事实记录失败: %v", err)
		}
		if recH.UnlockedAt != nil {
			t.Fatalf("锁事实记录 unlocked_at 未回滚: %v", recH.UnlockedAt)
		}
	})

	t.Run("钉钉权威批准先落待生效再按固定锁序原子解锁", func(t *testing.T) {
		orderK := createSEOrder("SE-" + suffix + "-K")
		linkMBL(orderK.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderK.ID, mbl.ID, "HBL-DINGTALK-"+suffix)
		if _, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderK.ID, 1, "ding-lock", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"}); err != nil {
			t.Fatalf("准备钉钉审批锁定订单失败: %v", err)
		}
		reason := "钉钉审批解锁"
		unlockResult, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, orderK.ID, 2, "ding-unlock", &reason, &biz.AuditEvent{Action: "order.unlock.request", OrganizationID: &org.ID, UserID: &normalUser.ID, Result: "success"})
		if err != nil {
			t.Fatalf("创建钉钉解锁申请失败: %v", err)
		}

		approvalRepo := NewDingTalkApprovalRepo(data)
		requestEntity, err := data.db.OrderUnlockRequest.Get(ctx, unlockResult.Request.ID)
		if err != nil {
			t.Fatalf("读取审批请求失败: %v", err)
		}
		dispatchEntity, err := requestEntity.QueryDispatch().Only(ctx)
		if err != nil {
			t.Fatalf("读取审批派发记录失败: %v", err)
		}
		leaseToken := uuid.NewString()
		if _, err := data.db.BackgroundTask.UpdateOneID(dispatchEntity.BackgroundTaskID).
			SetStatus(backgroundtaskent.StatusRUNNING).
			SetLeaseToken(leaseToken).
			SetLeaseExpiresAt(time.Now().Add(time.Minute)).
			Save(ctx); err != nil {
			t.Fatalf("建立审批派发测试租约失败: %v", err)
		}
		claimed := &biz.BackgroundTask{ID: dispatchEntity.BackgroundTaskID, OrganizationID: org.ID, LeaseToken: &leaseToken}
		dispatch, err := approvalRepo.PrepareDispatch(ctx, claimed)
		if err != nil || !dispatch.ShouldSend {
			t.Fatalf("准备审批派发失败: dispatch=%#v err=%v", dispatch, err)
		}
		if err := approvalRepo.FinishDispatch(ctx, claimed, &biz.DingTalkApprovalDispatchOutcome{ProcessInstanceID: "PROC-" + suffix}, time.Now().UTC()); err != nil {
			t.Fatalf("保存审批实例失败: %v", err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: "PROC-" + suffix, EncryptedPayloadHash: strings.Repeat("a", 64)}); err != nil {
			t.Fatalf("写入审批 Inbox 失败: %v", err)
		}
		job, err := approvalRepo.ClaimInbox(ctx, time.Minute, time.Now().UTC())
		if err != nil {
			t.Fatalf("领取审批 Inbox 失败: %v", err)
		}
		decision := &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionApproved, ApproverUserID: dtRoleUserID}
		requestID, shouldApply, err := approvalRepo.PrepareApproved(ctx, job, decision, time.Now().UTC())
		if err != nil || !shouldApply || requestID != unlockResult.Request.ID {
			t.Fatalf("保存已同意待生效失败: request=%s shouldApply=%t err=%v", requestID, shouldApply, err)
		}
		pendingApply, err := data.db.OrderUnlockRequest.Get(ctx, requestID)
		if err != nil || pendingApply.Status != orderunlockrequestent.StatusAPPROVED_PENDING_APPLY {
			t.Fatalf("批准必须先持久化 APPROVED_PENDING_APPLY: request=%#v err=%v", pendingApply, err)
		}
		lockedBeforeApply, err := data.db.Order.Get(ctx, orderK.ID)
		if err != nil || lockedBeforeApply.LockedAt == nil {
			t.Fatalf("待本地生效阶段订单必须仍锁定: order=%#v err=%v", lockedBeforeApply, err)
		}
		if err := approvalRepo.ApplyApproved(ctx, job, requestID, decision, time.Now().UTC()); err != nil {
			t.Fatalf("应用钉钉批准失败: %v", err)
		}
		appliedOrder, err := data.db.Order.Get(ctx, orderK.ID)
		if err != nil || appliedOrder.LockedAt != nil || appliedOrder.Version != 3 {
			t.Fatalf("钉钉批准未原子解锁并升级版本: order=%#v err=%v", appliedOrder, err)
		}
		appliedRequest, err := data.db.OrderUnlockRequest.Get(ctx, requestID)
		if err != nil || appliedRequest.Status != orderunlockrequestent.StatusAPPROVED || appliedRequest.DecidedBy == nil || *appliedRequest.DecidedBy != roleUser.ID {
			t.Fatalf("批准请求终态不正确: request=%#v err=%v", appliedRequest, err)
		}
		lockRecord, err := data.db.OrderLockRecord.Query().Where(orderlockrecordent.OrderIDEQ(orderK.ID), orderlockrecordent.GenerationEQ(1)).Only(ctx)
		if err != nil || lockRecord.UnlockMode == nil || *lockRecord.UnlockMode != orderlockrecordent.UnlockModeDINGTALK_APPROVED {
			t.Fatalf("锁事实未记录钉钉批准路径: record=%#v err=%v", lockRecord, err)
		}

		// 相同事件重投只命中唯一 Inbox，不会产生第二次解锁或版本递增。
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: "PROC-" + suffix, EncryptedPayloadHash: strings.Repeat("a", 64)}); err != nil {
			t.Fatalf("重复事件应幂等成功: %v", err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: "PROC-" + suffix, EncryptedPayloadHash: strings.Repeat("b", 64)}); err == nil {
			t.Fatal("相同 event_id 携带不同密文摘要必须冲突")
		}
		unchanged, err := data.db.Order.Get(ctx, orderK.ID)
		if err != nil || unchanged.Version != 3 {
			t.Fatalf("重复事件不应再次递增版本: order=%#v err=%v", unchanged, err)
		}
	})

	t.Run("钉钉拒绝只终结申请且保留锁和审计", func(t *testing.T) {
		orderL := createSEOrder("SE-" + suffix + "-L")
		linkMBL(orderL.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderL.ID, mbl.ID, "HBL-DINGTALK-REJECT-"+suffix)
		if _, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderL.ID, 1, "ding-reject-lock", &biz.AuditEvent{Action: "order.lock", OrganizationID: &org.ID, UserID: &roleUser.ID, Result: "success"}); err != nil {
			t.Fatal(err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, orderL.ID, 2, "ding-reject-unlock", nil, &biz.AuditEvent{Action: "order.unlock.request", OrganizationID: &org.ID, UserID: &normalUser.ID, Result: "success"})
		if err != nil {
			t.Fatal(err)
		}
		approvalRepo := NewDingTalkApprovalRepo(data)
		requestEntity, err := data.db.OrderUnlockRequest.Get(ctx, result.Request.ID)
		if err != nil {
			t.Fatal(err)
		}
		dispatchEntity, err := requestEntity.QueryDispatch().Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		leaseToken := uuid.NewString()
		_, err = data.db.BackgroundTask.UpdateOneID(dispatchEntity.BackgroundTaskID).SetStatus(backgroundtaskent.StatusRUNNING).SetLeaseToken(leaseToken).SetLeaseExpiresAt(time.Now().Add(time.Minute)).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		claimed := &biz.BackgroundTask{ID: dispatchEntity.BackgroundTaskID, OrganizationID: org.ID, LeaseToken: &leaseToken}
		if _, err := approvalRepo.PrepareDispatch(ctx, claimed); err != nil {
			t.Fatal(err)
		}
		instanceID := "PROC-REJECT-" + suffix
		if err := approvalRepo.FinishDispatch(ctx, claimed, &biz.DingTalkApprovalDispatchOutcome{ProcessInstanceID: instanceID}, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-REJECT-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: instanceID, EncryptedPayloadHash: strings.Repeat("c", 64)}); err != nil {
			t.Fatal(err)
		}
		job, err := approvalRepo.ClaimInbox(ctx, time.Minute, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err := approvalRepo.RecordRejected(ctx, job, &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionRejected, ApproverUserID: dtRoleUserID}, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		orderAfter, _ := data.db.Order.Get(ctx, orderL.ID)
		requestAfter, _ := data.db.OrderUnlockRequest.Get(ctx, result.Request.ID)
		if orderAfter.LockedAt == nil || orderAfter.Version != 2 || requestAfter.Status != orderunlockrequestent.StatusREJECTED {
			t.Fatalf("拒绝不得解锁: order=%#v request=%#v", orderAfter, requestAfter)
		}
		auditCount, err := data.db.AuditLog.Query().Where(auditlogent.ActionEQ("order.unlock.dingtalk_rejected"), auditlogent.ResourceIDEQ(orderL.ID.String())).Count(ctx)
		if err != nil || auditCount != 1 {
			t.Fatalf("拒绝审计数量=%d err=%v", auditCount, err)
		}
	})

	t.Run("批准人实时资格撤销后不得误解锁", func(t *testing.T) {
		orderM := createSEOrder("SE-" + suffix + "-M")
		linkMBL(orderM.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderM.ID, mbl.ID, "HBL-DINGTALK-STALE-"+suffix)
		if _, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderM.ID, 1, "ding-stale-lock", nil); err != nil {
			t.Fatal(err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, orderM.ID, 2, "ding-stale-unlock", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		approvalRepo := NewDingTalkApprovalRepo(data)
		requestEntity, err := data.db.OrderUnlockRequest.Get(ctx, result.Request.ID)
		if err != nil {
			t.Fatal(err)
		}
		dispatchEntity, err := requestEntity.QueryDispatch().Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		leaseToken := uuid.NewString()
		_, err = data.db.BackgroundTask.UpdateOneID(dispatchEntity.BackgroundTaskID).SetStatus(backgroundtaskent.StatusRUNNING).SetLeaseToken(leaseToken).SetLeaseExpiresAt(time.Now().Add(time.Minute)).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		claimed := &biz.BackgroundTask{ID: dispatchEntity.BackgroundTaskID, OrganizationID: org.ID, LeaseToken: &leaseToken}
		if _, err := approvalRepo.PrepareDispatch(ctx, claimed); err != nil {
			t.Fatal(err)
		}
		instanceID := "PROC-STALE-" + suffix
		if err := approvalRepo.FinishDispatch(ctx, claimed, &biz.DingTalkApprovalDispatchOutcome{ProcessInstanceID: instanceID}, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-STALE-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: instanceID, EncryptedPayloadHash: strings.Repeat("e", 64)}); err != nil {
			t.Fatal(err)
		}
		job, err := approvalRepo.ClaimInbox(ctx, time.Minute, time.Now().UTC().Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		decision := &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionApproved, ApproverUserID: dtRoleUserID}
		requestID, shouldApply, err := approvalRepo.PrepareApproved(ctx, job, decision, time.Now().UTC())
		if err != nil || !shouldApply {
			t.Fatalf("prepare approved: id=%s apply=%t err=%v", requestID, shouldApply, err)
		}
		if _, err := data.db.Membership.UpdateOneID(roleMembership.ID).SetEnabled(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		if err := approvalRepo.ApplyApproved(ctx, job, requestID, decision, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.Membership.UpdateOneID(roleMembership.ID).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		orderAfter, _ := data.db.Order.Get(ctx, orderM.ID)
		requestAfter, _ := data.db.OrderUnlockRequest.Get(ctx, requestID)
		if orderAfter.LockedAt == nil || orderAfter.Version != 2 || requestAfter.Status != orderunlockrequestent.StatusSTALE || requestAfter.FailureCode == nil || *requestAfter.FailureCode != "APPROVER_NOT_QUALIFIED" {
			t.Fatalf("资格撤销不得解锁: order=%#v request=%#v", orderAfter, requestAfter)
		}
	})

	t.Run("钉钉批准本地生效与角色直解竞争只产生一次解锁事实", func(t *testing.T) {
		orderN := createSEOrder("SE-" + suffix + "-N")
		linkMBL(orderN.ID, mbl.ID, seamasterbillorderlinkent.DocumentStructureHOUSE)
		createHBL(orderN.ID, mbl.ID, "HBL-DINGTALK-RACE-"+suffix)
		if _, err := orderLockRepo.LockOrder(ctx, rolePrincipal, orderN.ID, 1, "ding-race-lock", nil); err != nil {
			t.Fatal(err)
		}
		pending, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, orderN.ID, 2, "ding-race-pending", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		approvalRepo := NewDingTalkApprovalRepo(data)
		requestEntity, err := data.db.OrderUnlockRequest.Get(ctx, pending.Request.ID)
		if err != nil {
			t.Fatal(err)
		}
		dispatchEntity, err := requestEntity.QueryDispatch().Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		leaseToken := uuid.NewString()
		_, err = data.db.BackgroundTask.UpdateOneID(dispatchEntity.BackgroundTaskID).
			SetStatus(backgroundtaskent.StatusRUNNING).
			SetLeaseToken(leaseToken).
			SetLeaseExpiresAt(time.Now().Add(time.Minute)).
			Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		claimed := &biz.BackgroundTask{ID: dispatchEntity.BackgroundTaskID, OrganizationID: org.ID, LeaseToken: &leaseToken}
		if _, err := approvalRepo.PrepareDispatch(ctx, claimed); err != nil {
			t.Fatal(err)
		}
		instanceID := "PROC-RACE-" + suffix
		if err := approvalRepo.FinishDispatch(ctx, claimed, &biz.DingTalkApprovalDispatchOutcome{ProcessInstanceID: instanceID}, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-RACE-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: instanceID, EncryptedPayloadHash: strings.Repeat("f", 64)}); err != nil {
			t.Fatal(err)
		}
		job, err := approvalRepo.ClaimInbox(ctx, time.Minute, time.Now().UTC().Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		decision := &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionApproved, ApproverUserID: dtRoleUserID}
		requestID, shouldApply, err := approvalRepo.PrepareApproved(ctx, job, decision, time.Now().UTC())
		if err != nil || !shouldApply || requestID != pending.Request.ID {
			t.Fatalf("prepare approved: id=%s apply=%t err=%v", requestID, shouldApply, err)
		}

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		var applyErr, directErr error
		var directResult *biz.OrderUnlockResult
		go func() {
			defer wg.Done()
			<-start
			applyErr = approvalRepo.ApplyApproved(ctx, job, requestID, decision, time.Now().UTC())
		}()
		go func() {
			defer wg.Done()
			<-start
			directResult, directErr = orderLockRepo.RequestOrderUnlock(ctx, rolePrincipal, orderN.ID, 2, "ding-race-direct", nil, &biz.AuditEvent{
				Action:         "order.unlock.role_direct_race",
				OrganizationID: &org.ID,
				UserID:         &roleUser.ID,
				ResourceType:   "order",
				ResourceID:     orderN.ID.String(),
				Result:         "success",
			})
		}()
		close(start)
		wg.Wait()
		if applyErr != nil {
			t.Fatalf("ApplyApproved 不应泄漏死锁或驱动错误: %v", applyErr)
		}
		if directErr != nil {
			converted := errors.FromError(directErr)
			if converted == nil || converted.Reason != "ORDER_NOT_LOCKED" {
				t.Fatalf("直解失败只能是审批先解锁后的 ORDER_NOT_LOCKED，得到: %v", directErr)
			}
		}

		orderAfter, err := data.db.Order.Get(ctx, orderN.ID)
		if err != nil || orderAfter.LockedAt != nil || orderAfter.Version != 3 {
			t.Fatalf("竞争后订单必须仅解锁并递增一次版本: order=%#v err=%v", orderAfter, err)
		}
		lockRecord, err := data.db.OrderLockRecord.Query().Where(orderlockrecordent.OrderIDEQ(orderN.ID), orderlockrecordent.GenerationEQ(1)).Only(ctx)
		if err != nil || lockRecord.UnlockedAt == nil || lockRecord.UnlockRequestID == nil || lockRecord.OrderVersionAtUnlock == nil || *lockRecord.OrderVersionAtUnlock != 3 {
			t.Fatalf("锁事实必须只关闭一次并指向实际解锁请求: record=%#v err=%v", lockRecord, err)
		}
		approvedRequests, err := data.db.OrderUnlockRequest.Query().Where(orderunlockrequestent.OrderIDEQ(orderN.ID), orderunlockrequestent.StatusEQ(orderunlockrequestent.StatusAPPROVED)).All(ctx)
		if err != nil || len(approvedRequests) != 1 || approvedRequests[0].ID != *lockRecord.UnlockRequestID {
			t.Fatalf("必须恰好一个 APPROVED 请求且与锁事实一致: requests=%#v record=%#v err=%v", approvedRequests, lockRecord, err)
		}
		oldRequest, err := data.db.OrderUnlockRequest.Get(ctx, requestID)
		if err != nil {
			t.Fatal(err)
		}
		inbox, err := data.db.DingTalkApprovalInboxEvent.Get(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if directErr == nil {
			if directResult == nil || oldRequest.Status != orderunlockrequestent.StatusSTALE || oldRequest.SupersededByRequestID == nil || *oldRequest.SupersededByRequestID != directResult.Request.ID ||
				inbox.Status != dingtalkapprovalinboxeventent.StatusIGNORED || inbox.ResultCode == nil || *inbox.ResultCode != "REQUEST_SUPERSEDED" {
				t.Fatalf("直解胜出时旧审批与 Inbox 取代关系错误: direct=%#v old=%#v inbox=%#v", directResult, oldRequest, inbox)
			}
		} else if oldRequest.Status != orderunlockrequestent.StatusAPPROVED || oldRequest.SupersededByRequestID != nil ||
			inbox.Status != dingtalkapprovalinboxeventent.StatusPROCESSED || inbox.ResultCode == nil || *inbox.ResultCode != "APPROVED_APPLIED" {
			t.Fatalf("审批胜出时旧申请和 Inbox 终态错误: old=%#v inbox=%#v", oldRequest, inbox)
		}
		dingAuditCount, err := data.db.AuditLog.Query().Where(auditlogent.ActionEQ("order.unlock.dingtalk_approved"), auditlogent.ResourceIDEQ(orderN.ID.String())).Count(ctx)
		if err != nil {
			t.Fatal(err)
		}
		directAuditCount, err := data.db.AuditLog.Query().Where(auditlogent.ActionEQ("order.unlock.role_direct_race"), auditlogent.ResourceIDEQ(orderN.ID.String())).Count(ctx)
		if err != nil || dingAuditCount+directAuditCount != 1 {
			t.Fatalf("竞争只能写一条实际解锁审计: dingtalk=%d direct=%d err=%v", dingAuditCount, directAuditCount, err)
		}
	})

	t.Run("Inbox 处理租约未过期不重复领取且过期可接管", func(t *testing.T) {
		approvalRepo := NewDingTalkApprovalRepo(data)
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{EventID: "EVENT-LEASE-" + suffix, CorpID: "CORP", EventType: "bpms_instance_change", ProcessInstanceID: "PROC-MISSING-" + suffix, EncryptedPayloadHash: strings.Repeat("d", 64)}); err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC().Add(time.Second)
		first, err := approvalRepo.ClaimInbox(ctx, time.Minute, now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := approvalRepo.ClaimInbox(ctx, time.Minute, now.Add(30*time.Second)); err == nil {
			t.Fatal("未过期 PROCESSING Inbox 不得被第二个 Worker 领取")
		}
		second, err := approvalRepo.ClaimInbox(ctx, time.Minute, now.Add(61*time.Second))
		if err != nil || second.ID != first.ID || second.LeaseToken == first.LeaseToken {
			t.Fatalf("过期接管失败: first=%#v second=%#v err=%v", first, second, err)
		}
		if err := approvalRepo.IgnoreInbox(ctx, second, "TEST_DONE"); err != nil {
			t.Fatal(err)
		}
	})
}
