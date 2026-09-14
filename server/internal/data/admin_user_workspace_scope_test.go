package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
)

func TestAdminWorkspaceScopeIDs(t *testing.T) {
	hqID := uuid.New()
	companyAID := uuid.New()
	deptID := uuid.New()
	companyBID := uuid.New()
	disabledID := uuid.New()
	nodes := map[uuid.UUID]authOrgNode{
		hqID:       {ID: hqID, ParentID: nil, Kind: string(organization.KindHeadquarters), Enabled: true},
		companyAID: {ID: companyAID, ParentID: &hqID, Kind: string(organization.KindCompany), Enabled: true},
		deptID:     {ID: deptID, ParentID: &companyAID, Kind: string(organization.KindDepartment), Enabled: true},
		companyBID: {ID: companyBID, ParentID: &hqID, Kind: string(organization.KindCompany), Enabled: true},
		disabledID: {ID: disabledID, ParentID: &hqID, Kind: string(organization.KindCompany), Enabled: false},
	}

	headquartersScope := adminWorkspaceScopeIDs(nodes, hqID)
	if len(headquartersScope) != 4 {
		t.Fatalf("总部工作台范围应为全树启用组织（4 个）: %v", headquartersScope)
	}
	for _, excluded := range []uuid.UUID{disabledID} {
		for _, scopeID := range headquartersScope {
			if scopeID == excluded {
				t.Fatalf("总部工作台范围不应包含停用组织: %v", headquartersScope)
			}
		}
	}

	companyScope := adminWorkspaceScopeIDs(nodes, companyAID)
	if len(companyScope) != 2 {
		t.Fatalf("公司工作台范围应为公司子树（含公司节点与部门，共 2 个）: %v", companyScope)
	}
	if !containsUUID(companyScope, companyAID) || !containsUUID(companyScope, deptID) {
		t.Fatalf("公司工作台范围应包含公司节点本身与子树部门: %v", companyScope)
	}
	for _, scopeID := range companyScope {
		if scopeID == companyBID {
			t.Fatalf("公司工作台范围不应包含兄弟公司: %v", companyScope)
		}
	}

	if scope := adminWorkspaceScopeIDs(nodes, deptID); len(scope) != 0 {
		t.Fatalf("部门节点不是工作台，范围应为空: %v", scope)
	}
	if scope := adminWorkspaceScopeIDs(nodes, disabledID); len(scope) != 0 {
		t.Fatalf("停用工作台范围应为空: %v", scope)
	}
	if scope := adminWorkspaceScopeIDs(nodes, uuid.New()); len(scope) != 0 {
		t.Fatalf("未知组织范围应为空: %v", scope)
	}
}

func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestPreferAdminAnchorMembership(t *testing.T) {
	earliest := time.Now().Add(-3 * time.Hour)
	middle := time.Now().Add(-2 * time.Hour)
	latest := time.Now().Add(-time.Hour)
	newMembership := func(primary, enabled bool, createdAt time.Time) *ent.Membership {
		return &ent.Membership{Primary: primary, Enabled: enabled, CreatedAt: createdAt, ID: uuid.New()}
	}

	tests := []struct {
		name        string
		candidates  []*ent.Membership
		wantPrimary bool
		wantEnabled bool
		wantCreated time.Time
	}{
		{
			name:        "primary 优先于启用关系",
			candidates:  []*ent.Membership{newMembership(false, true, earliest), newMembership(true, false, latest)},
			wantPrimary: true,
			wantEnabled: false,
			wantCreated: latest,
		},
		{
			name:        "无 primary 时取最早创建的启用关系",
			candidates:  []*ent.Membership{newMembership(false, true, latest), newMembership(false, true, earliest), newMembership(false, false, middle)},
			wantPrimary: false,
			wantEnabled: true,
			wantCreated: earliest,
		},
		{
			name:        "全部停用时取最早创建的任一关系",
			candidates:  []*ent.Membership{newMembership(false, false, latest), newMembership(false, false, earliest)},
			wantPrimary: false,
			wantEnabled: false,
			wantCreated: earliest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			anchor := preferAdminAnchorMembership(test.candidates)
			if anchor == nil {
				t.Fatal("锚定成员关系不应为空")
			}
			if anchor.Primary != test.wantPrimary || anchor.Enabled != test.wantEnabled || !anchor.CreatedAt.Equal(test.wantCreated) {
				t.Fatalf("锚定成员关系 = primary=%v enabled=%v createdAt=%v, want primary=%v enabled=%v createdAt=%v",
					anchor.Primary, anchor.Enabled, anchor.CreatedAt, test.wantPrimary, test.wantEnabled, test.wantCreated)
			}
		})
	}
	if anchor := preferAdminAnchorMembership(nil); anchor != nil {
		t.Fatalf("空候选应返回 nil: %#v", anchor)
	}
}

