# 实施与定向验证

## 改动边界

- `server/internal/data/finance_commission_ledger.go`：仅在 ListScoped、ExportBatchScoped、GetByKey、GetScoped 的现有 WithAdjustments 中补 WithOrganization，以调整记录关联的真实公司输出名称；沿用转换器与既有服务 DTO。
- `server/internal/data/finance_commission_organization_integration_test.go`：正式迁移、随机 Schema、真实用例创建确认父提成并生成两条调整（一条草稿、一条取消），验证详情、幂等读取、列表、导出和独立调整公司名一致，范围外读取不可见，状态、版本和金额保持原值。
- `web/src/pages/finance/commissions/index.test.tsx`：补真实页面确认/取消回调与真实 App 确认框回归。仅替换明细排版，未 mock makeVersionActions 或弹窗构造；公司候选故意使用天津名称，内嵌记录使用北京公司完整名称，断言没有候选名称替代、没有缺失公司兜底，保留 expectedVersion 与必填取消原因透传。
- 另一个 `commissionAdjustmentParentProjection` 中 WithAdjustments 仅用于能力计算，不输出这些调整的名称，因此保持原样，不增加无用预载。
- 未改前端生产逻辑、金额计算、状态机、权限、契约、Schema 或生成物；未操作 Git，未修改开发业务数据。

## 验证结果

- `node /tmp/roncin-sea-operation-blockers/run-check.mjs go -C server test ./internal/data -run '^TestCommissionAdjustmentOrganizationProjectionPostgres$' -count=1 -v`
  - PASS，data 2.470 秒，6 个子场景全部通过，无 SKIP；日志 `/tmp/commission-company-go.log`。
  - 上述既有本地辅助脚本只注入专用测试库 `RONCIN_INTEGRATION_DATABASE_SOURCE`，不把开发 `DATABASE_SOURCE` 传给测试，不输出凭据。测试助手应用正式 SQL 迁移并隔离随机 Schema。
- `pnpm --dir web exec vitest run src/pages/finance/commissions/index.test.tsx`
  - 1 文件、10 测试 PASS，4.52 秒；日志 `/tmp/commission-company-web.log`。
- `pnpm --dir web tsc`：PASS；日志 `/tmp/commission-company-tsc.log`。
- `go -C server vet ./internal/data`：PASS；日志 `/tmp/commission-company-vet.log`。
- `pnpm --dir web exec biome check src/pages/finance/commissions/index.test.tsx`：PASS。
- gofmt、`git diff --check`：PASS。

## 首轮验证修正

- 真实用例创建调整只允许已确认父提成；测试夹具补真实 Confirm 后通过，不绕开状态约束。
- 测试 App 默认按钮文案为 OK；确认按钮查询与真实文案对齐。测试关闭 motion，避免 happy-dom 中退出动画阻塞闭框断言，没有增加帧等待。

## 交付边界

现有确认/取消弹窗只展示公司、调整编号及动作解释，没有直接展示具体金额；本次保持生产交互，金额不变由真实库断言。当前开发环境只读验收由主会话记录在 `research/live-acceptance.md`，本子代理没有重放现场财务操作或恢复订单。
