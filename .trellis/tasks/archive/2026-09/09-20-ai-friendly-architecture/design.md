# 技术设计：分层收敛与可执行边界

## 1. 分层
- 契约：继续消费 `services/roncin`、`enums.generated.ts`、`permissions.generated.ts`；不复制请求 DTO 或权限真相。
- UI：`components/ui` 继续承担视觉基础与现有模板，保留 `OrderFormTemplate` 等既有业务模板契约；不在模板中加入新领域请求。
- 领域能力：新增 `features`，只容纳真实跨模块共享的能力，公开入口显式导出；内部模块使用相对路径，避免从自身 barrel 导入形成环。
- 页面：保留路由组装、单模块私有表单、弹窗及 Hook。`pages/<领域>` 为普通模块边界；`pages/finance/<模块>` 是独立财务工作区边界。同模块内共享仍允许就近，不按每个 URL 强制建 feature。
- 平台：app、router、components/layout 等负责会话和导航，可供领域界面消费，不在本次强改既存平台依赖链。

## 2. 迁移映射
| 源 | 目标与公开面 |
| --- | --- |
| 账单页建账工作台及递归使用的私有组件/helper/测试 | `features/finance/bill-creation/`，`index.ts` 只导出工作台、Props 与实际使用类型 |
| 账单页 `billConstants` 的状态映射 | `features/finance/bill-status/index.ts`；`BillFormValues` 留原页面，列表/抽屉/结果表统一消费映射 |
| `pages/admin/audit-presentation.ts` 及测试 | `features/audit/`，只公开当前消费者需要的转换方法与类型 |
| `components/PartnerSelectOptionTags.tsx` | `features/partners/`，通过 `index.ts` 导出 |
| `hooks/useCreditLimitIntervention.ts` | `features/finance/credit-control/`，通过 `index.ts` 导出 |
| settings 的四个费用设置面板 | `pages/finance/fee-settings/components/` |
| settings 的汇率面板与导入/同步弹窗及测试 | `pages/finance/exchange-rates/components/` |
| settings 的编号规则整组、异常情况面板与测试 | `pages/admin/components/`；master-data 的无人使用转导出移除 |
| `pages/Welcome.tsx` | `pages/workbench/index.tsx`，更新路由 component 与测试引用，`/welcome` 不变 |

迁移先确定实际调用闭包，账单详情/编辑等页面私有组件不跟随工作台盲目上提。旧 settings 整页无路由消费者，相关面板迁出后删除；`/settings` 重定向继续保持。FeeSettingsPanel 的无人使用别名链移除。迁移前再次搜索所有生产与测试调用方。

## 3. 状态复用
费用状态复用现有 `orderFeeStatusMeta` 与 `statusTag`，不另造 StatusTag 组件。明确展示变更：建账候选的“已开账”改为已有标准“已进账单”，颜色遵循统一映射（已确认 green、已进账单 blue、草稿 gold、已作废 default）。账单状态由其领域元数据描述，不与费用状态合并。不增加枚举兼容逻辑。

## 4. 检查器
计划文件：`web/scripts/check-architecture.mjs` 与针对性 Node 测试；根 `check:architecture` 调用测试和扫描，接入 `check:web`。使用 AST 解析而非文本正则；显式添加锁文件已有的 `@babel/parser@7.29.8` 为 web 开发依赖，避免依赖隐式 hoist，不引入大型架构框架。

规则：
1. 不同页面模块之间不能互相导入，包含同一 finance 下不同子模块。
2. features 不能导入 pages。
3. 外部使用 feature 只能从公开入口进入：`features/<领域>/index.ts` 或 `features/<领域>/<能力>/index.ts`；更深目录的 index 不自动成为公共 API。不同能力之间也遵循公开入口，不建立 features 全局总 barrel。
4. `components/ui`、通用 `hooks`、`utils`、`constants` 不得导入 pages/features；其他非页面业务消费者不得直接导入 pages。路由适配的页面装载使用精确授权入口，不能对 router 目录整体无条件豁免。
5. 检查公开领域能力之间的有向环并报错；不把既有 UI 内部 barrel 的所有循环混入本次治理。

解析与范围：覆盖 `web/src` 的手写 TS/TSX（存在手写 JS 时同样处理），解析 import/export-from、import type、字面量 import()/require 以及 TS 导入类型引用；将 `@/`、`@root/` 与相对引用正规化，解析扩展名和目录入口，路径越界/无法解析的本地代码依赖明确报错。注释、普通字符串不作为依赖。

生产规则不限制测试合法访问被测私有实现；测试文件可排除产品边界检查，但不得对生产文件建立历史豁免基线。路由的 `import.meta.glob` 作为明确装配机制保留，并通过路由回归确认页面存在；不宣称静态扫描能判定任意运行时拼接路径。

测试在检查器实现后补充。正反例必须涵盖别名/相对路径、重导出、类型导入、动态导入、注释字符串误报、合法同模块引用、公开 API、路由授权和领域环，不允许只用扫描现有仓库通过作为唯一测试。

## 5. AI 导航
更新根 AGENTS.md 的前端目录/边界段，以及 Trellis 前端索引、目录、Hook、组件和质量指南。新增紧凑的复用能力导航，按“场景→入口→职责→不可放入内容”列出现有模板、状态映射、候选项、异步工具和新领域能力；入口链接必须能解析到真实文件。新增架构 ADR 记录取舍，并更新领域架构地图中的 Vite/React Router 与模板路径。

导航是发现入口，不抄录实现或复制权限/状态数据。保持“先搜索→私有就近→有真实跨模块使用再提取→检查边界”的单一路径，不改 Trellis 审批、运行时或平台 agent 模板。

## 6. 风险与回退
- 迁移会影响测试 mock 路径和相对导入：同步修改并运行被迁移测试及各调用入口回归。
- barrel 可能引入环或额外加载：公开入口按能力细分、显式导出，内部相对导入；构建检查路由 chunk 可用。
- 状态展示有受控变化，验收以 R3/AC2 为准。
- 无数据库/API 迁移或业务数据操作。按本任务提交边界逐组 git revert 可回退，不以重置工作区处理问题。
