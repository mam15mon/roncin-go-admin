# 五项业务审查的代码核实记录

日期：2026-10-13（北京时间）。基线：`39235097da72e177caac692d16b51480403d25d6`，本地 main，开始时工作区干净。远端 `git ls-remote origin refs/heads/main` 因 GitHub SSH 22 端口 Connection refused 失败。本记录只证明所检查本地代码行为，不证明远端或数据库已复现。

## R1：自动锁单归属与币种（确认缺陷，P1）

- `server/internal/data/order_auto_lock.go:581`：筛选本票的有效应收账单行，含 DRAFT/CONFIRMED。
- 同文件 `:598`、`:605`：lineSum 直接加 TotalAmount，无币种桶。
- 同文件 `:611`、`:622`：以 BillID 查询有效核销和对冲分摊，不含本票归属条件；`:630`–`:644` 合计整单结清金额。
- 同文件 `:653`–`:657`：lineSum - settledSum == 0 即合格。因此 100+200 合账收100，A 的资格判断可能为真；不共享主单的订单没有共享组保护。共享组也不能普遍兜底：100+100 合账收100，两票资格同时可能通过。
- `server/internal/data/ent/schema/finance_verification_allocation.go:17` 与 `finance_netting_allocation.go:26`：分摊字段归属 bill_id，无 order_id/order_fee_id。台账 `settlement.go:185`–`:225` 展示关联账单整体进度，并非持久化费用级结清分配。
- `server/internal/data/order_auto_lock_integration_test.go:896`：现有测试覆盖超出单票金额时不锁，`:926` 明确规定整账单全收仍不锁。这是既有保守策略，不能描述为新出现的回归；该策略未防住差额恰好为零。

建议先逐张确认相关应收账单结清，保留有效结清事实及其它资格条件，避免跨币种或跨账单轧差。整单全额收齐可以证明其行的应收已覆盖；部分收款若要区分票别则需要真正的分配契约。两者不要混淆。

## R2：更正范围（确认能力缺口，P1）

- `server/internal/biz/order_fee_supplement.go:109`–`:118`：仅正数 PAYABLE，无法表达应收或调减。
- `server/internal/data/order_fee_review.go:226` / `:390`–`:401`：APPROVED 的撤销资格要求不存在有效账单，含草稿账单；普通费用冻结契约见 `.trellis/spec/server/backend/order-fee-review.md`。
- 现有 API、biz/data 路径未找到通用费用更正申请；原审查关于取消财务链路后改费的方向合理，但不能保证所有订单都能简单退回，业务锁、生命周期、提成保护还会约束。

这是现有 MVP 未覆盖异常业务，而非应当删除保护。设计更正申请需要把经济调整与票据/资金执行区分，不能批准一笔更正就任意改写原核销或发票。

## R3：补录纠错（确认联动缺口，P1）

- `server/internal/data/order_fee_supplement_approval.go:249` 之后：按提成影响建议生成 DECREASE + CONFIRMED + LOCKED_FEE_SUPPLEMENT；并非每笔补录一定产生冲减。
- `server/internal/data/order_fee_supplement_cancel.go:129`–`:136`：只要存在 SourceFeeSupplementRequestID 匹配的调整便拒绝，甚至不是仅筛选 CONFIRMED。
- `server/internal/data/finance_commission_adjustment.go:338`–`:341`：系统来源冲减不能通过通用取消调整入口撤销。
- `server/internal/biz/finance_commission_lifecycle.go:128`–`:150`：已有 MANUAL INCREASE/DECREASE。因此「没有任何提成反向调整」不准确；缺的是成本更正、提成反向影响及原审批来源的统一闭环。

补录撤销只适用于无系统冲减、未建账等条件，不能把手工增提当成原错误成本已修复。

## R4：结案（确认门禁缺口，P1）

- `server/internal/biz/order_transition.go:151`：业务终点/终止、活动异常、未建账费用三类条件，无待审补录。
- `server/internal/data/order_write_lifecycle.go:142`–`:159`、`:174`–`:190`：只读资格与事务内复核均遗漏 PENDING 补录。
- `server/internal/data/order_fee_supplement_lock.go:112`–`:144`：只校验有效费用批准和锁事实，无结案检查；`order_fee_review.go:122`–`:140` 的快照验证同样无生命周期门禁。
- `server/internal/biz/order_fee_supplement_usecase.go:256` / `:443`：提交与批准使用该入口。原审查「先结案后批准补录，利润再变」有静态代码依据；此外先结案再创建补录也需要按最终规则控制。

不能把结案自动等同财务全清。推荐业务完结允许后续收付款，但改费先反结案；此为产品选择，尚待用户确认。

## R5：完整性声明（事实成立，增强建议）

- `server/internal/biz/order_fee_review.go:235`–`:236`：SUBMIT 要求 CompletenessConfirmed=true。
- 同文件 `:68`–`:80`：利润从已有费用 BaseCurrencyAmount 汇总，无法发现尚未录入的现实成本。

清单可记录应收核对、供应商成本核对、待回成本及估算依据、确认人与时间，帮助审批判断与追责；不应写成能证明费用绝不遗漏。是否允许待回成本时建账、估算是否生成正式费用及后续如何冲回是额外业务决定。

## 验证边界

本轮完成代码/Schema/既有测试静态交叉核查，没有修改产品代码、运行测试或操作数据库。数字算例是按现有代码推演。后续实现必须添加缺失触发场景与相邻路径验证，并在正式迁移的隔离 PostgreSQL 环境中记录实际 PASS。
