# 数据层迁移第二批：剩余手写请求链收敛

## Goal

延续 `09-20-data-layer-direction`（已归档）的模式映射表与规范（现已是
`.trellis/spec/web/frontend/state-management.md` 的唯一模式），迁移审计清单中
剩余的手写 `useState + useEffect + 竞态令牌` 服务端状态链。

## 范围（12 个目标文件，分 4 个互不相交批次）

| 批次 | 文件 |
|------|------|
| A orders 数据链 | `orders/use-order-create-options.ts`、`orders/use-order-fee-options.ts`、`orders/use-order-lock-state.ts`、`orders/fees.tsx`（含各自测试与消费方测试的 Provider 补齐） |
| B finance/workbench 页 | `finance/verifications/VerificationWorkbench.tsx`、`finance/cashflows/index.tsx`、`finance/invoices/index.tsx`、`finance/invoices/components/InvoiceCreateModal.tsx`、`finance/fees/index.tsx`、`finance/commissions/index.tsx`、`workbench/useWorkbenchOverview.ts`（含各自测试） |
| C 账单工作台 | `finance/bills/components/BillCreationWorkbench.tsx`（token/fingerprint 双令牌 + 手写防抖）+ `.test.tsx` + `.scale.test.tsx` |
| D 详情/拆票大页 | `partners/partner-detail.tsx`、`orders/split.tsx`（含各自测试） |

**明确不在范围**：`components/ui/master-data-template/useMasterDataCrud.ts`
（用户 2026-09-19 亲改的 ref 稳定回调实现，保留并观察；标记为后续候选）。

## Requirements

1. 严格遵循 `state-management.md` 唯一模式与第一周期的经验（同目录归档任务
   `09-20-data-layer-direction/design.md` 的模式映射表仍然有效）：
   - 行为等价、错误文案逐处保持（空 catch 静默 → `meta: { silent: true }`；
     自带文案 → `meta: { errorMessage }`；动态文案 → queryFn 包装 Error）。
   - v5 红线：useQuery 无 onSuccess/onError 回调、isLoading 语义、不在调用点
     设 retry、禁止再写任何竞态令牌。
2. Hook 迁移保持返回字段签名兼容，消费方零改动或最小改动。
3. **Provider 补齐是本批最大风险**：hook 被真实渲染的测试必须改
   `renderWithClient`（第一周期漏过 `orders-breadcrumbs.test.tsx`）。每个批次
   迁移前先 `grep -rn "<hook 名>" src` 找全真实渲染消费方及其测试。
4. 每批次独立提交，可单独回滚。
5. 测试 stderr 零噪音（act/deprecated = 0），用例数不减。

## Acceptance Criteria

- [x] 12 个目标文件全部迁移完成，手写竞态令牌清零。
- [x] 全量 vitest 通过（869 passed / 12 skipped，无 Provider 遗漏），
      stderr 零噪音。
- [x] `pnpm --dir web tsc`、改动文件 biome 通过；`pnpm run check:fast` 通过。
- [x] 分批次提交（A/B/C/D 四笔，Conventional Commits 规范）。
- [x] `useMasterDataCrud` 保持不动（用户自改实现），状态记录在案。

## Notes

- 模式映射表与 v5 经验：`.trellis/tasks/archive/2026-09/09-20-data-layer-direction/design.md`。
- 总部业务边界提交（23459708）改过 partner-detail、fees、split、
  use-order-lock-state 等，一切以工作区现状为准。
