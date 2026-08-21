-- 为已完成管理员初始化的数据库补齐后续新增权限。
-- 冷启动数据库此时还没有用户，权限仍由 bootstrap-admin 按当前 Manifest 创建。
INSERT INTO "permissions" (
  "id",
  "created_at",
  "updated_at",
  "key",
  "name",
  "group",
  "description"
)
SELECT
  new_permissions."id",
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  new_permissions."key",
  new_permissions."name",
  new_permissions."group",
  new_permissions."description"
FROM (
  VALUES
    ('2b8087e4-73c4-4f10-953c-7f2d943cd311'::uuid, 'system.master_data.read', '查看主数据', '系统管理', '查看订单表单所需的基础选项'),
    ('18b7fa2e-d728-4f8b-984b-12f163284c6f'::uuid, 'system.master_data.manage', '管理主数据', '系统管理', '维护币种、地区、港口、机场和订单基础目录'),
    ('5be39fe4-e18c-48e7-bc83-8f331f91ad16'::uuid, 'business.order.read', '查看订单', '订单管理', '查看当前组织的订单'),
    ('863ca67a-a2ec-44bc-9f61-1478d1b830f9'::uuid, 'business.order.manage', '管理订单', '订单管理', '创建、编辑和流转当前组织的订单'),
    ('70c08686-820e-48c8-8584-10c66b499d0c'::uuid, 'system.task.read', '查看后台任务', '系统管理', '查看当前组织的后台任务执行状态'),
    ('929ead00-03ae-4019-a089-e5d663a2848d'::uuid, 'system.task.manage', '管理后台任务', '系统管理', '回放当前组织的失败或死信后台任务')
) AS new_permissions("id", "key", "name", "group", "description")
WHERE EXISTS (SELECT 1 FROM "users")
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT administrator_roles."id", new_permissions."id"
FROM "roles" AS administrator_roles
CROSS JOIN "permissions" AS new_permissions
WHERE administrator_roles."code" = 'administrator'
  AND new_permissions."key" IN (
    'system.master_data.read',
    'system.master_data.manage',
    'business.order.read',
    'business.order.manage',
    'system.task.read',
    'system.task.manage'
  )
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
