-- 月度提成申请 Schema：
-- 1) 新增 finance_commission_applications：员工在组织内按提交自然月形成的
--    月度提成申请头。唯一索引「组织 + 员工 + 申请月」含被驳回行，结构上禁止
--    替代申请与并行申请；覆盖截止日固定为提交月份前一自然月最后一天，申请头
--    汇总（笔数、本位币/CNY 合计）为提交当时的不可变快照。提交人是永久审计
--    事实（NO ACTION），决策人是可逆审计引用（SET NULL），驳回原因必填由
--    业务层校验、列可空；
-- 2) 新增 finance_commission_application_lines：申请明细逐笔固化的提成事实
--    快照（归属日期、来源核销/对冲、人员身份、方案与金额），组织与员工冗余列
--    与申请头一致。外键全部 NO ACTION，禁止删除造成历史断链；
--    commission_id 全局唯一——一个提成事实至多进入一张申请，被驳回也不回流
--    公共池；「申请 + 提成」唯一索引同时支撑申请详情明细下钻。
-- 纯增量 DDL：不修改既有表、不做任何历史数据回填，旧提成没有申请关系时保持
-- 现状，可在员工下一次申请中被纳入；禁止生成虚假历史申请。

CREATE TABLE "finance_commission_applications" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "application_month" character varying(7) NOT NULL,
  "coverage_to" character varying(10) NOT NULL,
  "status" character varying NOT NULL DEFAULT 'PENDING_REVIEW',
  "version" bigint NOT NULL DEFAULT 1,
  "commission_count" bigint NOT NULL,
  "base_currency" character varying(3) NOT NULL,
  "total_commission_amount" numeric(28,8) NOT NULL,
  "total_cny_commission_amount" numeric(28,8) NOT NULL,
  "submitted_at" timestamp with time zone NOT NULL,
  "decided_at" timestamp with time zone NULL,
  "decision_reason" character varying(500) NULL,
  "organization_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  "submitted_by" uuid NOT NULL,
  "decided_by" uuid NULL,
  PRIMARY KEY ("id"),
  -- 申请状态取值；PENDING_REVIEW 只能进入 APPROVED 或 REJECTED 由状态机与
  -- 仓储校验保证。
  CONSTRAINT "finance_commission_applications_status_check"
    CHECK ("status" IN ('PENDING_REVIEW', 'REJECTED', 'APPROVED')),
  -- 汇总口径非负：申请只纳入合格提成事实。
  CONSTRAINT "finance_commission_applications_commission_count_non_negative"
    CHECK ("commission_count" >= 0),
  CONSTRAINT "finance_commission_applications_total_commission_amount_non_negative"
    CHECK ("total_commission_amount" >= 0),
  CONSTRAINT "finance_commission_applications_total_cny_commission_amount_non_negative"
    CHECK ("total_cny_commission_amount" >= 0),
  CONSTRAINT "finance_commission_applications_organizations_finance_commission_applications"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_applications_users_finance_commission_applications"
    FOREIGN KEY ("employee_id") REFERENCES "users" ("id") ON DELETE NO ACTION,
  -- 提交人是永久审计事实，删除用户不得清空申请历史。
  CONSTRAINT "finance_commission_applications_users_submitted_finance_commission_applications"
    FOREIGN KEY ("submitted_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  -- 决策人是可逆审计引用，删除用户时置空（SET NULL）即可。
  CONSTRAINT "finance_commission_applications_users_decided_finance_commission_applications"
    FOREIGN KEY ("decided_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

-- 一人一组织一提交月一行（含被驳回行）：结构上禁止替代申请与并行申请。
CREATE UNIQUE INDEX "financecommissionapplication_organization_id_employee_id_application_month"
  ON "finance_commission_applications" ("organization_id", "employee_id", "application_month");

-- 本人工作台与财务列表按员工 + 状态定位申请批次。
CREATE INDEX "financecommissionapplication_organization_id_employee_id_status"
  ON "finance_commission_applications" ("organization_id", "employee_id", "status");

-- 财务按提交月过滤申请批次。
CREATE INDEX "financecommissionapplication_organization_id_application_month"
  ON "finance_commission_applications" ("organization_id", "application_month");

CREATE INDEX "financecommissionapplication_updated_at"
  ON "finance_commission_applications" ("updated_at");

CREATE TABLE "finance_commission_application_lines" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "commission_date" character varying(10) NOT NULL,
  "verification_id" uuid NULL,
  "verification_no" character varying(64) NULL,
  "netting_id" uuid NULL,
  "netting_no" character varying(64) NULL,
  "personnel_role" character varying(20) NOT NULL,
  "rule_id" uuid NULL,
  "rule_version" bigint NOT NULL DEFAULT 1,
  "rule_name" character varying(100) NULL,
  "calculation_basis" character varying(30) NULL,
  "base_currency" character varying(3) NOT NULL,
  "commission_amount" numeric(28,8) NOT NULL,
  "cny_commission_amount" numeric(28,8) NOT NULL,
  "source_fingerprint" character varying(64) NOT NULL DEFAULT '',
  "commission_id" uuid NOT NULL,
  "application_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "employee_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  -- 明细金额快照非负：与提成单金额口径一致。
  CONSTRAINT "finance_commission_application_lines_commission_amount_non_negative"
    CHECK ("commission_amount" >= 0),
  CONSTRAINT "finance_commission_application_lines_cny_commission_amount_non_negative"
    CHECK ("cny_commission_amount" >= 0),
  CONSTRAINT "finance_commission_application_lines_finance_commissions_application_lines"
    FOREIGN KEY ("commission_id") REFERENCES "finance_commissions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_application_lines_finance_commission_applications_lines"
    FOREIGN KEY ("application_id") REFERENCES "finance_commission_applications" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_application_lines_organizations_finance_commission_application_lines"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "finance_commission_application_lines_users_finance_commission_application_lines"
    FOREIGN KEY ("employee_id") REFERENCES "users" ("id") ON DELETE NO ACTION
);

-- 申请内明细与提成事实一一对应；application_id 前导同时支撑申请详情明细
-- 下钻查询，无需单独的 application_id 索引。
CREATE UNIQUE INDEX "financecommissionapplicationline_application_id_commission_id"
  ON "finance_commission_application_lines" ("application_id", "commission_id");

-- 一个提成事实至多进入一张申请：跨申请、跨月份全局唯一，被驳回也不回流公共池。
CREATE UNIQUE INDEX "financecommissionapplicationline_commission_id"
  ON "finance_commission_application_lines" ("commission_id");

-- 本人跨申请明细查询与组织隔离过滤。
CREATE INDEX "financecommissionapplicationline_organization_id_employee_id"
  ON "finance_commission_application_lines" ("organization_id", "employee_id");

CREATE INDEX "financecommissionapplicationline_updated_at"
  ON "finance_commission_application_lines" ("updated_at");
