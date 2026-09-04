-- 本阶段不提供历史数据回填；发现旧海运业务数据或已有锁定/版本数据时必须停止迁移。
DO $$
DECLARE
  v_count integer;
BEGIN
  IF to_regclass(current_schema() || '.orders') IS NOT NULL
     AND EXISTS (SELECT 1 FROM "orders" WHERE "business_type" = 'SE' LIMIT 1) THEN
    RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：orders 存在 SE 业务数据';
  END IF;
  IF to_regclass(current_schema() || '.sea_cargo_allocations') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_cargo_allocations"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：sea_cargo_allocations 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_house_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_house_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：sea_house_bills 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bill_order_links') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bill_order_links"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：sea_master_bill_order_links 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bills') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bills"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：sea_master_bills 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.order_lock_records') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "order_lock_records"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：order_lock_records 存在数据';
    END IF;
  END IF;
  IF to_regclass(current_schema() || '.sea_master_bill_versions') IS NOT NULL THEN
    EXECUTE 'SELECT count(*) FROM "sea_master_bill_versions"' INTO v_count;
    IF v_count > 0 THEN
      RAISE EXCEPTION '海运出口单证版本与订单锁迁移已停止：sea_master_bill_versions 存在数据';
    END IF;
  END IF;
END $$;

-- 1. User 增加 is_bootstrap_admin
ALTER TABLE "users"
  ADD COLUMN "is_bootstrap_admin" boolean NOT NULL DEFAULT false;

-- 2. Order 增加 locked_by 和 lock_generation
ALTER TABLE "orders"
  ADD COLUMN "locked_by" uuid,
  ADD COLUMN "lock_generation" bigint NOT NULL DEFAULT 0;

ALTER TABLE "orders"
  ADD CONSTRAINT "orders_users_locked_orders" FOREIGN KEY ("locked_by") REFERENCES "users" ("id") ON DELETE SET NULL;

CREATE INDEX "orders_organization_id_locked_at" ON "orders" ("organization_id", "locked_at");

-- 3. sea_master_bills 增加 current_version_id 并扩展 status 枚举
ALTER TABLE "sea_master_bills"
  ADD COLUMN "current_version_id" uuid;

ALTER TABLE "sea_master_bills"
  DROP CONSTRAINT IF EXISTS "sea_master_bills_status_check";

ALTER TABLE "sea_master_bills"
  ADD CONSTRAINT "sea_master_bills_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED'));

-- 4. sea_house_bills 增加 current_version_id 并扩展 status 枚举
ALTER TABLE "sea_house_bills"
  ADD COLUMN "current_version_id" uuid;

ALTER TABLE "sea_house_bills"
  DROP CONSTRAINT IF EXISTS "sea_house_bills_status_check";

ALTER TABLE "sea_house_bills"
  ADD CONSTRAINT "sea_house_bills_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED', 'REPLACED'));

-- 5. background_tasks 扩展 kind 枚举
ALTER TABLE "background_tasks"
  DROP CONSTRAINT IF EXISTS "background_tasks_kind_check";

ALTER TABLE "background_tasks"
  ADD CONSTRAINT "background_tasks_kind_check" CHECK ("kind" IN ('MASTER_DATA_IMPORT', 'UNLOCODE_IMPORT', 'ORDER_REMINDER', 'INTEGRATION', 'DINGTALK_NOTIFICATION', 'OBJECT_STORAGE_DELETION', 'DINGTALK_APPROVAL_CREATE'));

