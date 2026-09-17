# 设计：锁单后费用补录与提成冲减调整

## 1. 设计边界

本功能是订单普通费用写入口之外的一条受控追加命令，不改变现有业务锁和财务锁的普通门禁：

- 普通费用新增、修改、流转、删除继续受现有统一门禁保护；
- 新增“费用补录申请”聚合，只有审批通过才能创建锁后费用；
- 补录只新增正数应付成本；应收方向由后端拒绝，不修改或删除既有费用；
- 待处理冲减建议直接复用 FinanceCommissionAdjustment 的 DECREASE + DRAFT，不建设第二张建议表；
- 财务确认继续走 DRAFT → CONFIRMED，实际少发后继续走 CONFIRMED → PAID；
- 不建设员工负余额、月度发放批次、工资扣款、会计期间或总账。

现有契约继续作为真相源：

- 普通费用门禁位于 server/internal/data/order_fee.go；
- 业务锁、锁定代次和 SE 单证快照位于 server/internal/data/order_lock.go；
- 财务锁遵循 .trellis/spec/server/backend/finance-commission-lock.md；
- 直接解锁资格遵循 .trellis/spec/server/backend/order-lock-and-document-version.md；
- 调整实体和状态机位于 server/internal/data/ent/schema/finance_commission_adjustment.go 与 server/internal/data/finance_commission.go；
- 外币费用遵循 .trellis/spec/server/backend/exchange-rate-single-rate.md。

## 2. 数据模型

### 2.1 新增 OrderFeeSupplementRequest

补录申请保存审批所需的不可变费用快照：

- organization_id、order_id、lock_generation；
- idempotency_key；
- 应付方向、费用项、结算对象、计费单位、数量、单价、税率、币种、费用发生日期和备注；
- reason、requested_by、requested_at；
- status：PENDING、APPROVED、REJECTED、WITHDRAWN；
- version；
- decided_by、decided_at、decision_reason。

数据库约束：

- organization_id + idempotency_key 唯一；
- 费用快照创建后不可变；
- PENDING 只能进入一个终态；
- WITHDRAWN 只能由 requested_by 在 PENDING 状态携带 expectedVersion 触发；审批、驳回和撤回统一锁定申请行并比较版本，竞争失败的一方返回状态冲突；
- 发起时订单必须存在业务锁、财务锁或二者之一；普通未锁订单继续使用普通新增费用；
- 创建和审批均要求方向为 PAYABLE，锁内复核快照，不允许客户端把方向改成 RECEIVABLE；
- 审批时锁代次必须仍与申请一致，否则返回冲突并要求重新申请。

### 2.2 复用 OrderFee

审批通过后创建普通订单费用：

- 状态直接为 CONFIRMED，补录审批承担费用真实性确认；
- 幂等键由补录申请 ID 确定性派生；
- 增加可空、唯一的 supplement_request_id 关联；
- 金额、税额、本位币金额与汇率快照复用现有计算；
- 创建后进入普通账单、对账、开票、收付和利润链路。

普通费用仓储不增加 skipLock 或 force 参数。新增内部专用仓储命令，并在事务中验证补录申请，避免形成通用绕过入口。

### 2.3 复用 FinanceCommissionAdjustment

待处理冲减建议直接落为现有提成调整记录：

- direction = DECREASE；
- status = DRAFT；
- source_type 新增 LOCKED_FEE_SUPPLEMENT；
- source_fee_supplement_request_id 关联补录申请；
- commission_id、order_id、employee_id 继续使用现有字段；
- commission_id + order_id + source_type + source_fee_supplement_request_id 唯一。
- LOCKED_FEE_SUPPLEMENT 的 DRAFT 调整允许财务复用 CancelCommissionAdjustment 忽略，必填 cancellation_reason 并保留 cancelled_by/cancelled_at；页面文案为【忽略建议】，不删除记录。

页面文案与状态严格对应：

| 页面文案 | 调整状态 | 业务含义 |
| --- | --- | --- |
| 待处理冲减建议 | DRAFT | 系统已经计算，财务尚未确认 |
| 已确认冲减 | CONFIRMED | 财务确认以后应少发，但尚不代表实际扣回 |
| 已扣回 | PAID | 财务声明线下已经少发或完成扣回 |
| 已取消 | CANCELLED | 保留记录，但不再参与有效提成 |

## 3. 权限与审批

### 3.1 发起权限

- 发起接口使用目标订单的 fee.create 权限和组织数据范围；
- 财务提成管理权限不会自动获得费用补录权；
- 发起人可以是销售或其他被配置了对应费用权限的业务人员。

### 3.2 审批资格

- 审批时实时复用目标订单业务类型的有效 lock grant 与数据范围；
- 不复制角色代码，不新增“主管”字段或负责人名单；
- 任一合格人员均可审批；
- 发起人本人合格时允许自行审批；
- 审批不会修改订单 locked_at、lock_generation 或解锁状态。

### 3.3 财务权限

- 查看与确认冲减继续使用 system.finance.commission.read/manage；
- 前端只消费后端权限结果，不硬编码“财务角色”名称。

## 4. API 与页面

### 4.1 订单费用侧

