-- 由 Ent 生成元数据中的 HBL 版本来源约束生成。
ALTER TABLE "sea_house_bill_versions"
  DROP CONSTRAINT "sea_house_bill_versions_source_check",
  ADD CONSTRAINT "sea_house_bill_versions_source_check" CHECK (source IN ('ORDER_LOCK', 'AMENDMENT', 'VOID', 'MODE_CHANGE', 'DRAFT_SAVE'));
