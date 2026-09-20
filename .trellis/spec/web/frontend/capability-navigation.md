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
| 往来单位候选标签 | [`web/src/features/partners/index.ts`](../../../../web/src/features/partners/index.ts) | `PartnerSelectOptionTags`（核销、收付等页面共用） | 往来单位领域展示回流通用 `components/` |
| 信用额度干预提示 | [`web/src/features/finance/credit-control/index.ts`](../../../../web/src/features/finance/credit-control/index.ts) | `useCreditLimitIntervention` 信用控制查询 | 业务查询 Hook 放全局 `hooks/` |
| 异步竞态防护 | [`web/src/hooks/`](../../../../web/src/hooks/) | 通用 `useLatestAsync`/`useAsyncGuard` 竞态工具 | 任何业务查询 Hook（领域 Hook 进 features） |
| 组织级候选项工具 | [`web/src/utils/options.ts`](../../../../web/src/utils/options.ts) | 候选项聚合与组织隔离工具（规则见 hook-guidelines「组织级异步联想隔离」） | 未出现跨模块需求前批量迁入领域代码 |
| 权限判定 | [`web/src/app/access.tsx`](../../../../web/src/app/access.tsx) | `useAccess` 唯一权限消费入口（键名来自生成物） | 页面硬编码第二套权限规则 |
| 后端请求 | [`web/src/services/roncin/`](../../../../web/src/services/roncin/) | OpenAPI 生成客户端（禁手改） | 页面自拼后端主机地址 |

## 何时提取新能力（单一路径）

1. **先搜索**：本表 + `grep` 符号名，确认能力不存在。
2. **私有就近**：单一页面用的实现留在该页面/页面 `components/`。
3. **真实跨模块使用再提取**：出现第二个模块消费时提取到
   `web/src/features/<领域>/<能力>/`，经 `index.ts` 公开入口导出最小面。
4. **验证边界**：提取后运行 `pnpm run check:architecture`，规则见
   [directory-structure.md](./directory-structure.md) 与
   [ADR 0016](../../adr/0016-frontend-module-boundaries.md)。
