CREATE TABLE "role_order_organization_accesses" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "role_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "writable" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id"),
  CONSTRAINT "role_order_organization_accesses_roles_accesses" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON DELETE NO ACTION,
  CONSTRAINT "role_order_organization_accesses_organizations_accesses" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "roleorderorganizationaccess_updated_at" ON "role_order_organization_accesses" ("updated_at");
CREATE UNIQUE INDEX "roleorderorganizationaccess_role_id_organization_id" ON "role_order_organization_accesses" ("role_id", "organization_id");
CREATE INDEX "roleorderorganizationaccess_organization_id" ON "role_order_organization_accesses" ("organization_id");
