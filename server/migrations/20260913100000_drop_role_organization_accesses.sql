-- 移除角色「可访问组织」功能：跨组织访问口径统一由多组织成员资格 + DataScope 承担，
-- role_organization_accesses 表零数据，直接删除。
DROP TABLE "role_organization_accesses";
