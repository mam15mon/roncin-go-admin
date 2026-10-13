# 费用更正核心实施

## 已实现

- 统一费用经济合同：`INCREASE/DECREASE` 保留原应收/应付属性；`EconomicAmount` 给出有符号经济金额，`SettlementDirection` 给出实际资金方向，`IsCurrentFee` 取当前有效费用，`RequiresAccounting` 保留应建账的冲回行。
- 更正申请支持 ADD / REPLACE / VOID。替换/作废固化原费用版本与快照；新费用在申请时预分配 ID 并保存来源链，预览、批准与提成重算消费同一 ID。VOID 不产生零元替代费用。
- 提交、更正预览、批准、退回、撤回、查询与分页 RPC；审批复用实时费用审核授权，提交按订单业务的费用 create/update 权限，组织与授权父锁稳定后再执行业务。拥有自己申请的读取/撤回资格仍要求实时有效身份；跨订单同幂等键不会返回别票申请。
- 预览令牌绑定申请版本、批准基线、当前及预计费用/利润、财务事实与提成计划。批准在共享事务内写费用冲回/替代、审核事件、新批准快照、提成调整、消息与审计。批准同版本同令牌重放不重复写入。
- 财务事实以相关账单锁下的核销、对冲与票据哈希绑定，预览不输出合并账单中其他订单的财务明细。
- 申请与已有补录互斥；结案/自动锁检查待审更正；业务锁定及 CLOSED 允许受控更正。来源/衍生费用阻止普通修改、删除与拆单转移；已更正补录不能从旧入口撤销删除。
- 常规费用与费用审核利润只展示当前经济有效行；财务费用台账保留需建账的 DECREASE，独立展示 economicEffect/correctionRequestId/hasPendingFeeCorrection，清算方向与业务方向分离。
- 更正历史保存批准时利润与提成影响快照，财务状态根据本次费用的真实账务/清偿及票据范围动态投影。支持 ADJUSTMENT_UNBILLED、INVOICE_PENDING、SETTLEMENT_PENDING、INVOICE_REISSUE_PENDING、COMPLETED；重开发票须 ISSUED 才算完成。

## Schema、生成与协作

- Ent 源 Schema 增加更正申请与双向边、OrderFee 来源/经济方向/建账责任、审核事件更正来源、账单用途、定向对冲来源、提成调整来源及计算快照、发票所覆盖费用 ID 集合。
- 正式迁移：`server/migrations/20261013120000_order_fee_correction.sql`。Schema 差异通过独立隔离 PostgreSQL Schema 加正式旧迁移后执行 Ent Diff 产生，审阅仅纳入本任务字段、索引、CHECK 与 NO ACTION 历史引用。发票 source_fee_ids 采用正式一次回填，再加 NOT NULL；无运行时历史回退。
- 已串行执行 Ent、API、HTTP 权限绑定、Wire、OpenAPI 与 web-client/Proto 常量生成；无手改生成物。临时 Ent Diff 命令目录已删除。
- 财务仓储和提成算法由同伴负责，核心通过强制注入的 `FeeCorrectionCommissionRepo.PreviewCorrectionImpact/ReconcileCorrectionImpact` 协作，不提供空实现兜底。
- 共同受控费用写入口预读提成父单的完整订单集合，按主键排序一次锁定；拿锁后再次校验集合，集合变化返回冲突，避免中途补锁其他订单。
- 创建待审提醒与审批结果提醒使用不同稳定任务键，解决同一申请自审时 outbox 幂等冲突造成整个批准失败的问题。

## 验证

专用真实 PostgreSQL 通过主会话提供的 runner 注入连接，未读取或输出凭据，夹具按隔离 Schema 执行正式迁移。

已通过：

1. `TestFeeCorrectionEconomicOperationsPostgres`：ADD / REPLACE / VOID；完整历史有符号金额与当前叶子汇总一致；原金额不变；未建账原费的冲回不进入财务；幂等审批；原费用不能更正第二次；批准快照可用于新建账门禁。
2. `TestFeeCorrectionRevenueAndPartyPostgres`：漏录应收、应收优惠、应付误录改应收且换结算对象。
3. `TestFeeCorrectionPendingAndBusinessClosurePostgres`：待审与补录互斥、无未建账费用仍阻止结案/自动锁；无回款也能业务结案；CLOSED 加业务锁仍可受控更正；已建账冲回进入财务待办且保留原业务方向。
4. `TestFeeCorrectionPermissionsVersionsAndConcurrencyPostgres`：无权限、停用账号、旧来源版本、他人申请、预览失效、跨订单幂等键拒绝；真正两个 goroutine 同屏障并发提交同订单，只准一个成功。
5. `TestFeeCorrectionApprovalRollbackPostgres`：在费用已写入后注入提成 Reconcile 失败，验证原费替代标记/版本、生成费用、订单内容/审核版本/批准事件、申请状态/版本、审核事件、提成调整、审计与 outbox 全回滚；随后正常审批成功。
6. `TestFeeCorrectionEconomicAndAccountingViews`：经济符号、结算方向及经济/建账视图分离。
7. 完整 `go -C server test ./internal/service ./internal/biz`；台账数据库分页/汇总定向测试；`git diff --check`。

## 审查重点

- 联合财务与提成真实库测试由各负责同伴维护，最终整体门禁由主会话统一执行。
- 本期未增加人工费用完整性清单，未将回款混入业务结案，也未开放普通解锁改费。
