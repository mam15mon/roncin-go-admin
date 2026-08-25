UPDATE "number_rules" SET "document_type" = 'house_bill' WHERE "document_type" = 'hbl';
UPDATE "number_rules" SET "document_type" = 'receipt_payment' WHERE "document_type" = 'payment';

UPDATE "number_rules"
SET "prefix" = '', "sequence_length" = 5, "updated_at" = NOW()
WHERE "document_type" = 'order'
  AND "prefix" = 'OR'
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 4
  AND "reset_policy" = 'daily';

UPDATE "number_rules"
SET "sequence_length" = 5, "updated_at" = NOW()
WHERE "document_type" = 'bill'
  AND "prefix" = 'BI'
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 4
  AND "reset_policy" = 'daily';

UPDATE "number_rules"
SET "prefix" = 'PR', "sequence_length" = 5, "updated_at" = NOW()
WHERE "document_type" = 'receipt_payment'
  AND "prefix" = 'PY'
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 4
  AND "reset_policy" = 'daily';

UPDATE "number_rules"
SET "prefix" = '', "sequence_length" = 5, "enabled" = false, "updated_at" = NOW()
WHERE "document_type" = 'house_bill'
  AND "prefix" = 'HBL'
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 4
  AND "reset_policy" = 'daily';

UPDATE "number_rules"
SET "prefix" = '', "sequence_length" = 5, "enabled" = false, "updated_at" = NOW()
WHERE "document_type" = 'invoice'
  AND "prefix" = 'IV'
  AND "date_format" = 'yyyyMMdd'
  AND "sequence_length" = 4
  AND "reset_policy" = 'daily';

DELETE FROM "number_sequences"
WHERE "rule_id" IN (
    SELECT "id" FROM "number_rules" WHERE "document_type" IN ('booking', 'mbl', 'statement')
);
DELETE FROM "number_rules" WHERE "document_type" IN ('booking', 'mbl', 'statement');

INSERT INTO "number_rules" (
    "id", "created_at", "updated_at", "document_type", "prefix",
    "date_format", "sequence_length", "reset_policy", "enabled", "organization_id"
)
SELECT
    gen_random_uuid(), NOW(), NOW(), defaults.document_type, defaults.prefix,
    defaults.date_format, defaults.sequence_length, defaults.reset_policy, defaults.enabled, organizations.id
FROM "organizations"
CROSS JOIN (
    VALUES
        ('order', '', 'yyyyMMdd', 5, 'daily', true),
        ('bill', 'BI', 'yyyyMMdd', 5, 'daily', true),
        ('quotation', 'QO', 'yyyyMMdd', 5, 'daily', true),
        ('write_off', 'WO', 'yyyyMMdd', 5, 'daily', true),
        ('receipt_payment', 'PR', 'yyyyMMdd', 5, 'daily', true),
        ('contract', 'CT', 'yyyyMMdd', 5, 'daily', true),
        ('internal_reference', '', 'yyyyMMdd', 5, 'daily', false),
        ('customer_reference', '', 'yyyyMMdd', 5, 'daily', false),
        ('house_bill', '', 'yyyyMMdd', 5, 'daily', false),
        ('coload_house_bill', '', 'none', 1, 'never', false),
        ('invoice', '', 'yyyyMMdd', 5, 'daily', false),
        ('freight_rate', 'FR', 'yyyyMM', 3, 'monthly', true)
) AS defaults(document_type, prefix, date_format, sequence_length, reset_policy, enabled)
ON CONFLICT ("organization_id", "document_type") DO NOTHING;
