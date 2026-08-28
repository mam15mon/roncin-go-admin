-- 用户删除权限调整为员工离职办理；权限码保持不变，现有角色授权无需迁移。
UPDATE "permissions"
SET
  "updated_at" = CURRENT_TIMESTAMP,
  "name" = '办理离职',
  "description" = '停用员工账号和全部组织权限，保留身份绑定与历史业务记录'
WHERE "key" = 'system.user.delete';