-- 6. 创建 sea_master_bill_versions
CREATE TABLE "sea_master_bill_versions" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "version_no" bigint NOT NULL,
  "source_entity_version" bigint NOT NULL,
  "issuer_partner_id" uuid NOT NULL,
  "transport_execution_id" uuid NOT NULL,
  "master_no" character varying(64) NOT NULL,
  "normalized_master_no" character varying(64) NOT NULL,
  "status" character varying(32) NOT NULL,
  "vessel_voyage_snapshot" character varying(100),
  "etd_snapshot" character varying(64),
  "eta_snapshot" character varying(64),
  "carrier_id" uuid,
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
  "created_by" uuid,
  "master_bill_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_master_bill_versions_organizations_sea_master_bill_versions" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_versions_partners_sea_master_bill_versions" FOREIGN KEY ("issuer_partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_versions_sea_master_bills_versions" FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_versions_sea_transport_executions_master_bill_versions" FOREIGN KEY ("transport_execution_id") REFERENCES "sea_transport_executions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_master_bill_versions_users_created_sea_master_bill_versions" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_master_bill_versions_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED')),
  CONSTRAINT "sea_master_bill_versions_source_check" CHECK ("source" IN ('ORDER_LOCK', 'AMENDMENT', 'SWITCH', 'VOID'))
);

CREATE UNIQUE INDEX "sea_mbl_version_master_version_no" ON "sea_master_bill_versions" ("master_bill_id", "version_no");
CREATE UNIQUE INDEX "sea_mbl_version_source_hash" ON "sea_master_bill_versions" ("master_bill_id", "source_entity_version", "content_hash");
CREATE INDEX "seamasterbillversion_organization_id_master_bill_id" ON "sea_master_bill_versions" ("organization_id", "master_bill_id");

-- 7. 创建 sea_house_bill_versions
CREATE TABLE "sea_house_bill_versions" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "version_no" bigint NOT NULL,
  "source_entity_version" bigint NOT NULL,
  "house_no" character varying(128) NOT NULL,
  "normalized_house_no" character varying(128) NOT NULL,
  "issuer_source" character varying(32) NOT NULL,
  "issuer_organization_id" uuid,
  "issuer_partner_id" uuid,
  "status" character varying(32) NOT NULL,
  "note" character varying(500),
  "content_hash" character varying(64) NOT NULL,
  "source" character varying(32) NOT NULL,
  "reason" character varying(500),
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
  "created_by" uuid,
  "house_bill_id" uuid NOT NULL,
  "master_bill_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_house_bill_versions_organizations_sea_house_bill_versions" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_versions_sea_house_bills_versions" FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_versions_organizations_issued_sea_house_bill_versions" FOREIGN KEY ("issuer_organization_id") REFERENCES "organizations" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_house_bill_versions_partners_sea_house_bill_versions" FOREIGN KEY ("issuer_partner_id") REFERENCES "partners" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_house_bill_versions_orders_sea_house_bill_versions" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_versions_sea_master_bills_house_bill_versions" FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_versions_users_created_sea_house_bill_versions" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_house_bill_versions_issuer_source_check" CHECK ("issuer_source" IN ('SELF_ORGANIZATION', 'CUSTOMER_PARTNER', 'OTHER_PARTNER')),
  CONSTRAINT "sea_house_bill_versions_issuer_check" CHECK (
    ("issuer_source" = 'SELF_ORGANIZATION' AND "issuer_organization_id" IS NOT NULL AND "issuer_partner_id" IS NULL) OR
    ("issuer_source" IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER') AND "issuer_organization_id" IS NULL AND "issuer_partner_id" IS NOT NULL)
  ),
  CONSTRAINT "sea_house_bill_versions_status_check" CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'RELEASED', 'VOIDED', 'REPLACED')),
  CONSTRAINT "sea_house_bill_versions_source_check" CHECK ("source" IN ('ORDER_LOCK', 'AMENDMENT', 'SWITCH', 'VOID'))
);

CREATE UNIQUE INDEX "sea_hbl_version_house_version_no" ON "sea_house_bill_versions" ("house_bill_id", "version_no");
CREATE UNIQUE INDEX "sea_hbl_version_source_hash" ON "sea_house_bill_versions" ("house_bill_id", "source_entity_version", "content_hash");
CREATE INDEX "seahousebillversion_organization_id_house_bill_id" ON "sea_house_bill_versions" ("organization_id", "house_bill_id");
CREATE INDEX "seahousebillversion_organization_id_order_id" ON "sea_house_bill_versions" ("organization_id", "order_id");

