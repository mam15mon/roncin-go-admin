package data

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func TestMobileLoginMigrationBackfillsNormalizationWithoutChangingIdentity(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261019100000")
	defer cleanup()
	ctx := context.Background()
	cases := []string{"13800138000", "+86 138-0013-8000", "0086 13800138000", "\t+86\u00a0138\u30000013\u00858000\n", "\u1680\u2000+86\u2001138\u20020013\u20038000\u2028\u2029\u202f\u205f", "13800138000", "", "invalid", "8613800138000", "12800138000"}
	ids := make([]uuid.UUID, len(cases))
	for i, raw := range cases {
		ids[i] = uuid.New()
		if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO users(id,created_at,updated_at,display_name,password_hash,dingtalk_mobile,dingtalk_userid,dingtalk_unionid) VALUES($1,now(),now(),'迁移保留用户','original-hash',$2,$3,$4)`, ids[i], raw, ids[i].String(), "union-"+ids[i].String()); err != nil {
			t.Fatal(err)
		}
	}
	if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal("重复迁移", err)
	}
	for i, raw := range cases {
		account, err := data.db.User.Get(ctx, ids[i])
		if err != nil {
			t.Fatal(err)
		}
		normalized := biz.NormalizeDingTalkMobile(raw)
		if !biz.ValidDingTalkMobile(normalized) {
			if account.DingtalkMobileNormalized != nil {
				t.Fatalf("无效号仍有投影 %d", i)
			}
		} else if account.DingtalkMobileNormalized == nil || *account.DingtalkMobileNormalized != normalized {
			t.Fatalf("回填和 Go 不等价，样本 %d", i)
		}
		if account.ID != ids[i] || account.PasswordHash == nil || *account.PasswordHash != "original-hash" || account.DingtalkMobile == nil || *account.DingtalkMobile != raw || account.DingtalkUserid == nil || *account.DingtalkUserid != ids[i].String() {
			t.Fatal("来源资料或身份被改变")
		}
	}
	checks := migrate.PasswordResetChallengesTable.Annotation.Checks
	for name, expr := range map[string]string{"password_reset_challenges_status_check": "status IN ('PENDING', 'READY', 'FAILED', 'CONSUMED')", "password_reset_challenges_failed_attempts_check": "failed_attempts BETWEEN 0 AND 5"} {
		if checks[name] != expr {
			t.Fatalf("Ent CHECK %s = %q", name, checks[name])
		}
		var count int
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid='password_reset_challenges'::regclass AND conname=$1`, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("正式 CHECK %s: %d %v", name, count, err)
		}
	}
	var unique bool
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT indisunique FROM pg_index WHERE indexrelid='user_dingtalk_mobile_normalized'::regclass`).Scan(&unique); err != nil || unique {
		t.Fatalf("手机号索引不应唯一 %v %v", unique, err)
	}
	var revisions int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261019100000_mobile_login_password_reset'`).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("迁移记录 %d，错误 %v", revisions, err)
	}
}
