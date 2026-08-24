-- 新增从当前组织删除员工的独立权限，并默认授予系统管理员。
INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description")
VALUES (
  md5('system.user.delete')::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  'system.user.delete',
  '删除用户',
  '系统管理 · 用户',
  '从当前组织移除用户并撤销其组织会话'
)
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT role."id", permission."id"
FROM "roles" AS role
JOIN "permissions" AS permission ON permission."key" = 'system.user.delete'
WHERE role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
