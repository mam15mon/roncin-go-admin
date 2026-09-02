-- 本阶段不提供历史数据回填；发现旧海运业务数据时必须停止迁移。
DO $$
DECLARE
  v_count integer;
BEGIN
  IF to_regclass(current_schema() || '.orders') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' LIMIT 1) THEN
    RAISE EXCEPTION '海运主分单内容迁移已停止：orders 存在 SE 业务数据';
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运主分单内容迁移已停止：sea_house_bills 存在数据';
    END IF;
  END IF;
END $$;

-- 1. 扩展 sea_master_bill_order_links 增加单证结构枚举
ALTER TABLE "sea_master_bill_order_links"
  ADD COLUMN "document_structure" character varying NOT NULL DEFAULT 'UNDETERMINED';

ALTER TABLE "sea_master_bill_order_links"
  ADD CONSTRAINT "sea_master_bill_order_links_document_structure_check"
  CHECK ("document_structure" IN ('UNDETERMINED', 'DIRECT', 'HOUSE'));

-- 2. 扩展 sea_master_bills 增加主单内容字段与检查约束
ALTER TABLE "sea_master_bills"
  ADD COLUMN "shipper_text" text,
  ADD COLUMN "consignee_text" text,
  ADD COLUMN "notify_party_text" text,
  ADD COLUMN "second_notify_party_text" text,
  ADD COLUMN "marks_text" text,
  ADD COLUMN "goods_description_text" text,
  ADD COLUMN "package_count" integer,
  ADD COLUMN "package_unit" character varying(64),
  ADD COLUMN "gross_weight_kg" double precision,
  ADD COLUMN "volume_cbm" double precision,
  ADD COLUMN "freight_terms" character varying(64),
  ADD COLUMN "transport_terms" character varying(64),
  ADD COLUMN "bill_form" character varying(64),
  ADD COLUMN "release_type" character varying(64),
  ADD COLUMN "clauses" text;

ALTER TABLE "sea_master_bills"
  ADD CONSTRAINT "sea_master_bills_package_count_check" CHECK ("package_count" IS NULL OR "package_count" >= 0),
  ADD CONSTRAINT "sea_master_bills_gross_weight_kg_check" CHECK ("gross_weight_kg" IS NULL OR "gross_weight_kg" >= 0),
  ADD CONSTRAINT "sea_master_bills_volume_cbm_check" CHECK ("volume_cbm" IS NULL OR "volume_cbm" >= 0);

-- 3. 创建海运真实分单表 sea_house_bills
CREATE TABLE "sea_house_bills" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "master_bill_id" uuid NOT NULL,
  "house_no" character varying(128) NOT NULL,
  "normalized_house_no" character varying(128) NOT NULL,
  "issuer_source" character varying(64) NOT NULL,
  "issuer_organization_id" uuid,
  "issuer_partner_id" uuid,
  "status" character varying(64) NOT NULL DEFAULT 'DRAFT',
  "version" bigint NOT NULL DEFAULT 1,
  "note" character varying(500),
  "shipper_text" text,
  "consignee_text" text,
  "notify_party_text" text,
  "second_notify_party_text" text,
  "marks_text" text,
  "goods_description_text" text,
  "package_count" integer,
  "package_unit" character varying(64),
  "gross_weight_kg" double precision,
  "volume_cbm" double precision,
  "freight_terms" character varying(64),
  "transport_terms" character varying(64),
  "bill_form" character varying(64),
  "release_type" character varying(64),
  "clauses" text,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_house_bills_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED')),
  CONSTRAINT "sea_house_bills_issuer_source_check" CHECK ("issuer_source" IN ('SELF_ORGANIZATION', 'CUSTOMER_PARTNER', 'OTHER_PARTNER')),
  CONSTRAINT "sea_house_bills_issuer_check" CHECK (
    ("issuer_source" = 'SELF_ORGANIZATION' AND "issuer_organization_id" IS NOT NULL AND "issuer_partner_id" IS NULL) OR
    ("issuer_source" IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER') AND "issuer_organization_id" IS NULL AND "issuer_partner_id" IS NOT NULL)
  ),
  CONSTRAINT "sea_house_bills_package_count_check" CHECK ("package_count" IS NULL OR "package_count" >= 0),
  CONSTRAINT "sea_house_bills_gross_weight_kg_check" CHECK ("gross_weight_kg" IS NULL OR "gross_weight_kg" >= 0),
  CONSTRAINT "sea_house_bills_volume_cbm_check" CHECK ("volume_cbm" IS NULL OR "volume_cbm" >= 0),
  CONSTRAINT "sea_house_bills_organizations_sea_house_bills"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bills_orders_sea_house_bills"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bills_sea_master_bills_house_bills"
    FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bills_organizations_issued_sea_house_bills"
    FOREIGN KEY ("issuer_organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bills_partners_issued_sea_house_bills"
    FOREIGN KEY ("issuer_partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seahousebill_updated_at" ON "sea_house_bills" ("updated_at");
CREATE INDEX "seahousebill_organization_id_order_id" ON "sea_house_bills" ("organization_id", "order_id");
CREATE INDEX "seahousebill_organization_id_master_bill_id" ON "sea_house_bills" ("organization_id", "master_bill_id");
CREATE UNIQUE INDEX "idx_sea_house_bills_self_org_unique" ON "sea_house_bills" ("organization_id", "issuer_organization_id", "normalized_house_no") WHERE (issuer_source = 'SELF_ORGANIZATION');
CREATE UNIQUE INDEX "idx_sea_house_bills_partner_unique" ON "sea_house_bills" ("organization_id", "issuer_partner_id", "normalized_house_no") WHERE (issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER'));
