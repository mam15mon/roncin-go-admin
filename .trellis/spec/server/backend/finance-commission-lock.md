# 提成经济重算与已发事实保护

## 1. 适用范围

提成财务锁已退役。已确认或已支付提成不再作为普通费用编辑门禁，台账不再输出 `finance_locked`。
订单审核锁、生命周期与逐费账单占用分别约束费用写入；提成通过事务内经济重算保持正确。
本文约束普通费用增改删、账单建账/更新/取消、补审、核销/对冲反转、确认与支付链路。

## 2. 来源与经济计划

- 提成单来源为核销单或对冲单之一；二者 XOR，活跃来源唯一索引防重复。幂等指纹含来源。
- 统一复用 `ProjectFinanceBillLineSettlements` 与 `ProjectCommissionEconomics`；不能用整张账单核销总额直接推断某票收入。
- 普通费用 Add/Update/Remove/Bulk 写入后，在同一事务调用 `reconcileOrdinaryFeeEconomics`；来源类型为 `FEE_MUTATION`。
- 单张/批量建账、草稿账单更新与取消同样改变普通费用采用的本币经济依据，必须在账单头行全部写完后、同一事务提交前执行 `reconcileBillEconomics`。触发标识使用账单或批次 ID、动作及版本，系统调整复用 `ECONOMIC_REALLOCATION` 并关联原父提成核销/对冲来源，账单触发事实写审计；不得借用未变化的费用内容版本造成幂等键碰撞。
- 费用补审来源为 `FEE_CORRECTION`；历史 `LOCKED_FEE_SUPPLEMENT` 与经济重分配来源仍按审计事实读取。
- `commissionSystemSource` 识别 FEE_MUTATION/FEE_CORRECTION/ECONOMIC_REALLOCATION 及历史补录来源。系统调整不能从普通确认/取消入口修改。
- 事务先展开关联父提成订单的传递闭包，按主键一次锁定联合集合，锁内复核集合未变化；变化返回 409，禁止中途反序补锁。

## 3. 已发与调整不变量

- PAID 是终态，不改状态、不删记录。原员工、岗位、比例、模式及实付证据保持。
- 重算理论目标后，只调整系统净额。未执行系统调整专用取消后净重建；已执行事实保留，并生成补付或追回差额。
- 普通费用经济重算保留手工奖励/处罚，不能把人工调整吞入系统目标。
- 原来源整体反转使用一份经济计划：未付父单取消；已付父单保留，按父单及已执行调整的实际支付敞口重建追回，不能叠加旧反转助手重复冲减。
- 未支付父单发放金额为原提成加所有 CONFIRMED 调整的有符号净额；锁内复核预览，父单及调整同时落支付事实，调整记 OFFSET_WITH_PARENT，不另付一次。零净额可抵扣结清，负净额拒绝。
- 已支付父单的后续调整独立执行。累计权益净额与历史实付是两个概念，不能互相替代。
- 取消 CONFIRMED 手工增提也会减少有效权益，须在既有 Order→父单→调整锁内复用
  `ensureCommissionAdjustmentBalances`，排除当前调整后分别检查订单行与父单净额。
  任一为负返回 `ErrCommissionAdjustmentExceeds`，状态、版本和审计整体不变；其他订单行
  的正余额不能替本行兜底。确认冲减使用同一计算口径并扣除本次冲减金额，DRAFT 不计余额。
  原始提成、已付事实、系统来源取消门禁与旧版本检查保持各自语义。

## 4. 并发与资格

- 创建/确认/月度批准需要员工快照时，先以 NO KEY UPDATE 锁 User，再锁申请头及排序后的订单；月度提交/重提先 User → Membership。
- 逐笔确认、整批确认及月申使用完整来源订单集合；不能逐员工或逐来源追加反序订单锁。
- PENDING 费用更正继续阻止受影响提成确认/发放；普通费用审核状态不再作为这些动作的前置。
- 逐笔确认、批量确认与月申批准的未建账门禁只检查当前有效且需建账的费用（复用 `accountingOrderFeePredicate`）；已被更正的原费及无需建账的冲回历史行不参与。不能要求用户给历史行建账来解除阻断；当前普通未建账费仍必须拦截。
- 确认/支付/来源反转与费用调整竞争必须在共同权威锁内重验，失败零部分写入。

## 5. 数值与验证

