# 收紧汇率精度口径至四位小数

## Goal

业务口径汇率固定 4 位小数（源头为中行牌价等外部牌价，仅发布约 4 位有效小数）。
当前前后端的汇率输入校验与业务校验统一放开到 8 位小数，存在虚假精度（后几位
无真实数据支撑）与误输入风险。本任务将**人工输入与校验口径**收紧到 4 位小数，
统一展示格式，并明确同步/交叉推导链路的精度策略。

## 背景（本次摸底结论）

- 业务确认：汇率实际数据就是 4 位小数；允许 8 位不会更准，只会引入虚假精度。
- 存储层 `numeric(18,8)` **不动**：容量零成本，契约为字符串透传，改列定义需要
  迁移且无业务收益。
- 金额类字段（`cashflows/index.tsx` 的 8 位金额正则）与汇率无关，不在范围内。

## 现状（8 位口径分布）

前端输入：

- `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx:36`
  （汇率主数据手工维护 + 校验）
- `web/src/utils/decimal.ts:4` `exchangeRatePattern`（订单费用表单
  `FeeFormModal` 的汇率 override 输入）

前端展示：

- `web/src/pages/finance/fees/components/feeLedgerColumns.tsx:69` `formatRate`
  （`toFixed(4)` + 去零，对 ≤4 位数据与 `trimDecimal` 等效）
- `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx`
  （牌价同步预览的差异展示，内部变精度）

后端校验（`server/internal/biz/exchange_rate.go:32` `exchangeRateValuePattern`
允许 ≤8 位）：

- 汇率主数据 upsert（`exchange_rate.go:549,554`）
- 牌价同步落库（`exchange_rate.go:508,513`）
- 订单费用汇率 override（`order_fee.go:877`）
- 现金流汇率 override（`finance_cashflow.go:156`）

后端交叉推导：

- `exchange_rate.go:514,555`：`(AR+AP)/2` 等推导用 `RoundBank(8)`，推导出的
  基准价是 8 位精度。

## Requirements

1. 前端两处汇率输入正则收紧为 `(\.[0-9]{1,4})?`（主数据维护、费用汇率
   override），输入 5-8 位小数被表单校验拒绝，错误提示明确"最多 4 位小数"。
2. 后端 `exchangeRateValuePattern` 同步收紧为 4 位，覆盖上述 4 个校验点；
   错误映射为现有参数错误语义，中文提示与前端一致。
3. 汇率展示口径统一为 `trimDecimal`（去尾零变长小数，≤4 位）：`formatRate`
   改为基于 `trimDecimal` 或等效实现，行为对 4 位内数据不变。
4. 牌价同步（BOC_SYNC / Excel 导入）来源数据若出现 >4 位小数：默认按
   "拒绝并提示"处理；若实现中发现源头确实会给出 >4 位有效数据，停下来向用户
   确认截断/拒绝策略，不得静默截断。

## 决策记录（用户已拍板，2026-09-24）

- **交叉推导精度**：全链路按 4 位收敛。人工输入、同步落库校验、内部推导
  （`(AR+AP)/2` 兜底、交叉汇率计算）统一 `RoundBank(4)`，保证推导结果能通过
  4 位校验，不存在"校验 4 位、推导 8 位"的双口径。项目尚未上线、无历史数据
  契约，不涉及存量折算口径迁移。
- 存储层 `numeric(18,8)` 不动（容量零成本，字符串透传）。

## 非目标

- 不修改存储层列定义（`numeric(18,8)` 保留），不做数据迁移。
- 不修改金额类字段的小数位口径（费用金额 8 位存储、两位展示不变）。
- 不处理重量/体积等数量类字段的展示位数。

## Acceptance Criteria

- [x] 汇率主数据面板输入 4 位小数通过，5 位被拒绝且提示"最多 4 位小数"。
- [x] 订单费用表单汇率 override 同上。
- [x] 后端汇率 upsert、同步落库、费用/现金流 override 对 >4 位小数返回参数
      错误（中文提示），≤4 位正常写入。
- [x] 牌价同步链路对 >4 位来源数据的行为符合 Requirement 4 的结论
      （BOC 源头仅 4 位，收紧后校验拒绝无实际影响）。
- [x] 汇率展示各入口（费用台账等）为去尾零变长小数，≤4 位。
- [x] 交叉推导精度策略按开放问题的用户决策落实并在变更说明中记录
      （决策：全链路 4 位，`RoundBank(4)`，含交叉盘与中间价推导、账单固化）。
- [x] 相关前后端定向测试通过（含 4/5 位边界用例）。

## Notes

- 轻量任务，PRD-only；若开放问题的决策导致折算金额口径变化，升级为
  complex 任务补 `design.md`。
