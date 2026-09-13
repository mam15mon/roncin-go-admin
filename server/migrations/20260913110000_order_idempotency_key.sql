-- 订单创建/草稿更新幂等键：单列「可变最新键」语义。
-- 创建时写入请求幂等键（可选输入，缺省由服务端生成随机键填充），草稿更新
-- 成功后覆写为本次键；同键 + 同 expected_version 的重放返回当前草稿。
-- 存量行用随机 UUID 回填（列为 NOT NULL，不允许空串），键为全局随机值，
-- 不会与业务键冲突。
ALTER TABLE "orders" ADD COLUMN "idempotency_key" varchar(128);
UPDATE "orders" SET "idempotency_key" = gen_random_uuid()::text;
ALTER TABLE "orders" ALTER COLUMN "idempotency_key" SET NOT NULL;

-- 组织内唯一，兜底并发同键创建（与建账 finance_bill 同构）。
CREATE UNIQUE INDEX "order_organization_id_idempotency_key" ON "orders"("organization_id", "idempotency_key");