-- 8. 关联 sea_master_bills / sea_house_bills 当前版本外键
ALTER TABLE "sea_master_bills"
  ADD CONSTRAINT "sea_master_bills_sea_master_bill_versions_current_version" FOREIGN KEY ("current_version_id") REFERENCES "sea_master_bill_versions" ("id") ON DELETE SET NULL;

ALTER TABLE "sea_house_bills"
  ADD CONSTRAINT "sea_house_bills_sea_house_bill_versions_current_version" FOREIGN KEY ("current_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE SET NULL;

-- 9. 创建 order_lock_records
CREATE TABLE "order_lock_records" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "order_no" character varying(64) NOT NULL,
  "generation" bigint NOT NULL,
  "locked_at" timestamp with time zone NOT NULL,
  "order_version_at_lock" bigint NOT NULL,
  "unlocked_at" timestamp with time zone,
  "order_version_at_unlock" bigint,
  "unlock_reason" character varying(500),
  "unlock_mode" character varying(32),
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "locked_by" uuid NOT NULL,
  "unlocked_by" uuid,
  "master_bill_id" uuid NOT NULL,
  "master_bill_version_id" uuid NOT NULL,
  "unlock_request_id" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_lock_records_orders_lock_records" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_records_organizations_order_lock_records" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_records_users_order_lock_records" FOREIGN KEY ("locked_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_records_users_unlocked_order_lock_records" FOREIGN KEY ("unlocked_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "order_lock_records_sea_master_bills_lock_records" FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_records_sea_master_bill_versions_lock_records" FOREIGN KEY ("master_bill_version_id") REFERENCES "sea_master_bill_versions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_records_unlock_mode_check" CHECK ("unlock_mode" IS NULL OR "unlock_mode" IN ('ROLE_DIRECT', 'ADMIN_EMERGENCY', 'DINGTALK_APPROVED'))
);

CREATE UNIQUE INDEX "order_lock_record_order_generation" ON "order_lock_records" ("order_id", "generation");
CREATE UNIQUE INDEX "order_lock_record_idempotency_key" ON "order_lock_records" ("organization_id", "idempotency_key");
CREATE INDEX "orderlockrecord_organization_id_order_id" ON "order_lock_records" ("organization_id", "order_id");
CREATE INDEX "orderlockrecord_organization_id_locked_at" ON "order_lock_records" ("organization_id", "locked_at");

-- 10. 创建 order_lock_house_bill_snapshots
CREATE TABLE "order_lock_house_bill_snapshots" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "house_no_snapshot" character varying(128) NOT NULL,
  "lock_record_id" uuid NOT NULL,
  "house_bill_id" uuid NOT NULL,
  "house_bill_version_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_lock_house_bill_snapshots_order_lock_records_house_bill_snapshots" FOREIGN KEY ("lock_record_id") REFERENCES "order_lock_records" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_house_bill_snapshots_organizations_order_lock_house_bill_snapshots" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_house_bill_snapshots_sea_house_bills_lock_snapshots" FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_lock_house_bill_snapshots_sea_house_bill_versions_lock_snapshots" FOREIGN KEY ("house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "order_lock_hbl_snapshot_unique" ON "order_lock_house_bill_snapshots" ("lock_record_id", "house_bill_id");
CREATE INDEX "orderlockhousebillsnapshot_organization_id_lock_record_id" ON "order_lock_house_bill_snapshots" ("organization_id", "lock_record_id");

