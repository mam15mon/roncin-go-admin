package data

import (
	"context"
	"slices"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// TestCompanyAdministratorCreatesMaximumRoleFromEffectivePermissions 验证角色表单按
// auth/me 有效能力全选后，真实用例可以创建、编辑最大公司角色；直接提交全字典仍拒绝。
func TestCompanyAdministratorCreatesMaximumRoleFromEffectivePermissions(t *testing.T) {
	f := newAuthPrincipalFixture(t)
	ctx := context.Background()
	if _, err := SyncPermissionManifest(ctx, f.data.sqlDB); err != nil {
		t.Fatalf("同步隔离测试库权限目录: %v", err)
	}
	if err := initializeCompanyAdministrator(ctx, f.data.db, f.branch); err != nil {
		t.Fatalf("创建公司管理员角色: %v", err)
	}
	companyRole, err := f.data.db.Role.Query().Where(role.OrganizationIDEQ(f.branch), role.CodeEQ("administrator")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	member, err := f.data.db.Membership.Create().SetUserID(f.bootstrapUserID).SetOrganizationID(f.branch).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.data.db.RoleAssignment.Create().SetMembershipID(member.ID).SetRoleID(companyRole.ID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	principal, err := f.repo.ResolvePrincipal(ctx, f.bootstrapUserID, f.branch)
	if err != nil {
		t.Fatalf("解析真实公司管理员能力: %v", err)
	}
	keys := principal.PermissionKeys()
	expected := make([]string, 0)
	allKeys := make([]string, 0)
	for _, definition := range access.Manifest() {
		allKeys = append(allKeys, definition.Key)
		if access.PermissionAllowedInWorkspace(definition.Key, false) {
			expected = append(expected, definition.Key)
		}
	}
	slices.Sort(expected)
	if len(keys) == 0 || !slices.Equal(keys, expected) || len(keys) >= len(allKeys) {
		t.Fatalf("公司管理员应拥有完整公司候选且不含全部系统目录: 有效=%d 公司=%d 字典=%d", len(keys), len(expected), len(allKeys))
	}
	t.Logf("公司全选有效权限 %d 项，完整目录 %d 项", len(keys), len(allKeys))

	repo := NewAdminRepo(f.data)
	uc := biz.NewAdminUsecase(repo, f.data)
	actorCtx := biz.WithPrincipal(ctx, principal)
	created, err := uc.CreateRole(actorCtx, f.branch, f.bootstrapUserID, &biz.AdminRole{Name: "管理员", Enabled: true}, keys)
	if err != nil {
		t.Fatalf("按有效能力全选创建最大公司角色应成功: %v", err)
	}
	if created.OrganizationID != f.branch || created.Name != "管理员" || !slices.Equal(created.PermissionKeys, keys) {
		t.Fatalf("创建角色必须保留公司归属与完整有效权限: %#v", created)
	}
	updated, err := uc.UpdateRole(actorCtx, f.branch, f.bootstrapUserID, created.ID, &biz.AdminRole{Name: "管理员（已更新）", Enabled: true}, keys)
	if err != nil || updated == nil || !slices.Equal(updated.PermissionKeys, keys) {
		t.Fatalf("按有效能力编辑最大公司角色应成功并保留权限: %v", err)
	}

	if _, err := uc.CreateRole(actorCtx, f.branch, f.bootstrapUserID, &biz.AdminRole{Name: "错误全字典角色", Enabled: true}, allKeys); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("即使初始管理员也不能向公司角色提交系统专属权限: %v", err)
	}
	if _, err := uc.UpdateRole(actorCtx, f.branch, f.bootstrapUserID, created.ID, &biz.AdminRole{Name: "错误更新", Enabled: true}, allKeys); err != biz.ErrAdminPrivilegeEscalation {
		t.Fatalf("编辑同样必须拒绝非法全字典权限: %v", err)
	}
	persisted, err := repo.GetRole(actorCtx, f.branch, created.ID)
	if err != nil || persisted == nil || persisted.Name != updated.Name || !slices.Equal(persisted.PermissionKeys, keys) {
		t.Fatalf("非法更新失败后原角色和权限必须保持不变: %v", err)
	}
	if exists, err := f.data.db.Role.Query().Where(role.OrganizationIDEQ(f.branch), role.NameEQ("错误全字典角色")).Exist(ctx); err != nil || exists {
		t.Fatalf("非法创建不得残留角色: exists=%v err=%v", exists, err)
	}
}
