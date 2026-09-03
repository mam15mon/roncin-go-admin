-- 本阶段不提供历史数据回填；发现旧海运业务数据或附件数据时必须停止迁移。
DO $$
DECLARE
  v_count integer;
BEGIN
  IF to_regclass(current_schema() || '.order_attachments') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "order_attachments"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口拆票改配迁移已停止：order_attachments 存在旧数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.orders') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' LIMIT 1) THEN
    RAISE EXCEPTION '海运出口拆票改配迁移已停止：orders 存在 SE 业务数据';
  END IF;
  IF to_regclass(current_schema() || '.sea_cargo_allocations') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_cargo_allocations"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口拆票改配迁移已停止：sea_cargo_allocations 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口拆票改配迁移已停止：sea_house_bills 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bill_order_links') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bill_order_links"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口拆票改配迁移已停止：sea_master_bill_order_links 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口拆票改配迁移已停止：sea_master_bills 存在数据';
    END IF;
  END IF;
END $$;

-- 1. 创建订单附件物理资产表 order_attachment_assets
CREATE TABLE "order_attachment_assets" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "object_key" character varying(1024) NOT NULL,
  "file_name" character varying(255) NOT NULL,
  "mime_type" character varying(127) NOT NULL,
  "file_size" bigint NOT NULL,
  "checksum" character varying(128),
  "organization_id" uuid NOT NULL,
  "uploaded_by" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_attachment_assets_organizations_attachment_assets" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_attachment_assets_users_uploaded_attachment_assets" FOREIGN KEY ("uploaded_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "order_attachment_assets_file_size_check" CHECK ("file_size" > 0)
);

CREATE INDEX "orderattachmentasset_updated_at" ON "order_attachment_assets" ("updated_at");
CREATE UNIQUE INDEX "order_attachment_asset_object_key" ON "order_attachment_assets" ("object_key");
CREATE INDEX "orderattachmentasset_organization_id_created_at" ON "order_attachment_assets" ("organization_id", "created_at");

-- 2. 重构 order_attachments 为订单—资产引用表
DROP INDEX IF EXISTS "order_attachment_object_key";

ALTER TABLE "order_attachments"
  DROP COLUMN IF EXISTS "file_name",
  DROP COLUMN IF EXISTS "mime_type",
  DROP COLUMN IF EXISTS "file_size",
  DROP COLUMN IF EXISTS "object_key",
  DROP COLUMN IF EXISTS "checksum",
  DROP COLUMN IF EXISTS "uploaded_by";

ALTER TABLE "order_attachments"
  ADD COLUMN "asset_id" uuid NOT NULL,
  ADD COLUMN "created_by" uuid;

ALTER TABLE "order_attachments"
  ADD CONSTRAINT "order_attachments_order_attachment_assets_attachments" FOREIGN KEY ("asset_id") REFERENCES "order_attachment_assets" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "order_attachments_users_created_order_attachments" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL;

CREATE UNIQUE INDEX "order_attachment_order_asset" ON "order_attachments" ("order_id", "asset_id");
CREATE INDEX "orderattachment_asset_id" ON "order_attachments" ("asset_id");

-- 3. 扩展 order_lifecycle_events 支持 ORIGIN 来源维度与引用类型/ID
ALTER TABLE "order_lifecycle_events"
  DROP CONSTRAINT IF EXISTS "order_lifecycle_events_dimension_check";

ALTER TABLE "order_lifecycle_events"
  ADD CONSTRAINT "order_lifecycle_events_dimension_check" CHECK ("dimension" IN ('FLOW', 'TERMINATION', 'CLOSURE', 'ORIGIN'));

ALTER TABLE "order_lifecycle_events"
  ADD COLUMN "reference_type" character varying(64),
  ADD COLUMN "reference_id" uuid;