-- 11. 创建 order_unlock_requests
CREATE TABLE "order_unlock_requests" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "order_no" character varying(64) NOT NULL,
  "lock_generation" bigint NOT NULL,
  "requested_at" timestamp with time zone NOT NULL,
  "reason" character varying(500),
  "expected_order_version" bigint NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "route" character varying(32) NOT NULL,
  "status" character varying(32) NOT NULL,
  "dingtalk_process_instance_id" character varying(128),
  "dingtalk_process_code" character varying(128),
  "decided_at" timestamp with time zone,
  "decision_source" character varying(64),
  "failure_code" character varying(64),
  "failure_message" character varying(500),
  "unlocked_at" timestamp with time zone,
  "result_order_version" bigint,
  "order_id" uuid NOT NULL,
  "lock_record_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "requested_by" uuid NOT NULL,
  "decided_by" uuid,
  "superseded_by_request_id" uuid,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_unlock_requests_orders_unlock_requests" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_requests_order_lock_records_unlock_requests" FOREIGN KEY ("lock_record_id") REFERENCES "order_lock_records" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_requests_organizations_order_unlock_requests" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_requests_users_order_unlock_requests" FOREIGN KEY ("requested_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_requests_users_decided_order_unlock_requests" FOREIGN KEY ("decided_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "order_unlock_requests_order_unlock_requests_superseded_requests" FOREIGN KEY ("superseded_by_request_id") REFERENCES "order_unlock_requests" ("id") ON DELETE SET NULL,
  CONSTRAINT "order_unlock_requests_route_check" CHECK ("route" IN ('ROLE_DIRECT', 'ADMIN_EMERGENCY', 'DINGTALK_APPROVAL')),
  CONSTRAINT "order_unlock_requests_status_check" CHECK ("status" IN ('PENDING_DISPATCH', 'PENDING_APPROVAL', 'APPROVED_PENDING_APPLY', 'APPROVED', 'REJECTED', 'CONFIGURATION_FAILED', 'DISPATCH_FAILED', 'DISPATCH_UNKNOWN', 'STALE'))
);

CREATE UNIQUE INDEX "order_unlock_request_idempotency_key" ON "order_unlock_requests" ("organization_id", "idempotency_key");
CREATE UNIQUE INDEX "order_unlock_request_process_instance_id" ON "order_unlock_requests" ("dingtalk_process_instance_id") WHERE "dingtalk_process_instance_id" IS NOT NULL;
CREATE UNIQUE INDEX "order_unlock_request_active_unique" ON "order_unlock_requests" ("order_id", "lock_generation") WHERE "status" IN ('PENDING_DISPATCH', 'PENDING_APPROVAL', 'APPROVED_PENDING_APPLY', 'DISPATCH_UNKNOWN');
CREATE INDEX "orderunlockrequest_organization_id_order_id_created_at" ON "order_unlock_requests" ("organization_id", "order_id", "created_at");

-- 12. 补充 order_lock_records.unlock_request_id 外键
ALTER TABLE "order_lock_records"
  ADD CONSTRAINT "order_lock_records_order_unlock_requests_applied_unlock_request" FOREIGN KEY ("unlock_request_id") REFERENCES "order_unlock_requests" ("id") ON DELETE SET NULL;

-- 13. 创建 order_unlock_approver_candidates
CREATE TABLE "order_unlock_approver_candidates" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "display_name_snapshot" character varying(100) NOT NULL,
  "dingtalk_userid_snapshot" character varying(64) NOT NULL,
  "request_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "membership_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_unlock_approver_candidates_order_unlock_requests_approver_candidates" FOREIGN KEY ("request_id") REFERENCES "order_unlock_requests" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_approver_candidates_users_order_unlock_approver_candidates" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_approver_candidates_memberships_order_unlock_approver_candidates" FOREIGN KEY ("membership_id") REFERENCES "memberships" ("id") ON DELETE NO ACTION,
  CONSTRAINT "order_unlock_approver_candidates_roles_order_unlock_approver_candidates" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "order_unlock_approver_candidate_unique" ON "order_unlock_approver_candidates" ("request_id", "user_id");
CREATE INDEX "orderunlockapprovercandidate_request_id" ON "order_unlock_approver_candidates" ("request_id");

-- 14. 创建 ding_talk_approval_dispatches
CREATE TABLE "ding_talk_approval_dispatches" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "process_code_snapshot" character varying(128) NOT NULL,
  "applicant_dingtalk_userid" character varying(64) NOT NULL,
  "candidate_dingtalk_userids" jsonb NOT NULL,
  "request_payload_hash" character varying(64) NOT NULL,
  "dispatch_status" character varying(32) NOT NULL,
  "process_instance_id" character varying(128),
  "response_digest" character varying(500),
  "error_category" character varying(64),
  "background_task_id" uuid NOT NULL,
  "unlock_request_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ding_talk_approval_dispatches_organizations_dingtalk_approval_dispatches" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_approval_dispatches_background_tasks_dingtalk_approval_dispatch" FOREIGN KEY ("background_task_id") REFERENCES "background_tasks" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_approval_dispatches_order_unlock_requests_dispatch" FOREIGN KEY ("unlock_request_id") REFERENCES "order_unlock_requests" ("id") ON DELETE NO ACTION,
  CONSTRAINT "ding_talk_approval_dispatches_dispatch_status_check" CHECK ("dispatch_status" IN ('PENDING', 'DISPATCHED', 'FAILED', 'UNKNOWN'))
);

