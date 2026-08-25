-- 加拼分单号由业务单据维护，不再作为系统自动编号规则。
-- 仅清理未启用且从未发号的默认配置；已启用或已有序列的历史配置保留在数据库中，
-- 并由数据层从编号规则设置列表中隐藏，避免静默删除用户数据。
DELETE FROM "number_rules" AS "rule"
WHERE "rule"."document_type" = 'coload_house_bill'
  AND "rule"."enabled" = false
  AND NOT EXISTS (
    SELECT 1
    FROM "number_sequences" AS "sequence"
    WHERE "sequence"."rule_id" = "rule"."id"
  );
