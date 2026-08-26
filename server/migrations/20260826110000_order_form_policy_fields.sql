-- 区分传统集运模式与拼载批次，并为监管申报截止时间提供独立字段。
ALTER TABLE "orders"
  ADD COLUMN "declaration_cutoff_at" character varying(64) NULL;

UPDATE "orders"
SET "shipment_mode" = 'TRADITIONAL_FORWARDING'
WHERE "shipment_mode" = 'CONSOLIDATION';

-- 旧页面曾把截申报时间误写入 loading_terms，仅迁移可确定为 RFC3339 的记录。
UPDATE "orders"
SET
  "declaration_cutoff_at" = "loading_terms",
  "loading_terms" = NULL
WHERE "loading_terms" ~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$';
