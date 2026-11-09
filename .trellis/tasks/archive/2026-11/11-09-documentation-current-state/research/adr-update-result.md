# 领域导航与 ADR 更新结果

- 日期：2026-11-09；依据规划阶段 `adr-doc-audit.md` 及当前源文件复核。
- 当前仅涉及文档，无产品代码、生成物、权限或数据变更。

## 已更新

| 文件 | 同步内容 | 实现依据 |
| --- | --- | --- |
| `.trellis/spec/adr/0001-contract-single-direction-flow.md` | make api 生成服务端 OpenAPI/权限规则，前端入口消费并生成 JSON、SDK、枚举与错误常量 | `server/Makefile`、`server/buf.gen.yaml`、`web/scripts/generate-openapi-input.mjs`、`web/scripts/generate-services.mjs`、`scripts/generate-proto-constants.mjs:106-108` |
| `.trellis/spec/adr/0009-server-state-react-query.md` | 主详情内存输入与离开保护，新建/HBL 才有持久草稿 | `web/src/pages/orders/detail.tsx:1312`、`web/src/components/ui/order-template/OrderFormTemplate.tsx:168`、`web/src/components/layout/formDraft.ts:136` |
| `.trellis/spec/adr/0010-order-kind-registry.md` | 保留建立时背景，明确当前 SE/AE 两种真实注册类型 | `web/src/pages/orders/order-kinds/registry.ts:9`、`web/config/routes.ts:140` |
| `.trellis/spec/adr/0013-order-lock-and-immutable-document-history.md` | ACTIVE 未锁允许内容编辑，完结本身不冻结；统一人工审核锁，建账无预审前置 | `server/internal/data/order_lock.go:39`、`server/internal/data/order_lock_apply.go:198`、`server/internal/data/finance_bill.go:355` |
| `.trellis/spec/domain/glossary.md` | 主单关联结束不是普通进度变化；订单三维状态与独立审核锁；修复提成归属规范定位 | `server/internal/data/ent/schema/order.go:65`、`server/internal/data/sea_order_change_reassignment.go:630`、`server/internal/data/order_write_lifecycle.go` |
| `.trellis/spec/domain/architecture-map.md` | 补全生成链；规范入口可点击；历史费用审核事件与 AE 子集关系 | `server/Makefile`、`server/internal/data/ent/schema/air_house_waybill.go`、`server/internal/data/ent/schema/air_master_waybill_order_link.go` |
| `.trellis/spec/domain/index.md` | 术语索引补空运 | 实际术语章节 |

## 保留项与限制

- 其余 12 份 ADR 已读；旧背景、提交日期、历史事故保留，不按文件年龄机械改写。
- 原全仓盘点中的历史任务/日志/框架/许可不批量改写。
- `registry.ts` 中 AE 未注册的旧注释是范围外源码注释，已记录，运行注册实际包含 AE；不为修正文档修改业务源码。
- 本报告不表示真实数据库、正式部署、OSS 或钉钉端到端已验收；验证结果另见最终文档核查报告。

## 命令参数传递补充验证

- 在临时目录创建最小 package.json 与只输出 argv 的 Node 脚本，使用当前 pnpm 运行后立即清理；不修改仓库代码，不读取环境文件、不连接数据库。
- `pnpm run probe -- -source /path/to/release.zip -release 2025-1` 实际收到 `["--","-source","/path/to/release.zip","-release","2025-1"]`。
- `pnpm run probe -source /path/to/release.zip -release 2025-1` 实际收到 `["-source","/path/to/release.zip","-release","2025-1"]`。
- `scripts/run-with-env.mjs` 直接转发 argv，`server/cmd/sync-unlocode/main.go` 使用标准 `flag.Parse`；因此同步示例不应在脚本名之后放终止标记 `--`，否则后续 source/release 不会按预期解析。正式同步入口带 `-apply`，文档应准确指定源文件及配套中文对照。
- 此验证只证明参数传递，不表示执行了港口写库或校验过其他发布包。

## Go 生成入口补充验证

- 当前 `server/` 根目录没有 Go 源文件；只读执行 `go -C server generate -n` 返回 1，并提示 `no Go files in .../server`，没有实际执行任何生成器。
- 根 AGENTS.md、后端质量规范及企业资源计划的建议命令遗漏包匹配范围，已统一为 `go -C server generate ./...`，与 `server/Makefile` 的 generate 目标一致。
- 此项属于纯文档命令修正，不改变生成器、协作审批流程或数据库行为；本次未运行真实生成。
