# 独立审查与自修记录

日期：2026-10-13。当前 task 的 PRD、最终 design 第7/8/9节、实施计划与 check.jsonl 所列规范为审查基准；没有创建重复任务、提交代码或操作已有开发数据库。

## 已修复问题

### 1. 本位币来源配额逐笔舍入导致负尾差

- 文件：`server/internal/biz/finance_commission_economics.go`。
- 原路径分别对每个来源 RoundBank(8)，最终来源吸收差额。应收原币10、本位币0.00000006、十笔来源各收1时，前九笔各分0.00000001，最后一笔变为-0.00000003。
- 改为先确定FIFO原币配额，再使用唯一 `FinanceSettlementDistribute` 最大余数算法分配本位币总额。原币/本位币量纲不同时，同乘正整数放大权重，比例和稳定顺序保持，满足分摊器容量条件。没有改变V3提成计算的Round(8)。
- 新增 `TestCommissionEconomicBaseQuotaRoundingConservation`，分别覆盖低汇率与高汇率，断言每来源非负、前六来源稳定取得余数、全部来源合计等于经济本位币基数。

### 2. 资金入口锁后未复核完整关联订单集合

- 文件：`server/internal/data/finance_settlement_projection.go`、`finance_verification.go`、`finance_netting.go`。
- 核销创建/反转、对冲读取上下文/锁账单/反转先预读关联父提成的订单，再等待Order锁；等待期间新跨票父单提交后，原集合可能失效，后续经济重算会触达未预锁的订单。
- 新增共用 `validateFinanceEconomicOrderSet`。拿齐预读Order锁后重新展开完整集合，逐项不符即409整体重试；不能在资金/提成父锁之后补锁订单。五个入口同步接入。

### 3. 关联父提成订单只展开一跳

- 文件：`server/internal/data/fee_correction_commission_source.go`。
- A属于AB父单，B又属于BC父单时，原展开仅返回A/B；后续重算B会触达BC父单却没有预锁C，不符合完整父单订单锁契约。
- 改为按批次展开父单/订单传递闭包，分别缓存已见订单与父单，最终主键排序。用于受控更正/补录以及资金入口的同一集合助手同步生效。
- 新增真实PG `TestFinanceEconomicOrderSetExpansionPostgres`：预读仅A，随后可见AB与BC父单，断言旧集合409、新集合包含A/B/C且重试成功。本用例验证集合变化和传递闭包；实际数据库锁等待另由下一用例验证。

## 新补真实PostgreSQL回归

均通过专用临时数据库运行器连接，使用隔离Schema与正式SQL迁移；没有Schema.Create或并发建表，没有SKIP。

- `TestFeeCorrectionCommissionRefundAndSourceReversePostgres`：CONFIRMED与PAID两子场景。原应收1000/成本400/实收1000产生提成60；优惠至800目标40；真实退款500后按净收500重算25；反退款恢复40。有有效退款时撤销原收款失败且核销状态、版本与提成零部分写入；撤销退款后撤原收款，未付父单与待执行调整取消，已付父单60历史保留且仅一笔60真实敞口追回。
- `TestFeeCorrectionApprovalSerializesCommissionPaymentPostgres`：审批持有Order锁暂停在提成重算，付款并发进入。通过pg_locks/pg_stat_activity确认付款实际等待本Schema的Order锁；审批提交后旧60付款预览拒绝，重新预览并实际净付10。
- `TestFinanceEconomicOrderSetExpansionPostgres`：上列关联集合变化/AB→BC传递闭包。

## 其他验证与结论

- `TestOrderFeeCorrectionListPaginationBounds`：7个边界，非法页码/容量在授权/仓储之前拒绝，1和200合法并原样传递。
- 修复后 Go biz/data/service 三包普通测试全部PASS（普通运行中的可选DB集成不计为真实库验收）；`go vet`三包PASS；`git diff --check` PASS。
- 新费用更正前端4文件Biome PASS；前端 `tsc` PASS。前端失败预览仍保留旧数据的问题已通知主会话，主会话补 `isError` 禁提交及回归。
- 中间组合真实PG 7顶层测试及其子场景PASS，18.389s；传递闭包最后变更后对应PG回归再次PASS，2.37s。最后完整门禁与组合真实PG由主会话统一执行。
- 没有遗留需要产品决定的审查问题。上述回归不代表穷举所有并发交错；具体已验证的锁等待、失败回滚及金额边界如上。
- 建议主会话同步规范：本位币配额同用最大余数；关联订单按父单传递闭包展开并锁后复核；finance-bill-currency的待审更正与受控费用批准汇率例外。已通知主会话统一维护，避免并行覆盖同一规范。
