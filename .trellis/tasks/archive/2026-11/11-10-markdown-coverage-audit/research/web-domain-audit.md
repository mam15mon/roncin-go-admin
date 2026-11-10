# Research: 前端、领域与 ADR 逐份补查

- Query：核查前轮未深审或未修改 Markdown 的现行事实，并逐份登记覆盖证据。
- Scope：internal；纯规划研究，仅写当前任务 research，未修改现行文档或业务代码。
- Date：2026-11-10。

## Files Found / 逐份覆盖

本组实际 Git 跟踪 **52 份**：web 规范 21、领域 4、ADR 17、web 目录 10。全部已读取；自动生成错误目录使用生成器只读一致性校验。正文阅读全文与关键实现静态核对不表示逐字段/全部业务矩阵已运行。

结论为 **41 份已核实无需修改（针对已声明的核查深度）／10 份待修正／1 份待确认**。机器可读逐路径台账见 `web-domain-ledger.jsonl`，包含基线 SHA-256、深度、证据和限制，不是只列文件名。

## Findings / 已确证遗漏

### W01：月度提成申请的提交月与覆盖月份混淆

- `.trellis/spec/adr/0014-monthly-commission-application.md:16-18` 声称按完整自然月申请该月全部，且“不做跨月拼凑”。
- 实际 `server/internal/data/ent/schema/finance_commission_application.go:42-47`：`application_month` 是提交发生月，`coverage_to` 固定提交月份前一自然月最后一天；`server/internal/biz/finance_commission_application.go:173-185` 同口径。
- `server/internal/data/finance_commission_application_candidates.go:178-232` 全量解析截止日以前全部未申请来源，可包含更早多个归属月；候选列表可按归属月筛选，但不等于提交只选一个归属月。
- 建议：改为每个提交月一次、覆盖截至上月末全部未申请合格提成；明确累积历史月份。同步 glossary.md:49 及 architecture-map 的流程节点，保留整单审批和显式重提。

### W02：快捷预设术语仍描述旧字段含义

- `domain/glossary.md:60` 将 PartnerShippingPreset 解释为港口/船司等出货预设。
- `server/internal/biz/partner_shipping_preset.go:26-31` 只有 SHIPPER、CONSIGNEE、NOTIFY_PARTY、ENGLISH_CARGO_NAME、HS_CODE、MARKS 六类。
- `server/internal/data/partner_shipping_preset.go:27-46` 查询 EnterpriseResource/PartnerLinks 后投影，当前不是独立旧表。
- 建议：说明其为客商常用单证主体/文本的业务投影（收发通知人、英文品名、HS、唛头），持久化由企业资源复用；不能因为旧表退役就删现有用例/API。

### W03：前端权限示例字段不存在

- `web/frontend/component-guidelines.md:252` 使用 `access.financeCommissionExport`；实际 `web/src/access.ts:277-278` 导出 `canExportFinanceCommissions`，私有 permissions 对象的 financeCommissionExport 不是 access 返回字段。
- 建议改用真实 access 字段，保留路由/按钮同源规则。

### W04：ADR 与规范定位未同步当前目录

- ADR0016:4,20 的 `.trellis/tasks/09-20-ai-friendly-architecture/...` 已不存在，真实证据位于 `.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/research/architecture-audit.md`。
- ADR0015:21 的 `.trellis/spec/domain|server|web/guides/` 容易误读，guides 实际为 spec 的同级子目录，应列真实路径或明确集合。
- type-safety.md:230 仍称“总部与公司切换”，当前已为系统管理/公司工作台；只作术语同步，不改变权限判定。

### W05：web 目录的问题模板仍指导上游工程

- `web/.github/ISSUE_TEMPLATE/bug_report.md` about 和环境项仍是 Ant Design Pro、umi 版本；feature_request/question 同样把需求/提问指向上游项目。
- 当前 `web/package.json:7-29` 与 `router/adaptRoutes.tsx:14-33` 是 Vite/React Router。
- 建议改成 Roncin 项目环境和问题收集或显式标示上游模板，不新增发布/报告动作。此目录嵌套在 web 下，不等于根 GitHub 已采用这些模板，不能虚构在线流程已生效。

