INSERT INTO "currencies" ("id", "created_at", "updated_at", "code", "name", "symbol", "minor_unit", "enabled")
VALUES
  ('0198cf68-5ba2-7d46-bbac-da40876e7207'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'AUD', '澳大利亚元', 'A$', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7208'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'SGD', '新加坡元', 'S$', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e7209'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'CHF', '瑞士法郎', 'CHF', 2, true),
  ('0198cf68-5ba2-7d46-bbac-da40876e720a'::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'THB', '泰铢', '฿', 2, true)
ON CONFLICT ("code") DO NOTHING;
