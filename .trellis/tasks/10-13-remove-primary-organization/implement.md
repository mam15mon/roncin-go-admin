# 实施计划：取消主要组织

## 步骤

1. [ ] 整体批准后激活本子任务；加载 auth/角色库/数据库/前端状态规范，读取 AGENTS 与当前 git status。
2. [ ] 全仓枚举 primary、PrimaryOrganizationID、is_default/isDefault、UpdateUser roleIds 运行时消费者和生成物。
3. [ ] 修改 Proto、Ent 源；实现登录候选排序与初始会话，移除 primary 维护；改用户资料接口与明确成员角色编辑。
4. [ ] 生成 Ent、API、OpenAPI/Web，追加正式迁移；同步 bootstrap、种子、夹具和前端调用。
5. [ ] 实现完成后补定向回归：单/多候选、显式非法组织、切换、资料保存不改角色、单成员角色编辑、迁移数据保留。
6. [ ] 同步 auth-session-org-switch 规范和旧任务默认工作台条款，记录验证；提交该完整变更。

## 命令

```bash
go -C server generate ./internal/data/ent
make -C server api
pnpm run generate:web-client
go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run 'Auth|Admin|Membership|Organization' -count=1
pnpm --dir web exec vitest run src/pages/user/login/index.test.tsx src/pages/user/login/dingtalk-callback.test.tsx src/pages/user/login/components/organization-picker.test.tsx src/pages/admin/components/users/UserFormModal.test.tsx src/pages/admin/components/users/UserMembershipModal.test.tsx src/pages/admin/components/users/userConstants.test.ts src/pages/admin/components/users/userColumns.test.tsx src/pages/admin/users.test.tsx
pnpm --dir web tsc
git diff --check
```

新迁移测试按实际命名加入定向命令；真实 PostgreSQL 用 -v 确认测试未跳过。改动文件单独 Biome 检查；本子任务触及 Schema/认证/跨层契约，完成阶段执行 check:fast。生成配置异常先修源，不改生成文件。

## 验收与回退

记录 PRD 各项对应测试；列出仍出现 primary 的合法历史/无关样式引用。未应用迁移可回退本组代码；应用后先评估逆向 Schema 方案，不能直接部署仍读 primary 的旧服务。未授权不清库、不重置用户数据。
