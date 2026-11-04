# 汇率组独立复审

## 范围与结论

基线 `a847e193`，审查 R4 / R5 / R8。未发现未处理的阻断问题，无需追加产品代码修复。其他账单、提成与 Ent 生成变更不在本次子审查结论内。

- R4：流水先归一币种，再使用服务端单据组织解析出的公司本币检查显式手工值；同币种非 1 返回 `EXCHANGE_RATE_SAME_CURRENCY_INVALID`，在仓储 Create 前停止。费用 Add / Update / 更正 ADD、REPLACE 共用 `resolveExchangeRate`，输入规范化及权限校验继续生效；未改汇率的费用更新保留既有快照，流水既有幂等重放不重复解析实时汇率。未硬编码 CNY，CNY / CAD 和合法外币路径均有断言。当前批量费用更新不接收汇率字段，未产生旁路。
- R5：`FinanceInvoiceUsecase.Issue` 从仓储读取发票方向，按 `invoice.Direction` 解析 AR / AP；单据组织、发票日、本币一致性及最终版本校验仍沿用既有流程。100 USD 的 AR 720 / AP 700 和缺价零写入均验证。
- R8：流水表单使用实际 `access.canUpdateExchangeRates`；服务端 `CreateCashflow` 仍独立要求 `FinanceExchangeRateUpdate` 及目标组织写范围。新增权限矩阵通过真实 `getAccess` 计算，验证仅费用覆盖权限无入口。
- `exchange-rate-single-rate.md` 已同步同币种手工恒等、发票方向、流水权限和验证要点。

## 独立验证

- `GOCACHE=/tmp/finance-review-go-cache go -C server test ./internal/biz -run '(Cashflow|OrderFee|FeeCorrectionRejectsInvalidIdentity|Invoice|ExchangeRate)' -count=1`：通过。
- `pnpm --dir web exec vitest run src/pages/finance/cashflows/cashflow-rate-permission.test.tsx src/pages/finance/cashflows/cashflow-save.test.tsx src/pages/finance/cashflows/cashflow-query-state.test.tsx`：3 文件、5 用例通过。
- 修改前端两文件 Biome：通过；`pnpm --dir web tsc`：通过。
- 汇率组 Go 文件 gofmt 检查、`go -C server vet ./internal/biz`、`git diff --check`：通过。

本组是 Biz 规则与前端入口权限修复，无数据库写实现变更，未运行数据库集成或漏洞扫描；整体全范围验收由父任务负责。