## 已核实的特殊文件

- `web/public/fonts/roncin-hbl/README.md`：最终 TTF SHA-256 为 `77ecc1242b6dfb247c483a7e94978197dbf0a7624d9be21cc0af79d4e328194a`，与文档一致；coverage 448 区间／30,890 码点；字宽项 98（95 ASCII + Ю/⸺/⸻ 三个超宽）。`fc-scan` 实测 family=Roncin HBL Sans SC、style=Regular、PostScript=RoncinHBLSansSC-Regular，与文档一致。未下载上游源，不验证历史 FontTools 版本或实际PDF视觉效果。
- `domain/error-catalog.md`：`node scripts/generate-error-catalog.mjs --check` 通过 403 唯一错误码，未手改；生成一致性不证明每个旧错误定义仍在调用路径。
- `organization-switch.md` 中 UmiRoute 名称是 `router/routeTypes.ts` 的实际接口形状，不代表仍采用 Umi 运行时，不应机械删词。
- `state-management.md` 的客户业务/客服只读必填确与 `SeaPersonnelSection.tsx:80-130` 一致；不能因后端允许显式三岗就误改为前端已全部可手选。

## 待确认 / 外部边界

`web/CODE_OF_CONDUCT.md:37-45` 保留 Contributor Covenant 原文和上游个人邮箱。仓库无法证实 Roncin 当前采用的投诉负责人，不擅改邮箱或联系外部。可在上层导航说明这是保留原文，不作为本项目当前支持入口；若要替换正式负责人需要用户提供事实。

## 逐份结论表

