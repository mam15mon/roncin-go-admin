ALTER TABLE "finance_commission_lines"
  ADD COLUMN "personnel_organization_id" uuid,
  ADD COLUMN "personnel_assigned_at" timestamptz;

UPDATE "finance_commission_lines"
SET "personnel_organization_id" = "organization_id",
    "personnel_assigned_at" = "created_at";

UPDATE "finance_commission_lines" AS "line"
SET "personnel_organization_id" = "assignment"."organization_id",
    "personnel_assigned_at" = "assignment"."assigned_at"
FROM "order_personnels" AS "assignment"
WHERE "assignment"."id" = "line"."personnel_assignment_id";

ALTER TABLE "finance_commission_lines"
  ALTER COLUMN "personnel_organization_id" SET NOT NULL,
  ALTER COLUMN "personnel_assigned_at" SET NOT NULL;
