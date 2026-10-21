# 实施计划：取消主要组织

## 步骤

1. [x] 整体批准后激活本子任务；加载 auth/角色库/数据库/前端状态规范，读取 AGENTS 与当前 git status。
2. [x] 全仓枚举 primary、PrimaryOrganizationID、is_default/isDefault、UpdateUser roleIds 运行时消费者和生成物。
3. [x] 修改 Proto、Ent 源；实现登录候选排序与初始会话，移除 primary 维护；改用户资料接口与明确成员角色编辑。
4. [x] 生成 Ent、API、OpenAPI/Web，追加正式迁移；同步 bootstrap、种子、夹具和前端调用。
5. [ ] 实现完成后补定向回归：单/多候选、显式非法组织、切换、资料保存不改角色、单成员角色编辑、迁移数据保留。
6. [x] 同步 auth-session-org-switch 规范和旧任务默认工作台条款，记录验证；提交该完整变更。

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


## 实施记录（2026-10-13）

已取消 Membership.primary、Credential.PrimaryOrganizationID 和全部管理/认证 DTO 的 primary / is_default 字段；删除编号和名称 reserved。UpdateUser 不再接收或写入角色，基础资料表单仅创建时提供角色选择；编辑已有用户的角色通过明确 membership ID 的独立入口完成。

密码和钉钉登录共用有效工作台候选及名称/ID 顺序，内部初始会话取首项；前端多候选需明确选择。选择组件依据本次认证响应同步保存的 currentOrganizationId 判定会话，避免 initialState 的 startTransition 更新晚于候选展示。移除默认标签及突出样式。部门/组仍只映射到所属启用公司，切换继续使用原有会话轮转。

已同步 bootstrap、sync-dev 和既有测试的字段、签名与陈旧行为断言；本轮没有新增用例或运行测试。正式迁移 `20261013100000_remove_membership_primary.sql` 通过 Ent 当前字段投影与旧 primary 字段的 Atlas PostgreSQL 离线差异规划生成，唯一语句为 `ALTER TABLE "memberships" DROP COLUMN "primary";`，临时生成程序已清理。未连接或修改实际数据库，空库/升级迁移执行仍未验证。

生成命令：

- `go -C server generate ./internal/data/ent`
- `make -C server api`
- `pnpm run generate:web-client`

非测试检查：

- `go -C server build ./...` 通过。
- `go -C server vet ./internal/biz ./internal/data ./internal/service ./cmd/bootstrap-admin ./cmd/sync-dev` 通过；该命令只做静态分析，不执行测试。
- `pnpm --dir web tsc` 通过。
- 修改的页面与既有前端夹具定向 Biome 检查通过，仅 `userColumns.test.tsx` 两条原有无用 Fragment 信息提示。
- 6 个修改的 Ant Design 产品组件逐文件 `antd lint <file> --format json` 均为 0 问题。
- `git diff --check` 通过。

待父任务统一检查/提交。本轮执行约束要求不新增或运行测试，因此不执行计划中的 Vitest、Go 测试、check:fast 或真实数据库迁移验收。后续管理员补齐和多组织批量加入子任务使用本次收敛后的无 primary 签名。
