-- 汇率覆盖权限同时保护订单费用和资金流水，更新权限展示语义。
UPDATE "permissions"
SET
  "name" = '覆盖财务汇率',
  "description" = '在订单费用或资金流水中手工覆盖系统汇率',
  "updated_at" = CURRENT_TIMESTAMP
WHERE "key" = 'system.finance.exchange_rate.override';
