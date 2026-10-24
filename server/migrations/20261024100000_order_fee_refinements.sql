-- 订单详情与费用录入规则调整：取消费用大类（含订单服务类型关联）、
-- 费用发生日期退役（改为不可变的录入时间 created_at 口径）。

-- 1. 费用科目与系统模板取消费用大类。
DROP INDEX IF EXISTS feesetting_organization_id_charge_category_id_abnormal_case_id;
ALTER TABLE fee_settings DROP CONSTRAINT IF EXISTS fee_settings_master_data_items_charge_category_fee_settings;
ALTER TABLE fee_settings DROP COLUMN IF EXISTS charge_category_id;
ALTER TABLE fee_setting_templates DROP COLUMN IF EXISTS charge_category_id;

-- 2. 订单服务类型（原费用大类在订单上的关联）整表下线。
DROP TABLE IF EXISTS "order_service_types";

-- 3. 主数据中的费用大类类型与数据下线；kind 为 varchar 列，直接删行。
DELETE FROM master_data_items WHERE kind = 'charge_category';

-- 4. 费用发生日期退役：录入时间以 created_at 为准，不再独立存储。
ALTER TABLE order_fees DROP COLUMN IF EXISTS expense_date;
ALTER TABLE order_fee_supplement_requests DROP COLUMN IF EXISTS expense_date;
