INSERT INTO "number_rules" ("id","created_at","updated_at","document_type","prefix","date_format","sequence_length","reset_policy","enabled","organization_id")
SELECT gen_random_uuid(), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'commission', 'CM', 'yyyyMMdd', 5, 'daily', TRUE, "id"
FROM "organizations"
ON CONFLICT ("organization_id","document_type") DO NOTHING;
