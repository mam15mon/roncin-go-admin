# 架构审查证据

日期：2026-09-20。只读审查；未修改产品代码。路径行号为规划时快照。

## 已有基础
- `web/src/components/ui/index.ts:1` 集中导出模板、输入组件和布局；订单列表、账单列表、管理中心已有模板消费。
- `web/src/constants/statusMeta.ts:1` 已有状态元数据、`statusTag`、`statusText`，不能重复创建新的全局状态渲染体系。
- `web/config/routes.ts:1` 是当前路由配置，`web/src/router/adaptRoutes.tsx:24` 使用 Vite glob 装载页面。
- `.github/workflows/ci.yml:34` 调用 `pnpm run check:web`；根 package.json 的该入口已包含权限和枚举生成物检查。
- `web/biome.json` 无依赖边界规则；当前尚无 `web/src/features/`。

## 跨页面依赖与归属
| 证据 | 问题 | 处置 |
| --- | --- | --- |
| `web/src/pages/orders/fees.tsx:17`、`web/src/pages/finance/fees/index.tsx:22`、`web/src/pages/finance/bills/index.tsx:34` | 三处使用账单页内部建账工作台 | 提取完整建账依赖闭包至 `features/finance/bill-creation/`；只公开工作台和调用方必需类型 |
| `web/src/pages/partners/components/AuditLogSection.tsx:19`、`web/src/pages/admin/audit.tsx:15` | 往来单位依赖管理页的审计展示 | 提取 `features/audit/`，保留纯展示转换与原测试 |
| `web/src/pages/finance/exchange-rates/index.tsx:4` | 财务汇率页引用设置页内部面板 | 汇率面板与导入、同步弹窗移到财务汇率页内部 |
| `web/src/pages/finance/fee-settings/index.tsx:13` | 财务费用设置引用设置页面板 | 四个费用面板与测试移到财务费用设置页内部 |
| `web/src/pages/admin/index.tsx:19` | 管理中心引用设置页的异常/编号规则 | 移到管理中心内部，包含编号规则子组件与测试 |
| `web/src/pages/master-data/components/NumberRulesPanel.tsx:1` | 无调用方的跨页面转导出 | 全仓确认无调用后移除，不增加兼容出口 |
| `web/src/pages/settings/index.tsx:21`、`web/config/routes.ts` 的 `/settings` 重定向 | 旧设置整页已无路由入口，仍聚合迁移前面板 | 移除未使用旧整页；保留现有 URL 重定向行为 |
| `web/src/pages/settings/components/FeeSettingsPanel.tsx:1`、`web/src/pages/finance/fee-settings/index.tsx:18` | 无调用方的别名转导出链 | 确认无调用后移除 |
| `web/src/pages/Welcome.tsx:6`、`web/config/routes.ts:40` | 工作台入口与私有组件分散 | 入口移到 `pages/workbench/index.tsx`，更新组件配置和测试，URL 不变 |
| `web/src/components/PartnerSelectOptionTags.tsx:14` | 通用 components 混入往来单位领域展示，核销和收付页复用 | 移到 `features/partners/` 公开入口 |
| `web/src/hooks/useCreditLimitIntervention.ts:13` | 全局 hooks 混入信用控制业务查询，核销和收付页复用 | 移到 `features/finance/credit-control/` 公开入口 |

## 状态重复
- `web/src/pages/finance/bills/components/billWorkbenchFeeColumns.tsx:35` 自写费用状态，已确认蓝色、已开账绿色；`web/src/constants/statusMeta.ts:60` 的相同状态为已确认绿色、已进账单蓝色。建账候选改用已有统一映射，明确记录展示变化。
- `web/src/pages/finance/bills/components/billConstants.ts:4` 账单状态映射与 `BillCreationResultTable.tsx:95` 的内联判断重复；提取 `features/finance/bill-status/`，复用现有通用渲染函数。费用与账单状态不合并。

## AI 指南漂移
- `.trellis/spec/web/frontend/index.md:1` 标题仍为 Umi。
- `.trellis/spec/web/frontend/hook-guidelines.md:3` 声称没有 hooks；第 7 行把跨页面业务 Hook 一律放到全局 hooks。
- `.trellis/spec/domain/architecture-map.md:11` 仍写 Umi，第 26 行订单模板路径已与实际不符。
- `.trellis/spec/web/frontend/directory-structure.md:18` 只说明就近，没有共享提取与公开入口规则。

## 检查设计证据与限制
- 取证扫描涵盖 src 的 TS/TSX 静态 import/export 与字面量动态 import，解析 `@/` 和相对路径；先排除测试与生成客户端，以免把测试访问被测对象当成业务越界。正式检查须用 AST，不能照搬取证正则。
- 页面模块边界：通常 `pages/<领域>/`；财务独立工作区细分为 `pages/finance/<模块>/`。同一模块内部允许就近组合；跨模块共享进入 features。
- 路由装配是加载页面的合法入口，不应被“非 pages 禁止依赖 pages”的粗暴规则误伤。
- 通用 UI 模板有既存布局、路由基础设施依赖，本期不强推所有代码形成简单单向层级；重点拦截 pages/features 业务依赖反流与 feature 私有入口导入。
- 当前工作区未安装可解析的 `web/node_modules/typescript`；依赖安装及运行验证留实施阶段。锁文件已有 `@babel/parser@7.29.8`，计划显式列为 web 开发依赖供 AST 检查使用，不依赖偶然提升的传递依赖。
- 本审查不声称穷尽所有语义重复；订单候选缓存、通用 options 聚合等既存设计作为导航记录，本期不重做其缓存/请求生命周期。
