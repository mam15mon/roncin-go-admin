-- 发票逐费用分配、命令幂等与修订历史，表结构由 Ent Schema WriteDriver 生成并审阅。
CREATE TABLE "finance_invoice_commands" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "actor_id" uuid NOT NULL, "kind" character varying NOT NULL, "idempotency_key" character varying NOT NULL, "request_fingerprint" character varying NOT NULL, "invoice_ids" jsonb NOT NULL, "completed" boolean NOT NULL DEFAULT false, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"));
CREATE INDEX "financeinvoicecommand_updated_at" ON "finance_invoice_commands" ("updated_at");
CREATE UNIQUE INDEX "finance_invoice_command_org_key" ON "finance_invoice_commands" ("organization_id", "idempotency_key");
CREATE TABLE "finance_invoice_fee_allocations" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "order_fee_id" uuid NOT NULL, "fee_code" character varying NOT NULL, "fee_name" character varying NOT NULL, "currency" character varying NOT NULL, "source_tax_rate" numeric(7,4) NULL, "amount" numeric(28,8) NOT NULL, "net_amount" numeric(28,8) NOT NULL, "tax_amount" numeric(28,8) NOT NULL, "bill_line_id" uuid NOT NULL, "invoice_id" uuid NOT NULL, "invoice_bill_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "finance_invoice_fee_allocation_amount_check" CHECK (amount > 0 AND tax_amount >= 0 AND tax_amount <= amount AND net_amount = amount - tax_amount));
CREATE INDEX "financeinvoicefeeallocation_updated_at" ON "finance_invoice_fee_allocations" ("updated_at");
CREATE UNIQUE INDEX "finance_invoice_fee_allocation_link_line" ON "finance_invoice_fee_allocations" ("invoice_bill_id", "bill_line_id");
CREATE INDEX "financeinvoicefeeallocation_bill_line_id" ON "finance_invoice_fee_allocations" ("bill_line_id");
CREATE INDEX "financeinvoicefeeallocation_order_fee_id" ON "finance_invoice_fee_allocations" ("order_fee_id");
CREATE TABLE "finance_invoice_revisions" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "actor_id" uuid NOT NULL, "command_id" uuid NOT NULL, "revision_number" bigint NOT NULL, "reason" character varying NOT NULL, "before_snapshot" jsonb NOT NULL, "after_snapshot" jsonb NOT NULL, "invoice_id" uuid NOT NULL, PRIMARY KEY ("id"));
CREATE INDEX "financeinvoicerevision_updated_at" ON "finance_invoice_revisions" ("updated_at");
CREATE UNIQUE INDEX "financeinvoicerevision_invoice_id_revision_number" ON "finance_invoice_revisions" ("invoice_id", "revision_number");
ALTER TABLE "finance_invoice_commands" ADD CONSTRAINT "finance_invoice_commands_organizations_finance_invoice_commands" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION;
ALTER TABLE "finance_invoice_fee_allocations" ADD CONSTRAINT "finance_invoice_fee_allocation_2deb99792eeb22053519bccb493d0efa" FOREIGN KEY ("bill_line_id") REFERENCES "finance_bill_lines" ("id") ON DELETE NO ACTION, ADD CONSTRAINT "finance_invoice_fee_allocation_20c37f764e6cc0b8515e987c989015de" FOREIGN KEY ("invoice_id") REFERENCES "finance_invoices" ("id") ON DELETE NO ACTION, ADD CONSTRAINT "finance_invoice_fee_allocation_e5bc6a319fd786ef3d9efaca75bb72f4" FOREIGN KEY ("invoice_bill_id") REFERENCES "finance_invoice_bills" ("id") ON DELETE NO ACTION;
ALTER TABLE "finance_invoice_revisions" ADD CONSTRAINT "finance_invoice_revisions_finance_invoices_revisions" FOREIGN KEY ("invoice_id") REFERENCES "finance_invoices" ("id") ON DELETE NO ACTION;

