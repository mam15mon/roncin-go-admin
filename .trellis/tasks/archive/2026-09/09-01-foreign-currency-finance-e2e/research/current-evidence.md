# 当前证据与测试缺口

## 已有证据

- `scripts/acceptance-finance-bill-batch.mjs` 已覆盖真实 HTTP 的订单、费用、账单、开票、
  资金流水、核销、提成及多组并发/状态机场景。
- `scripts/acceptance-finance-payable.mjs` 已覆盖基础应付链路，Playwright 已覆盖当前财务
  页面冒烟路径。
- 提成 PostgreSQL 事务集成测试已证明创建成功、汇率失败、保存失败和审计失败的真实
  提交/回滚语义。
- 汇率、账单、资金流水、核销和提成均已有领域或数据层局部测试。

## 尚不能证明的部分

- 现有应收与应付脚本显式提交 `exchangeRate: '1'`，费用也使用组织本位币，因此它们
  不能证明系统汇率从订单一直传递到提成。
- 现有长链没有从汇率配置 API 创建一组可识别 setting ID，再沿同一业务 ID 验证费用、
  账单、开票、收款、核销和提成快照。
- 现有应收脚本读取数据库首个客户，fresh bootstrap 数据库没有客户，故当前
  `acceptance:finance` 不能直接在空库复现。
- PostgreSQL 集成测试在缺少 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时会 SKIP；默认
  `go test ./...` 成功不等于这些用例真实执行。
- 当前 CNY 长链未断言阶段 1 新增的 CNY 提成快照字段。

## 本任务采用的最小闭环

- 临时库 A 保持组织本位币 CNY，只修复验收夹具依赖并固化现有基线。
- 临时库 B 在任何业务数据写入前设置本位币 USD，使用 EUR 业务币，既验证五类
  EUR→USD 系统汇率，又通过 CNY→USD WRITE_OFF 汇率验证提成 CNY 倒数派生。
- 两库串行运行并分别销毁，避免在已有财务数据的组织上变更本位币，也避免外币场景
  污染既有 CNY 验收。

## 后续仍独立保留的缺口

- 提成 `MarkPaid` 行为链；
- 应付外币连续链；
- 低权限与跨组织真实 HTTP 负向场景；
- CI 中自动提供临时 PostgreSQL 并强制集成测试不得 SKIP。
