UPDATE "finance_fee_ledger_preferences"
SET "row_colors" = "row_colors" || jsonb_build_object(
  'invoicedPartiallyVerified', '#E6FFFB',
  'partiallyVerifiedUninvoiced', '#FFF0F6'
);