CREATE UNIQUE INDEX "dingtalk_approval_dispatch_bg_task_unique" ON "ding_talk_approval_dispatches" ("background_task_id");
CREATE UNIQUE INDEX "dingtalk_approval_dispatch_unlock_req_unique" ON "ding_talk_approval_dispatches" ("unlock_request_id");
CREATE INDEX "dingtalkapprovaldispatch_organization_id_dispatch_status" ON "ding_talk_approval_dispatches" ("organization_id", "dispatch_status");

-- 15. 创建 ding_talk_approval_inbox_events
CREATE TABLE "ding_talk_approval_inbox_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "organization_id" uuid,
  "event_id" character varying(128) NOT NULL,
  "corp_id" character varying(64) NOT NULL,
  "event_type" character varying(64) NOT NULL,
  "process_instance_id" character varying(128) NOT NULL,
  "received_at" timestamp with time zone NOT NULL,
  "encrypted_payload_hash" character varying(64) NOT NULL,
  "parsed_summary" character varying(1000),
  "status" character varying(32) NOT NULL,
  "result_code" character varying(64),
  "error_message" character varying(500),
  PRIMARY KEY ("id"),
  CONSTRAINT "ding_talk_approval_inbox_events_status_check" CHECK ("status" IN ('RECEIVED', 'PROCESSED', 'IGNORED', 'FAILED'))
);

CREATE UNIQUE INDEX "dingtalk_approval_inbox_event_id_unique" ON "ding_talk_approval_inbox_events" ("event_id");
CREATE INDEX "dingtalkapprovalinboxevent_process_instance_id" ON "ding_talk_approval_inbox_events" ("process_instance_id");
CREATE INDEX "dingtalkapprovalinboxevent_status_received_at" ON "ding_talk_approval_inbox_events" ("status", "received_at");

-- 16. 创建 sea_document_void_events
CREATE TABLE "sea_document_void_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "document_type" character varying(32) NOT NULL,
  "previous_status" character varying(32) NOT NULL,
  "voided_status" character varying(32) NOT NULL,
  "reason" character varying(500) NOT NULL,
  "impact_summary" character varying(1000),
  "created_by" uuid NOT NULL,
  "order_id" uuid,
  "master_bill_id" uuid,
  "master_bill_version_id" uuid,
  "house_bill_id" uuid,
  "house_bill_version_id" uuid,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_document_void_events_organizations_sea_document_void_events" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_void_events_orders_sea_document_void_events" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_document_void_events_sea_master_bills_void_events" FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_document_void_events_sea_master_bill_versions_void_events" FOREIGN KEY ("master_bill_version_id") REFERENCES "sea_master_bill_versions" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_document_void_events_sea_house_bills_void_events" FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_document_void_events_sea_house_bill_versions_void_events" FOREIGN KEY ("house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE SET NULL,
  CONSTRAINT "sea_document_void_events_users_created_sea_document_void_events" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_document_void_events_document_type_check" CHECK (((document_type = 'MASTER' AND master_bill_id IS NOT NULL AND house_bill_id IS NULL) OR (document_type = 'HOUSE' AND house_bill_id IS NOT NULL AND master_bill_id IS NULL)))
);

