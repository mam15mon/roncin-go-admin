-- 修正本次发布前费用种子将天误设为整数的初始值；之后由管理员维护。
UPDATE "billing_units"
SET "quantity_must_be_integer" = false
WHERE "code" = 'DAY';
