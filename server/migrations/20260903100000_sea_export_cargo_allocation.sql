-- 本阶段不提供历史数据回填；发现旧海运业务数据时必须停止迁移。
DO $$
DECLARE
  v_count integer;
BEGIN
  IF to_regclass(current_schema() || '.orders') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' LIMIT 1) THEN
    RAISE EXCEPTION '海运箱货分配迁移已停止：orders 存在 SE 业务数据';
  END IF;
  IF to_regclass(current_schema() || '.order_cargo_items') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "order_cargo_items"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运箱货分配迁移已停止：order_cargo_items 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.order_containers') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "order_containers"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运箱货分配迁移已停止：order_containers 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_cargo_allocations') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_cargo_allocations"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运箱货分配迁移已停止：sea_cargo_allocations 存在数据';
    END IF;
  END IF;
END $$;

-- 1. 扩展 sea_master_bill_order_links 增加箱货分配状态、版本与确认信息
ALTER TABLE "sea_master_bill_order_links"
  ADD COLUMN "cargo_allocation_status" character varying(64) NOT NULL DEFAULT 'DRAFT',
  ADD COLUMN "cargo_allocation_version" bigint NOT NULL DEFAULT 1,
  ADD COLUMN "cargo_allocation_confirmed_at" timestamp with time zone,
  ADD COLUMN "cargo_allocation_confirmed_by" uuid;

ALTER TABLE "sea_master_bill_order_links"
  ADD CONSTRAINT "sea_master_bill_order_links_cargo_allocation_status_check"
  CHECK ("cargo_allocation_status" IN ('DRAFT', 'CONFIRMED')),
  ADD CONSTRAINT "sea_master_bill_order_links_users_cargo_allocation_confirmed_by"
  FOREIGN KEY ("cargo_allocation_confirmed_by") REFERENCES "users" ("id") ON DELETE NO ACTION;

-- 2. 扩展 order_cargo_items 增加组织、版本，规范十进制精度与检查约束
ALTER TABLE "order_cargo_items"
  ADD COLUMN "organization_id" uuid NOT NULL,
  ADD COLUMN "version" bigint NOT NULL DEFAULT 1;

ALTER TABLE "order_cargo_items"
  ALTER COLUMN "gross_weight_kg" TYPE numeric(18,3),
  ALTER COLUMN "volume_cbm" TYPE numeric(18,6),
  ALTER COLUMN "net_weight_kg" TYPE numeric(18,3);

ALTER TABLE "order_cargo_items"
  ADD CONSTRAINT "order_cargo_items_organizations_order_cargo_items"
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;

CREATE INDEX "ordercargoitem_organization_id" ON "order_cargo_items" ("organization_id");

ALTER TABLE "order_cargo_items"
  ADD CONSTRAINT "order_cargo_items_package_count_check" CHECK ("package_count" > 0),
  ADD CONSTRAINT "order_cargo_items_gross_weight_kg_check" CHECK ("gross_weight_kg" > 0),
  ADD CONSTRAINT "order_cargo_items_volume_cbm_check" CHECK ("volume_cbm" > 0),
  ADD CONSTRAINT "order_cargo_items_net_weight_kg_check" CHECK ("net_weight_kg" IS NULL OR "net_weight_kg" > 0);

-- 3. 扩展 order_containers 增加组织、版本、件数，移除旧提单单值引用，规范十进制精度与检查约束
ALTER TABLE "order_containers"
  DROP CONSTRAINT IF EXISTS "order_containers_order_shipping_documents_containers";

DROP INDEX IF EXISTS "ordercontainer_shipping_document_id";

ALTER TABLE "order_containers"
  DROP COLUMN IF EXISTS "shipping_document_id";

ALTER TABLE "order_containers"
  ADD COLUMN "organization_id" uuid NOT NULL,
  ADD COLUMN "version" bigint NOT NULL DEFAULT 1,
  ADD COLUMN "package_count" integer NOT NULL;

ALTER TABLE "order_containers"
  ALTER COLUMN "gross_weight_kg" TYPE numeric(18,3),
  ALTER COLUMN "volume_cbm" TYPE numeric(18,6);

ALTER TABLE "order_containers"
  ADD CONSTRAINT "order_containers_organizations_order_containers"
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;

CREATE INDEX "ordercontainer_organization_id" ON "order_containers" ("organization_id");

ALTER TABLE "order_containers"
  ADD CONSTRAINT "order_containers_package_count_check" CHECK ("package_count" > 0),
  ADD CONSTRAINT "order_containers_gross_weight_kg_check" CHECK ("gross_weight_kg" > 0),
  ADD CONSTRAINT "order_containers_volume_cbm_check" CHECK ("volume_cbm" > 0);

-- 4. 创建海运箱货分配事实表 sea_cargo_allocations
CREATE TABLE "sea_cargo_allocations" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "master_bill_order_link_id" uuid NOT NULL,
  "cargo_item_id" uuid NOT NULL,
  "house_bill_id" uuid NOT NULL,
  "container_id" uuid,
  "package_count" integer NOT NULL,
  "gross_weight_kg" numeric(18,3) NOT NULL,
  "volume_cbm" numeric(18,6) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_cargo_allocations_package_count_check" CHECK ("package_count" > 0),
  CONSTRAINT "sea_cargo_allocations_gross_weight_kg_check" CHECK ("gross_weight_kg" > 0),
  CONSTRAINT "sea_cargo_allocations_volume_cbm_check" CHECK ("volume_cbm" > 0),
  CONSTRAINT "sea_cargo_allocations_organizations_sea_cargo_allocations"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_cargo_allocations_orders_sea_cargo_allocations"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_cargo_allocations_links_cargo_allocations"
    FOREIGN KEY ("master_bill_order_link_id") REFERENCES "sea_master_bill_order_links" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_cargo_allocations_cargo_items_cargo_allocations"
    FOREIGN KEY ("cargo_item_id") REFERENCES "order_cargo_items" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_cargo_allocations_house_bills_cargo_allocations"
    FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_cargo_allocations_containers_cargo_allocations"
    FOREIGN KEY ("container_id") REFERENCES "order_containers" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seacargoallocation_organization_id_order_id"
  ON "sea_cargo_allocations" ("organization_id", "order_id");
CREATE INDEX "seacargoallocation_master_bill_order_link_id"
  ON "sea_cargo_allocations" ("master_bill_order_link_id");
CREATE INDEX "seacargoallocation_cargo_item_id"
  ON "sea_cargo_allocations" ("cargo_item_id");
CREATE INDEX "seacargoallocation_house_bill_id"
  ON "sea_cargo_allocations" ("house_bill_id");
CREATE INDEX "seacargoallocation_container_id"
  ON "sea_cargo_allocations" ("container_id");
CREATE UNIQUE INDEX "idx_sea_cargo_allocations_no_cntr_unique"
  ON "sea_cargo_allocations" ("master_bill_order_link_id", "cargo_item_id", "house_bill_id")
  WHERE "container_id" IS NULL;
CREATE UNIQUE INDEX "idx_sea_cargo_allocations_cntr_unique"
  ON "sea_cargo_allocations" ("master_bill_order_link_id", "cargo_item_id", "house_bill_id", "container_id")
  WHERE "container_id" IS NOT NULL;
