INSERT INTO "number_rules" (
    "id",
    "created_at",
    "updated_at",
    "document_type",
    "prefix",
    "date_format",
    "sequence_length",
    "reset_policy",
    "enabled",
    "organization_id"
)
SELECT
    gen_random_uuid(),
    NOW(),
    NOW(),
    defaults.document_type,
    defaults.prefix,
    'yyyyMMdd',
    4,
    'daily',
    true,
    organizations.id
FROM "organizations"
CROSS JOIN (
    VALUES
        ('order', 'OR'),
        ('booking', 'BK'),
        ('hbl', 'HBL'),
        ('mbl', 'MBL'),
        ('bill', 'BI'),
        ('statement', 'ST'),
        ('payment', 'PY'),
        ('invoice', 'IV')
) AS defaults(document_type, prefix)
ON CONFLICT ("organization_id", "document_type") DO NOTHING;