CREATE INDEX "seadocumentvoidevent_organization_id_document_type" ON "sea_document_void_events" ("organization_id", "document_type");
CREATE INDEX "seadocumentvoidevent_organization_id_master_bill_id" ON "sea_document_void_events" ("organization_id", "master_bill_id");
CREATE INDEX "seadocumentvoidevent_organization_id_house_bill_id" ON "sea_document_void_events" ("organization_id", "house_bill_id");

-- 17. 创建 sea_house_bill_switch_events
CREATE TABLE "sea_house_bill_switch_events" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "chain_id" uuid NOT NULL,
  "sequence" integer NOT NULL,
  "reason" character varying(500) NOT NULL,
  "surrender_info" character varying(500),
  "impact_summary" character varying(1000),
  "idempotency_key" character varying(128) NOT NULL,
  "request_fingerprint" character varying(128) NOT NULL,
  "created_by" uuid NOT NULL,
  "old_house_bill_id" uuid NOT NULL,
  "old_house_bill_version_id" uuid NOT NULL,
  "new_house_bill_id" uuid NOT NULL,
  "new_house_bill_version_id" uuid NOT NULL,
  "master_bill_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "sea_house_bill_switch_events_organizations_sea_house_bill_switch_events" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_orders_sea_house_bill_switch_events" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_sea_master_bills_switch_events" FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_sea_house_bills_old_switch_events" FOREIGN KEY ("old_house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_sea_house_bill_versions_old_switch_events" FOREIGN KEY ("old_house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_sea_house_bills_new_switch_events" FOREIGN KEY ("new_house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_sea_house_bill_versions_new_switch_events" FOREIGN KEY ("new_house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION,
  CONSTRAINT "sea_house_bill_switch_events_users_created_sea_house_bill_switch_events" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "sea_hbl_switch_idempotency_key" ON "sea_house_bill_switch_events" ("organization_id", "idempotency_key");
CREATE UNIQUE INDEX "sea_hbl_switch_old_hbl_unique" ON "sea_house_bill_switch_events" ("old_house_bill_id");
CREATE INDEX "seahousebillswitchevent_chain_id_sequence" ON "sea_house_bill_switch_events" ("chain_id", "sequence");
CREATE INDEX "seahousebillswitchevent_organization_id_order_id" ON "sea_house_bill_switch_events" ("organization_id", "order_id");

-- 18. 当前版本指针必须校验同一 document 身份
CREATE OR REPLACE FUNCTION check_sea_master_bill_current_version()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.current_version_id IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1 FROM "sea_master_bill_versions"
      WHERE "id" = NEW.current_version_id AND "master_bill_id" = NEW.id
    ) THEN
      RAISE EXCEPTION 'sea_master_bills.current_version_id % does not belong to master_bill %', NEW.current_version_id, NEW.id;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS check_sea_master_bill_current_version_trigger ON "sea_master_bills";
CREATE TRIGGER check_sea_master_bill_current_version_trigger
BEFORE INSERT OR UPDATE OF current_version_id ON "sea_master_bills"
FOR EACH ROW
EXECUTE FUNCTION check_sea_master_bill_current_version();

CREATE OR REPLACE FUNCTION check_sea_house_bill_current_version()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.current_version_id IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1 FROM "sea_house_bill_versions"
      WHERE "id" = NEW.current_version_id AND "house_bill_id" = NEW.id
    ) THEN
      RAISE EXCEPTION 'sea_house_bills.current_version_id % does not belong to house_bill %', NEW.current_version_id, NEW.id;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS check_sea_house_bill_current_version_trigger ON "sea_house_bills";
CREATE TRIGGER check_sea_house_bill_current_version_trigger
BEFORE INSERT OR UPDATE OF current_version_id ON "sea_house_bills"
FOR EACH ROW
EXECUTE FUNCTION check_sea_house_bill_current_version();
