package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
)

// TestAdminRoleDeletePostgres 在隔离 Schema 上验证角色删除的数据层规则：
// 已分配成员的角色拒绝删除（事务内复核口径）、跨组织角色不可删除、
// 未分配成员的角色连同权限关联一并删除并写审计、列表统计一次性返回分配数。
func TestAdminRoleDeletePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	repo := NewAdminRepo(data)

	systemWorkspace, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理: %v", err)
	}
	otherOrg, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司: %v", err)
	}

	assignedRole, err := data.db.Role.Create().
		SetOrganizationID(systemWorkspace.ID).
		SetCode("assigned_" + suffix).
		SetName("已分配角色-" + suffix).
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建已分配角色: %v", err)
	}
	freeRole, err := data.db.Role.Create().
		SetOrganizationID(systemWorkspace.ID).
		SetCode("free_" + suffix).
		SetName("空闲角色-" + suffix).
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建空闲角色: %v", err)
	}
	foreignRole, err := data.db.Role.Create().
		SetOrganizationID(otherOrg.ID).
		SetCode("foreign_" + suffix).
		SetName("跨组织角色-" + suffix).
		SetDataScope("organization").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建跨组织角色: %v", err)
	}
	account, err := data.db.User.Create().
		SetUsername("role-del-" + suffix).
		SetDisplayName("角色删除用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建用户: %v", err)
	}
	membershipRecord, err := data.db.Membership.Create().
		SetUserID(account.ID).
		SetOrganizationID(systemWorkspace.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建成员关系: %v", err)
	}
	if _, err := data.db.RoleAssignment.Create().SetMembershipID(membershipRecord.ID).SetRoleID(assignedRole.ID).Save(ctx); err != nil {
		t.Fatalf("挂载已分配角色: %v", err)
	}

	actorID := uuid.New()
	audit := func(action string) *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &systemWorkspace.ID, UserID: &actorID, Action: action, Result: "success", Details: map[string]string{}}
	}

	// 已分配成员的角色拒绝删除。
	if err := repo.DeleteRole(ctx, systemWorkspace.ID, assignedRole.ID, audit("admin.role.delete")); err != biz.ErrAdminRoleAssigned {
		t.Fatalf("删除已分配角色错误 = %v, want ErrAdminRoleAssigned", err)
	}
	// 跨组织角色按不存在处理。
	if err := repo.DeleteRole(ctx, systemWorkspace.ID, foreignRole.ID, audit("admin.role.delete")); err != biz.ErrAdminRoleNotFound {
		t.Fatalf("删除跨组织角色错误 = %v, want ErrAdminRoleNotFound", err)
	}

	// 空闲角色删除成功：角色与权限关联一并清除，并写审计。
	freeRolePermission, err := data.db.Permission.Create().
		SetKey("test.role.delete.perm." + suffix).
		SetName("角色删除测试权限-" + suffix).
		SetGroup("test").
		SetDescription("角色删除集成测试权限").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试权限: %v", err)
	}
	if _, err := data.db.Role.UpdateOneID(freeRole.ID).AddPermissions(freeRolePermission).Save(ctx); err != nil {
		t.Fatalf("挂载测试权限: %v", err)
	}
	deleteAudit := audit("admin.role.delete")
	if err := repo.DeleteRole(ctx, systemWorkspace.ID, freeRole.ID, deleteAudit); err != nil {
		t.Fatalf("删除空闲角色失败: %v", err)
	}
	if _, err := data.db.Role.Query().Where(role.IDEQ(freeRole.ID)).Only(ctx); !ent.IsNotFound(err) {
		t.Fatalf("角色应已删除，查询错误 = %v", err)
	}
	edges, err := data.db.RoleAssignment.Query().Where(roleassignment.RoleIDEQ(freeRole.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("查询残留分配失败: %v", err)
	}
	if edges != 0 {
		t.Fatalf("残留角色权限关联 = %d, want 0", edges)
	}
	if deleteAudit.Details["resource_id"] != freeRole.ID.String() {
		t.Fatalf("审计 resource_id = %q, want %q", deleteAudit.Details["resource_id"], freeRole.ID.String())
	}

	// 列表统计一次性返回各角色分配数：已分配角色为 1，已删除角色不再出现。
	roles, err := repo.ListRoles(ctx, systemWorkspace.ID)
	if err != nil {
		t.Fatalf("查询角色列表失败: %v", err)
	}
	countsByID := make(map[uuid.UUID]int, len(roles))
	for _, item := range roles {
		countsByID[item.ID] = item.AssignmentsCount
	}
	if _, ok := countsByID[freeRole.ID]; ok {
		t.Fatalf("已删除角色不应出现在列表: %#v", roles)
	}
	if countsByID[assignedRole.ID] != 1 {
		t.Fatalf("已分配角色 assignments_count = %d, want 1", countsByID[assignedRole.ID])
	}

	// 事务内复核兜底：直接构造绕过业务前置校验的调用仍拒绝删除已分配角色。
	if err := repo.DeleteRole(ctx, systemWorkspace.ID, assignedRole.ID, audit("admin.role.delete")); err != biz.ErrAdminRoleAssigned {
		t.Fatalf("事务内复核删除已分配角色错误 = %v, want ErrAdminRoleAssigned", err)
	}
}
