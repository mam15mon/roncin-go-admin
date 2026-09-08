-- 将 orders 表的 trade_term 列由必填（NOT NULL）改为选填（NULL）
ALTER TABLE "orders" ALTER COLUMN "trade_term" DROP NOT NULL;
