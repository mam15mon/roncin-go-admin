# R1 实施与验证记录

## 实现结果

- 新增 `UpdatePartnerRoles(partner_id, role_types)`，HTTP `PUT /api/v1/partners/{partner_id}/roles`，沿用 `business.partner.update`。
- `partner_id` 被既有 `requestPartner` 通用资源识别，仓储也以当前公司+主键锁定目标。
- 用例只接受非空、去重合法身份类型，全部作为启用身份；不接受资料、散客、启停或结算规则输入。
- 仓储复用 `WithTx` + `ForUpdate` + `replacePartnerRoles`，保留旧角色行及黑名单；仅更新角色集合、主档更新时间和独立中文可读审计。
- 前端身份弹窗调用生成专用客户端，写响应守卫及失败输入保留不变。
- 新审计 `partner.roles.update` 显示「修改客商业务身份」，变更前后角色均显示中文。

## 修改文件（生成物由主会话统一处理）

- `server/api/partner/v1/partner.proto`
- `server/internal/biz/partner.go`、`partner_test.go`
- `server/internal/data/partner.go`（该文件另含 R5 的后续独立修改）、`partner_roles_integration_test.go`
- `server/internal/service/partner_profile.go`、`partner_roles_test.go`
- `server/internal/server/auth_test.go`
- `web/src/pages/partners/components/RoleSwitchModal.tsx` 及两个测试
- `web/src/features/audit/audit-presentation.ts`、测试及 `AuditLogSection.tsx`

## 验证

- `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'TestPartnerUpdateRoles|TestUpdatePartnerRoles|TestRequestPartner' -count=1`：三包通过。
- `python3 /tmp/roncin-system-audit-pg/run-integration.py go -C server test -p 32 ./internal/data -run '^TestPartnerUpdateRolesPreservesAggregatePostgres$' -count=1`：真实独立 PostgreSQL、正式 SQL 迁移、隔离 Schema 通过（3.172 秒）。覆盖完整资料不变、停用/散客保留、增加/不变/移除/重启角色、结算规则不变、跨公司拒绝、黑名单移除失败回滚。
- R1+R3 相关前端 13 文件共 76 例全部通过，见 R3 记录中的完整命令。
- `pnpm --dir web tsc`：通过。
- 负责修改文件 Biome lint：通过；OrgEditModal 与 RoleSwitchModal 本地 antd lint：无问题。
- `git diff --check`：通过。

## 验证过程说明

真实库用例首次暴露夹具全量比较包含 Ent 私有配置、夹具额外子资料清理顺序的问题，已改为持久化公开字段比较与按依赖清理后重跑通过。未触碰业务库或运行自动建表。

## 待主会话收尾

独立 review、全栈门禁、规范确认、分组提交和 Git 交付。此子代理未暂存、提交、推送。
