-- 简化海运出口主分单数据模型：解耦 MBL 与运输执行，收敛单值 HBL，引入外部确认与运输版本，废除 Switch 与旧箱货分配。
-- 所有前置检查必须在 DDL 前完成，若发现任一相关 SE 历史数据，迁移执行器会在单一事务中原子回滚并中止。
DO $$
DECLARE
  v_count integer;
BEGIN
  IF to_regclass(current_schema() || '.orders') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' LIMIT 1) THEN
    RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：orders 存在 SE 业务数据';
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_master_bills 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bill_versions') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bill_versions"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_master_bill_versions 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bill_order_links') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bill_order_links"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_master_bill_order_links 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_house_bills 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bill_versions') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bill_versions"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_house_bill_versions 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bill_switch_events') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bill_switch_events"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_house_bill_switch_events 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_cargo_allocations') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_cargo_allocations"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_cargo_allocations 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_transport_executions') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_transport_executions"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_transport_executions 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_order_split_events') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_order_split_events"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_order_split_events 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_order_split_results') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_order_split_results"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_order_split_results 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_order_reassignment_events') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_order_reassignment_events"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_order_reassignment_events 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_document_void_events') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_document_void_events"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：sea_document_void_events 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.order_release_pods') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "order_release_pods" WHERE "sea_master_bill_id" IS NOT NULL OR "sea_house_bill_id" IS NOT NULL LIMIT 1) THEN
    RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：order_release_pods 存在海运单证引用数据';
  END IF;
  IF to_regclass(current_schema() || '.order_lock_records') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "order_lock_records" WHERE "business_type" = 'SE' OR "master_bill_id" IS NOT NULL OR "master_bill_version_id" IS NOT NULL LIMIT 1) THEN
    RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：order_lock_records 存在海运锁定数据';
  END IF;
  IF to_regclass(current_schema() || '.order_lock_house_bill_snapshots') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "order_lock_house_bill_snapshots"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '简化海运出口主分单模型迁移已停止：order_lock_house_bill_snapshots 存在数据';
    END IF;
  END IF;
END $$;

-- 1. 删除旧表：sea_cargo_allocations 与 sea_house_bill_switch_events
DROP TABLE "sea_cargo_allocations";
DROP TABLE "sea_house_bill_switch_events";

-- 2. sea_master_bills 解耦实际运输执行
ALTER TABLE "sea_master_bills"
  DROP CONSTRAINT "sea_master_bills_sea_transport_executions_master_bills";
DROP INDEX "seamasterbill_organization_id_transport_execution_id";
ALTER TABLE "sea_master_bills"
  DROP COLUMN "transport_execution_id";

-- 3. 创建运输执行版本表 sea_transport_execution_versions
CREATE TABLE "sea_transport_execution_versions" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "version_no" bigint NOT NULL,
  "source_entity_version" bigint NOT NULL,
  "origin_location_id" uuid,
  "discharge_location_id" uuid,
  "transit_location_id" uuid,
  "vessel_name" character varying(128) NOT NULL DEFAULT '',
  "voyage_no" character varying(64) NOT NULL DEFAULT '',
  "etd" timestamp with time zone,
  "eta" timestamp with time zone,
  "content_hash" character varying(64) NOT NULL,
  "source" character varying(32) NOT NULL,
  "reason" character varying(500),
  "idempotency_key" character varying(128),
  "request_fingerprint" character varying(128),
  "confirmed_by_party" character varying(128),
  "confirmed_at" timestamp with time zone,
  "confirmation_note" character varying(500),
  "confirmation_attachment_id" uuid,
  "organization_id" uuid NOT NULL,
  "transport_execution_id" uuid NOT NULL,
  "shipping_line_id" uuid NOT NULL,
  "created_by" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_transport_execution_versions_order_attachments_sea_transport_execution_versions" FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_transport_execution_versions_organizations_sea_transport_execution_versions" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_transport_execution_versions_sea_transport_executions_versions" FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_transport_execution_versions_shipping_lines_sea_transport_execution_versions" FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_transport_execution_versions_users_created_sea_transport_execution_versions" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_transport_execution_versions_source_check" CHECK ("source" IN ('ORDER_LOCK', 'SHARED_UPDATE', 'REASSIGNMENT'))
);

