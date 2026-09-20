# 数据层迁移第三批：useMasterDataCrud 与事件驱动型请求

## Goal

收敛最后一批手写数据链（前两批见归档任务 `09-20-data-layer-direction` /
`09-20-data-layer-migration-batch2`）：

1. **`useMasterDataCrud` 迁移 React Query**：用户 09-19 亲改的 ref 稳定回调
   实现（修复过无限重取），消费方为 5 个主数据面板
   （Airlines/Airports/ShippingLines/Ports/Countries）。用户已明确要求迁移。
2. **三处事件驱动型请求评估与迁移**：fees 标签筛选 `tagFilterRequestRef`、
   `ProFormSearchableSelect` 的 request 形态、`useCreditLimitIntervention` hook。

## Requirements

1. 模式与红线沿用 `state-management.md` 唯一规范：行为等价、错误提示逐处
   保持（silent / meta.errorMessage / 动态 Error）、v5 红线、测试用
   `renderWithClient`、hook 返回签名兼容消费方零改动或最小改动。
2. `useMasterDataCrud` 特别约定：
   - 返回 API 兼容：`{ data, setData, loading, total, activeTotal,
     disabledTotal, query, setQuery, reload, handleCreate, handleUpdate,
     handleToggleActive }` 字段名与语义不变。
   - 列表 query（query 状态进 key）+ 启停统计 query 并行；create/update/
     toggle 用 `useMutation` + invalidate（原「saveResponse 合并 + reload」
     语义由失效重查等价覆盖）；`setData` 保留为 `setQueryData` 薄封装。
   - 用户实现中「ref 稳定外部回调」的防死循环手段在 queryKey 模式下自然
     消失；其 hook 测试（含无限重取回归场景）等价保留。
3. 事件驱动三处的处置原则：
   - fees 标签筛选：改为防抖关键词 + 组织进 queryKey 的 useQuery，
     「已选项保活合并」用 useMemo 在渲染侧基于当前选中集合与服务端结果
     合成（不再是命令式 setState 累积）。
   - `useCreditLimitIntervention`：useQuery + `meta: { silent: true }`
     （原 catch 静默回退 false），返回布尔派生。
   - `ProFormSearchableSelect`：若确认其 request 形态是 pro-components 原生
     管理（框架自管异步），**判定为无需迁移**并在 spec 记录豁免理由；
     若存在调用方手写 effect 链则迁移调用方。
4. 消费方测试 Provider 补齐：迁移前 grep 全部真实渲染消费方。
5. 全量 vitest、tsc、check:fast 全绿；stderr 零噪音；用例数不减。

## Acceptance Criteria

- [x] `useMasterDataCrud` 迁移完成，5 个面板与 MasterDataTemplate 全绿。
- [x] fees 标签筛选、`useCreditLimitIntervention` 迁移完成；
      `ProFormSearchableSelect` 判定豁免并已在 spec 记录。
- [x] 全量 vitest（873 passed / 12 skipped）/ tsc / `pnpm run check:fast`
      通过，stderr 零噪音。
- [x] 分批提交；归档任务并记录日志。

## Notes

- `useMasterDataCrud` 是用户亲改代码：迁移须保留其修复成果的语义
  （内联回调不再引发无限重取——queryKey 模式天然满足），并在提交说明中注明。
- 事件驱动型请求中确属「pro 原生管理」的形态列入 spec 豁免清单，不算手写。
