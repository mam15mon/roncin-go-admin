CREATE TABLE "currencies" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "code" character varying NOT NULL,
  "name" character varying NOT NULL,
  "symbol" character varying NULL,
  "minor_unit" bigint NOT NULL DEFAULT 2,
  "enabled" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "currency_code" ON "currencies" ("code");
CREATE INDEX "currency_enabled_code" ON "currencies" ("enabled", "code");
CREATE INDEX "currency_updated_at" ON "currencies" ("updated_at");

INSERT INTO "currencies" ("id", "created_at", "updated_at", "code", "name", "symbol", "minor_unit", "enabled")
VALUES
  ('0198cf68-5ba2-7d46-bbac-da40876e7201'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'CNY', '人民币', '¥', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7202'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD', '美元', '$', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7203'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'EUR', '欧元', '€', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7204'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'HKD', '港币', 'HK$', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7205'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'JPY', '日元', '¥', 0, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7206'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'GBP', '英镑', '£', 2, true);

CREATE TABLE "administrative_regions" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "code" character varying NOT NULL,
  "name" character varying NOT NULL,
  "level" bigint NOT NULL,
  "parent_code" character varying NULL,
  "region_type" character varying NULL,
  "source" character varying NOT NULL DEFAULT 'MCA_DMFW',
  "source_version" character varying NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX "administrativeregion_code" ON "administrative_regions" ("code");
CREATE INDEX "administrativeregion_level_code" ON "administrative_regions" ("level", "code");
CREATE INDEX "administrativeregion_parent_code_level_code" ON "administrative_regions" ("parent_code", "level", "code");
CREATE INDEX "administrativeregion_name" ON "administrative_regions" ("name");
CREATE INDEX "administrativeregion_updated_at" ON "administrative_regions" ("updated_at");

CREATE TABLE "partner_profiles" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "name_en" character varying NULL,
  "address_en" character varying NULL,
  "country_code" character varying NOT NULL DEFAULT 'CN',
  "province_code" character varying NULL,
  "city_code" character varying NULL,
  "district_code" character varying NULL,
  "address_detail" character varying NULL,
  "nature" character varying NULL,
  "development_method" character varying NULL,
  "customer_types" jsonb NULL,
  "business_types" jsonb NULL,
  "remark" character varying NULL,
  "partner_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "partner_profiles_partners_profile" FOREIGN KEY ("partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "partnerprofile_partner_id" ON "partner_profiles" ("partner_id");
CREATE INDEX "partnerprofile_province_code_city_code_district_code" ON "partner_profiles" ("province_code", "city_code", "district_code");
CREATE INDEX "partnerprofile_updated_at" ON "partner_profiles" ("updated_at");

CREATE TABLE "partner_assignments" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "role" character varying NOT NULL,
  "organization_id" uuid NOT NULL,
  "partner_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "partner_assignments_organizations_partner_assignments" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "partner_assignments_partners_assignments" FOREIGN KEY ("partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION,
  CONSTRAINT "partner_assignments_users_partner_assignments" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "partnerassignment_partner_id_role" ON "partner_assignments" ("partner_id", "role");
CREATE INDEX "partnerassignment_user_id_role" ON "partner_assignments" ("user_id", "role");
CREATE INDEX "partnerassignment_organization_id_role" ON "partner_assignments" ("organization_id", "role");
CREATE INDEX "partnerassignment_updated_at" ON "partner_assignments" ("updated_at");

ALTER TABLE "partner_settlement_rules"
  ADD COLUMN "credit_limit_minor" bigint NULL,
  ADD COLUMN "credit_currency" character varying NULL;
