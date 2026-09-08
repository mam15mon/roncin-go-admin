ALTER TABLE "role_order_organization_accesses" RENAME TO "role_organization_accesses";

ALTER TABLE "role_organization_accesses"
  RENAME CONSTRAINT "role_order_organization_accesses_roles_accesses"
  TO "role_organization_accesses_roles_organization_accesses";

ALTER TABLE "role_organization_accesses"
  RENAME CONSTRAINT "role_order_organization_accesses_organizations_accesses"
  TO "role_organization_accesses_organizations_role_organization_accesses";

ALTER INDEX "roleorderorganizationaccess_updated_at"
  RENAME TO "roleorganizationaccess_updated_at";

ALTER INDEX "roleorderorganizationaccess_role_id_organization_id"
  RENAME TO "roleorganizationaccess_role_id_organization_id";

ALTER INDEX "roleorderorganizationaccess_organization_id"
  RENAME TO "roleorganizationaccess_organization_id";
