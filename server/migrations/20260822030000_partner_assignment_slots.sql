ALTER TABLE "partner_assignments"
  ADD COLUMN "sort_order" bigint NOT NULL DEFAULT 0;

DROP INDEX "partnerassignment_partner_id_role";

CREATE UNIQUE INDEX "partnerassignment_partner_id_role_sort_order"
  ON "partner_assignments" ("partner_id", "role", "sort_order");
