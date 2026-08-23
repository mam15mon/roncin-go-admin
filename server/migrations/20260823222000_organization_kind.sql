ALTER TABLE "organizations"
    ADD COLUMN "kind" character varying NULL;

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
SET "kind" = CASE
    WHEN organization_tree."depth" = 0 THEN 'headquarters'
    WHEN organization_tree."depth" = 1 THEN 'company'
    ELSE 'department'
END
FROM organization_tree
WHERE organization."id" = organization_tree."id";

ALTER TABLE "organizations"
    ALTER COLUMN "kind" SET NOT NULL,
    ADD CONSTRAINT "organizations_kind_check"
        CHECK ("kind" IN ('headquarters', 'company', 'department'));
