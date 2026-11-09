# 独立审查记录

## 范围与结论

2026-11-09 由独立 `trellis-check` 代理审查 `bfcbd9de` 至当前工作树的完整任务变更，覆盖 `server`（Proto、biz/service/data/server/access、Wire、Ent、正式迁移）、`web`（共享模板、订单/费用/账单/企业资源、权限、导出与生成客户端）及相关规范。已读取任务 PRD/design/implement、check.jsonl 研究材料和前后端 Quality Check。

未发现尚未解决的产品或迁移阻塞问题，可进入主会话最终完整门禁与交付。此结论不代替主会话的完整门禁、提交和迁移应用记录。

## 已修复发现

1. `web/tests/fixtures/role-permission-capabilities.json` 尚包含已退役的 `system.finance.fee.tag`。审查通知前端实现者移除；相关 `roles.grantable.test.tsx` 的权限总数同步为 280/262，角色保存验证保留。前端实现者报告角色 10 例通过。
2. `.trellis/spec/web/frontend/finance-ledger-snapshots.md` 仍把标签列为现行筛选。审查者删除该过时例子，保留翻页、排序、筛选、工作区与选择范围的全部约束。

## 行为核对

- 18 个标签 RPC、独占消息、HTTP/gRPC/Wire/provider、业务标签仓储和附加读取/过滤全部退役；9 个字段保留原名称和编号的 reserved，资源类型 4 不再合法，5..10 编号未重用。生成客户端和权限键同步，无空接口或兼容层。
- 仅移除独立费用标签权限；订单更新、费用更新、账单更新和企业资源 CRUD 权限保留。旧权限与角色关联复用正式迁移后的 Manifest 同步清理。
- 正式迁移只移除五张标签专表、TAG 资源及指向其 ID 的客商关系。非标签明细引用、双向关系类型漂移和未知依赖均使迁移回滚；不使用 CASCADE、不改历史迁移、不重置数据库。
- 迁移前态测试通过正式 SQL 前缀建立标签/非标签混合夹具，逐表整行快照保护订单、费用、账单与账单行、客商、组织/用户/角色、全部保留资源类别及历史审计；重复迁移和权限同步验证保持幂等。
- 拆票只删标签复制/转移；费用归属、金额守恒、版本、加锁和事务不变。费用单条/批量删除仍保留账单历史来源置空、金额快照和审计/原子性验证。
- 前端标签列、筛选、批量按钮、候选、快捷创建、CSV、列偏好、查询类型和资源 type=4 入口均已去除。普通状态/进度、人员客商候选、汇差标记、TagsView 和历史审计中文词典保留。
- 账单整条公司/标签工具栏已删除，列表 organizationId 仅取当前认证工作区；旧恢复条件不能覆盖，缺身份不调用列表 API。AppLayout 与 OrganizationWorkspace 按用户+组织 key 重建查询、选择与弹窗，沿用既有请求取消和缓存隔离；官方 onLoad 采纳快照与财务汇总保持。
- 企业资源类型切换仍以类型 key 隔离编辑会话，原有 Form.submit/onFinish 校验路径保留。其余资源/关联/图片删除任务未被误删。
- 全仓精确标识复核后的余项仅为 Proto reserved、退役/负断言测试、历史迁移和允许保留的审计展示；不按普通 Tag 字样清零。

## 审查者实际验证

| 检查 | 结果 |
| --- | --- |
| `pnpm --dir web tsc` | PASS |
| 最终 44 个前端手写变更文件的 `biome check` | PASS，无自动修复 |
| `go -C server test ./internal/server -run '^TestBusinessTagRetirement' -count=1` | PASS，真实生产构造注册检查及 Proto reserved 反射 |
| `bill-query-state.test.tsx`、`utils/api.test.ts` | 2 套 8 例 PASS |
| `OrganizationWorkspace.query.test.tsx`、`OrganizationWorkspace.modal.test.tsx`、`workspaceMenus.test.tsx` | 3 套 12 例 PASS |
| `git diff --check` | PASS |

## 实现者验证证据复核

以下真实数据库与生成验证由对应实现者执行，审查者检查测试代码并复核结果，不将默认环境下的 SKIP 计作通过。

- 存储实现者在专用隔离 PostgreSQL 执行业务标签升级、四种异常回滚、冷启动，以及费用删除/批量维护、拆票并发与金额守恒、图片删除、资源客商关联、费用分页汇总和迁移冷启动：data 包 32.222 秒、migration 包 5.297 秒，0 SKIP。证据日志 `/tmp/roncin-retire-tags-storage-tests.log`。
- 最终合法 BOOKING 备注夹具再次执行三组新迁移测试：升级 2.90 秒、四种异常 11.02 秒、冷启动 2.65 秒，包合计 16.616 秒，0 SKIP。证据日志 `/tmp/roncin-retire-tags-migration-final.log`。
- Ent 生成目录 SHA256 重跑幂等；Proto/Wire/客户端/枚举/权限共 120 个生成文件重跑 SHA256 相同。`go mod tidy` 后无 go.mod/go.sum 差异。mutation.go 大默认 diff 为重复文本的差异对齐，histogram 对比与真实标签删除一致。
- 后端实现者已执行受影响 biz/service/server/access/cmd/server 包测试、vet 与 Buf lint；前端实现者最终 27 套 157 例定向、tsc、Biome 均通过。完整前后端门禁由主会话统一记录。

## 未修复发现与应用边界

无未修复发现。本次独立审查未向真实开发库应用退役迁移；专用数据库测试通过不表示真实开发/生产数据已经清理。正式迁移文件是否应用、最终完整门禁及合并推送结果由主会话验收记录明确说明。
