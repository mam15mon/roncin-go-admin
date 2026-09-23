-- 初始化既有离散单位，随后费用目录种子按同一规则写入新单位。
ALTER TABLE "billing_units"
  ADD COLUMN "quantity_must_be_integer" boolean NOT NULL DEFAULT false;

UPDATE "billing_units"
SET "quantity_must_be_integer" = true
WHERE "code" IN (
  'PIAO', 'BL', 'CONT', 'SET', 'DOC', 'CHE',
  '12GP', '20GP', '20HC', '20OT', '20FR', '20RF', '20TK', '20HT', '20RH',
  '40GP', '40HC', '40HQ', '40FR', '40PF', '40RF', '40OT', '40RH', '45HC'
);
