-- 为现有总部补齐订单箱型箱量使用的标准箱型主数据。
WITH headquarters AS (
  SELECT "id"
  FROM "organizations"
  WHERE "kind" = 'headquarters'
), container_specs("sort_order", "code", "teu_factor") AS (
  VALUES
    (10, '20GP', '1'),
    (20, '40HQ', '2'),
    (30, '20HC', '1'),
    (40, '20OT', '1'),
    (50, '20FR', '1'),
    (60, '20RF', '1'),
    (70, '20TK', '1'),
    (80, '20HT', '1'),
    (90, '20RH', '1'),
    (100, '40FR', '2'),
    (110, '40GP', '2'),
    (120, '40PF', '2'),
    (130, '40RF', '2'),
    (140, '40OT', '2'),
    (150, '40RH', '2'),
    (160, '45HC', '2.25'),
    (170, '12GP', '0.6'),
    (180, '40HC', '2')
)
INSERT INTO "master_data_items" (
  "id", "created_at", "updated_at", "kind", "code", "name", "teu_factor",
  "source", "sort_order", "enabled", "organization_id", "attributes"
)
SELECT
  md5('master_data_item:' || headquarters."id" || ':container_spec:' || container_specs."code")::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  'container_spec',
  container_specs."code",
  container_specs."code",
  container_specs."teu_factor",
  'system',
  container_specs."sort_order",
  true,
  headquarters."id",
  '{}'::jsonb
FROM headquarters
CROSS JOIN container_specs
ON CONFLICT ("organization_id", "kind", "code") DO NOTHING;
