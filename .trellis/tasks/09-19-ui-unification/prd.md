# 全局 UI 规范统一治理

## Goal

针对冲刺上线阶段暴露的 UI 不一致硬伤，统一全局 Tag 色彩语义、标准化 Modal/Drawer 弹窗尺寸体系，并规范财务台账顶层骨架与公司筛选器，实现高品质纯白高密度企业级视觉一致性。

## Requirements

1. **子任务 1：统一业务状态与收支方向 Tag 色彩语义 (`09-19-ui-tag-semantics`)**
   - 建立全站统一的业务 Tag 色彩规范：
     - 方向：应收 (RECEIVABLE) 统一 `green`，应付 (PAYABLE) 统一 `volcano`。
     - 状态：草稿 (DRAFT) 统一 `default`（灰），已确认 (CONFIRMED) 统一 `processing`/`blue`，已开账/已开票/已核销 (SETTLED/ISSUED) 统一 `success`/`green`，作废/驳回 (CANCELLED/REJECTED) 统一 `error`/`red`。
   - 消除提成行明细中应收变蓝、应付变橙的冲突，消除工作台已确认变绿与账单已确认变蓝的冲突。
   - 在 `@/components/ui` 或公共模块沉淀标准常量与映射函数。

2. **子任务 2：标准化 Modal 与 Drawer 弹窗尺寸 (`09-19-ui-modal-drawer-sizes`)**
   - 在 `@/components/ui` 定义标准 T-shirt 尺寸常量（`MODAL_SIZES` / `DRAWER_SIZES`）：
     - Modal: `S: 520px` (单列简易表单/输入/确认), `M: 680px` (标准双列或编辑表单), `L: 960px` (复杂网格或大表单)。
     - Drawer: `M: 860px` (标准明细下钻), `L: 1080px` (复杂台账/全功能抽屉), `XL: 1200px` (双表协同工作台，如建账/拆单)。
   - 收敛全站分散的 18 种 Modal 宽度与 10 种 Drawer 宽度至标准常量。

3. **子任务 3：规范财务台账顶层骨架与公司筛选器 (`09-19-ui-finance-ledger-shell`)**
   - 改造 `FinanceLedgerTemplate`：
     - 支持外部传入或内部集成顶层公司/标签筛选器插槽，不再强制要求各业务页在模板外部手写裸 `<Card>`。
     - 恢复 `PageContainer` 标题与副标题渲染能力，对齐订单列表与系统管理的吸顶页头规范。
   - 重构 `bills/index.tsx`、`cashflows/index.tsx`、`invoices/index.tsx`、`verifications/index.tsx`、`fees/index.tsx` 的外挂 Card 结构，统一纳入标准页骨架。

## Acceptance Criteria

- [ ] 全站无裸露外挂在 PageContainer 外部的顶层业务 Card。
- [ ] 财务台账各页面具备规范的 PageHeader 标题与层级展示。
- [ ] 全站 Modal 与 Drawer 宽度全部引用标准 T-shirt 常量，无任意裸写数字。
- [ ] 财务与订单全域的 RECEIVABLE / PAYABLE 以及 DRAFT / CONFIRMED / CANCELLED 状态 Tag 色彩完全一致。
- [ ] 前端测试基线 `pnpm --dir web test` 全部通过，`pnpm --dir web tsc` 零错误，`biome` 校验通过。
