-- 本阶段不提供历史数据回填；发现旧主分单业务数据时必须停止迁移。
DO $$
BEGIN
  IF to_regclass('public.order_consolidations') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "order_consolidations" LIMIT 1) THEN
    RAISE EXCEPTION '海运共享主单基础迁移已停止：order_consolidations 存在历史数据';
  END IF;
  IF to_regclass('public.order_shipping_documents') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "order_shipping_documents" LIMIT 1) THEN
    RAISE EXCEPTION '海运共享主单基础迁移已停止：order_shipping_documents 存在历史数据';
  END IF;
END $$;

-- 1. 创建海运独立运输执行表
CREATE TABLE "sea_transport_executions" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "carrier_id" uuid,
  "origin_location_id" uuid,
  "discharge_location_id" uuid,
  "transit_location_id" uuid,
  "vessel_name" character varying(128) NOT NULL DEFAULT '',
  "voyage_no" character varying(64) NOT NULL DEFAULT '',
  "etd" timestamp with time zone,
  "eta" timestamp with time zone,
  "version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_transport_executions_organizations_sea_transport_executions"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seatransportexecution_updated_at" ON "sea_transport_executions" ("updated_at");
CREATE INDEX "seatransportexecution_organization_id_carrier_id" ON "sea_transport_executions" ("organization_id", "carrier_id");
CREATE INDEX "seatransportexecution_organization_id_origin_location_id_discharge_location_id" ON "sea_transport_executions" ("organization_id", "origin_location_id", "discharge_location_id");

-- 2. 创建海运共享 MBL 主单表
CREATE TABLE "sea_master_bills" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "issuer_partner_id" uuid NOT NULL,
  "transport_execution_id" uuid NOT NULL,
  "master_no" character varying(64) NOT NULL,
  "normalized_master_no" character varying(64) NOT NULL,
  "status" character varying NOT NULL DEFAULT 'DRAFT',
  "version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_master_bills_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED')),
  CONSTRAINT "sea_master_bills_organizations_sea_master_bills"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bills_sea_transport_executions_master_bills"
    FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seamasterbill_updated_at" ON "sea_master_bills" ("updated_at");
CREATE UNIQUE INDEX "seamasterbill_organization_id_issuer_partner_id_normalized_master_no" ON "sea_master_bills" ("organization_id", "issuer_partner_id", "normalized_master_no");
CREATE INDEX "seamasterbill_organization_id_transport_execution_id" ON "sea_master_bills" ("organization_id", "transport_execution_id");

-- 3. 创建海运操作票—共享 MBL 当前/历史成员关联表
CREATE TABLE "sea_master_bill_order_links" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "master_bill_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "status" character varying NOT NULL DEFAULT 'ACTIVE',
  "started_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "ended_at" timestamp with time zone,
  "ended_reason" character varying(255),
  "version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_master_bill_order_links_status_check" CHECK ("status" IN ('ACTIVE', 'ENDED')),
  CONSTRAINT "sea_master_bill_order_links_organizations_sea_master_bill_order_links"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_order_links_sea_master_bills_order_links"
    FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_order_links_orders_sea_master_bill_links"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seamasterbillorderlink_updated_at" ON "sea_master_bill_order_links" ("updated_at");
CREATE INDEX "seamasterbillorderlink_organization_id_master_bill_id" ON "sea_master_bill_order_links" ("organization_id", "master_bill_id");
CREATE INDEX "seamasterbillorderlink_organization_id_order_id" ON "sea_master_bill_order_links" ("organization_id", "order_id");
CREATE UNIQUE INDEX "idx_sea_mbl_order_links_active_order" ON "sea_master_bill_order_links" ("order_id") WHERE status = 'ACTIVE';

-- 4. 从 order_shipping_documents 移除 consolidation_id 依赖，并删除旧 order_consolidations
ALTER TABLE "order_shipping_documents" DROP CONSTRAINT IF EXISTS "order_shipping_documents_order_consolidations_shipping_documents";
DROP INDEX IF EXISTS "ordershippingdocument_order_id_consolidation_id";
ALTER TABLE "order_shipping_documents" DROP COLUMN IF EXISTS "consolidation_id";
ALTER TABLE "order_shipping_documents" DROP COLUMN IF EXISTS "master_no";
ALTER TABLE "order_shipping_documents" DROP COLUMN IF EXISTS "master_document_type";
ALTER TABLE "order_shipping_documents" DROP COLUMN IF EXISTS "master_release_method";

DROP TABLE IF EXISTS "order_consolidations";