// TestAdminUserWorkspaceScopePostgres 在隔离 Schema 上验证用户管理的工作台范围口径：
// 总部工作台可见/可编辑全树用户（含仅有公司、部门成员关系的用户）；公司工作台只见
// 本公司子树；移除工作台节点成员关系后用户仍可编辑；角色按锚定成员关系所在组织校验。
func TestAdminUserWorkspaceScopePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	adminRepo := &adminRepo{data: data}

	headquarters, err := data.db.Organization.Create().
		SetCode("HQ").
		SetName("总部").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部: %v", err)
	}
	companyA, err := data.db.Organization.Create().
		SetCode("COMPANY-A").
		SetName("公司A").
		SetKind("company").
		SetParentID(headquarters.ID).
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司A: %v", err)
	}
	deptA1, err := data.db.Organization.Create().
		SetCode("DEPT-A1").
		SetName("公司A部门1").
		SetKind("department").
		SetParentID(companyA.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门: %v", err)
	}
	companyB, err := data.db.Organization.Create().
		SetCode("COMPANY-B").
		SetName("公司B").
		SetKind("company").
		SetParentID(headquarters.ID).
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司B: %v", err)
	}
	roleHQ, err := data.db.Role.Create().SetOrganizationID(headquarters.ID).SetCode("hq_operator").SetName("总部操作员").SetDataScope("organization").Save(ctx)
	if err != nil {
		t.Fatalf("创建总部角色: %v", err)
	}
	roleA, err := data.db.Role.Create().SetOrganizationID(companyA.ID).SetCode("a_operator").SetName("A公司操作员").SetDataScope("organization").Save(ctx)
	if err != nil {
		t.Fatalf("创建公司A角色: %v", err)
	}
	roleDept, err := data.db.Role.Create().SetOrganizationID(deptA1.ID).SetCode("dept_operator").SetName("部门操作员").SetDataScope("organization").Save(ctx)
	if err != nil {
		t.Fatalf("创建部门角色: %v", err)
	}
	roleB, err := data.db.Role.Create().SetOrganizationID(companyB.ID).SetCode("b_operator").SetName("B公司操作员").SetDataScope("organization").Save(ctx)
	if err != nil {
		t.Fatalf("创建公司B角色: %v", err)
	}

	newUser := func(name, username string) *ent.User {
		account, createErr := data.db.User.Create().SetUsername(username).SetDisplayName(name).SetEnabled(true).Save(ctx)
		if createErr != nil {
			t.Fatalf("创建用户 %s: %v", name, createErr)
		}
		return account
	}
	newMembership := func(userID, organizationID uuid.UUID, primary bool) *ent.Membership {
		record, createErr := data.db.Membership.Create().SetUserID(userID).SetOrganizationID(organizationID).SetPrimary(primary).SetEnabled(true).Save(ctx)
		if createErr != nil {
			t.Fatalf("创建成员关系: %v", createErr)
		}
		return record
	}

	// deptUser 仅在公司A的部门有成员关系：总部工作台可见可编辑，公司B工作台不可见。
	deptUser := newUser("部门用户", "dept.user")
	deptMembership := newMembership(deptUser.ID, deptA1.ID, true)
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(deptMembership.ID).SetRoleID(roleDept.ID).Save(ctx); err != nil {
		t.Fatalf("分配部门角色: %v", err)
	}

	// removedUser 回归用户报告场景：总部成员关系被移除后仍保留公司A成员关系。
	removedUser := newUser("被移除用户", "removed.user")
	hqMembership := newMembership(removedUser.ID, headquarters.ID, true)
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(hqMembership.ID).SetRoleID(roleHQ.ID).Save(ctx); err != nil {
		t.Fatalf("分配总部角色: %v", err)
	}
	companyAMembership := newMembership(removedUser.ID, companyA.ID, false)
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(companyAMembership.ID).SetRoleID(roleA.ID).Save(ctx); err != nil {
		t.Fatalf("分配公司A角色: %v", err)
	}

	// outsiderUser 仅在公司B有成员关系：公司A工作台不可见、不可办理离职。
	outsiderUser := newUser("外部用户", "outsider.user")
	newMembership(outsiderUser.ID, companyB.ID, true)

	// 1. 总部工作台列表包含仅有部门/公司成员关系的用户。
	headquartersList, err := adminRepo.ListUsers(ctx, headquarters.ID, biz.AdminUserListOptions{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("总部工作台列表: %v", err)
	}
	headquartersRows := listUserIDs(headquartersList)
	for _, expected := range []uuid.UUID{deptUser.ID, removedUser.ID, outsiderUser.ID} {
		if !headquartersRows[expected] {
			t.Fatalf("总部工作台列表缺少用户（全树口径） %v: %v", expected, headquartersRows)
		}
	}

	// 2. 公司工作台只见本公司子树；跨公司用户 404。
	companyAList, err := adminRepo.ListUsers(ctx, companyA.ID, biz.AdminUserListOptions{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("公司A工作台列表: %v", err)
	}
	companyARows := listUserIDs(companyAList)
	if !companyARows[deptUser.ID] || !companyARows[removedUser.ID] {
		t.Fatalf("公司A工作台应包含子树用户: %v", companyARows)
	}
	if companyARows[outsiderUser.ID] {
		t.Fatalf("公司A工作台不应包含公司B用户: %v", companyARows)
	}
	companyBList, err := adminRepo.ListUsers(ctx, companyB.ID, biz.AdminUserListOptions{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("公司B工作台列表: %v", err)
	}
	if listUserIDs(companyBList)[deptUser.ID] {
		t.Fatal("公司B工作台不应包含公司A部门用户")
	}
	if _, err := adminRepo.GetUser(ctx, companyB.ID, deptUser.ID); err != biz.ErrAdminUserNotFound {
		t.Fatalf("跨公司读取用户 error = %v, want ErrAdminUserNotFound", err)
	}

	// 3. 锚定成员关系按 primary 优先：部门用户锚定部门关系，角色与 CurrentOrganizationID 随锚定关系。
	deptView, err := adminRepo.GetUser(ctx, headquarters.ID, deptUser.ID)
	if err != nil {
		t.Fatalf("总部工作台读取部门用户: %v", err)
	}
	if deptView.CurrentOrganizationID != deptA1.ID || !deptView.CurrentMembershipEnabled {
		t.Fatalf("部门用户锚定视图 = org=%v enabled=%v, want org=%v enabled=true", deptView.CurrentOrganizationID, deptView.CurrentMembershipEnabled, deptA1.ID)
	}
	if !containsUUID(deptView.RoleIDs, roleDept.ID) {
		t.Fatalf("部门用户角色展示应来自锚定关系: %v", deptView.RoleIDs)
	}

	// 4. 回归场景：移除总部成员关系后，总部工作台仍可编辑该用户（不再报用户不存在）。
	if err := adminRepo.DeleteUserMembership(ctx, removedUser.ID, hqMembership.ID, adminLifecycleAudit(headquarters.ID, "admin.user.membership.delete")); err != nil {
		t.Fatalf("移除总部成员关系: %v", err)
	}
	removedList, err := adminRepo.ListUsers(ctx, headquarters.ID, biz.AdminUserListOptions{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("移除后总部工作台列表: %v", err)
	}
	if !listUserIDs(removedList)[removedUser.ID] {
		t.Fatal("移除总部成员关系后用户仍应出现在总部工作台列表")
	}
	updated, err := adminRepo.UpdateUser(ctx, headquarters.ID, removedUser.ID, &biz.AdminUser{
		ID:          removedUser.ID,
		DisplayName: "被移除用户-新名",
		Enabled:     true,
	}, []uuid.UUID{roleA.ID}, adminLifecycleAudit(headquarters.ID, "admin.user.update"))
	if err != nil {
		t.Fatalf("移除总部成员关系后编辑用户: %v", err)
	}
	if updated.CurrentOrganizationID != companyA.ID || updated.DisplayName != "被移除用户-新名" {
		t.Fatalf("移除后编辑结果 = %#v", updated)
	}
	anchors, err := data.db.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(companyAMembership.ID)).All(ctx)
	if err != nil || len(anchors) != 1 || anchors[0].RoleID != roleA.ID {
		t.Fatalf("角色应写入锚定的公司A成员关系: %v, error = %v", anchors, err)
	}

	// 5. 角色按锚定组织校验：公司角色挂公司关系成功、错配组织被拒。
	if _, err := adminRepo.UpdateUser(ctx, headquarters.ID, deptUser.ID, &biz.AdminUser{ID: deptUser.ID, DisplayName: "部门用户", Enabled: true}, []uuid.UUID{roleB.ID}, adminLifecycleAudit(headquarters.ID, "admin.user.update")); err != biz.ErrAdminRoleNotFound {
		t.Fatalf("错配组织角色 error = %v, want ErrAdminRoleNotFound", err)
	}
	if _, err := adminRepo.UpdateUser(ctx, companyA.ID, deptUser.ID, &biz.AdminUser{ID: deptUser.ID, DisplayName: "部门用户-新名", Enabled: true}, []uuid.UUID{roleDept.ID}, adminLifecycleAudit(companyA.ID, "admin.user.update")); err != nil {
		t.Fatalf("锚定组织角色更新: %v", err)
	}

	// 6. 重置密码/办理离职按工作台范围内启用成员关系校验。
	if err := adminRepo.ResetUserPassword(ctx, headquarters.ID, deptUser.ID, "new-password-hash", nil, adminLifecycleAudit(headquarters.ID, "admin.user.password.reset")); err != nil {
		t.Fatalf("总部工作台重置密码: %v", err)
	}
	if err := adminRepo.ResetUserPassword(ctx, companyB.ID, deptUser.ID, "new-password-hash", nil, adminLifecycleAudit(companyB.ID, "admin.user.password.reset")); err != biz.ErrAdminUserNotFound {
		t.Fatalf("跨公司重置密码 error = %v, want ErrAdminUserNotFound", err)
	}
	if err := adminRepo.TerminateUser(ctx, companyA.ID, outsiderUser.ID, adminLifecycleAudit(companyA.ID, "admin.user.terminate")); err != biz.ErrAdminUserNotFound {
		t.Fatalf("公司A办理公司B用户离职 error = %v, want ErrAdminUserNotFound", err)
	}
	if err := adminRepo.TerminateUser(ctx, companyB.ID, outsiderUser.ID, adminLifecycleAudit(companyB.ID, "admin.user.terminate")); err != nil {
		t.Fatalf("公司B办理离职: %v", err)
	}
}

// listUserIDs 把用户列表转为按 ID 索引的存在性集合。
func listUserIDs(list *biz.AdminUserList) map[uuid.UUID]bool {
	rows := make(map[uuid.UUID]bool, len(list.Items))
	for _, item := range list.Items {
		rows[item.ID] = true
	}
	return rows
}
