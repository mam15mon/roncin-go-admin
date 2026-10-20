-- 补齐上一迁移因时间戳不相等而遗漏的默认发票规则。
-- 正式历史迁移和 Ent 默认创建都可能使两列时间不同；人工修改以成功审计为依据。
UPDATE "number_rules" AS rule
SET "enabled" = true, "updated_at" = CURRENT_TIMESTAMP
WHERE rule."document_type" = 'invoice'
  AND rule."enabled" = false
  AND rule."prefix" = ''
  AND rule."date_format" = 'yyyyMMdd'
  AND rule."sequence_length" = 5
  AND rule."reset_policy" = 'daily'
  AND NOT EXISTS (
    SELECT 1 FROM "audit_logs" AS audit
    WHERE audit."organization_id" = rule."organization_id"
      AND audit."action" = 'number_rule.update'
      AND audit."result" = 'success'
      AND audit."details" ->> 'number_rule.id' = rule."id"::text
  );
