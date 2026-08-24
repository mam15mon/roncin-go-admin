-- 费用设置、计费单位及货物或应税劳务名称主数据。
CREATE TABLE "billing_units" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "code" character varying(32) NOT NULL,
  "name" character varying(64) NOT NULL,
  "sort_order" bigint NOT NULL DEFAULT 100,
  "enabled" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_units_organizations_billing_units"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "billingunit_organization_id_code" ON "billing_units" ("organization_id", "code");
CREATE INDEX "billingunit_organization_id_enabled_sort_order" ON "billing_units" ("organization_id", "enabled", "sort_order");
CREATE INDEX "billingunit_updated_at" ON "billing_units" ("updated_at");

CREATE TABLE "taxable_services" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" character varying(128) NOT NULL,
  "short_name" character varying(64) NULL,
  "goods_code" character varying(64) NULL,
  "default_tax_rate" numeric(5,2) NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "taxable_services_tax_rate_range" CHECK ("default_tax_rate" >= 0 AND "default_tax_rate" <= 100),
  CONSTRAINT "taxable_services_organizations_taxable_services"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "taxableservice_organization_id_name" ON "taxable_services" ("organization_id", "name");
CREATE INDEX "taxableservice_organization_id_enabled_name" ON "taxable_services" ("organization_id", "enabled", "name");
CREATE INDEX "taxableservice_updated_at" ON "taxable_services" ("updated_at");

CREATE TABLE "fee_settings" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "fee_code" character varying(32) NOT NULL,
  "name_zh" character varying(64) NOT NULL,
  "name_en" character varying(128) NULL,
  "alias_name" character varying(64) NULL,
  "service_type_id" uuid NULL,
  "default_currency" character varying(3) NOT NULL,
  "billing_unit_id" uuid NOT NULL,
  "abnormal_case_id" uuid NULL,
  "tax_rate" numeric(5,2) NOT NULL,
  "taxable_service_id" uuid NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "sort_order" bigint NOT NULL DEFAULT 100,
  PRIMARY KEY ("id"),
  CONSTRAINT "fee_settings_tax_rate_range" CHECK ("tax_rate" >= 0 AND "tax_rate" <= 100),
  CONSTRAINT "fee_settings_organizations_fee_settings"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "fee_settings_master_data_service_type"
    FOREIGN KEY ("service_type_id") REFERENCES "master_data_items" ("id") ON DELETE NO ACTION,
  CONSTRAINT "fee_settings_billing_units_fee_settings"
    FOREIGN KEY ("billing_unit_id") REFERENCES "billing_units" ("id") ON DELETE NO ACTION,
  CONSTRAINT "fee_settings_master_data_abnormal_case"
    FOREIGN KEY ("abnormal_case_id") REFERENCES "master_data_items" ("id") ON DELETE NO ACTION,
  CONSTRAINT "fee_settings_taxable_services_fee_settings"
    FOREIGN KEY ("taxable_service_id") REFERENCES "taxable_services" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "feesetting_organization_id_fee_code" ON "fee_settings" ("organization_id", "fee_code");
CREATE INDEX "feesetting_organization_id_enabled_sort_order" ON "fee_settings" ("organization_id", "enabled", "sort_order");
CREATE INDEX "feesetting_organization_id_service_type_id_abnormal_case_id" ON "fee_settings" ("organization_id", "service_type_id", "abnormal_case_id");
CREATE INDEX "feesetting_updated_at" ON "fee_settings" ("updated_at");

INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description") VALUES
  (md5('system.finance.fee_setting.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.fee_setting.read', '查看费用设置', '财务管理 · 费用设置', '查看费用设置及关联基础资料'),
  (md5('system.finance.fee_setting.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.fee_setting.create', '新建费用设置', '财务管理 · 费用设置', '新建费用设置及关联基础资料'),
  (md5('system.finance.fee_setting.update')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.fee_setting.update', '编辑费用设置', '财务管理 · 费用设置', '编辑和停用费用设置及关联基础资料')
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT role."id", permission."id"
FROM "roles" AS role
JOIN "permissions" AS permission ON permission."key" LIKE 'system.finance.fee_setting.%'
WHERE role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
