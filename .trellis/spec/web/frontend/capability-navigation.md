# 复用能力导航

> 开发新页面前先查本表：已有能力直接消费公开入口，不复制实现。
> 导航只列「场景 → 入口 → 职责 → 不可放入内容」；细节读入口文件与关联 spec。
> 新能力只有出现真实跨模块使用才提取进来；提取后必须更新本表。

## 按场景找入口

| 场景 | 入口 | 职责 | 不可放入内容 |
|------|------|------|--------------|
| 页面骨架、分节表单/详情、CRUD 列表 | [`web/src/components/ui/index.ts`](../../../../web/src/components/ui/index.ts) | `PageHeaderShell`/`SectionCard`/`StickyFooterBar`、`OrderFormTemplate`、`MasterDataTemplate`、`FinanceLedgerTemplate`、`OrderListTemplate` 等视觉与结构模板 | 领域请求、领域状态映射；模板内不发起新领域请求 |
| 业务状态标签/文案 | [`web/src/constants/statusMeta.ts`](../../../../web/src/constants/statusMeta.ts) | `orderFeeStatusMeta` 等各实体状态元数据与通用 `statusTag`/`statusText`/`makeValueEnum` | 页面自写「状态→颜色/文字」映射；不新建平行的全局状态渲染体系 |
| 账单状态映射 | [`web/src/features/finance/bill-status/index.ts`](../../../../web/src/features/finance/bill-status/index.ts) | `billStatusMeta`（账单列表、抽屉、建账结果表共用） | 与费用状态合并；账单语义只归本能力 |
| 账单建账工作台 | [`web/src/features/finance/bill-creation/index.ts`](../../../../web/src/features/finance/bill-creation/index.ts) | `BillCreationWorkbench` 及 Props（订单费用、财务费用、账单列表三入口共用） | 账单页私有表单/抽屉（`BillEditModal` 等留页面）；未经真实共用不上提新组件 |
| 审计记录展示 | [`web/src/features/audit/index.ts`](../../../../web/src/features/audit/index.ts) | 审计操作/对象/明细的展示转换（管理端审计页与业务页审计分区共用） | 在管理页与业务页各自复制转换逻辑 |
| 汇兑损益展示 | [`web/src/features/finance/exchange-gain-loss/index.ts`](../../../../web/src/features/finance/exchange-gain-loss/index.ts) | `ExchangeGainLossTag`（对冲汇差、核销已实现损益共用展示） | 损益口径与计算逻辑（留在调用方） |
| 业务标签列表渲染 | [`web/src/components/business-tag/BusinessTagList.tsx`](../../../../web/src/components/business-tag/BusinessTagList.tsx) | 分组色描边 Tag 列表（账单、费用台账、订单费用面板共用） | 列宽、搜索等列配置（留在调用方） |
| 往来单位候选标签 | [`web/src/features/partners/index.ts`](../../../../web/src/features/partners/index.ts) | `PartnerSelectOptionTags`、`searchPartnerOptions` 与 `PartnerOption`（核销、收付、订单等共用） | 往来单位领域展示回流通用 `components/` |
| 信用额度干预提示 | [`web/src/features/finance/credit-control/index.ts`](../../../../web/src/features/finance/credit-control/index.ts) | `useCreditLimitIntervention` 与 `disableCreditExceededOptions` 信用控制查询和候选禁用 | 业务查询 Hook 放全局 `hooks/` |
| 异步竞态防护 | [`web/src/hooks/`](../../../../web/src/hooks/) | 通用 `useLatestAsync`/`useAsyncGuard` 竞态工具 | 任何业务查询 Hook（领域 Hook 进 features） |
| 订单候选缓存 | [`features/orders/options`](../../../../web/src/features/orders/options/index.ts) | 字典、首批港口/机场、人员候选缓存；组织切换/退出调用清理入口 | 把缓存键隔离误认为页面迟到响应保护 |
| 币种候选 | [`features/master-data/currencies`](../../../../web/src/features/master-data/currencies/index.ts) | `getCurrencies` / `getCurrencyOptions`，启停后显式强制刷新 | 在每个页面重复维护币种请求缓存 |
| 船公司候选 | [`features/master-data/shipping-lines`](../../../../web/src/features/master-data/shipping-lines/index.ts) | `searchShippingLineOptions` 服务端关键字搜索 | 循环翻页拼接全量 |
| 通用候选类型 | [`types/select-option.ts`](../../../../web/src/types/select-option.ts) | `SelectOption` 基础字段，业务类型按领域组合 | 散客、信用超额等业务字段 |
| 权限判定 | [`web/src/app/access.tsx`](../../../../web/src/app/access.tsx) | `useAccess` 唯一权限消费入口（键名来自生成物） | 页面硬编码第二套权限规则 |
| 后端请求 | [`web/src/services/roncin/`](../../../../web/src/services/roncin/) | OpenAPI 生成客户端（禁手改） | 页面自拼后端主机地址 |

## 何时提取新能力（单一路径）

1. **先搜索**：本表 + `rg` 符号名，确认能力不存在。
2. **私有就近**：单一页面用的实现留在该页面/页面 `components/`。
3. **真实跨模块使用再提取**：出现第二个模块消费时提取到
   `web/src/features/<领域>/<能力>/`，经 `index.ts` 公开入口导出最小面。
4. **验证边界**：提取后运行 `pnpm run check:architecture`，规则见
   [directory-structure.md](./directory-structure.md) 与
   [ADR 0016](../../adr/0016-frontend-module-boundaries.md)。
