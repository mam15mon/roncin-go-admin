package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// 前态夹具用原生 SQL，不依赖新生成字段；正式迁移运行在独立 Schema。
func TestDingTalkDirectoryMigrationPreservesIdentityAndHistory(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261009120000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	sys, company := uuid.New(), uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO organizations(id,created_at,updated_at,code,name,kind,enabled,base_currency,search_keywords) VALUES ($1,now(),now(),'mig-sys','系统','system',true,'CNY',''),($2,now(),now(),'mig-co','公司','company',true,'CNY','')`, sys, company); err != nil {
		t.Fatal(err)
	}
	ids, members := map[string]uuid.UUID{}, map[string]uuid.UUID{}
	cases := []struct {
		name                              string
		enabled, bootstrap, memberEnabled bool
		org                               uuid.UUID
		username, password, wecom         *string
		registration                      bool
	}{
		{name: "pending", memberEnabled: true, org: sys, registration: true},
		{name: "bootstrap", enabled: true, bootstrap: true, memberEnabled: true, org: sys},
		{name: "active", enabled: true, memberEnabled: true, org: company, registration: true},
		{name: "terminated", memberEnabled: false, org: company, registration: true},
		{name: "wecom", memberEnabled: true, org: sys, wecom: directoryTestString("wecom-member")},
		{name: "local", memberEnabled: true, org: sys, username: directoryTestString("local-admin"), password: directoryTestString("test-hash")},
	}
	for _, c := range cases {
		id, member := uuid.New(), uuid.New()
		ids[c.name], members[c.name] = id, member
		if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO users(id,created_at,updated_at,display_name,enabled,is_bootstrap_admin,username,password_hash,wecom_userid,dingtalk_unionid,dingtalk_userid,search_keywords) VALUES($1,now(),now(),$2,$3,$4,$5,$6,$7,$8,$9,'')`, id, c.name, c.enabled, c.bootstrap, c.username, c.password, c.wecom, "union-"+c.name, "user-"+c.name); err != nil {
			t.Fatal(err)
		}
		if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO memberships(id,created_at,updated_at,user_id,organization_id,enabled,"primary") VALUES($1,now(),now(),$2,$3,$4,true)`, member, id, c.org, c.memberEnabled); err != nil {
			t.Fatal(err)
		}
		if c.registration {
			if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO audit_logs(id,created_at,updated_at,user_id,action,result,resource_type,resource_id,request_id,trace_id,ip_address) VALUES($1,now(),now(),$2,'auth.dingtalk.register','success','','','','','')`, uuid.New(), id); err != nil {
				t.Fatal(err)
			}
		}
	}
	taskID, deliveryID := uuid.New(), uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO background_tasks(id,created_at,updated_at,organization_id,kind,idempotency_key,status,attempts,max_attempts,next_run_at) VALUES($1,now(),now(),$2,'DINGTALK_NOTIFICATION','migration-history','PENDING',0,3,now())`, taskID, sys); err != nil {
		t.Fatal(err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO notification_deliveries(id,created_at,updated_at,background_task_id,recipient_user_id,channel,template,resource_type,resource_id,reference_code,parameter) VALUES($1,now(),now(),$2,$3,'DINGTALK','DINGTALK_REGISTRATION_PENDING','USER',$3,'','')`, deliveryID, taskID, ids["pending"]); err != nil {
		t.Fatal(err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t, "20261009120000_dingtalk_directory_provisioning.sql")); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		var id uuid.UUID
		var pending, enabled bool
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT id,pending_assignment,enabled FROM users WHERE id=$1`, ids[c.name]).Scan(&id, &pending, &enabled); err != nil {
			t.Fatal(err)
		}
		if id != ids[c.name] || pending != (c.name == "pending") || enabled != c.enabled {
			t.Fatalf("%s 身份/状态被误改", c.name)
		}
		var count int
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM memberships WHERE id=$1`, members[c.name]).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 1
		if c.name == "pending" {
			want = 0
		}
		if count != want {
			t.Fatalf("%s 成员关系被误改: %d", c.name, count)
		}
	}
	var status, message string
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT status,last_error FROM background_tasks WHERE id=$1`, taskID).Scan(&status, &message); err != nil {
		t.Fatal(err)
	}
	if status != "DEAD_LETTER" || message != "入职流程已替换" {
		t.Fatal("旧专属通知未终结")
	}
	var checks int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conname='users_pending_assignment_check' AND conrelid='users'::regclass`).Scan(&checks); err != nil || checks != 1 {
		t.Fatal("正式 CHECK 缺失", err)
	}
}

func directoryTestString(s string) *string { return &s }

func TestDingTalkDirectoryMigrationRejectsAmbiguousAccountAtomically(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261009120000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	sys, person, member := uuid.New(), uuid.New(), uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO organizations(id,created_at,updated_at,code,name,kind,enabled,base_currency,search_keywords) VALUES($1,now(),now(),'ambiguous-sys','系统','system',true,'CNY','')`, sys); err != nil {
		t.Fatal(err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO users(id,created_at,updated_at,display_name,enabled,is_bootstrap_admin,dingtalk_unionid,dingtalk_userid,search_keywords) VALUES($1,now(),now(),'来源不明账号',false,false,'ambiguous-union','ambiguous-user','')`, person); err != nil {
		t.Fatal(err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO memberships(id,created_at,updated_at,user_id,organization_id,enabled,"primary") VALUES($1,now(),now(),$2,$3,true,true)`, member, person, sys); err != nil {
		t.Fatal(err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t, "20261009120000_dingtalk_directory_provisioning.sql")); err == nil {
		t.Fatal("缺少来源审计不应被猜测归类")
	}
	var count int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM memberships WHERE id=$1`, member).Scan(&count); err != nil || count != 1 {
		t.Fatal("失败迁移修改了原成员关系", err)
	}
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='users' AND column_name='pending_assignment'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("失败迁移未原子回滚DDL", err)
	}
}
