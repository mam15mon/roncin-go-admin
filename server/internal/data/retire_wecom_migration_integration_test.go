package data

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

func TestRetireWeComMigrationPreservesAccountsAndRelationsPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261009130000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, err := data.db.Organization.Create().SetCode("RETIRE-SYSTEM").SetName("下线验证系统").SetKind("system").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	keptPermission, err := data.db.Permission.Query().Where(permissionent.KeyEQ(access.UserRead)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	retiredPermission, err := data.db.Permission.Query().Where(permissionent.KeyEQ("system.user.authorize_wecom")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	role, err := data.db.Role.Create().SetOrganizationID(org.ID).SetCode("retirement-reader").SetName("保留角色").SetDataScope("all").AddPermissions(keptPermission, retiredPermission).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const plainPassword = "retirement-local-password"
	passwordHash, err := password.Hash(plainPassword)
	if err != nil {
		t.Fatal(err)
	}
	combinedID, retiredOnlyID, dingTalkID := uuid.New(), uuid.New(), uuid.New()
	for _, account := range []struct {
		id      uuid.UUID
		name    string
		oldID   *string
		unionID *string
		userID  *string
		enabled bool
		hash    *string
	}{
		{combinedID, "local-combined", directoryTestString("old-combined"), nil, nil, true, &passwordHash},
		{retiredOnlyID, "retired-only", directoryTestString("old-only"), nil, nil, false, nil},
		{dingTalkID, "dingtalk-linked", nil, directoryTestString("kept-union"), directoryTestString("kept-user"), true, nil},
	} {
		// 旧绑定通过正式迁移前态准备，当前 Ent 真相源不再含企业微信字段。
		if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO users(id,created_at,updated_at,username,display_name,password_hash,wecom_userid,wecom_name,dingtalk_unionid,dingtalk_userid,dingtalk_name,enabled,is_bootstrap_admin,search_keywords)
			VALUES($1,now(),now(),$2,'保留姓名',$3,$4,'已删除来源',$5,$6,'保留钉钉姓名',$7,false,$8)`, account.id, account.name, account.hash, account.oldID, account.unionID, account.userID, account.enabled, searchtext.Build("保留姓名", "已删除来源", "保留钉钉姓名")); err != nil {
			t.Fatal(err)
		}
		member, err := data.db.Membership.Create().SetUserID(account.id).SetOrganizationID(org.ID).SetEnabled(true).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if account.enabled {
			if _, err := data.db.RoleAssignment.Create().SetMembershipID(member.ID).SetRoleID(role.ID).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := data.db.Session.Create().SetUserID(combinedID).SetOrganizationID(org.ID).SetTokenHash(strings.Repeat("a", 64)).SetExpiresAt(time.Now().Add(time.Hour)).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := data.db.AuditLog.Create().SetUserID(retiredOnlyID).SetOrganizationID(org.ID).SetAction("auth.wecom.register").SetResult("success").Save(ctx); err != nil {
		t.Fatal(err)
	}
	queries := map[string]string{
		"users":            `SELECT jsonb_agg(to_jsonb(t) - 'wecom_userid' - 'wecom_name' - 'search_keywords' ORDER BY id)::text FROM users t`,
		"organizations":    `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM organizations t`,
		"memberships":      `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM memberships t`,
		"roles":            `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM roles t`,
		"role_assignments": `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM role_assignments t`,
		"sessions":         `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM sessions t`,
		"audit_logs":       `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM audit_logs t`,
	}
	before := make(map[string]string, len(queries))
	for name, query := range queries {
		var snapshot string
		if err := data.sqlDB.QueryRowContext(ctx, query).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		before[name] = snapshot
	}
	dir := filepath.Join("..", "..", "migrations")
	if err := migration.Apply(ctx, data.sqlDB, dir); err != nil {
		t.Fatal(err)
	}
	assertRetiredIdentitySchema(t, data)
	if err := BackfillSelectorSearchKeywords(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncPermissionManifest(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	for name, query := range queries {
		var after string
		if err := data.sqlDB.QueryRowContext(ctx, query).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if after != before[name] {
			t.Fatalf("下线迁移改写了 %s 的保留数据", name)
		}
	}
	var keywords string
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT search_keywords FROM users WHERE id=$1`, combinedID).Scan(&keywords); err != nil {
		t.Fatal(err)
	}
	if want := searchtext.Build("保留姓名", "保留钉钉姓名"); keywords != want {
		t.Fatalf("检索键未按保留姓名来源重建: got=%q want=%q", keywords, want)
	}
	if exists, err := data.db.Permission.Query().Where(permissionent.IDEQ(retiredPermission.ID)).Exist(ctx); err != nil || exists {
		t.Fatalf("退役权限未移除: exists=%v err=%v", exists, err)
	}
	grants, err := data.db.Role.Query().WithPermissions().All(ctx)
	if err != nil || len(grants) != 1 || len(grants[0].Edges.Permissions) != 1 || grants[0].Edges.Permissions[0].Key != access.UserRead {
		t.Fatalf("角色有效权限未保留: err=%v", err)
	}
	// 再执行迁移与回填应幂等，且账密、钉钉身份和旧会话仍可用。
	if err := migration.Apply(ctx, data.sqlDB, dir); err != nil {
		t.Fatal(err)
	}
	if err := BackfillSelectorSearchKeywords(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	repo := NewAuthRepo(data)
	if _, err := repo.FindSession(ctx, strings.Repeat("a", 64), time.Now()); err != nil {
		t.Fatalf("已有会话失效: %v", err)
	}
	credential, err := repo.FindDingTalkCredential(ctx, &biz.DingTalkIdentity{UnionID: "kept-union", UserID: "kept-user"})
	if err != nil || credential.UserID != dingTalkID || !credential.Enabled {
		t.Fatalf("钉钉绑定失效: err=%v", err)
	}
	uc := biz.NewAuthUsecase(repo, &biz.SessionPolicy{TTL: time.Hour}, nil, nil)
	result, err := uc.Login(ctx, "local-combined", plainPassword, uuid.Nil, "migration-test", "127.0.0.1")
	if err != nil || result.Principal.UserID != combinedID {
		t.Fatalf("本地账密登录失效: %v", err)
	}
}

func TestRetireWeComMigrationColdStartPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	assertRetiredIdentitySchema(t, data)
	ctx := context.Background()
	account, err := data.db.User.Create().SetUsername("cold-start-local").SetDisplayName("冷启动用户").SetDingtalkName("冷启动钉钉姓名").Save(ctx)
	if err != nil {
		t.Fatalf("冷启动 Ent 写入失败: %v", err)
	}
	if account.SearchKeywords != searchtext.Build("冷启动用户", "冷启动钉钉姓名") {
		t.Fatal("当前用户检索 Hook 未按保留字段构建")
	}
	var revisions int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261009130000_retire_wecom' AND length(checksum)=64`).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("冷启动迁移记录不完整: revisions=%d err=%v", revisions, err)
	}
}

func assertRetiredIdentitySchema(t *testing.T, data *Data) {
	t.Helper()
	ctx := context.Background()
	var count int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='users' AND column_name IN ('wecom_userid','wecom_name')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("退役身份字段仍存在: count=%d err=%v", count, err)
	}
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND indexname='users_wecom_userid_key'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("退役身份索引仍存在: count=%d err=%v", count, err)
	}
}