CREATE UNIQUE INDEX "sea_te_version_transport_version_no" ON "sea_transport_execution_versions" ("transport_execution_id", "version_no");
CREATE UNIQUE INDEX "sea_te_version_source_hash" ON "sea_transport_execution_versions" ("transport_execution_id", "source_entity_version", "content_hash");
CREATE INDEX "seatransportexecutionversion_organization_id_transport_execution_id" ON "sea_transport_execution_versions" ("organization_id", "transport_execution_id");
CREATE UNIQUE INDEX "sea_te_version_idempotency_key" ON "sea_transport_execution_versions" ("organization_id", "idempotency_key");

-- 4. sea_transport_executions 增加当前版本引用
ALTER TABLE "sea_transport_executions"
  ADD COLUMN "current_version_id" uuid;

ALTER TABLE "sea_transport_executions"
  ADD CONSTRAINT "sea_transport_executions_sea_transport_execution_versions_current_version"
  FOREIGN KEY ("current_version_id") REFERENCES "sea_transport_execution_versions" ("id") ON DELETE SET NULL;

-- 5. orders 增加 booking_no 字段与索引
ALTER TABLE "orders"
  ADD COLUMN "booking_no" character varying(100);

CREATE INDEX "order_organization_id_booking_no" ON "orders" ("organization_id", "booking_no");

-- 6. sea_master_bill_order_links 调整：增加 transport_execution_id，移除旧箱货分配，收敛为 DIRECT | HOUSE 两态
ALTER TABLE "sea_master_bill_order_links"
  DROP COLUMN "cargo_allocation_status",
  DROP COLUMN "cargo_allocation_version",
  DROP COLUMN "cargo_allocation_confirmed_at",
  DROP COLUMN "cargo_allocation_confirmed_by";

ALTER TABLE "sea_master_bill_order_links"
  ADD COLUMN "transport_execution_id" uuid NOT NULL;

ALTER TABLE "sea_master_bill_order_links"
  ADD CONSTRAINT "sea_master_bill_order_links_sea_transport_executions_order_links"
  FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION;

CREATE INDEX "seamasterbillorderlink_organization_id_transport_execution_id"
  ON "sea_master_bill_order_links" ("organization_id", "transport_execution_id");

ALTER TABLE "sea_master_bill_order_links"
  DROP CONSTRAINT "sea_master_bill_order_links_document_structure_check";

ALTER TABLE "sea_master_bill_order_links"
  ALTER COLUMN "document_structure" DROP DEFAULT,
  ADD CONSTRAINT "sea_master_bill_order_links_document_structure_check"
  CHECK ("document_structure" IN ('DIRECT', 'HOUSE'));

-- 7. sea_house_bills 调整：删除 REPLACED 状态，增加当前有效 HBL 条件唯一索引
ALTER TABLE "sea_house_bills"
  DROP CONSTRAINT "sea_house_bills_status_check";

ALTER TABLE "sea_house_bills"
  ADD CONSTRAINT "sea_house_bills_status_check"
  CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED'));

CREATE UNIQUE INDEX "idx_sea_house_bills_current_order_unique"
  ON "sea_house_bills" ("order_id")
  WHERE "status" IN ('DRAFT', 'CONFIRMED', 'RELEASED');

-- 8. sea_house_bill_versions 调整：增加外部确认字段、更新 source/status 枚举
ALTER TABLE "sea_house_bill_versions"
  ADD COLUMN "confirmed_by_party" character varying(128),
  ADD COLUMN "confirmed_at" timestamp with time zone,
  ADD COLUMN "confirmation_note" character varying(500),
  ADD COLUMN "confirmation_attachment_id" uuid;

ALTER TABLE "sea_house_bill_versions"
  ADD CONSTRAINT "sea_house_bill_versions_order_attachments_sea_house_bill_versions"
  FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION;

ALTER TABLE "sea_house_bill_versions"
  DROP CONSTRAINT "sea_house_bill_versions_source_check";

ALTER TABLE "sea_house_bill_versions"
  ADD CONSTRAINT "sea_house_bill_versions_source_check"
  CHECK ("source" IN ('ORDER_LOCK', 'AMENDMENT', 'VOID', 'MODE_CHANGE'));

ALTER TABLE "sea_house_bill_versions"
  DROP CONSTRAINT "sea_house_bill_versions_status_check";

ALTER TABLE "sea_house_bill_versions"
  ADD CONSTRAINT "sea_house_bill_versions_status_check"
  CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED'));

-- 9. sea_master_bill_versions 调整：移除单一航次快照，增加外部确认字段
ALTER TABLE "sea_master_bill_versions"
  DROP CONSTRAINT "sea_master_bill_versions_sea_transport_executions_master_bill_versions";

