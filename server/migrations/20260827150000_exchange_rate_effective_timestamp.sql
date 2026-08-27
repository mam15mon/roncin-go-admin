-- 将汇率有效期从日期字符串升级为带时区秒级时间。
-- 历史日期是左闭右开边界，按上海时区当天 00:00:00 转换后语义保持不变。
ALTER TABLE "exchange_rate_settings"
  ALTER COLUMN "effective_from" TYPE timestamptz(0)
    USING (("effective_from" || ' 00:00:00 Asia/Shanghai')::timestamptz(0)),
  ALTER COLUMN "effective_to" TYPE timestamptz(0)
    USING (CASE
      WHEN "effective_to" IS NULL THEN NULL
      ELSE ("effective_to" || ' 00:00:00 Asia/Shanghai')::timestamptz(0)
    END);

ALTER TABLE "exchange_rate_settings"
  ADD CONSTRAINT "exchange_rate_settings_effective_interval"
  CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from");
