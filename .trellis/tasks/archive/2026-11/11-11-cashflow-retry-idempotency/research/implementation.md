# 资金登记幂等重试实施

- 资金载荷与指纹同源规范化：金额/显式汇率用 Decimal 归一，币种大写、文本 trim、空可选值省略。
- 当前页面内意图到幂等键映射覆盖 A→B→A 和未知结果关闭重开；成功后仅删除本次已确认意图，不清除其他未知意图。
- 旧弹窗迟到成功不影响新弹窗，且保留同一键供再次回放确认；工作区卸载复用已有 owner signal 隔离。
- 修改：cashflows/index.tsx、cashflowCreateIntent.ts、cashflow-save.test.tsx。
- `pnpm --dir web exec vitest run src/pages/finance/cashflows/cashflow-save.test.tsx`：7 项通过。
- 修改文件 Biome 通过。未做真实库写入；后端原有公司+幂等键唯一索引与回放机制沿用。
- 边界：保障同一页面挂载生命周期内的重试，整个页面刷新后的持久恢复不在本次最小范围。