-- 同一账单允许多张有效的部分登记；余额由订单/账单锁内逐费分配守恒校验。
DROP INDEX "finance_invoice_bills_active_bill_unique";

ALTER TABLE "finance_invoices" ADD COLUMN "revision_number" bigint NOT NULL DEFAULT 1;
ALTER TABLE "finance_invoice_lines" ALTER COLUMN "tax_rate" DROP NOT NULL;
ALTER TABLE "finance_invoice_lines" ADD COLUMN "revision_number" bigint NOT NULL DEFAULT 1;
DROP INDEX "financeinvoiceline_invoice_id_line_no";
CREATE UNIQUE INDEX "finance_invoice_line_revision_no" ON "finance_invoice_lines" ("invoice_id", "revision_number", "line_no");

-- 不猜票号、日期、汇率或来源，不清理开发数据。无法确定的旧记录须先由财务处理。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM finance_invoices
      WHERE status = 'DRAFT' AND (tax_invoice_no IS NULL OR length(trim(tax_invoice_no))=0
        OR invoice_date IS NULL OR exchange_rate IS NULL OR exchange_rate_source IS NULL
        OR exchange_rate_date IS NULL OR base_currency_amount IS NULL OR issued_at IS NULL OR issued_by IS NULL)) THEN
    RAISE EXCEPTION '发票直登记迁移受阻：存在缺号码、日期、汇率或真实登记人/时间的草稿，请先核对并按原流程完成或取消后重试';
  END IF;
  IF EXISTS (
    SELECT 1 FROM finance_invoice_bills link
    LEFT JOIN LATERAL (
      SELECT count(*) AS line_count, COALESCE(sum(line.total_amount),0) AS amount,
        COALESCE(sum(line.tax_amount),0) AS tax_amount
      FROM finance_bill_lines line
      WHERE line.bill_id=link.bill_id
        AND line.order_fee_id IN (SELECT value::uuid FROM jsonb_array_elements_text(link.source_fee_ids))
    ) source ON TRUE
    WHERE source.line_count=0 OR source.line_count<>jsonb_array_length(link.source_fee_ids)
      OR source.amount<>link.amount OR source.tax_amount<>link.tax_amount
  ) THEN
    RAISE EXCEPTION '发票逐费分配迁移受阻：原整额关联与来源费用快照不一致，请核对原票据';
  END IF;
  IF EXISTS (
    SELECT 1 FROM finance_invoices invoice
    LEFT JOIN LATERAL (SELECT sum(link.amount) AS amount,sum(link.tax_amount) AS tax_amount
      FROM finance_invoice_bills link WHERE link.invoice_id=invoice.id) links ON TRUE
    WHERE links.amount IS NULL OR links.amount<>invoice.total_amount OR links.tax_amount<>invoice.tax_amount
  ) THEN
    RAISE EXCEPTION '发票逐费分配迁移受阻：发票与账单关联金额不一致，请核对原票据';
  END IF;
END $$;

-- 只有票据与真实登记人/时间均完整的旧草稿可以转换，不能用创建时间冒充登记事实。
UPDATE finance_invoices SET status='ISSUED' WHERE status='DRAFT';

-- 旧登记只有整额来源；按已固化的原账单行逐项补齐，绝不重算当前费用或按比例分摊。
INSERT INTO finance_invoice_fee_allocations
  (id,created_at,updated_at,invoice_id,invoice_bill_id,bill_line_id,order_fee_id,
   fee_code,fee_name,currency,source_tax_rate,amount,net_amount,tax_amount)
SELECT md5(link.id::text||':'||line.id::text)::uuid,link.created_at,link.updated_at,
  link.invoice_id,link.id,line.id,line.order_fee_id,
  line.fee_code,line.fee_name,line.currency,line.tax_rate,line.total_amount,line.net_amount,line.tax_amount
FROM finance_invoice_bills link
JOIN finance_bill_lines line ON line.bill_id=link.bill_id
  AND line.order_fee_id IN (SELECT value::uuid FROM jsonb_array_elements_text(link.source_fee_ids));
