DROP INDEX "financeinvoice_tax_invoice_no";

CREATE UNIQUE INDEX "financeinvoice_org_tax_invoice_no"
  ON "finance_invoices"("organization_id", "tax_invoice_no")
  WHERE "tax_invoice_no" IS NOT NULL;
