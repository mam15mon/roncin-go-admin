-- HBL 号由实际签发方提供并在海运单证中录入，不再保留自动编号规则。
DELETE FROM "number_sequences"
WHERE "rule_id" IN (
  SELECT "id" FROM "number_rules" WHERE "document_type" = 'house_bill'
);

DELETE FROM "number_rules" WHERE "document_type" = 'house_bill';
