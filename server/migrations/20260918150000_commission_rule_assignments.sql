-- 提成方案员工有效期分配 Schema：
-- 1) finance_commission_rules 增加 legacy_readonly 标记：迁移停用的无分配旧角色
--    规则成为历史只读方案，只能复制为有明确起始日与真实员工分配的新方案；
-- 2) 新增 finance_commission_rule_assignments：保存方案与员工的有效期分配。
--    分配起始日必填且不可变，终止日可空（正无穷）；未生效分配以 cancelled_at /
--    cancelled_by 撤销，已生效分配以 effective_to 终止并记录 terminated_at /
--    terminated_by，禁止物理删除造成历史断链。方案、组织、员工与创建人外键
--    NO ACTION，撤销/终止操作者为可逆审计引用（SET NULL）；
-- 3) 将迁移前没有任何员工分配的旧角色规则统一置为 enabled = false 并标记
--    legacy_readonly = true，使其不能继续生成新提成或开启工作台资格。
--    不清空数据、不重算既有提成快照、不自动给同角色员工补挂方案。

ALTER TABLE "finance_commission_rules"
  ADD COLUMN "legacy_readonly" boolean NOT NULL DEFAULT false;

CREATE TABLE "finance_commission_rule_assignments" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "rule_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "effective_from" character varying(10) NOT NULL,
  "effective_to" character varying(10) NULL,
  "cancelled_at" timestamp with time zone NULL,
  "cancelled_by" uuid NULL,
  "terminated_at" timestamp with time zone NULL,
  "terminated_by" uuid NULL,
  "created_by" uuid NOT NULL,
  PRIMARY KEY ("id"),
  -- 有效期闭区间合法性：终止日为空视为正无穷，存在时不得早于起始日。
  CONSTRAINT "finance_commission_rule_assignments_effective_period_check"
    CHECK ("effective_to" IS NULL OR "effective_from" <= "effective_to"),
  -- 撤销审计一致性：撤销时间与撤销操作者必须成对出现。
  CONSTRAINT "finance_commission_rule_assignments_cancel_audit_check"
    CHECK (("cancelled_at" IS NULL AND "cancelled_by" IS NULL) OR ("cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL)),
  -- 终止审计一致性：终止时间与终止操作者必须成对出现。
  CONSTRAINT "finance_commission_rule_assignments_terminate_audit_check"
    CHECK (("terminated_at" IS NULL AND "terminated_by" IS NULL) OR ("terminated_at" IS NOT NULL AND "terminated_by" IS NOT NULL)),
  CONSTRAINT "finance_commission_rule_assignments_finance_commission_rules_assignments"
    FOREIGN KEY ("rule_id") REFERENCES "finance_commission_rules" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_rule_assignments_organizations_finance_commission_rule_assignments"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_rule_assignments_users_finance_commission_rule_assignments"
    FOREIGN KEY ("employee_id") REFERENCES "users" ("id") ON DELETE NO ACTION,
  -- 创建人是永久审计事实，删除用户不得清空分配历史。
  CONSTRAINT "finance_commission_rule_assignments_users_created_finance_commission_rule_assignments"
    FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_rule_assignments_users_cancelled_finance_commission_rule_assignments"
    FOREIGN KEY ("cancelled_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "finance_commission_rule_assignments_users_terminated_finance_commission_rule_assignments"
    FOREIGN KEY ("terminated_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

-- 同一方案、员工和分配起始日唯一：允许退出后在另一不重叠区间重新加入同一方案，
-- 禁止重复登记同一区间的起点。
CREATE UNIQUE INDEX "financecommissionruleassignment_organization_id_rule_id_employee_id_effective_from"
  ON "finance_commission_rule_assignments" ("organization_id", "rule_id", "employee_id", "effective_from");

-- 候选与门禁查询：按组织 + 员工定位当前/未来分配（成员停用前检查、计提候选
-- 解析与工作台资格判定）。
CREATE INDEX "financecommissionruleassignment_organization_id_employee_id_effective_to"
  ON "finance_commission_rule_assignments" ("organization_id", "employee_id", "effective_to");

-- 方案名单查询：按方案定位未结束的分配段。
CREATE INDEX "financecommissionruleassignment_rule_id_effective_to"
  ON "finance_commission_rule_assignments" ("rule_id", "effective_to");

CREATE INDEX "financecommissionruleassignment_updated_at"
  ON "finance_commission_rule_assignments" ("updated_at");

-- 停用没有任何员工分配的旧角色规则：迁移执行时即全部既有规则。它们此前按
-- 「组织 + 人员角色」隐式扩散到组织内所有同角色员工，新模型要求方案必须绑定
-- 真实员工分配，禁止无分配的启用方案继续生成新提成或开启工作台资格。
UPDATE "finance_commission_rules"
   SET "enabled" = false,
       "legacy_readonly" = true
WHERE NOT EXISTS (
  SELECT 1
    FROM "finance_commission_rule_assignments" a
   WHERE a."rule_id" = "finance_commission_rules"."id"
);
