ALTER TABLE "organizations"
    DROP CONSTRAINT "organizations_kind_check";

WITH RECURSIVE organization_tree AS (
    SELECT "id", 0 AS "depth"
    FROM "organizations"
    WHERE "parent_id" IS NULL

    UNION ALL

    SELECT child."id", parent."depth" + 1
    FROM "organizations" AS child
    INNER JOIN organization_tree AS parent ON child."parent_id" = parent."id"
)
UPDATE "organizations" AS organization
SET "kind" = 'team'
FROM organization_tree
WHERE organization."id" = organization_tree."id"
  AND organization_tree."depth" >= 3;

ALTER TABLE "organizations"
    ADD CONSTRAINT "organizations_kind_check"
        CHECK ("kind" IN ('headquarters', 'company', 'department', 'team'));