订单费用契约新增：

- CreateOrderFeeSupplementRequest；
- ListOrderFeeSupplementRequests；
- ApproveOrderFeeSupplementRequest；
- RejectOrderFeeSupplementRequest；
- WithdrawOrderFeeSupplementRequest。

订单费用页面行为：

- 锁定后普通新增、编辑、删除继续禁用；
- 有 fee.create 权限时显示【补录费用】；
- 补录表单复用普通费用字段，额外要求补录原因；
- 页面展示申请状态、发起人、审批人和生成的费用；
- PENDING 且当前用户为发起人时显示【撤回申请】；撤回成功后状态为 WITHDRAWN，不再显示审批动作；
- 后端返回可审批能力，前端据此显示通过或驳回；
- 审批候选收到通知并跳转订单费用页；MVP 不新增全局审批中心。

### 4.2 提成侧

财务契约增加服务端分页的调整列表，支持状态、来源、员工、组织以及订单号或提成号关键字过滤。

提成页面增加【待处理冲减】视图，默认条件为：

- direction = DECREASE；
- status = DRAFT；
- source_type = LOCKED_FEE_SUPPLEMENT。

该视图不依赖原提成月份，因此历史订单的草稿一直保留到处理。财务点击【确认冲减】复用现有确认接口；点击【忽略建议】复用现有取消调整接口并填写必填原因；实际少发后点击现有【标记已扣回】。

若新增静态列表路由，必须声明在同前缀的 /{id} 参数路由之前。

## 5. 审批事务与并发

审批通过由 biz.Transactor.WithinTransaction 建立共享事务，仓储通过 Data.client(ctx) 使用同一事务客户端。

固定执行顺序：

1. FOR UPDATE 锁定订单；
2. FOR UPDATE 锁定与订单匹配的补录申请，校验 version、status 和 lock_generation；审批、驳回、撤回均复用这条锁序，确保并发时只有一个 PENDING 终态迁移成功；
3. 实时校验审批人的 lock grant 与组织范围；
4. 在锁内校验结算对象、费用项、币种和汇率，创建 CONFIRMED 费用；
5. 查询包含该订单的 CONFIRMED 或 PAID 提成父单，按父单 UUID 升序 FOR UPDATE；
6. 计算并创建 DECREASE + DRAFT 调整；
7. 更新申请为 APPROVED，记录审批人和时间；生成费用通过 supplement_request_id 反向关联申请；
8. 写审计和通知 outbox 后提交。

任何一步失败全部回滚，不允许“费用成功但调整缺失”。驳回只写申请终态和审计。

幂等由以下约束共同保证：

- 申请组织级幂等键；
- 申请到费用的一对一唯一键；
- 申请到每个原提成调整的一对一唯一键；
- 申请版本与状态机；
- 订单级串行锁和提成父单固定加锁顺序。

## 6. 提成影响计算

只处理包含目标订单提成线且父单为 CONFIRMED 或 PAID 的提成。查询结果按父单 UUID 固定顺序锁定，但金额不按这个顺序分配：

- 每条原提成订单行是独立的历史 earning event；分别使用该行冻结的人员、计提口径、比例、规则版本、计算版本、已实现收入、历史总应收和历史总应付计算本次补录的边际差额；
- FinanceCommissionLine 增加不可变的 total_receivable_snapshot、total_payable_snapshot（numeric(28,8)）；原提成主单的 rule_name、rule_version、calculation_version，以及订单行的 calculation_basis、rate_percent、realized_revenue、allocated_cost、realized_profit、commission_amount 与上述分母共同构成复算输入；
- 锁后复算调用按 calculation_version 路由的纯函数，禁止调用会根据 rule_id 查询 FinanceCommissionRule 的 calculateCommission/loadCommissionCalculationSource；当前规则修改、停用或删除不影响历史复算；
- 本次补录应付的本位币金额只增加历史 total_payable，不改变历史 total_receivable；原提成行确认时已纳入快照的费用不再叠加，禁止使用当前订单费用重建历史总应收；
- 对每条原提成行，按原快照分母及其确认后至本次审批前已生效的补录应付计算 before，再计入本笔费用计算 after；在逐次提成金额均按 8 位舍入后取非负差额。补录申请和订单费用的关联及批准时间作为边界，不根据冲减建议是否确认或忽略倒推成本事实；
- REALIZED_PROFIT 按现有部分实现公式比较前后金额，负差生成调整草稿；
- REALIZED_REVENUE 在实现收入未变化时不生成草稿；
- 差额为零不生成；
- 使用 decimal 与现有 8 位金额精度，禁止浮点数；
- 每个原提成父单分别创建调整，不合并不同父单，不采用 FIFO/LIFO，也不把一张父单的不足额顺延到另一张父单。

系统建议不得把某笔原提成的累计有效金额降到零以下。创建草稿时综合该父单现有非取消冲减和未处理系统建议，最多生成该父单仍可冲减的金额；理论影响超出部分只写入该父单审计，不转扣员工的其他原提成，也不形成员工欠款。

财务确认时继续执行现有有效金额校验。如果期间有人工调整占用了可冲减额度，返回版本或金额冲突，要求刷新后处理，不静默修改建议金额。

