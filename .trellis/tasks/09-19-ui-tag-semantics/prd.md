# 统一业务状态与收支方向 Tag 色彩语义

## Goal

收敛全站订单、财务账单、费用、提成、核销及工作台中混乱的业务状态 Tag 和收支方向 Tag 色彩，建立统一的 Tag 色彩设计规范。

## Requirements

1. **核心语义定义**：
   - **收支方向 (Direction)**：
     - `RECEIVABLE` (应收)：统一 `green`
     - `PAYABLE` (应付)：统一 `volcano`
   - **单据/费用通用生命周期状态 (Status)**：
     - `DRAFT` (草稿)：统一 `default`（灰）
     - `CONFIRMED` / `ACTIVE` (已确认/有效)：统一 `processing` 或 `blue`
     - `SETTLED` / `ISSUED` / `PAID` (已开账/已开票/已核销/已发放)：统一 `success` 或 `green`
     - `CANCELLED` / `REJECTED` / `VOID` (已取消/已驳回/已作废)：统一 `error` 或 `red`
     - `LOCKED` (已锁定/风控)：统一 `warning` 或 `orange`
2. **重点修复冲突点**：
   - 修复 `web/src/pages/finance/commissions/components/CommissionLineTable.tsx` 中 `RECEIVABLE` 被标为 `blue`、`PAYABLE` 被标为 `orange` 的问题。
   - 修复 `web/src/pages/finance/bills/components/billWorkbenchFeeColumns.tsx` 与 `BillCreationResultTable.tsx` 中“已确认”有时为 `blue` 有时为 `green` 的不一致。
   - 修复 `BillCreationResultTable.tsx` 中草稿为 `gold` 的问题，统一为 `default`。
3. **沉淀公共组件/映射**：
   - 在 `web/src/components/ui` 或相关公共域提供统一的 `directionTagMeta` 与 `statusTagMeta`，供各业务组件复用。

## Acceptance Criteria

- [x] 提成明细中的费用方向 Tag 与账单、费用明细中的方向 Tag 完全一致（应收绿、应付火山红）。
- [x] 账单工作台、创建结果表中的“已确认”与“草稿”色彩与全站统一。
- [x] 受影响页面的相关单元测试保持 100% 通过。
- [x] `pnpm --dir web tsc` 零错误。
