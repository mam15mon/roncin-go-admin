-- 单证变更命令需要在不可变事实表上持久化幂等请求。
-- 作废事件旧行无法推导原始幂等键及请求指纹，因此明确阻断有存量事件的迁移。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "sea_document_void_events" LIMIT 1) THEN
    RAISE EXCEPTION '海运单证变更幂等迁移已停止：sea_document_void_events 存在无法回填的历史数据';
  END IF;
END $$;

ALTER TABLE "sea_master_bill_versions"
  ADD COLUMN "idempotency_key" character varying(128),
  ADD COLUMN "request_fingerprint" character varying(128);

ALTER TABLE "sea_house_bill_versions"
  ADD COLUMN "idempotency_key" character varying(128),
  ADD COLUMN "request_fingerprint" character varying(128);

CREATE UNIQUE INDEX "sea_mbl_version_idempotency_key"
  ON "sea_master_bill_versions" ("organization_id", "idempotency_key");

CREATE UNIQUE INDEX "sea_hbl_version_idempotency_key"
  ON "sea_house_bill_versions" ("organization_id", "idempotency_key");

ALTER TABLE "sea_document_void_events"
  ADD COLUMN "previous_master_bill_version_id" uuid,
  ADD COLUMN "previous_house_bill_version_id" uuid,
  ADD COLUMN "idempotency_key" character varying(128) NOT NULL,
  ADD COLUMN "request_fingerprint" character varying(128) NOT NULL;

ALTER TABLE "sea_document_void_events"
  ALTER COLUMN "order_id" SET NOT NULL,
  DROP CONSTRAINT "sea_document_void_events_document_type_check",
  ADD CONSTRAINT "sea_document_void_events_document_type_check" CHECK (
    (document_type = 'MASTER'
      AND master_bill_id IS NOT NULL
      AND master_bill_version_id IS NOT NULL
      AND previous_master_bill_version_id IS NOT NULL
      AND house_bill_id IS NULL
      AND house_bill_version_id IS NULL
      AND previous_house_bill_version_id IS NULL)
    OR
    (document_type = 'HOUSE'
      AND house_bill_id IS NOT NULL
      AND house_bill_version_id IS NOT NULL
      AND previous_house_bill_version_id IS NOT NULL
      AND master_bill_id IS NULL
      AND master_bill_version_id IS NULL
      AND previous_master_bill_version_id IS NULL)
  ),
  ADD CONSTRAINT "sea_document_void_events_status_check" CHECK (voided_status = 'VOIDED');

ALTER TABLE "sea_document_void_events"
  DROP CONSTRAINT "sea_document_void_events_sea_master_bills_void_events",
  DROP CONSTRAINT "sea_document_void_events_sea_house_bills_void_events",
  DROP CONSTRAINT "sea_document_void_events_sea_master_bill_versions_void_events",
  DROP CONSTRAINT "sea_document_void_events_sea_house_bill_versions_void_events",
  ADD CONSTRAINT "sea_document_void_events_sea_master_bills_void_events"
    FOREIGN KEY ("master_bill_id") REFERENCES "sea_master_bills" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "sea_document_void_events_sea_house_bills_void_events"
    FOREIGN KEY ("house_bill_id") REFERENCES "sea_house_bills" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "sea_document_void_events_sea_master_bill_versions_void_events"
    FOREIGN KEY ("master_bill_version_id") REFERENCES "sea_master_bill_versions" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "sea_document_void_events_sea_house_bill_versions_void_events"
    FOREIGN KEY ("house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION;

ALTER TABLE "sea_document_void_events"
  ADD CONSTRAINT "sea_document_void_events_sea_master_bill_versions_previous_void_events"
    FOREIGN KEY ("previous_master_bill_version_id") REFERENCES "sea_master_bill_versions" ("id") ON DELETE NO ACTION,
  ADD CONSTRAINT "sea_document_void_events_sea_house_bill_versions_previous_void_events"
    FOREIGN KEY ("previous_house_bill_version_id") REFERENCES "sea_house_bill_versions" ("id") ON DELETE NO ACTION;

CREATE UNIQUE INDEX "sea_document_void_idempotency_key"
  ON "sea_document_void_events" ("organization_id", "idempotency_key");

DROP INDEX "seahousebillswitchevent_chain_id_sequence";
CREATE UNIQUE INDEX "seahousebillswitchevent_chain_id_sequence"
  ON "sea_house_bill_switch_events" ("chain_id", "sequence");