## 7. 审计与通知

- 向审批资格快照内人员发送待审批通知；
- 审批通过后向产生冲减建议的员工发送知情通知；
- 通知失败使用现有 outbox 重试，不回滚已提交业务事务；
- 调整可追溯到补录申请、订单费用、原提成、审批人和财务确认人；
- 被忽略建议额外记录取消人、取消时间和必填原因；补录申请撤回记录发起人、时间、原版本和稳定审计动作；
- 业务审计记录申请快照、前后提成金额、建议金额及未形成员工欠款的超出金额；
- 日志只记录 ID、状态、金额币种和稳定错误码，不记录完整敏感请求。

## 8. 迁移、兼容与回滚

- Ent Schema 是数据库真相源；新增表、字段、枚举、CHECK、外键和索引必须生成正式迁移；
- Proto 先行，生成 Go、OpenAPI 和前端客户端，禁止手改生成物；
- 项目尚未上线，不做旧数据双读或兼容分支；新增历史分母字段前盘点现存提成行，若有必须继续参与补录复算的已确认/已发放行，须用可审阅迁移还原分母并验证，不得填写猜测值、自动清库或静默跳过；
- 人工调整、核销冲减和对冲冲减行为保持不变；
- 上线后若关闭功能，只关闭创建和审批入口，保留申请、费用和调整的读取与审计；
- 不自动删除任何已批准补录或提成调整事实。

## 9. 自动业务锁定

### 9.1 触发与资格

- 应收核销创建生效或应收对冲确认生效后，收集本次有效分摊涉及的订单 ID，去重并按 UUID 排序，逐一发出自动锁定检查；一张跨订单账单不得只检查某个锚点订单。
- 费用草稿确认或作废后，只在该订单已经存在至少一笔有效应收结清事实时重试；纯成本、从未产生应收或仅有作废应收的订单不因“当前余额为零”进入自动锁定。
- 已确认应收的未结金额按订单关联的应收账单行减去有效核销分摊和已确认对冲分摊推导；未建账的已确认应收视为未结清。已反转核销、已反转对冲、已取消账单或失效分摊不计入已结清金额。
- 资格要求订单至少有一笔有效应收结清事实、当前未结清已确认应收为零、不存在费用草稿，并满足现有订单状态及 SE 快照校验。不得把整张跨订单账单余额代替订单余额。

### 9.2 锁定来源与审计身份

- Order 与 OrderLockRecord 增加 lock_source：MANUAL / AUTO_SETTLEMENT；锁定状态响应向前端返回来源。
- 人工锁定保持 locked_by 必填，并展示实际锁定人；自动锁定的 locked_by 为空，页面固定展示【系统自动锁定】，不创建可登录的“系统用户”，也不复用触发人的权限或伪装 bootstrap admin。
- 自动锁定记录 trigger_type（VERIFICATION、NETTING、FEE_CONFIRM、FEE_CANCEL）、trigger_resource_id 与 triggered_by；triggered_by 只表示导致本次检查的业务操作人，不表示其执行或批准了锁单。
- 数据库 CHECK 保证 MANUAL 必须有 locked_by 且自动触发字段为空，AUTO_SETTLEMENT 必须无 locked_by 且触发类型、资源与操作人完整；历史人工锁定迁移统一标记 MANUAL。

### 9.3 执行与失败边界

- 自动与人工锁定共用一个不包含调用人授权判断的内部锁定核心，复用订单行锁、状态校验、SE 单证快照、锁代次、锁定记录与审计；人工命令在进入核心前继续校验调用人的 lock grant，自动命令只接受内部可信事件并执行自动资格重验。
- 核销、对冲或费用状态流转先按自身事务提交；随后自动锁定使用独立事务，因此锁定失败不得回滚或改变原业务事实。触发接口不得把锁定失败伪装成原业务失败。
- 本期不引入持久事件投递：若进程在原事务提交后、自动检查调用开始前中断，无法为这一次遗漏检查写审计；后续相关事件仍会重验，人工锁定保留兜底。若未来要求消除此窗口，应独立设计事务 outbox 与可靠消费，不得把锁定并回原财务事务。
- 自动事务先锁定订单，再重验触发事实、应收结清、费用草稿、生命周期和当前锁状态；已经锁定时幂等结束。手动与自动锁定并发时最多产生一个锁代次和一份 SE 快照，败方重读后按已锁定成功处理。
- 每次自动检查均写成功、资格不满足或执行失败的结构化审计，包含稳定原因码、触发类型、触发资源和目标订单；失败不做全量定时扫描，仅由后续费用草稿确认/作废或有效应收核销/对冲事件再次检查，人工锁定保留兜底。
- 反核销、反对冲只恢复财务余额并沿用现有提成冲减联动，不触发自动业务解锁；自动锁定记录、锁代次和 SE 快照保持不可变，后续修改仍走现有显式解锁申请/直接解锁流程。
- 原补录费用、审批申请和提成冲减建议与自动锁定共用 Order 优先的互斥顺序；不得为自动锁定放宽普通订单写门禁或财务锁判定。
