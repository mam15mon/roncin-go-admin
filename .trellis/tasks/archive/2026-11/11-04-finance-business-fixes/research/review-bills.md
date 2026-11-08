# 账单与对冲独立复审

## 范围与结论

基线 `a847e193`，覆盖 R2/R3/R9 的账单仓储、订单闭包锁、经济调整来源、对冲确认、两份 Ent Schema 与四份生成物，以及新增真实 PostgreSQL 回归。独立审查未发现未处理阻断问题，未追加产品代码修正。

- 单张/批量建账、更新和取消在账单头行全部写入后、事务提交前重算普通费用经济依据。批量完成后统一按订单重算，触发动作/账单或批次/版本进入幂等键，避免复用未变化的费用版本。
- 原 CONFIRMED/PAID 计提与已执行调整保留；系统净差额采用现有经济调整机制，人工奖励不被净重建吞并。账单触发调整关联原核销/对冲来源满足正式 CHECK，并以审计保留触发账单。
- 建账入口及已有账单读取/写入入口都先展开父提成的传递订单闭包，一次按 ID 排序锁定后重验；账单来源集合另行重验，不把闭包多出的订单误判为来源漂移。沿用订单→账单/费用→提成父单及调整锁序。
- 对冲确认先按锁内最新账单复核状态、余额、方向及范围，再刷新分摊和单头本币/汇差；同事务激活分摊。已确认/重复确认仍拒绝，失败审计回滚全部快照和生效标记。
- 单张建账继续核对费用原币金额、税率、币种、方向、归属与有效性，移除不成立的跨日期本币等式；账单头行金额由服务端账单日汇率构建。
- Ent 仅移除四个金额字段的 Immutable；列形、索引和 CHECK 不变，无 SQL/API 变更。生成差异限于对应 create/update setter 与 upsert 行为；依赖无变化。

## 独立验证

使用本次一次性 PostgreSQL 实例、正式 SQL 迁移和每个测试独立 Schema，未访问开发库。

- 新增 `TestFinanceBillEconomicReconcilePostgres`、`TestFinanceBillCancelSerializesCommissionPaymentPostgres`、`TestFinanceNettingRefreshDraftRatesPostgres` 实际 PASS。覆盖单张/批量×确认/已付、净额变化、幂等、审计失败回滚、旧支付预览竞争、800/700/-100 与重复确认不变。
- 相邻 `TestFinanceBillCreateSharedTransactionPostgres`、`TestFinanceBillBatchCreatePostgres`、`TestFinanceBillBatchNettingCreatePostgres`、`TestFinanceNettingTransactionPostgres`、`TestFinanceEconomicOrderSetExpansionPostgres`、`TestCommissionBillLockOrderConcurrentPostgres`、`TestFinanceNettingPersistsBothBaseAmountsAndGainLoss` 实际 PASS。
- 上述独立测试共 19.668 秒，日志 `/tmp/finance-bill-independent-review.log`，无 FAIL/SKIP。
- `go -C server vet ./internal/data`、本组修改 Go 文件 `gofmt -l` 和 `git diff --check` 通过；测试编译同时完成 Go 类型检查。

完整门禁与其他分组 review 由父任务记录；本记录不替代整体验收。实际漏洞扫描按用户要求未运行。
