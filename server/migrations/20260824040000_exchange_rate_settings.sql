-- 组织级订单费用结算汇率主数据；金额和汇率使用 numeric，禁止浮点。
CREATE TABLE "exchange_rate_settings" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "rate_type" character varying NOT NULL,
  "from_currency" character varying(3) NOT NULL,
  "to_currency" character varying(3) NOT NULL,
  "time_standard" character varying NOT NULL,
  "effective_from" character varying(10) NOT NULL,
  "effective_to" character varying(10) NULL,
  "receivable_rate" numeric(18,8) NOT NULL,
  "payable_rate" numeric(18,8) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "exchange_rate_settings_rate_type_check" CHECK ("rate_type" IN ('SETTLEMENT')),
  CONSTRAINT "exchange_rate_settings_time_standard_check" CHECK ("time_standard" IN ('EXPENSE_DATE')),
  CONSTRAINT "exchange_rate_settings_currency_different" CHECK ("from_currency" <> "to_currency"),
  CONSTRAINT "exchange_rate_settings_rates_positive" CHECK ("receivable_rate" > 0 AND "payable_rate" > 0),
  CONSTRAINT "exchange_rate_settings_organizations_exchange_rates"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "exchange_rate_setting_unique_effective_from"
  ON "exchange_rate_settings" ("organization_id", "rate_type", "from_currency", "to_currency", "time_standard", "effective_from");
CREATE INDEX "exchange_rate_setting_active_lookup"
  ON "exchange_rate_settings" ("organization_id", "rate_type", "from_currency", "to_currency", "time_standard", "is_active");
CREATE INDEX "exchange_rate_setting_effective_range"
  ON "exchange_rate_settings" ("organization_id", "effective_from", "effective_to");
CREATE INDEX "exchange_rate_setting_updated_at" ON "exchange_rate_settings" ("updated_at");

INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description")
VALUES
  (md5('system.finance.exchange_rate.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.exchange_rate.read', '查看汇率', '财务管理 · 汇率', '查看组织结算汇率主数据'),
  (md5('system.finance.exchange_rate.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.exchange_rate.create', '新建汇率', '财务管理 · 汇率', '新建组织结算汇率'),
  (md5('system.finance.exchange_rate.update')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.exchange_rate.update', '编辑汇率', '财务管理 · 汇率', '修改组织结算汇率'),
  (md5('system.finance.exchange_rate.disable')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.exchange_rate.disable', '停用汇率', '财务管理 · 汇率', '停用组织结算汇率')
ON CONFLICT ("key") DO UPDATE SET "updated_at" = EXCLUDED."updated_at", "name" = EXCLUDED."name", "group" = EXCLUDED."group", "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r."id", p."id"
FROM "roles" r CROSS JOIN "permissions" p
WHERE r."code" = 'administrator' AND p."key" LIKE 'system.finance.exchange_rate.%'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
