-- 单笔费用手工汇率覆盖与全公司汇率维护拆分授权。
INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description")
VALUES (
  md5('system.finance.exchange_rate.override')::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  'system.finance.exchange_rate.override',
  '覆盖费用汇率',
  '财务管理 · 汇率',
  '在订单费用中手工覆盖系统结算汇率'
)
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT role."id", permission."id"
FROM "roles" AS role
JOIN "permissions" AS permission ON permission."key" = 'system.finance.exchange_rate.override'
WHERE role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
