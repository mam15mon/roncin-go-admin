-- 根据 Ent FinanceInvoiceExemption 源模型通过 WriteDriver 生成并审阅。
-- 每次决定保留历史，仅未撤销决定参与费用保护与完成状态。
CREATE TABLE "finance_invoice_exemptions" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "marked_by" uuid NOT NULL, "reason" character varying NOT NULL, "version" bigint NOT NULL DEFAULT 1, "revoked_at" timestamptz NULL, "revoked_by" uuid NULL, "revocation_reason" character varying NULL, "bill_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "finance_invoice_exemption_reason_check" CHECK (length(trim(reason)) > 0), CONSTRAINT "finance_invoice_exemption_revoke_check" CHECK ((revoked_at IS NULL AND revoked_by IS NULL AND revocation_reason IS NULL) OR (revoked_at IS NOT NULL AND revoked_by IS NOT NULL AND revocation_reason IS NOT NULL AND length(trim(revocation_reason)) > 0)));

CREATE INDEX "financeinvoiceexemption_updated_at" ON "finance_invoice_exemptions" ("updated_at");

CREATE UNIQUE INDEX "finance_invoice_exemption_active_bill" ON "finance_invoice_exemptions" ("bill_id") WHERE revoked_at IS NULL;

CREATE INDEX "financeinvoiceexemption_organization_id_bill_id_created_at" ON "finance_invoice_exemptions" ("organization_id", "bill_id", "created_at");

ALTER TABLE "finance_invoice_exemptions" ADD CONSTRAINT "finance_invoice_exemptions_finance_bills_invoice_exemptions" FOREIGN KEY ("bill_id") REFERENCES "finance_bills" ("id") ON DELETE NO ACTION, ADD CONSTRAINT "finance_invoice_exemptions_org_b1038250fae2ebe660b43531c5005ac9" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;
