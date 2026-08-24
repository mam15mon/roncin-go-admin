-- 将既有“票”单位放回用户提供清单中的顺序。
UPDATE "billing_units"
SET "sort_order" = 40,
    "updated_at" = CURRENT_TIMESTAMP
WHERE "organization_id" = (
  SELECT "id" FROM "organizations" WHERE "code" = 'HQ' AND "kind" = 'headquarters'
)
AND "code" = 'PIAO';
