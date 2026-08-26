-- 区分传统集运模式与拼载批次，并为监管申报截止时间提供独立字段。
UPDATE "orders"
SET "shipment_mode" = 'TRADITIONAL_FORWARDING'
WHERE "shipment_mode" = 'CONSOLIDATION';

ALTER TABLE "orders"
  ADD COLUMN "declaration_cutoff_at" character varying(64) NULL;