-- 4. 创建海运出口拆分事件表 sea_order_split_events
CREATE TABLE "sea_order_split_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "source_order_no" character varying(64) NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "note" character varying(500),
  "source_order_version" bigint NOT NULL,
  "source_link_id" uuid NOT NULL,
  "source_link_version" bigint NOT NULL,
  "source_allocation_version" bigint NOT NULL,
  "before_snapshot" jsonb NOT NULL,
  "conservation_snapshot" jsonb NOT NULL,
  "source_order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "created_by" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_order_split_events_orders_sea_order_split_events" FOREIGN KEY ("source_order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_events_organizations_sea_order_split_events" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_events_users_created_sea_order_split_events" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

CREATE UNIQUE INDEX "sea_order_split_event_idempotency_key" ON "sea_order_split_events" ("organization_id", "idempotency_key");
CREATE INDEX "seaordersplitevent_organization_id_source_order_id_created_at" ON "sea_order_split_events" ("organization_id", "source_order_id", "created_at");
CREATE INDEX "seaordersplitevent_organization_id_request_fingerprint" ON "sea_order_split_events" ("organization_id", "request_fingerprint");

-- 5. 创建海运出口拆分结果表 sea_order_split_results
CREATE TABLE "sea_order_split_results" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "order_no" character varying(64) NOT NULL,
  "result_role" character varying(32) NOT NULL,
  "sequence" integer NOT NULL,
  "client_result_key" character varying(128) NOT NULL,
  "result_snapshot" jsonb NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "initial_master_bill_id" uuid NOT NULL,
  "final_master_bill_id" uuid NOT NULL,
  "split_event_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_order_split_results_orders_sea_order_split_results" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_results_organizations_sea_order_split_results" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_results_sea_master_bills_initial_sea_order_split_results" FOREIGN KEY ("initial_master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_results_sea_master_bills_final_sea_order_split_results" FOREIGN KEY ("final_master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_results_sea_order_split_events_results" FOREIGN KEY ("split_event_id") REFERENCES "sea_order_split_events" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_split_results_result_role_check" CHECK ("result_role" IN ('ORIGINAL', 'CREATED'))
);

CREATE UNIQUE INDEX "sea_order_split_result_sequence" ON "sea_order_split_results" ("split_event_id", "sequence");
CREATE UNIQUE INDEX "sea_order_split_result_client_key" ON "sea_order_split_results" ("split_event_id", "client_result_key");
CREATE INDEX "seaordersplitresult_organization_id_order_id" ON "sea_order_split_results" ("organization_id", "order_id");

-- 6. 创建海运出口改配事件表 sea_order_reassignment_events
CREATE TABLE "sea_order_reassignment_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "order_no" character varying(64) NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "previous_transport_execution_id" uuid NOT NULL,
  "target_transport_execution_id" uuid NOT NULL,
  "previous_link_id" uuid NOT NULL,
  "target_link_id" uuid NOT NULL,
  "previous_link_version" bigint NOT NULL,
  "target_link_version" bigint NOT NULL,
  "reason" character varying(500) NOT NULL,
  "responsibility_type" character varying(32) NOT NULL,
  "responsible_partner_name" character varying(255),
  "before_snapshot" jsonb NOT NULL,
  "after_snapshot" jsonb NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "responsible_partner_id" uuid,
  "previous_master_bill_id" uuid NOT NULL,
  "target_master_bill_id" uuid NOT NULL,
  "split_event_id" uuid,
  "split_result_id" uuid,
  "created_by" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_order_reassignment_events_orders_sea_order_reassignment_events" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_reassignment_events_organizations_sea_order_reassignment_events" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_reassignment_events_partners_sea_order_reassignments" FOREIGN KEY ("responsible_partner_id") REFERENCES "partners" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_order_reassignment_events_sea_master_bills_previous_sea_order_reassignments" FOREIGN KEY ("previous_master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_reassignment_events_sea_master_bills_target_sea_order_reassignments" FOREIGN KEY ("target_master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_order_reassignment_events_sea_order_split_events_reassignments" FOREIGN KEY ("split_event_id") REFERENCES "sea_order_split_events" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_order_reassignment_events_sea_order_split_results_reassignment_events" FOREIGN KEY ("split_result_id") REFERENCES "sea_order_split_results" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_order_reassignment_events_users_created_sea_order_reassignment_events" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_order_reassignment_events_responsibility_type_check" CHECK ("responsibility_type" IN ('CARRIER', 'CUSTOMER', 'CUSTOMS', 'OWN_COMPANY', 'FORCE_MAJEURE', 'OTHER'))
);

CREATE UNIQUE INDEX "sea_order_reassignment_event_idempotency_key" ON "sea_order_reassignment_events" ("organization_id", "idempotency_key");
CREATE INDEX "seaorderreassignmentevent_organization_id_order_id_created_at" ON "sea_order_reassignment_events" ("organization_id", "order_id", "created_at");
CREATE INDEX "seaorderreassignmentevent_organization_id_request_fingerprint" ON "sea_order_reassignment_events" ("organization_id", "request_fingerprint");
