# 初步证据索引

## 已确认的测试层次

- 真实 PostgreSQL 分阶段测试：
  - `server/internal/data/order_transaction_integration_test.go`
  - `server/internal/data/finance_bill_transaction_integration_test.go`
  - `server/internal/data/finance_verification_transaction_integration_test.go`
  - `server/internal/data/finance_commission_transaction_integration_test.go`
- 真实 API 验收脚本：
  - `scripts/acceptance-finance-bill-batch.mjs`
  - `scripts/acceptance-finance-payable.mjs`
- 浏览器验收：`web/tests/e2e/finance-bill-batch.e2e.ts`。
- 组合入口：根 `package.json` 的 `acceptance:finance`。

## 初步观察，待正式审计复核

1. `acceptance-finance-bill-batch.mjs` 从订单 API 创建开始，继续创建并确认费用、批量
   建账单、开票、资金流水和核销；脚本还包含提成相关编号规则和后续逻辑，需要在正式
   审计中精确确认最终提成创建/状态断言及多币种覆盖范围。
2. `acceptance-finance-payable.mjs` 覆盖应付费用、应付账单、付款、方向错配拒绝与核销，
   终点是应付核销后的财务进度，不包含提成。
3. Playwright 会操作核销页面，但提成部分主要查看既有可解释提成并打开生成弹窗；需要
   区分“页面可见/可操作”与“浏览器实际创建提成”。
4. 分阶段 PostgreSQL 测试真实覆盖事务、并发和回滚，但核销及提成夹具存在直接创建
   上游账单、资金流水或核销数据的情况，不能自动累计为一条完整链路。
5. 这些 PostgreSQL 测试依赖 `RONCIN_INTEGRATION_DATABASE_SOURCE`，默认未配置会
   Skip；正式报告必须核对最近的显式运行记录。

以上仅是规划阶段的证据入口，不是最终覆盖结论。
