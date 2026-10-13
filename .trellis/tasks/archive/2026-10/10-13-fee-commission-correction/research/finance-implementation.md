# 费用更正财务闭环实施记录

## 已实现

- 正金额保持经济方向；建账采用 `SettlementDirection`，账单用途 NORMAL/CORRECTION_REVERSAL。单笔及批量分组将更正步骤隔离，已替代旧费不能重建，旧账单不能取消或改经济快照；原草稿仍可确认。候选只展示需建账的正常/冲回，待审补录与更正分别投影。
- `biz/finance_settlement_projection.go` 是账单行余额与提成贡献的共同内核；data读取完整有效/已反转核销与对冲事件。核销按created_at，对冲按confirmed_at，反转按reversed_at；稳定重放，撤销释放原分摊，不重分后来的历史。`FinanceSettlementDistribute` 八位最大余数分配，输入ID固定顺序作为同余数归属，金额严格守恒且不超行余额。10行各1tick余额核6tick证明逐行银行家舍入可使前9行超额，必须共用最大余数。
- 更正对冲端点保持既有bill+expectedVersion输入，服务端从反向账单唯一推导更正request与source_fee_id。只允许同一步骤的原/冲回/替代，不吞合并账单其他行。替代清偿只能转移真实已收/已付覆盖；前一步替代的合法内部覆盖可以继承，当前原债抵销排除。未收原债不能先抵替代。逆向退款及来源反转后校验覆盖余额，后续依赖未反转前不能撤掉已消费的覆盖。
- 原混合账单中被替代费用未结时，普通核销/对冲返回409先内部清偿；清完后只给剩余正常行分摊。真实资金流水登记仍独立。资金创建/确认/反转先锁完整Order联合集合，再Cashflow/Source/Bill，写事实后同事务调用commission统一重算hook；删除旧独立反转冲减调用，避免双扣。
- 开票候选保留历史账额，新增可开票经济金额与有效正常行；反向账不可普通开票。InvoiceBill新增source_fee_ids固化实际费集合，原票整红后旧A+B账重开B不会挡新A；原票仍占原A则替代票拒绝。签发先Order后Invoice/Bill，并重验金额、费集合、聚合票行完整经济快照。批准不改原票状态。

Schema、正式迁移、Ent/API/Web生成统一由core完成，本代理只改settlement.proto源及财务service映射，无并发生成。

## 实际验证

1. `go -C server test ./internal/biz -run 'Test.*(FinanceBill|FinanceNetting|FinanceInvoice|FinanceSettlementProjection)' -count=1` PASS；纯投影4用例包含合并清偿、撤销后不重分、原未清拒收、tiny舍入守恒。
2. `python3 /tmp/roncin-correction-tests-afn2u0r2/run.py go -C server test -p 32 ./internal/data -run 'TestFinanceCorrection' -count=1 -v` PASS，4顶层+2子场景，9.787s，零SKIP。包括真实不同订单A1000/B200合账收400，只抵原A666.66666667，B再收133.33333333，退233.33333333全清；历史原额1200保持；未收先抵替代拒绝；原票整红后200+100重开；更正前草稿不能签发；1000收500→800→700连续继承。
3. 相邻发票终态4顶层/15子场景实际PostgreSQL PASS（状态机、释放、审计失败回滚、并发取消/红冲及账单取消），约8.6s。随后netting相邻旧夹具补真实fee/billline，`TestFinanceNettingTransactionPostgres` 2子场景PASS，2.588s（幂等并发、并发确认）。真实隔离库仅使用正式SQL迁移，没有重置开发库。
4. `go -C server vet ./internal/biz ./internal/data ./internal/service` PASS；`git diff --check` PASS。

## 交主会话检查

- 业务更正授权及更正预览不泄露其他订单内容由core/main验证；财务接口继续按既有组织grant范围校验整账。
- 新完整贡献要求真实账单行，旧netting并发fixture原只有账头，已补行；SQL mock新增purpose/sourcefee列已同步，财务数据层定向回归PASS。全biz/service包PASS；全data包剩core台账空列表新增pending查询SQL mock期望未同步，已交core处理，不属于财务金额算法失败。
- 最终全量门禁、独立Trellis检查、规范和Git提交由主会话统一；本代理没有提交、归档或变更任务状态。
