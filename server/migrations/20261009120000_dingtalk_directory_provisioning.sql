-- 钉钉通讯录来源资料与待分配状态；账号 ID 和业务引用保留。
ALTER TABLE users
 ADD COLUMN pending_assignment boolean NOT NULL DEFAULT false,
 ADD COLUMN dingtalk_job_number varchar(100),
 ADD COLUMN dingtalk_mobile varchar(64),
 ADD COLUMN dingtalk_departments jsonb,
 ADD COLUMN dingtalk_last_synced_at timestamptz,
 ADD CONSTRAINT users_pending_assignment_check CHECK (NOT pending_assignment OR NOT enabled);
ALTER TABLE background_tasks DROP CONSTRAINT background_tasks_kind_check;
ALTER TABLE background_tasks ADD CONSTRAINT background_tasks_kind_check CHECK (kind IN ('MASTER_DATA_IMPORT', 'UNLOCODE_IMPORT', 'ORDER_REMINDER', 'INTEGRATION', 'DINGTALK_NOTIFICATION', 'OBJECT_STORAGE_DELETION', 'DINGTALK_APPROVAL_CREATE', 'DINGTALK_DIRECTORY_SYNC'));
CREATE TABLE "ding_talk_directory_syncs" (
 id uuid PRIMARY KEY,
 created_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL,
 background_task_id uuid NOT NULL REFERENCES "background_tasks"(id),
 actor_id uuid NOT NULL,
 corp_id varchar(128) NOT NULL,
 active boolean NOT NULL DEFAULT true,
 scope_summary varchar(500) NOT NULL DEFAULT '',
 full_scope boolean NOT NULL DEFAULT false,
 available_fields jsonb,
 total_count bigint NOT NULL DEFAULT 0,
 created_count bigint NOT NULL DEFAULT 0,
 updated_count bigint NOT NULL DEFAULT 0,
 unchanged_count bigint NOT NULL DEFAULT 0,
 failed_count bigint NOT NULL DEFAULT 0,
 failures jsonb,
 completed_at timestamptz
);
CREATE INDEX dingtalkdirectorysync_updated_at ON ding_talk_directory_syncs(updated_at);
CREATE UNIQUE INDEX dingtalkdirectorysync_background_task_id ON ding_talk_directory_syncs(background_task_id);
CREATE UNIQUE INDEX dingtalk_directory_sync_active_corp_unique ON ding_talk_directory_syncs(corp_id) WHERE active = true;
-- 仅迁移身份完整、禁用、非初始化管理员且无角色/业务组织资格的旧注册占位。
CREATE TEMP TABLE dingtalk_pending_migration ON COMMIT DROP AS
 SELECT u.id FROM users u
 WHERE NOT u.enabled AND NOT u.is_bootstrap_admin
 AND NULLIF(u.dingtalk_unionid,'') IS NOT NULL AND NULLIF(u.dingtalk_userid,'') IS NOT NULL
 AND u.wecom_userid IS NULL AND NULLIF(u.username,'') IS NULL AND u.password_hash IS NULL
 AND EXISTS(SELECT 1 FROM memberships m JOIN organizations o ON o.id=m.organization_id WHERE m.user_id=u.id AND m.enabled AND o.kind='system')
 AND NOT EXISTS(SELECT 1 FROM memberships m JOIN role_assignments ra ON ra.membership_id=m.id WHERE m.user_id=u.id)
 AND NOT EXISTS(SELECT 1 FROM memberships m JOIN organizations o ON o.id=m.organization_id WHERE m.user_id=u.id AND m.enabled AND o.kind<>'system');
-- 旧注册确证来自注册事务内以本人 user_id 记录的成功审计；无证据停止迁移。
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM dingtalk_pending_migration p WHERE NOT EXISTS(SELECT 1 FROM audit_logs a WHERE a.user_id=p.id AND a.action='auth.dingtalk.register' AND a.result='success')) THEN
 RAISE EXCEPTION '钉钉占位账号缺少注册来源审计，无法唯一分类，请核对后再迁移';
 END IF;
END $$;
-- 最后一次登记后若已经开通/离职，保留原状态。
DELETE FROM dingtalk_pending_migration p WHERE EXISTS(
 SELECT 1 FROM audit_logs a WHERE a.result='success'
 AND (a.details->>'resource_id'=p.id::text OR a.resource_id=p.id::text)
 AND a.action IN ('admin.user.dingtalk.authorize','admin.dingtalk.registration.approve','admin.user.terminate','admin.dingtalk.registration.reject')
 AND a.created_at > (SELECT max(r.created_at) FROM audit_logs r WHERE r.user_id=p.id AND r.action='auth.dingtalk.register' AND r.result='success')
);
UPDATE users SET pending_assignment=true FROM dingtalk_pending_migration p WHERE users.id=p.id;
DELETE FROM memberships m USING dingtalk_pending_migration p,organizations o WHERE m.user_id=p.id AND m.organization_id=o.id AND o.kind='system';
-- 旧邀请/注册专属通知为静态历史，上线后不可继续派发或重试。
UPDATE background_tasks t SET status='DEAD_LETTER',attempts=max_attempts,lease_token=NULL,lease_expires_at=NULL,last_error='入职流程已替换',updated_at=now()
 FROM notification_deliveries n WHERE n.background_task_id=t.id
 AND n.template IN ('DINGTALK_REGISTRATION_PENDING','DINGTALK_REGISTRATION_REJECTED','DINGTALK_INVITATION_ACTIVATED') AND t.status<>'SUCCEEDED';
-- 删除旧入口权限的角色关系；其余权限由迁移命令按 Manifest 同步。
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE key='system.user.dingtalk_invitation.manage');
DELETE FROM permissions WHERE key='system.user.dingtalk_invitation.manage';