| 路径 | 状态 | 证据与核查范围 |
| --- | --- | --- |
| `.trellis/spec/adr/0001-contract-single-direction-flow.md` | 已核实无需修改 | `server/Makefile:16-19`、`server/buf.gen.yaml:14`、`scripts/generate-proto-constants.mjs:106-108`、`web/scripts/generate-services.mjs:12-15`；生成顺序、输出及禁手改要求与当前入口一致；未真实生成。 |
| `.trellis/spec/adr/0002-permission-manifest-single-source.md` | 已核实无需修改 | `server/internal/access/manifest.go`、`scripts/generate-permission-keys.mjs:6-13`、`web/src/access.ts:120-154`、`server/cmd/migrate/main.go:44-51`；权限键与有效能力入口已核对，未执行迁移或权限同步。 |
| `.trellis/spec/adr/0003-ent-schema-and-sql-migrations.md` | 已核实无需修改 | `server/cmd/migrate/main.go:18`、`server/internal/data/integration_test_helper_test.go:26-80`、`server/migrations/20260918093000_notification_delivery_template_check.sql`；正式迁移、开发校验和选项与初始化约束已核对；历史事故日期及原验收未重跑。 |
| `.trellis/spec/adr/0004-unified-transaction-wrapper.md` | 已核实无需修改 | `server/internal/data/transaction.go:29-75`、`server/internal/data/transaction.go:100-121`；共享事务合并、关闭上下文及原生SQL边界已核对，未运行并发测试。 |
| `.trellis/spec/adr/0005-pessimistic-plus-optimistic-locking.md` | 已核实无需修改 | `server/internal/data/order_write_draft.go:182-235`、`server/internal/data/order_lock.go:39-68`；版本/行锁/状态门禁范本已静态核对；不是全仓并发行为实测。 |
| `.trellis/spec/adr/0006-pagination-limit-and-masterdata-strategy.md` | 已核实无需修改 | `server/internal/biz/pagination.go:3-14`、`server/api/masterdata/v1/masterdata.proto`、`web/src/pages/master-data/components/PortsPanel.tsx`；公共200上限与主数据双模式入口已核对，保留根规范要求。 |
| `.trellis/spec/adr/0007-pinyin-search-keys.md` | 已核实无需修改 | `server/internal/platform/searchtext/searchtext.go:10-55`、`server/internal/data/ent/schema/search_keywords.go`；全拼/分词/首字母生成入口已读，保留按实体维护索引回填要求。 |
| `.trellis/spec/adr/0008-frontend-template-system.md` | 已核实无需修改 | `web/src/components/ui/index.ts`、`web/config/defaultSettings.ts:14`、`web/config/layout.ts:3-16`、`web/config/theme.ts:13`；模板导出及流式布局、尺寸和背景与当前配置一致；未浏览器测量。 |
| `.trellis/spec/adr/0009-server-state-react-query.md` | 已核实无需修改 | `web/src/utils/queryClient.ts:11-69`、`web/src/pages/orders/detail.tsx:1312-1328`、`web/src/components/layout/formDraft.ts:136`；公开/认证查询与详情无持久草稿边界已核对，未运行恢复测试。 |
| `.trellis/spec/adr/0010-order-kind-registry.md` | 已核实无需修改 | `web/src/pages/orders/order-kinds/registry.ts:9-12`、`web/src/pages/orders/order-kinds/types.ts`、`web/config/routes.ts:130-167`；SE/AE注册、未知kind不兜底和三类职责一致；历史背景不追改。 |
| `.trellis/spec/adr/0011-antd6-testing-conventions.md` | 已核实无需修改 | `web/package.json`、`web/vite.config.ts:133-168`、`.trellis/spec/web/frontend/quality-guidelines.md:90-159`；阅读全文，对照当前Vitest零用例配置和测试范例；antd历史内部版本细节未重跑所有交互，不声称库级验收。 |
| `.trellis/spec/adr/0012-same-origin-deployment.md` | 已核实无需修改 | `server/internal/server/http.go:78`、`server/internal/webassets/assets.go`、`web/vite.config.ts:101-110`；静态资源/代理注册入口存在；正式域名与证书未验证。 |
| `.trellis/spec/adr/0013-order-lock-and-immutable-document-history.md` | 已核实无需修改 | `server/internal/data/order_lock.go:39-68`、`server/internal/data/order_lock_apply.go:127-234`、`server/internal/data/order_write_lifecycle.go:133-173`；人工锁与内容门禁、退关结束Link静态核对；未执行审批或数据库测试。 |
| `.trellis/spec/adr/0014-monthly-commission-application.md` | 待修正 | `server/internal/data/ent/schema/finance_commission_application.go:42-47`、`server/internal/biz/finance_commission_application.go:173-185`、`server/internal/data/finance_commission_application_candidates.go:178-232`；应纠正提交月/覆盖截止日/累积历史归属月，现文禁止跨月描述错误。 |
| `.trellis/spec/adr/0015-trellis-workflow-and-spec-knowledge.md` | 待修正 | `.trellis/spec/domain/`、`.trellis/spec/server/backend/`、`.trellis/spec/web/frontend/`、`.trellis/spec/guides/`；知识层路径domain|server|web/guides含混，应列真实兄弟目录；历史动因保留。 |
| `.trellis/spec/adr/0016-frontend-module-boundaries.md` | 待修正 | `web/scripts/check-architecture.mjs:1-57`、`.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/research/architecture-audit.md`；规则与AST实现一致；两处未归档任务路径作为证据引用已失效，应更新到真实archive路径。 |
| `.trellis/spec/adr/index.md` | 已核实无需修改 | `.trellis/spec/adr/0001-contract-single-direction-flow.md`、`.trellis/spec/adr/0002-permission-manifest-single-source.md`、`.trellis/spec/adr/0003-ent-schema-and-sql-migrations.md`、`.trellis/spec/adr/0004-unified-transaction-wrapper.md`；全文核对16项状态与路径；ADR0014正文待修但目录摘要未扩张错误细节。 |
| `.trellis/spec/domain/architecture-map.md` | 待修正 | `server/internal/data/ent/schema/finance_commission_application.go:42-47`、`server/internal/data/finance_commission_application_candidates.go:178-232`、`web/src/pages/orders/order-kinds/registry.ts:9-12`；月度申请流程节点建议明确提交月与上月末覆盖；其他域导航对照主单/财务/工作区入口。 |
| `.trellis/spec/domain/error-catalog.md` | 已核实无需修改 | `scripts/generate-error-catalog.mjs`、`node scripts/generate-error-catalog.mjs --check（通过403码）`；自动生成文档通过源扫描一致性，不人工逐条重写，也不将历史未调用错误当现行行为。 |
| `.trellis/spec/domain/glossary.md` | 待修正 | `server/internal/biz/partner_shipping_preset.go:26-31`、`server/internal/data/partner_shipping_preset.go:27-46`、`server/internal/data/ent/schema/finance_commission_application.go:42-47`；快捷预设实际是常用单证主体/文本企业资源投影，不是港口船司；月度申请术语同步提交月覆盖口径。 |
| `.trellis/spec/domain/index.md` | 已核实无需修改 | `.trellis/spec/domain/glossary.md`、`.trellis/spec/domain/architecture-map.md`、`.trellis/spec/domain/error-catalog.md`；三个导航目标存在，说明与生成目录边界一致。 |
| `.trellis/spec/web/backend/database-guidelines.md` | 已核实无需修改 | `.trellis/spec/server/backend/database-guidelines.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/backend/directory-structure.md` | 已核实无需修改 | `.trellis/spec/server/backend/directory-structure.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/backend/error-handling.md` | 已核实无需修改 | `.trellis/spec/server/backend/error-handling.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/backend/index.md` | 已核实无需修改 | `.trellis/spec/server/backend/index.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/backend/logging-guidelines.md` | 已核实无需修改 | `.trellis/spec/server/backend/logging-guidelines.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/backend/quality-guidelines.md` | 已核实无需修改 | `.trellis/spec/server/backend/quality-guidelines.md`、`.trellis/config.yaml`；明确导航文件；全文核对目标存在，未维护第二套规则。 |
| `.trellis/spec/web/frontend/capability-navigation.md` | 已核实无需修改 | `web/src/components/ui/index.ts`、`web/src/features/finance/bill-creation/index.ts`、`web/src/features/orders/fee-review/index.ts`、`web/src/components/layout/useRouteLeaveGuard.ts`；全部表项全文阅读及链接扫描，关键公开入口对照；不声称每个导出的所有业务矩阵已执行。 |
| `.trellis/spec/web/frontend/column-settings.md` | 已核实无需修改 | `web/src/components/ui/column-settings/types.ts:1-34`、`web/src/components/ui/column-settings/freeze.ts`、`web/src/components/ui/responsive-table/layout.ts:18-71`、`web/src/components/ui/responsive-table/useResizableColumns.tsx`；偏好字段与响应式阈值、50–600等调宽规则已静态对照；几何与全屏交互未在浏览器复验。 |
| `.trellis/spec/web/frontend/component-guidelines.md` | 待修正 | `web/src/access.ts:277-278`、`web/config/layout.ts`、`web/src/components/ui/responsive-table/layout.ts:18-71`、`web/src/pages/orders/templates/components/sea/SeaPersonnelSection.tsx:80-130`；全文阅读；权限例access.financeCommissionExport不存在，应为canExportFinanceCommissions。其余视觉/模板规则关键入口对照，不虚构全视口几何验证。 |
| `.trellis/spec/web/frontend/dingtalk-qr-login.md` | 已核实无需修改 | `web/src/pages/user/login/components/dingtalk-qr-login.tsx:130-200`、`web/src/pages/user/login/components/dingtalk-qr-login.tsx:304-366`、`web/config/dingtalkLoginBootstrap.ts`、`server/internal/server/http.go:78`；本地QR、ReactQuery轮询、隐藏暂停、刷新串行与预热代码对照；外部内部协议及历史真机事实保留未重跑。 |
| `.trellis/spec/web/frontend/directory-structure.md` | 已核实无需修改 | `web/src/router/adaptRoutes.tsx:27-33`、`web/src/router/adaptRoutes.tsx:65-90`、`web/scripts/check-architecture.mjs:1-57`、`web/vite.config.ts`；目录/模块边界/页面扫描与当前文件一致。 |
| `.trellis/spec/web/frontend/duplicate-report.md` | 已核实无需修改 | `scripts/report-duplicates.mjs:28-34`、`scripts/report-duplicates.mjs:333-396`、`.github/workflows/duplicate-report.yml:4-6`、`package.json`；默认阈值、JSON版本、基线校验和UTC计划已对照；未执行全仓重复扫描。 |
| `.trellis/spec/web/frontend/finance-current-workspace.md` | 已核实无需修改 | `web/src/access.ts:168-172`、`web/src/components/layout/OrganizationWorkspace.tsx:15-32`、`web/src/features/finance/bill-creation/`、`server/api/finance/v1/settlement.proto`；全文读取并对照认证公司与共享用途/退役组织候选契约；未逐弹窗重跑请求验收。 |
| `.trellis/spec/web/frontend/finance-ledger-snapshots.md` | 已核实无需修改 | `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:94-104`、`web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:399-452`、`web/src/features/finance/ledger-summary/index.ts`；WeakMap/onLoad采纳及选择范围入口静态对照；未执行真实框架竞态与聚合数据库验证。 |
| `.trellis/spec/web/frontend/hook-guidelines.md` | 已核实无需修改 | `web/src/features/orders/options/index.ts`、`web/src/features/orders/options/orderOptionsCache.ts`、`web/src/utils/queryClient.ts`、`web/src/pages/orders/order-kinds/registry.ts:9-12`；全文与现有候选/公开入口/查询策略对照；保留规范的身份检查要求，不因未全面运行所有调用方削弱要求。 |
| `.trellis/spec/web/frontend/index.md` | 已核实无需修改 | `.trellis/spec/web/frontend/`、`AGENTS.md`、`web/package.json`；全文核对专项规范导航、生成入口及质量检查边界。 |
| `.trellis/spec/web/frontend/organization-switch.md` | 已核实无需修改 | `web/src/router/routeTypes.ts:10-31`、`web/src/router/organizationSwitchDestination.ts`、`web/src/components/layout/tabCloseGuard.ts:140-196`、`web/src/components/layout/useRouteLeaveGuard.ts:7-42`；UmiRoute只是集中配置形状非Umi运行时；落点/guard/异步确认边界已核对。 |
| `.trellis/spec/web/frontend/quality-guidelines.md` | 已核实无需修改 | `web/package.json`、`web/vite.config.ts:133-168`、`web/src/router/adaptRoutes.tsx:27-33`、`web/tests/setupTests.ts`；全文检查命令和测试加载范围；当前passWithNoTests支持零用例警示，保留禁TDD和按风险验证。 |
| `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md` | 已核实无需修改 | `web/src/pages/orders/detail.tsx:1312-1328`、`web/src/pages/orders/use-order-change-notice.ts:112-150`、`web/src/components/ui/order-template/OrderFormTemplate.tsx:168-181`、`web/src/pages/orders/templates/components/sea/SeaTransportSection.tsx:50`；新建候选/详情基线/15秒检查与持久化开关重点核对；未执行全部共享航程组合。 |
| `.trellis/spec/web/frontend/state-management.md` | 已核实无需修改 | `web/src/utils/queryClient.ts:11-69`、`web/src/components/layout/OrganizationWorkspace.tsx:43-63`、`web/src/components/ui/list-query-state/index.tsx:44-94`、`web/src/pages/orders/templates/components/sea/SeaPersonnelSection.tsx:80-130`；全篇阅读并静态对照主要状态分区；客户业务/客服当前确实只读必填，保留规范，不按后端允许手选推断UI已放开。 |
| `.trellis/spec/web/frontend/type-safety.md` | 待修正 | `server/internal/server/jsoncodec.go:35-36`、`web/src/access.ts:135-154`、`web/src/pages/master-data/components/masterDataMapper.ts`、`scripts/generate-proto-constants.mjs:106-108`；全文阅读并对照非optional零值、能力来源及生成物；残留“总部”术语宜统一为系统管理工作台。 |
| `web/.claude/skills/antd/SKILL.md` | 已核实无需修改 | `web/package.json`、`web/src/utils/requestClient.ts`、`scripts/generate-proto-constants.mjs:106-108`、`AGENTS.md`；全文核对pnpm、当前Vite边界、真实生成物保护及外部发送授权说明；未执行迁移或升级。 |
| `web/.claude/skills/pro-upgrade/SKILL.md` | 已核实无需修改 | `web/package.json`、`web/src/utils/requestClient.ts`、`scripts/generate-proto-constants.mjs:106-108`、`AGENTS.md`；全文核对pnpm、当前Vite边界、真实生成物保护及外部发送授权说明；未执行迁移或升级。 |
| `web/.github/ISSUE_TEMPLATE/bug_report.md` | 待修正 | `web/package.json:7-29`、`web/src/router/adaptRoutes.tsx:14-33`；上游问题模板仍称Ant Design Pro，bug还要求umi版本；应标明模板来源或改为本项目技术及环境问题收集，避免误导。 |
| `web/.github/ISSUE_TEMPLATE/feature_request.md` | 待修正 | `web/package.json:7-29`、`web/src/router/adaptRoutes.tsx:14-33`；上游问题模板仍称Ant Design Pro，bug还要求umi版本；应标明模板来源或改为本项目技术及环境问题收集，避免误导。 |
| `web/.github/ISSUE_TEMPLATE/question.md` | 待修正 | `web/package.json:7-29`、`web/src/router/adaptRoutes.tsx:14-33`；上游问题模板仍称Ant Design Pro，bug还要求umi版本；应标明模板来源或改为本项目技术及环境问题收集，避免误导。 |
| `web/CLAUDE.md` | 已核实无需修改 | `web/package.json:7-29`、`web/src/access.ts:135-172`、`web/src/router/adaptRoutes.tsx:27-33`、`web/src/components/layout/formDraft.ts:136`；全文对照Vite/ReactRouter、pnpm脚本、生成物、当前能力及草稿边界；未验证部署状态。 |
| `web/CODE_OF_CONDUCT.md` | 待确认 | `web/CODE_OF_CONDUCT.md:37-45`、`web/package.json:5-6`；Contributor Covenant原文仍用上游个人邮箱；本项目是否采用及负责人无法从源码确认，禁止猜邮箱或发送消息，原文不擅改。 |
| `web/README.md` | 已核实无需修改 | `web/package.json:7-29`、`web/src/access.ts:135-172`、`web/src/router/adaptRoutes.tsx:27-33`、`web/src/components/layout/formDraft.ts:136`；全文对照Vite/ReactRouter、pnpm脚本、生成物、当前能力及草稿边界；未验证部署状态。 |
| `web/README.zh-CN.md` | 已核实无需修改 | `web/package.json:7-29`、`web/src/access.ts:135-172`、`web/src/router/adaptRoutes.tsx:27-33`、`web/src/components/layout/formDraft.ts:136`；全文对照Vite/ReactRouter、pnpm脚本、生成物、当前能力及草稿边界；未验证部署状态。 |
| `web/public/fonts/roncin-hbl/README.md` | 已核实无需修改 | `web/public/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf`、`web/src/pages/orders/hbl/pdf/fontCoverage.json`、`web/src/pages/orders/hbl/pdf/fontGlyphWidths.json`、`web/src/pages/orders/hbl/pdf/font.ts:7-13`；SHA匹配，448区间30890码点，98宽度项=95ASCII+3超宽；fc-scan家族/Regular/PostScript名匹配。未下载上游或实际渲染PDF，不证明历史工具版本。 |

## Caveats / Not Found

- 现行文档未因“前轮没修改”自动判错，也未因链接存在就判内容最新；台账明确每份静态核查内容。
- 所有关键证据路径已按当前工作树存在性复核；行号是本轮核对时点，实施后可能变化。
- 不联网、不调用真实钉钉/OSS、不运行数据库、业务测试、构建或真实生成。历史验收日期及外部协议可用性保留原始语义，未伪造当前通过。
- 本组仍属规划研究，所有“待修正”未写入现行文档；交由实施阶段处理后更新状态为已修正。
