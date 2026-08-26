ALTER TABLE "partner_invoice_profiles"
  ADD COLUMN "is_default" boolean NOT NULL DEFAULT true,
  ADD COLUMN "enabled" boolean NOT NULL DEFAULT true;

DROP INDEX "partnerinvoiceprofile_organization_id_partner_id";

CREATE INDEX "partnerinvoiceprofile_organization_id_partner_id"
  ON "partner_invoice_profiles"("organization_id", "partner_id");
CREATE UNIQUE INDEX "partner_invoice_profile_title_key"
  ON "partner_invoice_profiles"("organization_id", "partner_id", "invoice_title");
CREATE UNIQUE INDEX "partner_invoice_profile_default_key"
  ON "partner_invoice_profiles"("organization_id", "partner_id")
  WHERE "is_default";
