ALTER TABLE "orders"
    ADD COLUMN "booking_notes" character varying NULL,
    ADD COLUMN "allocation_notes" character varying NULL,
    ADD COLUMN "operation_notes" character varying NULL;

ALTER TABLE "order_personnel"
    ADD COLUMN "organization_id" uuid NULL;

UPDATE "order_personnel" AS personnel
SET "organization_id" = orders."organization_id"
FROM "orders"
WHERE orders."id" = personnel."order_id";

ALTER TABLE "order_personnel"
    ALTER COLUMN "organization_id" SET NOT NULL,
    ADD CONSTRAINT "order_personnel_organizations_order_personnel"
        FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;

DROP INDEX IF EXISTS "order_personnel_order_id_user_id_role";
CREATE UNIQUE INDEX "order_personnel_order_id_role" ON "order_personnel" ("order_id", "role");
CREATE INDEX "order_personnel_organization_id_role" ON "order_personnel" ("organization_id", "role");
