# 全仓文档核查验收汇总

- 核查日期：2026-11-09。
- 需求：查找项目文档，按当前仓库更新过时内容。
- 规划基线：`ec71fb19`；规划与研究提交：`d32a7f67`。
- 本任务只更新文档与任务记录，未改业务代码、API、Schema、依赖、权限、配置、生成物或数据。

## 范围与结果

基线已跟踪文档 **1,111 份**，完整路径见 [盘点清单](document-inventory.md)。另登记 47 份被忽略的本地技能文档，不跟随外部软链接，也不强行加入 Git。

正式文档共 **63 份变更**：通用入口与业务说明 16、服务端说明与规范 24、前端说明与规范 16、领域导航与架构决策 7。包括新增 docs/README.md，未对所有文档机械修改日期。

| 主要差异 | 处理 |
| --- | --- |
| 功能与技术栈 | SE/AE 开放范围、标签退役、Vite 工程、Go 版本、数据库会话与共享主单/每票分单语义 |
| 开发命令 | 初始化幂等、正式 SQL 隔离测试、主数据全局目录、港口源和中文文件；修复 pnpm 多余 -- 与 Go generate 漏 ./... |
| 财务及订单 | 统一人工审核锁、无需独立预审建账、逐源费用待审阻断、NETTING、提成经济重算、完结本身不冻结 |
| 前端状态 | 详情无持久草稿但保留 dirty/离开保护，新建/HBL 仍持久化；ETA 保留输入并校验 |
| 海运模型 | DIRECT/HOUSE 两态、模式切换与外部确认；Switch、UNDETERMINED 及旧链路明确退役 |
| 历史材料 | 原计划/验收按历史语义保留，就近指向替代实现；客户静态行政区划等未落实项如实记录 |
| 维护入口 | docs/README.md 汇总开发、当前规范、历史计划及文档同步要求 |

## 验证

- MarkdownIt 解析 124 份现行说明、规范及项目技能：**397 个本地链接和标题锚点，0 个失效**；含引用式链接，不访问远程 URL。
- 独立复核显式 pnpm run：**188 处引用、33 种脚本**，均存在于根/web package.json；命令所属工作目录按文档上下文核对。
- `node scripts/generate-error-catalog.mjs --check` 通过：**403 个唯一错误码**，生成文档未手改。
- `git diff --check`、变更范围检查及原未跟踪目录的 **4 个文件 SHA-256** 对比通过。
- 临时 Node 脚本只读验证 pnpm 参数传递，Go generate -n 只打印/检查不执行生成器；相关修正依据见 [补充验证](adr-update-result.md)。
- 独立 check 完成全文差异及关键源码复核，并修复主分单共享语义、GIN 术语、系统工作台名及生成物漏列；见 [独立复核记录](check-result.md)。

## 保留与限制

- 874 份历史任务文档、5 份原会话日志、框架模板、许可及社区模板保留时间语义，不伪造当前业务验收。
- 初查发现的 5 个现行失效链接已修复；另 1 个原历史任务链接只记录原貌，不追改旧任务。
- 不为纯文档修改运行产品 lint/类型检查、Go/Vitest、构建、数据库迁移、同步写入、OSS/钉钉或正式部署验收；这些结果不记为“通过”。
- 当前实现仍有已记录的范围外旧源码注释和静态行政区划来源；本次不改变实现迁就文档。
- 本记录表示本次源码核对时点；后续功能、命令和契约变更须同步维护现行文档。

## 正式文档变更清单

- `.trellis/spec/adr/0001-contract-single-direction-flow.md`
- `.trellis/spec/adr/0009-server-state-react-query.md`
- `.trellis/spec/adr/0010-order-kind-registry.md`
- `.trellis/spec/adr/0013-order-lock-and-immutable-document-history.md`
- `.trellis/spec/domain/architecture-map.md`
- `.trellis/spec/domain/glossary.md`
- `.trellis/spec/domain/index.md`
- `.trellis/spec/server/backend/capability-navigation.md`
- `.trellis/spec/server/backend/commission-scheme-assignment.md`
- `.trellis/spec/server/backend/database-guidelines.md`
- `.trellis/spec/server/backend/dingtalk-notification-markdown.md`
- `.trellis/spec/server/backend/directory-structure.md`
- `.trellis/spec/server/backend/exchange-rate-single-rate.md`
- `.trellis/spec/server/backend/finance-bill-currency.md`
- `.trellis/spec/server/backend/index.md`
- `.trellis/spec/server/backend/operating-company-commission-attribution.md`
- `.trellis/spec/server/backend/order-lock-and-document-version.md`
- `.trellis/spec/server/backend/organization-shared-masterdata.md`
- `.trellis/spec/server/backend/partner-casual-contract.md`
- `.trellis/spec/server/backend/partner-role-blacklist.md`
- `.trellis/spec/server/backend/quality-guidelines.md`
- `.trellis/spec/server/backend/retire-role-data-scope.md`
- `.trellis/spec/server/backend/sea-document-change-history.md`
- `.trellis/spec/server/backend/sea-export-document-contract.md`
- `.trellis/spec/web/backend/database-guidelines.md`
- `.trellis/spec/web/backend/directory-structure.md`
- `.trellis/spec/web/backend/error-handling.md`
- `.trellis/spec/web/backend/logging-guidelines.md`
- `.trellis/spec/web/backend/quality-guidelines.md`
- `.trellis/spec/web/frontend/column-settings.md`
- `.trellis/spec/web/frontend/component-guidelines.md`
- `.trellis/spec/web/frontend/index.md`
- `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md`
- `.trellis/spec/web/frontend/state-management.md`
- `.trellis/spec/web/frontend/type-safety.md`
- `AGENTS.md`
- `PLAN.md`
- `README.md`
- `docs/README.md`
- `docs/airport-port-sync.md`
- `docs/cross-repository-transactions.md`
- `docs/eyun-config-management.md`
- `docs/local-development.md`
- `docs/migration-matrix.md`
- `docs/order-attachment-oss.md`
- `docs/plans/业务标签建设计划.md`
- `docs/plans/企业资源备忘录建设计划.md`
- `docs/plans/客户页面.md`
- `docs/plans/提成管理增强计划.md`
- `docs/plans/海运出口订单闭环验收.md`
- `docs/plans/费用明细批量转账单.md`
- `server/AGENTS.md`
- `server/README.md`
- `server/internal/biz/README.md`
- `server/internal/data/README.md`
- `server/internal/service/README.md`
- `server/migrations/README.md`
- `server/seeds/README.md`
- `web/.claude/skills/antd/SKILL.md`
- `web/.claude/skills/pro-upgrade/SKILL.md`
- `web/CLAUDE.md`
- `web/README.md`
- `web/README.zh-CN.md`

## 文档提交

- `ec175848` docs: 同步项目介绍与开发业务文档（16 份）
- `5e3cd721` docs: 更新服务端现行契约与协作说明（24 份）
- `71f2a988` docs: 更新前端工程与表单状态文档（16 份）
- `d41ead36` docs: 同步领域导航与架构决策（7 份）

核查记录与任务状态另以 docs 提交；任务完成后归档及记录会话，不推送远程。
