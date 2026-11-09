# 前端文档同步结果

- 日期：2026-11-09。
- 范围：前端项目入口、两个仓库自带项目技能，以及 `.trellis/spec/web/`。仅文档；没有修改业务代码、配置、生成物或数据库，也没有 Git 写操作。
- 研究依据：[规划审计](./web-doc-audit.md)。本次为现状静态核对，不等同于重新执行全部业务验收。

## 已落实

- W1：web README 改为当前 Roncin/Vite 工程中文说明，中文副本转为单一入口；删除上游模板的错误命令和失效 cheatsheet 引用。
- W2：CLAUDE 改为中文项目协作入口，权限消费 permissionCapabilities，实现后补验证，生成/提交要求统一到根规范。
- W3：航程规范明确主详情不恢复/写入持久草稿，保留内存输入和编辑基线、未保存保护及保存/刷新语义；新建、独立 HBL 的持久草稿契约保留。
- W4：web/backend 五份空模板替换为对应 server 规范导航，不复制另一套规则。
- W5：组件规范的页面类型表恢复完整结构。
- W6/W7：两个仓库自带技能改为中文，命令使用当前本地 pnpm CLI；升级保护 Vite/React Router、权限和工作区，正确保护 Roncin 客户端和全部当前常量生成物；外部 issue 发送仍需要用户明确授权。
- 补充：组件规范当前注册类型改为海运和空运出口（`registry.ts:9-12`）；ETA 在 ETD 后移时保留原输入并依赖校验报错（`SeaTransportSection.tsx:597-647`、`SeaScheduleDateFields.test.tsx:90-107`）。
- 补充：列设置规范同步订单审核费用核对表的现有固定全列例外；状态管理中的旧财务公司/标签测试矩阵和标签 CSV 说明同步退役契约；类型安全列出真实 OpenAPI 输入和 Proto 常量生成命令。
- 未扩大行为规范：保留主动变动检查、异步隔离、并发提交基线等原有正确约束，不因某页尚未迁移就削弱要求。

## 修改文件

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
- `web/.claude/skills/antd/SKILL.md`
- `web/.claude/skills/pro-upgrade/SKILL.md`
- `web/CLAUDE.md`
- `web/README.md`
- `web/README.zh-CN.md`

## 逐份现行规范阅读与对照

| 文档 | 实现或配置对照与处理 |
| --- | --- |
| web/frontend/index.md | 当前 package、路由和各入口；纠正不存在的 web/types 导航，生成清单统一指向类型安全规范 |
| capability-navigation.md | 读取全部能力清单与所有链接；抽查 fee-review、bill-creation、layout guards、options 和 ui 入口，未确认新陈旧点 |
| column-settings.md | ColumnSettingsEntry/useColumnSettings、freeze.ts、responsive-table/layout.ts/useResizableColumns.tsx 的字段身份、作用域、宽度边界、固定区和生命周期；补现有审核表例外 |
| component-guidelines.md | 读取全篇；对照 ui 模板、routeUtils/TAB_KEY_RULES、registry、SeaTransportSection、layout.ts、默认主题和根 patches 声明；修复表结构、注册范围、ETA规则，弃用清零结论改明确历史时点 |
| dingtalk-qr-login.md | QR组件状态、active/visibility、独立state query、取消/过期、配置refetch入口及相关测试源码；保留历史真机记录，不宣称本轮真实钉钉已验证 |
| directory-structure.md | 实际目录、adaptRoutes 的 import.meta.glob 排除及 HydrateFallback、架构检查入口；无确证过时内容 |
| duplicate-report.md | 根 package.json 相关命令与 .github/workflows/duplicate-report.yml 北京时间周一11点；历史标签已标退役，无需重写历史结果 |
| finance-current-workspace.md | bill-creation 当前身份/来源等值校验及访问门控、实际认证入口；无确证过时内容 |
| finance-ledger-snapshots.md | FinanceLedgerTemplate 的 request、onLoad 与 onAcceptedSnapshot，领域汇总公开入口；无确证过时内容 |
| hook-guidelines.md | 通用hooks、订单options公开入口与组织/品类隔离要求；无确证过时内容，保留规范要求 |
| organization-switch.md | organizationSwitchDestination、workspaceTabs、useRouteLeaveGuard 的location.key及dirty、tabCloseGuard；现有详情持久草稿退役说明已正确 |
| quality-guidelines.md | 根/web脚本、vite.config.ts passWithNoTests、页扫描排除、patch与相关测试辅助函数；无确证过时内容 |
| sea-master-voyage-inheritance.md | new/detail 模板身份、完整 Form store 组包、版本基线、OrderFormTemplate 草稿身份与现有测试；修正主详情存储描述，不改内存草稿/版本语义 |
| state-management.md | queryClient 默认策略、OrganizationWorkspace独立实例、requestClient、详情dirty/路由guard及源码测试；修正剩余公司/业务标签矩阵及CSV口径 |
| type-safety.md | access.ts:135-154、masterDataMapper.ts:18-26、生成脚本实际输出与jsoncodec；补全生成输入和常量命令，保留真实ProtoJSON契约 |
| web/backend/index.md | 已正确指向server，保留 |
| web/backend其余五份规范 | 原占位模板改对应server规范导航，目标存在 |

前端 README/中文入口/CLAUDE 与两项目技能另与 package.json、实际目录、根协作约束逐项核对。第三方许可证、CODE_OF_CONDUCT、issue模板、字体说明与 Trellis 通用框架文档未改写。

## 已执行检查

- `git diff --check -- web .trellis/spec/web`：通过。
- 26 份负责范围文档的 141 个本地 Markdown 内联链接目标及 Markdown 标题锚点扫描：除新增统一入口 `docs/README.md` 尚由并行通用文档代理创建外，其余通过；最终由主会话在整合后复扫该入口。
- 新增说明涉及的根/web pnpm scripts 及具体生成文件、定向测试路径存在性断言：通过。技能中的根目录命令与web目录CLI分别按明示上下文核对。
- `pnpm --dir web exec antd --help`、`antd bug --help`、`antd lint --help`：只读执行通过，CLI本地版本6.6.2；没有调用submit、迁移、升级或真实外部报告。
- 本地CLI的 `ANTD_NO_AUTO_REPORT` 行为在当前dist只读检索确认。

## 限制与后续

- 不运行 Vitest、tsc、全量lint、构建、数据库或真实钉钉/浏览器验收：本次只改文档，按批准方案以文档适用验证收口。
- 链接扫描只检查围栏外字面量 Markdown 内联目标和标题slug，不验证远程URL可达性；源码中的旧AE未开放注释记录在主会话ADR报告，本任务不改代码。
- 全仓 `git diff --check` 曾发现并行通用文档尾空格，已通知主会话；本范围检查无误。主会话负责最终统一审阅、Git提交和归档。
