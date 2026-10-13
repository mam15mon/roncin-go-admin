# 费用更正与提成联动实现记录

日期：2026-10-13。实现依据为 `commission-design.md` 第 9 节及主会话确认：按实际已收金额重算，部分回款优惠示例结果为 25 元。本文记录实现与实际验证，不替代任务状态及最终全栈门禁结果。

## 已实现的业务闭环

- `biz/finance_commission_economics.go` 汇总当前有效经济费用，按经济根、结算对象及币种建桶。历史贡献消费财务唯一账单行结算投影；FIFO 占据当前有效应收额度，后续超收不重复计提。
- 只有来源发生时的应收增加费用能够提供收入；当前叶子也必须提供相同对象及币种的应收额度。方向、结算对象或币种更正不能把旧成本追回、旧客户收款当成新收入。
- 内部费用更正对冲不提供收入。真实退款分摊并扣减原来源贡献，退款反转恢复原分摊；来源反转有仍有效的退款依赖时拒绝，避免改变原退款归属。
- `data/fee_correction_commission_projection.go` 负责加载费用经济链、正常账单冻结本位币及统一财务投影。没有另造账单结算或资金归属算法。
- `data/fee_correction_commission.go` 实现更正预览、批准后锁内复核与系统净额重建。原规则、比例、计算版本、员工及原费用快照保留；未知或缺失原快照明确失败。
- 系统来源 `LOCKED_FEE_SUPPLEMENT`、`FEE_CORRECTION`、`ECONOMIC_REALLOCATION` 与人工调整分开。未执行系统调整取消并保留历史，已执行事实不改；根据累计实际系统净额创建本次增提或冲减。减少额仍受行及父单有效额度约束，并审计未覆盖超额。
- `data/fee_correction_commission_source.go` 统一处理核销创建/撤销、对冲确认/反转之后的重算。先锁完整订单集合，再按 UUID 锁父提成及调整。原来源整体反转时取消尚未执行的调整，已支付父单仅追回实际已付净额；不再并行调用旧反转冲减路径。
- 普通补录审批复用相同经济目标和系统净额算法，支持更正后的继续补录、连续补录及恢复增提。原补录冲减也纳入累计系统净额。
- 首次生成提成、候选项、订单提成估算及工作台消费相同收入额度，首次生成持久化费用内容版本与实现收入投影快照。
- 系统调整不允许从普通确认/取消入口人工改变。待审批费用更正同补录一样阻止执行提成；批准后按明确增减方向执行。

## 主要变更文件

新增 `biz/finance_commission_economics.go`，以及 `data/fee_correction_commission.go`、`fee_correction_commission_projection.go`、`fee_correction_commission_source.go`。已有 `finance_commission_calculation.go`、`finance_commission_write.go`、`finance_commission_mapping.go`、`finance_commission_payment.go`、`finance_commission_adjustment.go`、候选及摘要/工作台文件同步消费新口径。

补录适配修改 `biz/order_fee_supplement.go`、`biz/order_fee_supplement_usecase.go`、`data/order_fee_supplement_approval.go`、`data/order_fee_supplement_notifications.go` 及 `service/order_fee_supplement.go`。服务 DTO 与工作台新增只读来源和方向；契约与生成物由核心/财务实现代理集中生成。

原 `finance_verification.go` 底部的独立 `reconcileCommissionsForSourceReversal` 已删除；来源入口由财务实现代理接入统一 hook。财务投影与 `FinanceSettlementDistribute` 的实现归财务代理，本代理直接复用。

## 实际验证

真实 PostgreSQL 使用专用测试运行器注入独立测试配置，正式 SQL 迁移初始化隔离 Schema；未读出连接配置或秘密，未清空开发库。

已通过的新真实数据库测试：

- `TestFeeCorrectionCommissionPartialRevenuePostgres`：应收 1000、成本 400、实收 500，原提成 30；优惠后应收 800，目标 25，新冲减 5。
- `TestFeeCorrectionCommissionPaidNetRebuildPostgres`：原成本 0、原提成 100，错误补录成本 1000 再更正为 100；覆盖未支付父单、支付父单尚未追回、支付父单已独立追回、支付时已抵扣，以及人工冲减 80 并存。未执行系统历史取消后重建为 -10；已执行 -100 保留并新增 +90；人工 80 保留，最终仅保留系统 -10。
- `TestFeeCorrectionCommissionFirstGenerationPostgres`：更正在首次计提前批准，新父提成按当前费用生成并持久化版本/收入证据；后续普通补录继续按经济总额重算，系统确认及取消入口拒绝人工动作。

已通过的邻接真实 PostgreSQL 回归包括：

- `TestCommissionLifecycleAndDeduplicationPostgres` 全部 8 场景：创建、取消重建、确认及支付、核销反转的草稿/已确认/已支付处理、并发唯一性和数据库约束映射。
- `TestCommissionPaymentSourceEvidencePostgres` 两场景：取消待执行增提并仅追回实际已付；缺实际付款证据拒绝反转。
- 连续补录、收入计提口径补录及取消、已确认冲减支付、月度多订单父单锁顺序、授权与自审防护、撤回/批准竞态、回滚、Schema 与 FK/唯一索引验证。
- `TestOrderClosurePendingSupplementClosedWorkflowPostgres`：已结案订单仍可受控补录和审批，普通费用修改仍受冻结，待审费用与未建账费用分别阻止后续结案。共享提成测试夹具已改为真实账单行及核销来源。
- 订单提成摘要隐私及跨仓储事务原子性相关回归。

纯业务测试覆盖 25/80/0 目标、FIFO 连续封顶、多来源退款及退款反转、来源反转依赖、对象/币种更换、五类系统净重建、未知计算版本与十个微量来源退款守恒。`go -C server test -p 32 ./internal/biz ./internal/data ./internal/service` 已通过；该无配置命令中的数据库测试会跳过，因此不把它当作真实数据库验证证据。

## 独立检查交接

独立检查代理继续负责两个补充验证：本位币来源额度的最大余数分配，避免大量小额来源四舍五入造成最后负尾差；真实退款/退款反转和既有确认或支付父提成的完整联动。它直接拥有该算法修复与对应测试，不重复上述业务例子。

最终全栈门禁、Schema/Proto 生成物一致性、正式迁移审阅及 Git 提交由主会话统一处理。本代理不修改任务状态，不单独提交共享工作区。