ALTER TABLE "sea_master_bill_versions"
  DROP COLUMN "transport_execution_id",
  DROP COLUMN "origin_location_id",
  DROP COLUMN "discharge_location_id",
  DROP COLUMN "transit_location_id",
  DROP COLUMN "vessel_name",
  DROP COLUMN "voyage_no",
  DROP COLUMN "etd",
  DROP COLUMN "eta";

ALTER TABLE "sea_master_bill_versions"
  ADD COLUMN "confirmed_by_party" character varying(128),
  ADD COLUMN "confirmed_at" timestamp with time zone,
  ADD COLUMN "confirmation_note" character varying(500),
  ADD COLUMN "confirmation_attachment_id" uuid;

ALTER TABLE "sea_master_bill_versions"
  ADD CONSTRAINT "sea_master_bill_versions_order_attachments_sea_master_bill_versions"
  FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION;

ALTER TABLE "sea_master_bill_versions"
  DROP CONSTRAINT "sea_master_bill_versions_source_check";

ALTER TABLE "sea_master_bill_versions"
  ADD CONSTRAINT "sea_master_bill_versions_source_check"
  CHECK ("source" IN ('ORDER_LOCK', 'AMENDMENT', 'VOID'));

-- 10. order_lock_records 增加运输执行及版本快照，更新 CHECK 约束
ALTER TABLE "order_lock_records"
  ADD COLUMN "transport_execution_id" uuid,
  ADD COLUMN "transport_execution_version_id" uuid;

ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_sea_transport_executions_lock_records"
  FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION;

ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_sea_transport_execution_versions_lock_records"
  FOREIGN KEY ("transport_execution_version_id") REFERENCES "sea_transport_execution_versions" ("id") ON DELETE NO ACTION;

ALTER TABLE "order_lock_records"
  DROP CONSTRAINT "order_lock_records_business_type_document_refs_check";

ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_business_type_document_refs_check"
  CHECK ((business_type = 'SE' AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL AND transport_execution_id IS NOT NULL AND transport_execution_version_id IS NOT NULL) OR (business_type IN ('SI', 'AE', 'AI', 'LAND', 'RAIL') AND master_bill_id IS NULL AND master_bill_version_id IS NULL AND transport_execution_id IS NULL AND transport_execution_version_id IS NULL));

-- 11. sea_order_reassignment_events 增加外部确认字段
ALTER TABLE "sea_order_reassignment_events"
  ADD COLUMN "confirmed_by_party" character varying(128) NOT NULL,
  ADD COLUMN "confirmed_at" timestamp with time zone NOT NULL,
  ADD COLUMN "confirmation_note" character varying(500) NOT NULL,
  ADD COLUMN "confirmation_attachment_id" uuid;

ALTER TABLE "sea_order_reassignment_events"
  ADD CONSTRAINT "sea_order_reassignment_events_order_attachments_sea_order_reassignment_events"
  FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION;

-- 12. sea_document_void_events 增加外部确认字段
ALTER TABLE "sea_document_void_events"
  ADD COLUMN "confirmed_by_party" character varying(128) NOT NULL,
  ADD COLUMN "confirmed_at" timestamp with time zone NOT NULL,
  ADD COLUMN "confirmation_note" character varying(500) NOT NULL,
  ADD COLUMN "confirmation_attachment_id" uuid;

ALTER TABLE "sea_document_void_events"
  ADD CONSTRAINT "sea_document_void_events_order_attachments_sea_document_void_events"
  FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION;

