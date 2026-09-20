package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// TestOrderLockGrantScopes_PostgresFlows 覆盖 ACC-01 的订单锁资格矩阵：
// 组织内角色、上级组织树角色、ALL 范围 administrator、bootstrap admin、
// 无权限用户与权限撤销后的旧候选回调。资格口径必须与候选快照、直接解锁、
// 锁单命令和钉钉回调复核完全一致。
func TestOrderLockGrantScopes_PostgresFlows(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		source = "postgresql://roncin:roncin_local_dev@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable"
		t.Setenv("RONCIN_INTEGRATION_DATABASE_SOURCE", source)
	}
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]

	// 组织树：parent → child（目标组织）、parent → sibling（兄弟组织）。
	parent, err := data.db.Organization.Create().
		SetCode("GRANT-P-" + suffix).
		SetName("锁资格上级组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建上级组织失败: %v", err)
	}
	child, err := data.db.Organization.Create().
		SetCode("GRANT-C-" + suffix).
		SetName("锁资格目标组织-" + suffix).
		SetKind("department").
		SetParentID(parent.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建目标组织失败: %v", err)
	}
	sibling, err := data.db.Organization.Create().
		SetCode("GRANT-S-" + suffix).
		SetName("锁资格兄弟组织-" + suffix).
		SetKind("department").
		SetParentID(parent.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建兄弟组织失败: %v", err)
	}

	ensurePermission := func(key, name string) *ent.Permission {
		permission, queryErr := data.db.Permission.Query().Where(permissionent.KeyEQ(key)).First(ctx)
		if ent.IsNotFound(queryErr) {
			permission, queryErr = data.db.Permission.Create().
				SetKey(key).
				SetName(name).
				SetDescription("锁资格矩阵集成测试").
				SetGroup("order").
				Save(ctx)
		}
		if queryErr != nil {
			t.Fatalf("准备权限 %s 失败: %v", key, queryErr)
		}
		return permission
	}
	lockPerm := ensurePermission("business.order.si.lock", "海运进口订单锁定")
	updatePerm := ensurePermission("business.order.si.update", "海运进口订单修改")

	newUser := func(label string, bootstrap bool) *ent.User {
		u, userErr := data.db.User.Create().
			SetDisplayName("锁资格" + label + "-" + suffix).
			SetDingtalkUserid("dt-grant-" + label + "-" + suffix).
			SetEnabled(true).
			SetIsBootstrapAdmin(bootstrap).
			Save(ctx)
		if userErr != nil {
			t.Fatalf("创建用户 %s 失败: %v", label, userErr)
		}
		return u
	}
	newMembership := func(u *ent.User, org *ent.Organization) *ent.Membership {
		m, membershipErr := data.db.Membership.Create().
			SetUserID(u.ID).
			SetOrganizationID(org.ID).
			SetPrimary(true).
			SetEnabled(true).
			Save(ctx)
		if membershipErr != nil {
			t.Fatalf("创建成员关系失败: %v", membershipErr)
		}
		return m
	}
	newRole := func(org *ent.Organization, code string, scope roleent.DataScope, permissions ...*ent.Permission) *ent.Role {
		r, roleErr := data.db.Role.Create().
			SetOrganizationID(org.ID).
			SetCode(code).
			SetName(code + "-" + suffix).
			SetDataScope(scope).
			AddPermissions(permissions...).
			Save(ctx)
		if roleErr != nil {
			t.Fatalf("创建角色 %s 失败: %v", code, roleErr)
		}
		return r
	}
	assign := func(m *ent.Membership, r *ent.Role) {
		if _, assignmentErr := data.db.RoleAssignment.Create().SetMembershipID(m.ID).SetRoleID(r.ID).Save(ctx); assignmentErr != nil {
			t.Fatalf("分配角色 %s 失败: %v", r.Code, assignmentErr)
		}
	}

	// 组织内业务角色：目标组织 ORG 范围，持 lock+update。
	orgUser := newUser("org", false)
	orgMembership := newMembership(orgUser, child)
	orgRole := newRole(child, "si_locker_"+suffix, roleent.DataScopeOrganization, lockPerm, updatePerm)
	assign(orgMembership, orgRole)

	// 上级组织树角色：membership 在 parent，TREE 范围覆盖 child。
	treeUser := newUser("tree", false)
	treeMembership := newMembership(treeUser, parent)
	treeRole := newRole(parent, "si_supervisor_"+suffix, roleent.DataScopeOrganizationTree, lockPerm)
	assign(treeMembership, treeRole)

	// ALL 范围 administrator：角色代码恰为 administrator，membership 在 parent。
	// 旧口径按角色代码排除 administrator，新口径只看权限与范围。
	allAdminUser := newUser("alladmin", false)
	allAdminMembership := newMembership(allAdminUser, parent)
	allAdminRole := newRole(parent, "administrator", roleent.DataScopeAll, lockPerm)
	assign(allAdminMembership, allAdminRole)

	// 无锁权限的 ALL administrator：角色代码同样恰为 administrator，证明资格
	// 来自真实持有的权限与范围，而不是角色代码。
	adminNoPermUser := newUser("adminnoperm", false)
	adminNoPermMembership := newMembership(adminNoPermUser, sibling)
	noPermAdminRole := newRole(sibling, "administrator", roleent.DataScopeAll)
	assign(adminNoPermMembership, noPermAdminRole)

	// 普通编辑人：目标组织 ORG 范围，仅 update，用于发起审批并验证无锁资格。
	normalUser := newUser("normal", false)
	normalMembership := newMembership(normalUser, child)
	normalRole := newRole(child, "si_operator_"+suffix, roleent.DataScopeOrganization, updatePerm)
	assign(normalMembership, normalRole)

	// 上级组织 ORG 范围：只覆盖 parent 自身，不得覆盖 child。
	parentOrgScopeUser := newUser("parentorg", false)
	parentOrgScopeMembership := newMembership(parentOrgScopeUser, parent)
	parentOrgScopeRole := newRole(parent, "si_parent_only_"+suffix, roleent.DataScopeOrganization, lockPerm)
	assign(parentOrgScopeMembership, parentOrgScopeRole)

	// 兄弟组织 TREE 范围：只覆盖 sibling 子树，不得覆盖 child。
	siblingTreeUser := newUser("siblingtree", false)
	siblingTreeMembership := newMembership(siblingTreeUser, sibling)
	siblingTreeRole := newRole(sibling, "si_sibling_tree_"+suffix, roleent.DataScopeOrganizationTree, lockPerm)
	assign(siblingTreeMembership, siblingTreeRole)

	bootstrapUser := newUser("bootstrap", true)

	users := []*ent.User{orgUser, treeUser, allAdminUser, adminNoPermUser, normalUser, parentOrgScopeUser, siblingTreeUser, bootstrapUser}
	t.Cleanup(func() {
		_, _ = data.sqlDB.ExecContext(ctx, `
			DELETE FROM ding_talk_approval_inbox_events WHERE organization_id = $1;
			DELETE FROM order_unlock_approver_candidates WHERE request_id IN (SELECT id FROM order_unlock_requests WHERE organization_id = $1);
			DELETE FROM ding_talk_approval_dispatches WHERE organization_id = $1;
			DELETE FROM background_tasks WHERE organization_id = $1;
			DELETE FROM order_unlock_requests WHERE organization_id = $1;
			DELETE FROM order_lock_records WHERE organization_id = $1;
			DELETE FROM orders WHERE organization_id = $1;
			DELETE FROM partners WHERE organization_id = $1;
			DELETE FROM role_assignments WHERE membership_id IN (SELECT id FROM memberships WHERE organization_id IN ($2, $3, $4));
			DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE organization_id IN ($2, $3, $4));
			DELETE FROM roles WHERE organization_id IN ($2, $3, $4);
			DELETE FROM memberships WHERE organization_id IN ($2, $3, $4);
			DELETE FROM users WHERE id = ANY($5::uuid[]);
			DELETE FROM organizations WHERE id IN ($2, $3, $4);
		`, child.ID, child.ID, sibling.ID, parent.ID, pqUUIDArray(users))
	})

	customer, err := data.db.Partner.Create().
		SetOrganizationID(child.ID).
		SetCode("GRANT-CUST-" + suffix).
		SetLegalName("锁资格测试客户-" + suffix).
		SetNormalizedName("锁资格测试客户-" + suffix).
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

	orderSeq := 0
	createSIOrder := func() *ent.Order {
		orderSeq++
		o, orderErr := data.db.Order.Create().
			SetIdempotencyKey(uuid.NewString()).
			SetOrganizationID(child.ID).
			SetOrderNo("SI-GRANT-" + suffix + "-" + string(rune('A'+orderSeq))).
			SetCustomerID(customer.ID).
			SetBusinessType(orderent.BusinessTypeSI).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetShipmentType(orderent.ShipmentTypeFCL).
			SetFlowStatus(orderent.FlowStatusDRAFT).
			SetTerminationStatus(orderent.TerminationStatusACTIVE).
			SetClosureStatus(orderent.ClosureStatusOPEN).
			SetVersion(1).
			Save(ctx)
		if orderErr != nil {
			t.Fatalf("创建 SI 订单失败: %v", orderErr)
		}
		return o
	}

	// 中间件会按订单组织重写有效身份：所有 Principal 的当前组织均为目标组织 child。
	principalFor := func(u *ent.User, grants ...biz.RoleGrant) *biz.Principal {
		return &biz.Principal{
			UserID:           u.ID,
			DisplayName:      u.DisplayName,
			IsBootstrapAdmin: u.IsBootstrapAdmin,
			Organization:     biz.Organization{Kind: biz.OrganizationKindCompany, ID: child.ID},
			OrganizationNodes: []biz.OrganizationScopeNode{
				{Kind: biz.OrganizationKindCompany, ID: parent.ID},
				{Kind: biz.OrganizationKindCompany, ID: child.ID, ParentID: &parent.ID},
				{Kind: biz.OrganizationKindCompany, ID: sibling.ID, ParentID: &parent.ID},
			},
			RoleGrants: grants,
		}
	}
	grantFor := func(r *ent.Role, permissionKeys ...string) biz.RoleGrant {
		permissionSet := make(map[string]struct{}, len(permissionKeys))
		for _, key := range permissionKeys {
			permissionSet[key] = struct{}{}
		}
		return biz.RoleGrant{
			RoleID:      r.ID,
			RoleCode:    r.Code,
			DataScope:   biz.DataScope(r.DataScope),
			Permissions: permissionSet,
		}
	}

	orgPrincipal := principalFor(orgUser, grantFor(orgRole, "business.order.si.lock", "business.order.si.update"))
	treePrincipal := principalFor(treeUser, grantFor(treeRole, "business.order.si.lock"))
	allAdminPrincipal := principalFor(allAdminUser, grantFor(allAdminRole, "business.order.si.lock"))
	normalPrincipal := principalFor(normalUser, grantFor(normalRole, "business.order.si.update"))
	bootstrapPrincipal := principalFor(bootstrapUser)

	orderLockRepo := NewOrderLockRepo(data, &conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:             true,
		CorpId:              "CORP-GRANT-TEST",
		ApprovalProcessCode: "PROC-GRANT-TEST",
		EventToken:          "EVENT-TOKEN-GRANT",
		EventAesKey:         "EVENT-AES-GRANT",
	}})

	t.Run("组织内角色按ORG范围覆盖目标组织并可直解", func(t *testing.T) {
		order := createSIOrder()
		state, err := orderLockRepo.GetOrderLockState(ctx, child.ID, order.ID, orgPrincipal)
		if err != nil || !state.CanLock || len(state.LockBlockedReasons) != 0 {
			t.Fatalf("组织内角色锁状态 = %#v, err=%v", state, err)
		}
		if _, err := orderLockRepo.LockOrder(ctx, orgPrincipal, order.ID, 1, "grant-org-lock-"+suffix, nil); err != nil {
			t.Fatalf("组织内角色锁定失败: %v", err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, orgPrincipal, order.ID, 2, "grant-org-unlock-"+suffix, nil, nil)
		if err != nil || result.Request.Route != biz.UnlockRouteRoleDirect || result.Request.Status != biz.UnlockStatusApproved {
			t.Fatalf("组织内角色直解结果异常: %#v err=%v", result, err)
		}
	})

	t.Run("上级组织树角色覆盖下级订单并按真实membership快照", func(t *testing.T) {
		order := createSIOrder()
		state, err := orderLockRepo.GetOrderLockState(ctx, child.ID, order.ID, treePrincipal)
		if err != nil || !state.CanLock {
			t.Fatalf("上级树角色锁状态 = %#v, err=%v", state, err)
		}
		if _, err := orderLockRepo.LockOrder(ctx, treePrincipal, order.ID, 1, "grant-tree-lock-"+suffix, nil); err != nil {
			t.Fatalf("上级树角色锁定失败: %v", err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, treePrincipal, order.ID, 2, "grant-tree-unlock-"+suffix, nil, nil)
		if err != nil || result.Request.Route != biz.UnlockRouteRoleDirect {
			t.Fatalf("上级树角色直解结果异常: %#v err=%v", result, err)
		}
	})

	t.Run("ALL范围administrator不再被角色代码排除且无权限administrator仍拒绝", func(t *testing.T) {
		order := createSIOrder()
		if _, err := orderLockRepo.LockOrder(ctx, allAdminPrincipal, order.ID, 1, "grant-alladmin-lock-"+suffix, nil); err != nil {
			t.Fatalf("ALL 范围 administrator 持锁权限仍被拒绝: %v", err)
		}
		state, err := orderLockRepo.GetOrderLockState(ctx, child.ID, order.ID, allAdminPrincipal)
		if err != nil || !state.CanRoleDirectUnlock {
			t.Fatalf("ALL 范围 administrator 应具备角色直解入口: %#v err=%v", state, err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, allAdminPrincipal, order.ID, 2, "grant-alladmin-unlock-"+suffix, nil, nil)
		if err != nil || result.Request.Route != biz.UnlockRouteRoleDirect {
			t.Fatalf("ALL 范围 administrator 直解结果异常: %#v err=%v", result, err)
		}

		noPermOrder := createSIOrder()
		adminNoPermPrincipal := principalFor(adminNoPermUser, grantFor(noPermAdminRole))
		_, err = orderLockRepo.LockOrder(ctx, adminNoPermPrincipal, noPermOrder.ID, 1, "grant-adminnoperm-lock-"+suffix, nil)
		if kErr := errors.FromError(err); kErr == nil || kErr.Reason != "ORDER_LOCK_ROLE_REQUIRED" {
			t.Fatalf("无锁权限的 administrator 期望 ORDER_LOCK_ROLE_REQUIRED，得到: %v", err)
		}
	})

	t.Run("bootstrap显式具备锁单与应急解锁且不进入候选池", func(t *testing.T) {
		order := createSIOrder()
		state, err := orderLockRepo.GetOrderLockState(ctx, child.ID, order.ID, bootstrapPrincipal)
		if err != nil || !state.CanLock || len(state.LockBlockedReasons) != 0 {
			t.Fatalf("bootstrap 锁状态 = %#v, err=%v", state, err)
		}
		lockResult, err := orderLockRepo.LockOrder(ctx, bootstrapPrincipal, order.ID, 1, "grant-bootstrap-lock-"+suffix, nil)
		if err != nil {
			t.Fatalf("bootstrap 锁定失败: %v", err)
		}
		dbOrder, err := data.db.Order.Get(ctx, order.ID)
		if err != nil || dbOrder.LockedBy == nil || *dbOrder.LockedBy != bootstrapUser.ID {
			t.Fatalf("bootstrap 锁定人字段异常: order=%#v err=%v", dbOrder, err)
		}
		lockedState, err := orderLockRepo.GetOrderLockState(ctx, child.ID, order.ID, bootstrapPrincipal)
		if err != nil || !lockedState.CanAdminEmergencyUnlock || lockedState.CanRoleDirectUnlock {
			t.Fatalf("bootstrap 解锁入口异常: %#v err=%v", lockedState, err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, bootstrapPrincipal, order.ID, 2, "grant-bootstrap-unlock-"+suffix, nil, nil)
		if err != nil || result.Request.Route != biz.UnlockRouteAdminEmergency || result.Request.Status != biz.UnlockStatusApproved {
			t.Fatalf("bootstrap 应急解锁结果异常: %#v err=%v", result, err)
		}
		if lockResult.LockRecord.UnlockMode != nil {
			t.Fatalf("锁定时不应有解锁模式: %#v", lockResult.LockRecord)
		}
	})

	t.Run("ORG范围只覆盖本组织且树范围不覆盖兄弟组织", func(t *testing.T) {
		parentOrgOrder := createSIOrder()
		parentOrgPrincipal := principalFor(parentOrgScopeUser, grantFor(parentOrgScopeRole, "business.order.si.lock"))
		_, err := orderLockRepo.LockOrder(ctx, parentOrgPrincipal, parentOrgOrder.ID, 1, "grant-parentorg-lock-"+suffix, nil)
		if kErr := errors.FromError(err); kErr == nil || kErr.Reason != "ORDER_LOCK_ROLE_REQUIRED" {
			t.Fatalf("上级组织 ORG 范围锁定 child 订单期望拒绝，得到: %v", err)
		}

		siblingOrder := createSIOrder()
		siblingTreePrincipal := principalFor(siblingTreeUser, grantFor(siblingTreeRole, "business.order.si.lock"))
		_, err = orderLockRepo.LockOrder(ctx, siblingTreePrincipal, siblingOrder.ID, 1, "grant-sibling-lock-"+suffix, nil)
		if kErr := errors.FromError(err); kErr == nil || kErr.Reason != "ORDER_LOCK_ROLE_REQUIRED" {
			t.Fatalf("兄弟组织 TREE 范围锁定 child 订单期望拒绝，得到: %v", err)
		}

		normalOrder := createSIOrder()
		_, err = orderLockRepo.LockOrder(ctx, normalPrincipal, normalOrder.ID, 1, "grant-normal-lock-"+suffix, nil)
		if kErr := errors.FromError(err); kErr == nil || kErr.Reason != "ORDER_LOCK_ROLE_REQUIRED" {
			t.Fatalf("无锁权限用户期望 ORDER_LOCK_ROLE_REQUIRED，得到: %v", err)
		}
	})

	t.Run("候选池按统一口径生成且撤权后旧候选回调稳定拒绝", func(t *testing.T) {
		order := createSIOrder()
		if _, err := orderLockRepo.LockOrder(ctx, orgPrincipal, order.ID, 1, "grant-pipeline-lock-"+suffix, nil); err != nil {
			t.Fatalf("准备锁定失败: %v", err)
		}
		result, err := orderLockRepo.RequestOrderUnlock(ctx, normalPrincipal, order.ID, 2, "grant-pipeline-request-"+suffix, nil, nil)
		if err != nil || result.Request.Route != biz.UnlockRouteDingTalkApproval || result.Request.Status != biz.UnlockStatusPendingDispatch {
			t.Fatalf("普通编辑人应进入钉钉审批: %#v err=%v", result, err)
		}

		reqRecord, err := data.db.OrderUnlockRequest.Query().
			Where(orderunlockrequestent.OrderIDEQ(order.ID), orderunlockrequestent.LockGenerationEQ(1)).
			WithApproverCandidates().
			Only(ctx)
		if err != nil {
			t.Fatalf("读取解锁请求失败: %v", err)
		}
		candidatesByUser := make(map[uuid.UUID]*ent.OrderUnlockApproverCandidate, len(reqRecord.Edges.ApproverCandidates))
		for _, candidate := range reqRecord.Edges.ApproverCandidates {
			candidatesByUser[candidate.UserID] = candidate
		}
		if len(candidatesByUser) != 3 {
			t.Fatalf("候选池应恰有组织内/树/ALL 三类合格审批人，实际: %d -> %#v", len(candidatesByUser), candidatesByUser)
		}
		for _, u := range []*ent.User{orgUser, treeUser, allAdminUser} {
			if _, ok := candidatesByUser[u.ID]; !ok {
				t.Fatalf("候选池缺少合格审批人 %s", u.DisplayName)
			}
		}
		if _, ok := candidatesByUser[bootstrapUser.ID]; ok {
			t.Fatal("bootstrap admin 不得进入普通审批候选池")
		}
		if candidate := candidatesByUser[treeUser.ID]; candidate.MembershipID != treeMembership.ID || candidate.RoleID != treeRole.ID {
			t.Fatalf("树范围候选必须快照真实的上级 membership/role: %#v", candidate)
		}
		if candidate := candidatesByUser[orgUser.ID]; candidate.MembershipID != orgMembership.ID {
			t.Fatalf("组织内候选必须快照真实 membership: %#v", candidate)
		}

		approvalRepo := NewDingTalkApprovalRepo(data)
		requestEntity, err := data.db.OrderUnlockRequest.Get(ctx, result.Request.ID)
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
			t.Fatalf("建立审批派发租约失败: %v", err)
		}
		claimed := &biz.BackgroundTask{ID: dispatchEntity.BackgroundTaskID, OrganizationID: child.ID, LeaseToken: &leaseToken}
		if _, err := approvalRepo.PrepareDispatch(ctx, claimed); err != nil {
			t.Fatalf("准备审批派发失败: %v", err)
		}
		instanceID := "PROC-GRANT-APPLY-" + suffix
		if err := approvalRepo.FinishDispatch(ctx, claimed, &biz.DingTalkApprovalDispatchOutcome{ProcessInstanceID: instanceID}, time.Now().UTC()); err != nil {
			t.Fatalf("保存审批实例失败: %v", err)
		}
		if err := approvalRepo.StoreCallback(ctx, &biz.DingTalkApprovalCallbackEvent{
			EventID:              "EVENT-GRANT-APPLY-" + suffix,
			CorpID:               "CORP-GRANT-TEST",
			EventType:            "bpms_instance_change",
			ProcessInstanceID:    instanceID,
			EncryptedPayloadHash: strings.Repeat("a", 64),
		}); err != nil {
			t.Fatalf("写入审批 Inbox 失败: %v", err)
		}
		job, err := approvalRepo.ClaimInbox(ctx, time.Minute, time.Now().UTC().Add(time.Second))
		if err != nil {
			t.Fatalf("领取审批 Inbox 失败: %v", err)
		}
		decision := &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionApproved, ApproverUserID: "dt-grant-tree-" + suffix}
		requestID, shouldApply, err := approvalRepo.PrepareApproved(ctx, job, decision, time.Now().UTC())
		if err != nil || !shouldApply || requestID != result.Request.ID {
			t.Fatalf("保存已同意待生效失败: id=%s apply=%t err=%v", requestID, shouldApply, err)
		}

		// 撤销树角色的锁权限：回调必须以同一资格口径复核并稳定拒绝。
		if _, err := data.db.Role.UpdateOneID(treeRole.ID).RemovePermissions(lockPerm).Save(ctx); err != nil {
			t.Fatalf("撤销树角色锁权限失败: %v", err)
		}
		if err := approvalRepo.ApplyApproved(ctx, job, requestID, decision, time.Now().UTC()); err != nil {
			t.Fatalf("撤权后回调执行失败: %v", err)
		}
		requestAfter, err := data.db.OrderUnlockRequest.Get(ctx, requestID)
		if err != nil || requestAfter.Status != orderunlockrequestent.StatusSTALE ||
			requestAfter.FailureCode == nil || *requestAfter.FailureCode != "APPROVER_NOT_QUALIFIED" {
			t.Fatalf("撤权后回调未稳定拒绝: request=%#v err=%v", requestAfter, err)
		}
		orderAfter, err := data.db.Order.Get(ctx, order.ID)
		if err != nil || orderAfter.LockedAt == nil || orderAfter.Version != 2 {
			t.Fatalf("撤权后回调不得解锁订单: order=%#v err=%v", orderAfter, err)
		}
		lockRecord, err := data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.OrderIDEQ(order.ID), orderlockrecordent.GenerationEQ(1)).Only(ctx)
		if err != nil || lockRecord.UnlockedAt != nil {
			t.Fatalf("撤权后锁事实不应关闭: record=%#v err=%v", lockRecord, err)
		}
	})
}

// pqUUIDArray 把测试用户 ID 集合转为 PostgreSQL uuid[] 字面量。
func pqUUIDArray(users []*ent.User) string {
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID.String())
	}
	return "{" + strings.Join(ids, ",") + "}"
}
