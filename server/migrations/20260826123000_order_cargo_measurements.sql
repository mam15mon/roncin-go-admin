-- 委托件重尺与货物明细实际件重尺分开存储，供拼箱批次汇总和散杂 RT 使用。
-- 旧页面曾把截申报时间误写入 loading_terms，仅迁移可确定为 RFC3339 的记录。
UPDATE "orders"
SET
  "declaration_cutoff_at" = "loading_terms",
  "loading_terms" = NULL
WHERE "loading_terms" ~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$';

ALTER TABLE "orders"
  ADD COLUMN "total_gross_weight_kg" double precision NULL,
  ADD COLUMN "total_volume_cbm" double precision NULL,
  ADD CONSTRAINT "orders_total_gross_weight_kg_non_negative" CHECK ("total_gross_weight_kg" IS NULL OR "total_gross_weight_kg" >= 0),
  ADD CONSTRAINT "orders_total_volume_cbm_non_negative" CHECK ("total_volume_cbm" IS NULL OR "total_volume_cbm" >= 0);
