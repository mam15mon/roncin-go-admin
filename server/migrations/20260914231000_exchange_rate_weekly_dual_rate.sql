-- 汇率周汇率结构重建（阶段一，design.md §8 第 5 步）：
-- exchange_rate_settings 扩展应收/应付双轨点差列（ar_rate=现汇卖出价/应收汇率，
-- ap_rate=现汇买入价/应付汇率），保留 rate 基准价列（中行折算价口径）；
-- 存量任意区间/开口区间行清理重建为自然周行（周一 00:00:00 至周日 23:59:59，
-- Asia/Shanghai），开发数据处置已获用户授权（prd.md §8.5）。
-- 执行次序：先按「归一化后的目标周键」幂等去重，再做周归一化 UPDATE——保证
-- 归一化期间部分唯一索引（基线/org 两个作用域）不会因同作用域同周多行而中断，
-- 任意存量库态均可一次跑通。
-- order_fees / finance_bill_lines 的汇率快照迁移落在应用层枚举扩展
-- （exchange_rate_source 增加 WEEKLY / INHERITED_LAST_WEEK / BOC_SYNC；
-- BASE_CURRENCY 为历史保留值，阶段二随 ResolveBaseRate 退役一并清理）：
-- Ent 枚举在 PostgreSQL 中为 varchar，无 DDL 变更。

-- 1. 双轨点差列（阶段一允许为空，周同步与手工录入在阶段二接入补齐）。
ALTER TABLE "exchange_rate_settings"
  ADD COLUMN "ar_rate" numeric(18,8) NULL,
  ADD COLUMN "ap_rate" numeric(18,8) NULL;

-- 2. 幂等去重：同作用域（基线/同组织）同货币对归一化后落入同一自然周的存量行，
--    按 (created_at, id) 保留最新一行。去重键与步骤 3 的目标周锚点完全一致，
--    确保归一化 UPDATE 不触发部分唯一索引冲突。
DELETE FROM "exchange_rate_settings" stale
USING "exchange_rate_settings" keep
WHERE stale."from_currency" = keep."from_currency"
  AND stale."to_currency" = keep."to_currency"
  AND COALESCE(stale."organization_id"::text, '') = COALESCE(keep."organization_id"::text, '')
  AND date_trunc('week', stale."effective_from" AT TIME ZONE 'Asia/Shanghai')
      = date_trunc('week', keep."effective_from" AT TIME ZONE 'Asia/Shanghai')
  AND (stale."created_at", stale."id") < (keep."created_at", keep."id");

-- 3. 周归一化：effective_from 锚定周一 00:00:00，effective_to 锚定同一周周日
--    23:59:59（业务时区 Asia/Shanghai），每行恒为单周行；基准价同步补齐双轨初值。
--    原跨周/开口区间行的后续周不展开，由周同步重建。
UPDATE "exchange_rate_settings"
SET "effective_from" = (date_trunc('week', "effective_from" AT TIME ZONE 'Asia/Shanghai')) AT TIME ZONE 'Asia/Shanghai',
    "effective_to" = (date_trunc('week', "effective_from" AT TIME ZONE 'Asia/Shanghai') + interval '6 days 23 hours 59 minutes 59 seconds') AT TIME ZONE 'Asia/Shanghai',
    "ar_rate" = "rate",
    "ap_rate" = "rate";