- **不含税口径（V4）**：提成基数是不含税已实现利润。应收、应付、来源额度的本位币金额一律经 `biz.CommissionNetBase(base, net, total)`，即 `base × net ÷ total`（`RoundBank(8)`，总额为零返回 0）折算，沿用该费用或账单行自身汇率快照，不另取汇率、不取发票或账单税额。已建账且非更正、非补录取**账单行**自身的 net/total，其余取费用自身 net/total；`REALIZED_REVENUE` 依据同样走不含税。禁止在新取值点直接使用含税 `BaseCurrencyAmount` 参与提成计算。
- **计算标识与上线前取舍**：当前 `CommissionCalculationVersion = CUSTOMER_REALIZED_PROFIT_V4`。用户明确尚未上线，旧开发计算版本不作为兼容目标，不并存双公式，也不要求保留旧开发 DRAFT 的旧公式金额。`refreshApplicationLines` 在驳回申请重提时可通过 `refreshCommissionSnapshot` 按当前公式刷新草稿及申请金额；不为此增加旧版本阻断。现有 `CommissionEconomicTarget`、费用更正与 `CommissionSupplementCostSensitivity` 仍会拒绝非当前版本，这是现状而非所有路径统一的冻结契约。正常并发版本、事务、实际支付事实及审计要求继续有效；不自动清库。
- **指纹**：费用与账单行来源指纹含 `net_amount`（账单行另含 `total_amount`），税率或未税额变化会被来源变化检测发现。
- 样例：应收含税 10,600/未税 10,000，应付含税 6,360/未税 6,000，核销占 40%，比例 10%：已实现利润 1,600.00，提成 160.00。回归见 `biz/finance_commission_net_base_test.go` 与真实库 `TestCommissionNetBasePostgres`（账单行与费用行税率不同、已建账与未建账并存）。
- 展示字段 `CommissionFeeDetail.BaseCurrencyAmount` 等仍为含税值，仅供展示，不参与计算；页面与导出须标清含税/不含税。

- 账单例：收入1000、本币成本费用快照600/账单快照700，10%提成确认30；取消成本账单后目标40，保留原30并生成+10，支付预览净额40。重新建账与更改账单日也必须产生正确净差额；事务失败不得留下账单或提成部分变更。
- 原实际发放 60，未锁订单新增成本后目标 50，修改成本后目标 40，删除新增成本后目标回到 60：原支付 60 不变，系统差额分别为 -10、-20、0。
- 连续补审：已实际追回 100 后理论目标恢复 90，保留追回事实，另生 +90 补付；未执行的旧 -100 则取消并净重建为 -10。
- 真实 PostgreSQL 必须覆盖普通新增/修改/删除、部分来源反转、重建账重核销、连续更正、手工调整并存、已确认/已付父单及已执行调整。验证实际金额与失败零写入，SKIP 不算验收。
- 不得再新增“提成存在则冻结整票费用”或净额释放费用锁的并行状态机。
- 手工调整回归：原提成 100，确认增提 100，再确认冲减 150，此时取消增提必须拒绝；
  撤销依赖的手工冲减后可正常取消增提。同时覆盖安全取消、旧版本、系统来源不可取消、
  父单总额仍正而目标订单行会负的情形。错误是只在冲减确认时检查非负，取消增提不检查。

## 6. 内嵌调整的公司投影

- `commissionRepo.ListScoped`、`ExportBatchScoped`、`GetScoped`、`GetByKey` 对外返回 `FinanceCommission.Adjustments` 时，必须在 `WithAdjustments` 中预载 `WithOrganization`；父提成的 `WithOrganization` 不会预载调整的组织边。映射复用 `commissionAdjustmentToBiz`，每条调整的 `organizationId/organizationName` 来自该记录的真实组织。
- 独立调整列表/详情与内嵌调整必须返回相同公司名称；确认/取消弹窗消费调整记录名称，不用当前工作台或父单名称兜底。纯能力计算的 `commissionAdjustmentParentProjection` 不输出名称，无需额外预载。
- 不改变父单或调整的组织范围谓词：范围外详情保持不可见，列表/导出过滤，幂等键读取按目标公司限定。
- 回归 `TestCommissionAdjustmentOrganizationProjectionPostgres` 必须用正式迁移的 PostgreSQL 验证四个父提成读取入口、独立调整读取及其他公司拒绝；前端通过真实页面动作构造确认/取消弹窗，核对公司名和 `expectedVersion`、取消原因透传。只测试独立调整列表无法覆盖内嵌投影缺失。
- 错误：`WithOrganization().WithAdjustments()` 后认为嵌套公司边已经加载；正确：`WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) { q.WithOrganization().Order(adjustment.ByCreatedAt()) })`。
