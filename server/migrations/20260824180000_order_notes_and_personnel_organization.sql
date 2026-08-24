ALTER TABLE "orders"
    ADD COLUMN "booking_notes" character varying NULL,
    ADD COLUMN "allocation_notes" character varying NULL,
    ADD COLUMN "operation_notes" character varying NULL;

ALTER TABLE "order_personnels"
    ADD COLUMN "organization_id" uuid NULL;

UPDATE "order_personnels" AS personnel
SET "organization_id" = orders."organization_id"
FROM "orders"
WHERE orders."id" = personnel."order_id";

ALTER TABLE "order_personnels"
    ALTER COLUMN "organization_id" SET NOT NULL,
    ADD CONSTRAINT "order_personnels_organizations_order_personnel"
        FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;

DROP INDEX IF EXISTS "orderpersonnel_order_id_user_id_role";
CREATE UNIQUE INDEX "orderpersonnel_order_id_role" ON "order_personnels" ("order_id", "role");
CREATE INDEX "orderpersonnel_organization_id_role" ON "order_personnels" ("organization_id", "role");
