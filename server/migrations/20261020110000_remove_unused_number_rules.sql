-- 清理未接入业务出号的四类编号规则；订单参考号字段仍由业务表单维护。
DELETE FROM "number_sequences"
WHERE "rule_id" IN (
  SELECT "id" FROM "number_rules"
  WHERE "document_type" IN ('quotation', 'freight_rate', 'internal_reference', 'customer_reference')
);

DELETE FROM "number_rules"
WHERE "document_type" IN ('quotation', 'freight_rate', 'internal_reference', 'customer_reference');
