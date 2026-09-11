-- 汇率体系极简化：推平多汇率类型与收付双轨，收敛为总部唯一折本币基准汇率。
-- 存量规则：仅保留 rate_type = 'BASE_CURRENCY' 的行并取 receivable_rate 作为单一汇率；
-- 塌缩后新唯一键冲突时迁移直接失败，人工裁定后重跑，禁止静默合并。

-- 1. 汇率主数据表收敛单一汇率列
DELETE FROM "exchange_rate_settings" WHERE "rate_type" <> 'BASE_CURRENCY';

ALTER TABLE "exchange_rate_settings" ADD COLUMN "rate" numeric(18,8);
UPDATE "exchange_rate_settings" SET "rate" = "receivable_rate";
ALTER TABLE "exchange_rate_settings" ALTER COLUMN "rate" SET NOT NULL;

ALTER TABLE "exchange_rate_settings" DROP CONSTRAINT "exchange_rate_settings_rate_type_check";
DROP INDEX "exchange_rate_setting_unique_effective_from";
DROP INDEX "exchange_rate_setting_active_lookup";
ALTER TABLE "exchange_rate_settings"
  DROP COLUMN "receivable_rate",
  DROP COLUMN "payable_rate",
  DROP COLUMN "rate_type";
CREATE UNIQUE INDEX "exchange_rate_setting_unique_effective_from"
  ON "exchange_rate_settings" ("organization_id", "from_currency", "to_currency", "effective_from");
CREATE INDEX "exchange_rate_setting_active_lookup"
  ON "exchange_rate_settings" ("organization_id", "from_currency", "to_currency", "is_active");

-- 2. 物理下线取值时间标准与自定义继承策略两张配置表
DROP TABLE "exchange_rate_time_standards";
DROP TABLE "exchange_rate_custom_settings";

-- 3. 核销单头删除汇率快照；单头本位币金额口径收敛为行级流水本位币合计
ALTER TABLE "finance_verifications"
  DROP CONSTRAINT "finance_verifications_exchange_rate_setting_fk",
  DROP CONSTRAINT "finance_verifications_base_amount_composition",
  DROP CONSTRAINT "finance_verifications_exchange_rate_positive",
  DROP CONSTRAINT "finance_verifications_exchange_rate_source_check";
DROP INDEX "finance_verifications_exchange_rate_setting_id";
ALTER TABLE "finance_verifications"
  DROP COLUMN "exchange_rate",
  DROP COLUMN "exchange_rate_source",
  DROP COLUMN "exchange_rate_date",
  DROP COLUMN "exchange_rate_setting_id";

-- 4. 核销分摊行删除核销日汇率折算列
ALTER TABLE "finance_verification_allocations" DROP COLUMN "write_off_base_amount";

-- 5. 对冲单新增应付侧本位币与对冲汇差；存量行按 0 回填，业务计算由应用层补齐
ALTER TABLE "finance_nettings" ADD COLUMN "payable_base_amount" numeric(28,8);
ALTER TABLE "finance_nettings" ADD COLUMN "exchange_gain_loss" numeric(28,8);
UPDATE "finance_nettings" SET "payable_base_amount" = '0', "exchange_gain_loss" = '0';
ALTER TABLE "finance_nettings"
  ALTER COLUMN "payable_base_amount" SET NOT NULL,
  ALTER COLUMN "exchange_gain_loss" SET NOT NULL,
  ADD CONSTRAINT "financenetting_payable_base_amount_non_negative"
    CHECK ("payable_base_amount" >= 0);
