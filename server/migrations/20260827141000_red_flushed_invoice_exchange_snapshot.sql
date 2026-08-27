-- 红冲会停用发票与账单的有效关联；补齐已红冲历史发票的开票汇率快照。
UPDATE "finance_invoices" AS invoice
SET
  "base_currency_amount" = source."base_currency_amount",
  "exchange_rate" = ROUND(source."base_currency_amount" / invoice."total_amount", 8),
  "exchange_rate_source" = 'DERIVED',
  "exchange_rate_date" = invoice."invoice_date"
FROM (
  SELECT link."invoice_id", SUM(bill."base_currency_amount") AS "base_currency_amount"
  FROM "finance_invoice_bills" AS link
  JOIN "finance_bills" AS bill ON bill."id" = link."bill_id"
  GROUP BY link."invoice_id"
) AS source
WHERE source."invoice_id" = invoice."id"
  AND invoice."status" = 'RED_FLUSHED'
  AND invoice."invoice_date" IS NOT NULL
  AND invoice."exchange_rate" IS NULL;
