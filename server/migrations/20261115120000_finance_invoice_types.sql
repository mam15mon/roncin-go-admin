-- 发票票种细分：普票/专票拆分为数电、纸质、电子三种载体，仅用于登记记录与查询。
-- 项目无生产历史契约，开发数据一次性映射：NORMAL→DIGITAL_NORMAL，SPECIAL→DIGITAL_SPECIAL。
ALTER TABLE "finance_invoices" DROP CONSTRAINT IF EXISTS "finance_invoices_type_check";
UPDATE "finance_invoices" SET "invoice_type" = CASE "invoice_type" WHEN 'NORMAL' THEN 'DIGITAL_NORMAL' WHEN 'SPECIAL' THEN 'DIGITAL_SPECIAL' ELSE "invoice_type" END;
ALTER TABLE "finance_invoices" ADD CONSTRAINT "finance_invoices_type_check" CHECK ("invoice_type" IN ('DIGITAL_NORMAL', 'DIGITAL_SPECIAL', 'PAPER_NORMAL', 'PAPER_SPECIAL', 'ELECTRONIC_NORMAL', 'ELECTRONIC_SPECIAL'));

ALTER TABLE "partner_invoice_profiles" DROP CONSTRAINT IF EXISTS "partner_invoice_profile_type_check";
UPDATE "partner_invoice_profiles" SET "default_invoice_type" = CASE "default_invoice_type" WHEN 'NORMAL' THEN 'DIGITAL_NORMAL' WHEN 'SPECIAL' THEN 'DIGITAL_SPECIAL' ELSE "default_invoice_type" END;
ALTER TABLE "partner_invoice_profiles" ALTER COLUMN "default_invoice_type" SET DEFAULT 'DIGITAL_NORMAL';
ALTER TABLE "partner_invoice_profiles" ADD CONSTRAINT "partner_invoice_profile_type_check" CHECK ("default_invoice_type" IN ('DIGITAL_NORMAL', 'DIGITAL_SPECIAL', 'PAPER_NORMAL', 'PAPER_SPECIAL', 'ELECTRONIC_NORMAL', 'ELECTRONIC_SPECIAL'));

-- 修订历史快照中的票种同步映射，保证历史展示一致。
UPDATE "finance_invoice_revisions" SET
  "before_snapshot" = CASE WHEN "before_snapshot"->>'invoiceType' IN ('NORMAL', 'SPECIAL') THEN jsonb_set("before_snapshot", '{invoiceType}', to_jsonb('DIGITAL_' || ("before_snapshot"->>'invoiceType'))) ELSE "before_snapshot" END,
  "after_snapshot" = CASE WHEN "after_snapshot"->>'invoiceType' IN ('NORMAL', 'SPECIAL') THEN jsonb_set("after_snapshot", '{invoiceType}', to_jsonb('DIGITAL_' || ("after_snapshot"->>'invoiceType'))) ELSE "after_snapshot" END;
