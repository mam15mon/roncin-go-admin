-- 只修正尚未被人工修改的旧发票默认规则，保留管理员主动关闭的配置。
UPDATE "number_rules"
SET "enabled" = true, "updated_at" = CURRENT_TIMESTAMP
WHERE "document_type" = 'invoice'
  AND "enabled" = false
  AND "prefix" = ''
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 5
  AND "reset_policy" = 'daily'
  AND "created_at" = "updated_at";