-- 13. 创建不可变模式变更事件表 sea_document_mode_change_events
CREATE TABLE "sea_document_mode_change_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "previous_mode" character varying(32) NOT NULL,
  "target_mode" character varying(32) NOT NULL,
  "reason" character varying(500) NOT NULL,
  "impact_summary" character varying(1000),
  "confirmed_by_party" character varying(128) NOT NULL,
  "confirmed_at" timestamp with time zone NOT NULL,
  "confirmation_note" character varying(500) NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "order_id" uuid NOT NULL,
  "confirmation_attachment_id" uuid,
  "organization_id" uuid NOT NULL,
  "previous_house_bill_id" uuid,
  "target_house_bill_id" uuid,
  "previous_house_bill_version_id" uuid,
  "target_house_bill_version_id" uuid,
  "created_by" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_document_mode_change_events_orders_sea_document_mode_change_events" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_order_attachments_sea_document_mode_change_events" FOREIGN KEY ("confirmation_attachment_id") REFERENCES "order_attachments" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_organizations_sea_document_mode_change_events" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_sea_house_bills_previous_mode_change_events" FOREIGN KEY ("previous_house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_sea_house_bills_target_mode_change_events" FOREIGN KEY ("target_house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_sea_house_bill_versions_previous_mode_change_events" FOREIGN KEY ("previous_house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_sea_house_bill_versions_target_mode_change_events" FOREIGN KEY ("target_house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_users_created_sea_document_mode_change_events" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_mode_change_events_mode_check" CHECK (previous_mode <> target_mode AND ((previous_mode = 'HOUSE' AND target_mode = 'DIRECT' AND previous_house_bill_id IS NOT NULL AND previous_house_bill_version_id IS NOT NULL AND target_house_bill_id IS NULL AND target_house_bill_version_id IS NULL) OR (previous_mode = 'DIRECT' AND target_mode = 'HOUSE' AND target_house_bill_id IS NOT NULL AND target_house_bill_version_id IS NOT NULL AND previous_house_bill_id IS NULL AND previous_house_bill_version_id IS NULL)))
);

CREATE INDEX "seadocumentmodechangeevent_organization_id_order_id" ON "sea_document_mode_change_events" ("organization_id", "order_id");
CREATE UNIQUE INDEX "sea_doc_mode_change_idempotency_key" ON "sea_document_mode_change_events" ("organization_id", "idempotency_key");
CREATE INDEX "seadocumentmodechangeevent_organization_id_request_fingerprint" ON "sea_document_mode_change_events" ("organization_id", "request_fingerprint");

-- 14. 创建共享物理箱表 sea_shared_containers
CREATE TABLE "sea_shared_containers" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "container_no" character varying(64) NOT NULL,
  "container_spec_id" uuid NOT NULL,
  "seal_no" character varying(64),
  "package_count" integer NOT NULL,
  "gross_weight_kg" numeric(18,3) NOT NULL,
  "volume_cbm" numeric(18,6) NOT NULL,
  "status" character varying(32) NOT NULL DEFAULT 'DRAFT',
  "confirmed_at" timestamp with time zone,
  "note" character varying(500),
  "version" bigint NOT NULL DEFAULT 1,
  "organization_id" uuid NOT NULL,
  "transport_execution_id" uuid NOT NULL,
  "confirmed_by" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_shared_containers_organizations_sea_shared_containers" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_containers_sea_transport_executions_shared_containers" FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_containers_users_confirmed_sea_shared_containers" FOREIGN KEY ("confirmed_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_shared_containers_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED'))
);

CREATE INDEX "seasharedcontainer_updated_at" ON "sea_shared_containers" ("updated_at");
CREATE UNIQUE INDEX "sea_shared_container_execution_no" ON "sea_shared_containers" ("transport_execution_id", "container_no");
CREATE INDEX "seasharedcontainer_organization_id_transport_execution_id" ON "sea_shared_containers" ("organization_id", "transport_execution_id");
CREATE INDEX "seasharedcontainer_organization_id_container_spec_id" ON "sea_shared_containers" ("organization_id", "container_spec_id");

-- 15. 创建共享箱件重尺分配表 sea_shared_container_allocations
CREATE TABLE "sea_shared_container_allocations" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "package_count" integer NOT NULL,
  "gross_weight_kg" numeric(18,3) NOT NULL,
  "volume_cbm" numeric(18,6) NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "order_id" uuid NOT NULL,
  "cargo_item_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "house_bill_id" uuid NOT NULL,
  "shared_container_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_shared_container_allocations_orders_sea_shared_container_allocations" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_container_allocations_order_cargo_items_shared_container_allocations" FOREIGN KEY ("cargo_item_id") REFERENCES "order_cargo_items" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_container_allocations_organizations_sea_shared_container_allocations" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_container_allocations_sea_house_bills_shared_container_allocations" FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_shared_container_allocations_sea_shared_containers_allocations" FOREIGN KEY ("shared_container_id") REFERENCES "sea_shared_containers" ("id") ON DELETE NO ACTION
);

CREATE INDEX "seasharedcontainerallocation_updated_at" ON "sea_shared_container_allocations" ("updated_at");
CREATE UNIQUE INDEX "sea_shared_cntr_alloc_unique" ON "sea_shared_container_allocations" ("shared_container_id", "cargo_item_id");
CREATE INDEX "seasharedcontainerallocation_organization_id_shared_container_id" ON "sea_shared_container_allocations" ("organization_id", "shared_container_id");
CREATE INDEX "seasharedcontainerallocation_organization_id_order_id" ON "sea_shared_container_allocations" ("organization_id", "order_id");
CREATE INDEX "seasharedcontainerallocation_organization_id_house_bill_id" ON "sea_shared_container_allocations" ("organization_id", "house_bill_id");
