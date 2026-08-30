package data

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestSyncPermissionManifestPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开集成测试数据库: %v", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	// 关库注册为最早的 t.Cleanup（LIFO 中最后执行），保证数据清理先于连接关闭。
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("初始化集成测试 Schema: %v", err)
	}
	if _, err := SyncPermissionManifest(ctx, db); err != nil {
		t.Fatalf("初始化权限清单: %v", err)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	organization, err := client.Organization.Create().
		SetCode("PERMISSION-SYNC-" + suffix).
		SetName("权限同步测试组织").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	t.Cleanup(func() { cleanupPermissionSyncOrganization(t, client, organization.ID) })

	roleUpdate, err := client.Permission.Query().Where(permissionent.KeyEQ(access.RoleUpdate)).Only(ctx)
	if err != nil {
		t.Fatalf("查询角色编辑权限: %v", err)
	}
	stale, err := client.Permission.Create().
		SetKey("test.stale." + suffix).
		SetName("待清理权限").
		SetGroup("测试").
		SetDescription("仅用于验证清单外权限清理").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建清单外权限: %v", err)
	}
	customRole, err := client.Role.Create().
		SetOrganizationID(organization.ID).
		SetCode("operator").
		SetName("操作员").
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(roleUpdate, stale).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建待归一化角色: %v", err)
	}
	administrator, err := client.Role.Create().
		SetOrganizationID(organization.ID).
		SetCode("administrator").
		SetName("系统管理员").
		SetDataScope(roleent.DataScopeAll).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建管理员角色: %v", err)
	}

	summary, err := SyncPermissionManifest(ctx, db)
	if err != nil {
		t.Fatalf("同步权限清单: %v", err)
	}
	if summary.Removed < 1 {
		t.Fatalf("移除清单外权限数 = %d, want >= 1", summary.Removed)
	}
	if summary.Attached < len(access.Manifest())+2 {
		t.Fatalf("补齐角色权限数 = %d, want >= %d", summary.Attached, len(access.Manifest())+2)
	}
	if exists, err := client.Permission.Query().Where(permissionent.KeyEQ(stale.Key)).Exist(ctx); err != nil || exists {
		t.Fatalf("清单外权限仍然存在: exists=%v error=%v", exists, err)
	}

	customPermissions, err := customRole.QueryPermissions().All(ctx)
	if err != nil {
		t.Fatalf("查询归一化角色权限: %v", err)
	}
	customKeys := make(map[string]struct{}, len(customPermissions))
	for _, item := range customPermissions {
		customKeys[item.Key] = struct{}{}
	}
	for _, key := range []string{access.RoleUpdate, access.RoleRead, access.PermissionRead} {
		if _, ok := customKeys[key]; !ok {
			t.Fatalf("归一化角色缺少权限 %s: %v", key, customKeys)
		}
	}
	adminPermissionCount, err := administrator.QueryPermissions().Count(ctx)
	if err != nil {
		t.Fatalf("查询管理员权限数: %v", err)
	}
	if adminPermissionCount != len(access.Manifest()) {
		t.Fatalf("管理员权限数 = %d, want %d", adminPermissionCount, len(access.Manifest()))
	}

	second, err := SyncPermissionManifest(ctx, db)
	if err != nil {
		t.Fatalf("重复同步权限清单: %v", err)
	}
	if second.Created != 0 || second.Updated != 0 || second.Removed != 0 || second.Attached != 0 {
		t.Fatalf("重复同步不是幂等操作: %+v", second)
	}
}

// cleanupPermissionSyncOrganization 先删除组织下的角色及其权限关联再删除组织，
// 避免直接删组织被外键挡下而残留测试数据。
func cleanupPermissionSyncOrganization(t *testing.T, client *ent.Client, organizationID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	roles, err := client.Role.Query().Where(roleent.OrganizationIDEQ(organizationID)).All(ctx)
	if err != nil {
		t.Errorf("查询测试角色失败: %v", err)
		return
	}
	for _, item := range roles {
		if err := item.Update().ClearPermissions().Exec(ctx); err != nil {
			t.Errorf("清理角色权限关联失败: %v", err)
			return
		}
	}
	if _, err := client.Role.Delete().Where(roleent.OrganizationIDEQ(organizationID)).Exec(ctx); err != nil {
		t.Errorf("清理测试角色失败: %v", err)
		return
	}
	if err := client.Organization.DeleteOneID(organizationID).Exec(ctx); err != nil {
		t.Errorf("清理测试组织失败: %v", err)
	}
}
