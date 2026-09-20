# 执行计划：数据层迁移第二批

> 模式映射表、v5 红线、错误提示约定见
> `.trellis/tasks/archive/2026-09/09-20-data-layer-direction/design.md` 与
> `.trellis/spec/web/frontend/state-management.md`。每批次独立提交。

- [x] 1. 批次 A（orders 数据链）：`use-order-create-options`、
      `use-order-fee-options`、`use-order-lock-state`、`orders/fees.tsx`；
      消费方测试 Provider 补齐（split.test.tsx、order-commission-summary.test.tsx；
      detail 系三个半 mock 文件经实证覆写了 useOrderLockState 无需改动）。
- [x] 2. 批次 B（finance/workbench 页）：`VerificationWorkbench`、
      `cashflows/index`、`invoices/index`、`InvoiceCreateModal`、
      `finance/fees/index`、`commissions/index`、`useWorkbenchOverview`。
- [x] 3. 批次 C（BillCreationWorkbench）：主组件 + test + scale.test，
      预览竞态与防抖迁移，逐用例核对等待语义（复跑 4 次稳定）。
- [x] 4. 批次 D（partner-detail + split）：聚合 effect 与防抖预览迁移。
- [x] 5. 收口：全量 `pnpm --dir web test` 869/12 双零；`pnpm --dir web tsc` 通过；
      biome 改动文件通过。
- [x] 6. 分批提交：A=04fdfc08、B=0edffa15、C=4b487253、D=ab72886f。
- [x] 7. `pnpm run check:fast` 终验通过（869 passed / 12 skipped，
      act=0、deprecated=0）。
- [x] 8. 归档任务 + 记录日志；`useMasterDataCrud` 保持用户实现不动（已记录，
      后续收敛候选）。

## 后续候选（本批外，另立任务时参考）

- `components/ui/master-data-template/useMasterDataCrud.ts`（用户 09-19 亲改的
  ref 稳定回调实现，待用户决定是否收敛到 React Query）
- `fees/index.tsx` 的 `tagFilterRequestRef` 事件驱动搜索、
  `ProFormSearchableSelect` 的 request 属性、`useCreditLimitIntervention`
  hook（事件驱动型，非典型查询链）
