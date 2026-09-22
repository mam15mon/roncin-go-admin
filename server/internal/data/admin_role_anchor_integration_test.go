package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// TestAdminRoleAnchorPostgres 覆盖角色库锚定不变量（验收 A2/A3）：
// 写入侧只允许工作台（系统管理/公司）锚定，部门/团队锚定显式失败且不改动数据；
// 读取侧无法解析出工作台时显式返回组织不存在，不回退为传入组织。
func TestAdminRoleAnchorPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	repo := NewAdminRepo(data)

	_, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理: %v", err)
	}
	company, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司: %v", err)
	}
	department, err := data.db.Organization.Create().
		SetCode("DEPT-" + suffix).
		SetName("部门-" + suffix).
		SetKind("department").
		SetParentID(company.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门: %v", err)
	}
	team, err := data.db.Organization.Create().
		SetCode("TEAM-" + suffix).
		SetName("组-" + suffix).
		SetKind("team").
		SetParentID(department.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组: %v", err)
	}

	actorID := uuid.New()
	audit := func(organizationID uuid.UUID) *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "admin.role.write", Result: "success", Details: map[string]string{}}
	}

	// A2：部门锚定创建角色显式拒绝，且不落库。
	if _, err := repo.CreateRole(ctx, department.ID, &biz.AdminRole{
		Code: "dept_" + suffix, Name: "部门角色-" + suffix, DataScope: biz.DataScopeOrganization, Enabled: true,
	}, nil, audit(department.ID)); err != biz.ErrAdminRoleAnchorInvalid {
		t.Fatalf("部门锚定创建角色错误 = %v, want ErrAdminRoleAnchorInvalid", err)
	}
	deptRoleCount, err := data.db.Role.Query().Where(role.OrganizationIDEQ(department.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("统计部门角色失败: %v", err)
	}
	if deptRoleCount != 0 {
		t.Fatalf("部门锚定创建角色不应落库，部门角色数 = %d", deptRoleCount)
	}

	// 存量部门/团队锚定角色（模拟迁移前数据）：更新与删除都必须显式拒绝且不改动数据。
	legacyDepartmentRole, err := data.db.Role.Create().
		SetOrganizationID(department.ID).
		SetCode("legacy_dept_" + suffix).
		SetName("存量部门角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建存量部门角色: %v", err)
	}
	legacyTeamRole, err := data.db.Role.Create().
		SetOrganizationID(team.ID).
		SetCode("legacy_team_" + suffix).
		SetName("存量组角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建存量组角色: %v", err)
	}
	if _, err := repo.UpdateRole(ctx, department.ID, legacyDepartmentRole.ID, &biz.AdminRole{
		Name: "改名后-" + suffix, DataScope: biz.DataScopeAll, Enabled: false,
	}, nil, audit(department.ID)); err != biz.ErrAdminRoleAnchorInvalid {
		t.Fatalf("部门锚定更新角色错误 = %v, want ErrAdminRoleAnchorInvalid", err)
	}
	if err := repo.DeleteRole(ctx, department.ID, legacyDepartmentRole.ID, audit(department.ID)); err != biz.ErrAdminRoleAnchorInvalid {
		t.Fatalf("部门锚定删除角色错误 = %v, want ErrAdminRoleAnchorInvalid", err)
	}
	if err := repo.DeleteRole(ctx, team.ID, legacyTeamRole.ID, audit(team.ID)); err != biz.ErrAdminRoleAnchorInvalid {
		t.Fatalf("组锚定删除角色错误 = %v, want ErrAdminRoleAnchorInvalid", err)
	}
	reloadedLegacy, err := data.db.Role.Get(ctx, legacyDepartmentRole.ID)
	if err != nil {
		t.Fatalf("读取存量部门角色失败: %v", err)
	}
	if reloadedLegacy.Name != "存量部门角色-"+suffix || !reloadedLegacy.Enabled || reloadedLegacy.DataScope != role.DataScopeOrganization {
		t.Fatalf("部门锚定写入被拒绝后数据仍应保持原样: %#v", reloadedLegacy)
	}

	// A3：无法解析出工作台的组织显式返回组织不存在，不回退为传入组织。
	unknown := uuid.New()
	if _, err := repo.ListRoles(ctx, unknown); err != biz.ErrAdminOrganizationNotFound {
		t.Fatalf("未知组织查询角色列表错误 = %v, want ErrAdminOrganizationNotFound", err)
	}
	if _, err := repo.GetRole(ctx, unknown, legacyDepartmentRole.ID); err != biz.ErrAdminOrganizationNotFound {
		t.Fatalf("未知组织查询角色详情错误 = %v, want ErrAdminOrganizationNotFound", err)
	}

	// 对照：公司（工作台）可以正常创建角色，部门上下文共享读取该角色。
	createdRole, err := repo.CreateRole(ctx, company.ID, &biz.AdminRole{
		Code: "company_" + suffix, Name: "公司角色-" + suffix, DataScope: biz.DataScopeOrganization, Enabled: true,
	}, nil, audit(company.ID))
	if err != nil {
		t.Fatalf("公司锚定创建角色失败: %v", err)
	}
	if createdRole.OrganizationID != company.ID {
		t.Fatalf("公司锚定角色所属组织 = %v, want %v", createdRole.OrganizationID, company.ID)
	}
	departmentRoles, err := repo.ListRoles(ctx, department.ID)
	if err != nil {
		t.Fatalf("部门上下文查询角色失败: %v", err)
	}
	foundShared := false
	for _, item := range departmentRoles {
		if item.ID == createdRole.ID {
			foundShared = true
			break
		}
	}
	if !foundShared {
		t.Fatalf("部门上下文应共享所属公司的角色库")
	}
}

