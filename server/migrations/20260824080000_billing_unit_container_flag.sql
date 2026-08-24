-- 计费单位补充箱型标识，并初始化总部共用计费单位。
ALTER TABLE "billing_units"
  ADD COLUMN "is_container_unit" boolean NOT NULL DEFAULT false;

UPDATE "billing_units"
SET "is_container_unit" = false,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "organization_id" = (
  SELECT "id" FROM "organizations" WHERE "code" = 'HQ' AND "kind" = 'headquarters'
)
AND "code" = 'PIAO';

WITH headquarters AS (
  SELECT "id"
  FROM "organizations"
  WHERE "code" = 'HQ' AND "kind" = 'headquarters'
), units("sort_order", "code", "name", "is_container_unit") AS (
  VALUES
    (10, '20GP', '20GP', true),
    (20, '40HQ', '40HQ', true),
    (30, 'CBM', 'CBM', false),
    (50, 'CHE', '车', false),
    (60, '20HC', '20HC', true),
    (70, '20OT', '20OT', true),
    (80, '20FR', '20FR', true),
    (90, '20RF', '20RF', true),
    (100, '20TK', '20TK', true),
    (110, '20HT', '20HT', true),
    (120, '20RH', '20RH', true),
    (130, '40FR', '40FR', true),
    (140, '40GP', '40GP', true),
    (150, '40PF', '40PF', true),
    (160, '40RF', '40RF', true),
    (170, '40OT', '40OT', true),
    (180, '40RH', '40RH', true),
    (190, '45HC', '45HC', true),
    (200, 'BILL', 'bill', false),
    (210, 'SHIP', 'ship', false),
    (220, 'SHIPMENT', 'shipment', false),
    (230, 'HOUR', 'hour', false),
    (240, 'DAY', 'day', false),
    (250, '12GP', '12GP', true),
    (260, 'RT', 'RT', false),
    (270, 'BOARDS', 'BOARDS', false),
    (280, 'PLYWOOD_PALLETS', 'PLYWOOD PALLETS', false),
    (290, '40HC', '40HC', true),
    (300, 'KG', 'kg', false)
)
INSERT INTO "billing_units" (
  "id", "created_at", "updated_at", "organization_id", "code", "name",
  "is_container_unit", "sort_order", "enabled"
)
SELECT
  md5('billing_unit:HQ:' || units."code")::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  headquarters."id",
  units."code",
  units."name",
  units."is_container_unit",
  units."sort_order",
  true
FROM headquarters
CROSS JOIN units;
