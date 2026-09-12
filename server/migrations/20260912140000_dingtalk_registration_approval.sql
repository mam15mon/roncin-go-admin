-- 钉钉注册审批与通知路由：
--   1. 新表 ding_talk_invitations（通道 A 邀请）：只存管理员决策字段（手机号/目标
--      组织/初始角色），活跃邀请按 (organization_id, mobile) 部分唯一；
--   2. users 增加 dingtalk_requested_organization_id（通道 B 注册自选目标组织，
--      NULL 表示存量兜底，通知总部管理员）；
--   3. 通知模板枚举扩展注册待审批 / 注册被拒绝 / 邀请已激活。

CREATE TABLE "ding_talk_invitations" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "mobile" character varying(32) NOT NULL,
  "display_name" character varying(100) NULL DEFAULT '',
  "status" character varying NOT NULL DEFAULT 'PENDING',
  "consumed_at" timestamp with time zone NULL,
  "expires_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  "invited_by" uuid NOT NULL,
  "consumed_by" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ding_talk_invitations_status_check"
    CHECK ("status" IN ('PENDING', 'CONSUMED', 'EXPIRED', 'REVOKED')),
  CONSTRAINT "ding_talk_invitations_organizations_dingtalk_invitations"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_invitations_roles_dingtalk_invitations"
    FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_invitations_users_created_dingtalk_invitations"
    FOREIGN KEY ("invited_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_invitations_users_consumed_dingtalk_invitations"
    FOREIGN KEY ("consumed_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

CREATE INDEX "dingtalkinvitation_updated_at"
  ON "ding_talk_invitations" ("updated_at");

-- 活跃邀请唯一：同一组织同手机号只能存在一条 PENDING；过期/消费/撤销后可重建。
CREATE UNIQUE INDEX "dingtalkinvitation_organization_id_mobile"
  ON "ding_talk_invitations" ("organization_id", "mobile") WHERE "status" = 'PENDING';

CREATE INDEX "dingtalkinvitation_expires_at"
  ON "ding_talk_invitations" ("expires_at");

CREATE INDEX "dingtalkinvitation_invited_by"
  ON "ding_talk_invitations" ("invited_by");

ALTER TABLE "users"
  ADD COLUMN "dingtalk_requested_organization_id" uuid NULL;

ALTER TABLE "users"
  ADD CONSTRAINT "users_organizations_dingtalk_registration_requests"
    FOREIGN KEY ("dingtalk_requested_organization_id") REFERENCES "organizations" ("id") ON DELETE SET NULL;

CREATE INDEX "user_dingtalk_requested_organization_id"
  ON "users" ("dingtalk_requested_organization_id");

-- 通知模板枚举扩展：注册待审批 / 注册被拒绝 / 邀请已激活。
-- DROP 带 IF EXISTS：开发库存在早期建库漂移（该 CHECK 可能缺失），
-- 完整迁移链冷启动下原约束存在时同样无副作用。
ALTER TABLE "notification_deliveries"
  DROP CONSTRAINT IF EXISTS "notification_deliveries_template_check";

ALTER TABLE "notification_deliveries"
  ADD CONSTRAINT "notification_deliveries_template_check"
    CHECK ("template" IN ('ORDER_PERSONNEL_ASSIGNED', 'USER_AUTHORIZED', 'DINGTALK_REGISTRATION_PENDING', 'DINGTALK_REGISTRATION_REJECTED', 'DINGTALK_INVITATION_ACTIVATED'));