// TestRoleWorkspaceAnchorBackfillPostgres 覆盖迁移期存量归一的幂等性与失败语义（验收 A4）：
// 部门/团队锚定角色改挂到最近的工作台祖先；重复执行结果不变；断链与唯一索引冲突
// 都必须报错终止迁移，不静默跳过、不改名停用。
func TestRoleWorkspaceAnchorBackfillPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	_, err := data.db.Organization.Create().
		SetCode("HQ-" + suffix).
		SetName("系统管理-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建系统管理: %v", err)
	}
	company, err := data.db.Organization.Create().
		SetCode("CO-" + suffix).
		SetName("公司-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司: %v", err)
	}
	department, err := data.db.Organization.Create().
		SetCode("DEPT-" + suffix).
		SetName("部门-" + suffix).
		SetKind("department").
		SetParentID(company.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门: %v", err)
	}
	team, err := data.db.Organization.Create().
		SetCode("TEAM-" + suffix).
		SetName("组-" + suffix).
		SetKind("team").
		SetParentID(department.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组: %v", err)
	}

	legacyDepartmentRole, err := data.db.Role.Create().
		SetOrganizationID(department.ID).
		SetCode("backfill_dept_" + suffix).
		SetName("存量部门角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建存量部门角色: %v", err)
	}
	legacyTeamRole, err := data.db.Role.Create().
		SetOrganizationID(team.ID).
		SetCode("backfill_team_" + suffix).
		SetName("存量组角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建存量组角色: %v", err)
	}

	// 首次归一：部门与团队锚定角色都改挂到该组织树最近的工作台（公司）。
	if err := BackfillRoleWorkspaceAnchors(ctx, data.sqlDB); err != nil {
		t.Fatalf("归一角色工作台锚点失败: %v", err)
	}
	movedDepartmentRole, err := data.db.Role.Get(ctx, legacyDepartmentRole.ID)
	if err != nil {
		t.Fatalf("读取归一后的部门角色失败: %v", err)
	}
	movedTeamRole, err := data.db.Role.Get(ctx, legacyTeamRole.ID)
	if err != nil {
		t.Fatalf("读取归一后的组角色失败: %v", err)
	}
	if movedDepartmentRole.OrganizationID != company.ID {
		t.Fatalf("部门角色归一后所属组织 = %v, want %v", movedDepartmentRole.OrganizationID, company.ID)
	}
	if movedTeamRole.OrganizationID != company.ID {
		t.Fatalf("组角色归一后所属组织 = %v, want %v", movedTeamRole.OrganizationID, company.ID)
	}

	// 幂等：重复执行不报错、不再改动结果。
	if err := BackfillRoleWorkspaceAnchors(ctx, data.sqlDB); err != nil {
		t.Fatalf("重复归一角色工作台锚点失败: %v", err)
	}
	repeatedDepartmentRole, err := data.db.Role.Get(ctx, legacyDepartmentRole.ID)
	if err != nil {
		t.Fatalf("重复归一后读取部门角色失败: %v", err)
	}
	if repeatedDepartmentRole.OrganizationID != company.ID {
		t.Fatalf("重复归一后部门角色所属组织 = %v, want %v", repeatedDepartmentRole.OrganizationID, company.ID)
	}

	// 断链：部门没有上级组织时无法解析工作台，归一必须报错终止。
	orphan, err := data.db.Organization.Create().
		SetCode("ORPHAN-" + suffix).
		SetName("断链部门-" + suffix).
		SetKind("department").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建断链部门: %v", err)
	}
	orphanRole, err := data.db.Role.Create().
		SetOrganizationID(orphan.ID).
		SetCode("backfill_orphan_" + suffix).
		SetName("断链部门角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建断链部门角色: %v", err)
	}
	if err := BackfillRoleWorkspaceAnchors(ctx, data.sqlDB); err == nil || !strings.Contains(err.Error(), "无法解析出工作台") {
		t.Fatalf("断链部门归一错误 = %v, want 包含「无法解析出工作台」", err)
	}
	if err := data.db.Role.DeleteOneID(orphanRole.ID).Exec(ctx); err != nil {
		t.Fatalf("清理断链部门角色失败: %v", err)
	}
	if err := data.db.Organization.DeleteOneID(orphan.ID).Exec(ctx); err != nil {
		t.Fatalf("清理断链部门失败: %v", err)
	}

	// 唯一索引冲突：目标工作台已存在同编码角色时不静默改名，而是报错等待人工确认。
	conflictWorkspaceRole, err := data.db.Role.Create().
		SetOrganizationID(company.ID).
		SetCode("backfill_conflict_" + suffix).
		SetName("公司同编码角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建公司同编码角色: %v", err)
	}
	conflictDepartmentRole, err := data.db.Role.Create().
		SetOrganizationID(department.ID).
		SetCode("backfill_conflict_" + suffix).
		SetName("部门同编码角色-" + suffix).
		SetDataScope("organization").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建部门同编码角色: %v", err)
	}
	if err := BackfillRoleWorkspaceAnchors(ctx, data.sqlDB); err == nil || !strings.Contains(err.Error(), "归一角色工作台锚点失败") {
		t.Fatalf("唯一索引冲突归一错误 = %v, want 包含「归一角色工作台锚点失败」", err)
	}
	keptConflictRole, err := data.db.Role.Get(ctx, conflictDepartmentRole.ID)
	if err != nil {
		t.Fatalf("读取冲突角色失败: %v", err)
	}
	if keptConflictRole.OrganizationID != department.ID {
		t.Fatalf("归一失败后冲突角色应保持原锚定组织 %v, got %v", department.ID, keptConflictRole.OrganizationID)
	}
	if err := data.db.Role.DeleteOneID(conflictDepartmentRole.ID).Exec(ctx); err != nil {
		t.Fatalf("清理部门同编码角色失败: %v", err)
	}
	if err := data.db.Role.DeleteOneID(conflictWorkspaceRole.ID).Exec(ctx); err != nil {
		t.Fatalf("清理公司同编码角色失败: %v", err)
	}

	// 清理冲突后归一应通过迁移后自检断言。
	if err := BackfillRoleWorkspaceAnchors(ctx, data.sqlDB); err != nil {
		t.Fatalf("清理冲突后归一应通过自检: %v", err)
	}
}
