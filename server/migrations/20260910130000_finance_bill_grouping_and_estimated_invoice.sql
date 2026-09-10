DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM "finance_bills") OR EXISTS (SELECT 1 FROM "finance_bill_lines") THEN
    RAISE EXCEPTION '本迁移不兼容旧开发数据：finance_bills 或 finance_bill_lines 已有记录，需经授权清理或重建后再执行迁移';
  END IF;
END $$;

ALTER TABLE "finance_bill_batches"
  ADD COLUMN "grouping_mode" character varying NOT NULL DEFAULT 'NORMAL',
  ADD CONSTRAINT "finance_bill_batches_grouping_mode_check" CHECK ("grouping_mode" IN ('NORMAL', 'NETTING'));

ALTER TABLE "finance_bills"
  ADD COLUMN "estimated_invoice_currency" character varying NULL,
  ADD COLUMN "estimated_invoice_rate" numeric(18,8) NULL,
  ADD COLUMN "estimated_invoice_amount" numeric(28,8) NULL,
  ADD CONSTRAINT "finance_bills_estimated_invoice_snapshot_check" CHECK (("estimated_invoice_currency" IS NULL AND "estimated_invoice_rate" IS NULL AND "estimated_invoice_amount" IS NULL) OR ("estimated_invoice_currency" IS NOT NULL AND "estimated_invoice_rate" IS NOT NULL AND "estimated_invoice_amount" IS NOT NULL AND "estimated_invoice_rate" > 0 AND "estimated_invoice_amount" >= 0));
