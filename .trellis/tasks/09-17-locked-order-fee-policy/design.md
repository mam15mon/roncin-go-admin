# 设计：锁单后费用补录与提成冲减调整

## 1. 设计边界

本功能是订单普通费用写入口之外的一条受控追加命令，不改变现有业务锁和财务锁的普通门禁：

- 普通费用新增、修改、流转、删除继续受现有统一门禁保护；
- 新增“费用补录申请”聚合，只有审批通过才能创建锁后费用；
- 补录只新增正数应付成本；应收方向由后端拒绝，不修改或删除锁前既有费用；审批生成的补录费用仅允许通过本设计的受限专用动作作废；
- 待处理冲减建议直接复用 FinanceCommissionAdjustment 的 DECREASE + DRAFT，不建设第二张建议表；
- 财务确认继续走 DRAFT → CONFIRMED，实际少发后继续走 CONFIRMED → PAID；
- 不建设员工负余额、月度发放批次、工资扣款、会计期间或总账。

现有契约继续作为真相源：

- 普通费用门禁位于 server/internal/data/order_fee.go；
- 业务锁、锁定代次和 SE 单证快照位于 server/internal/data/order_lock.go；
- 共享 MBL 活动成员、固定锁序与成员集合重验遵循 .trellis/spec/server/backend/order-lock-and-document-version.md 和 sea-export-document-contract.md；
- 财务锁遵循 .trellis/spec/server/backend/finance-commission-lock.md；
- 直接解锁资格遵循 .trellis/spec/server/backend/order-lock-and-document-version.md；
- 调整实体和状态机位于 server/internal/data/ent/schema/finance_commission_adjustment.go 与 server/internal/data/finance_commission.go；
- 外币费用遵循 .trellis/spec/server/backend/exchange-rate-single-rate.md。

## 2. 数据模型

### 2.1 新增 OrderFeeSupplementRequest

补录申请保存审批所需的不可变费用快照：

- organization_id、order_id、lock_basis（BUSINESS、FINANCIAL、BOTH）、可空 business_lock_generation、可空 financial_lock_evidence_version、financial_lock_evidence_hash 和 financial_lock_net_amount_snapshot；
- idempotency_key、request_fingerprint；
- 应付方向、费用项、结算对象、计费单位、数量、单价、税率、币种、费用发生日期和备注；
- reason、requested_by、requested_at；
- status：PENDING、APPROVED、REJECTED、WITHDRAWN；
- version；
- decided_by、decided_at、decision_reason。

数据库约束：

- organization_id + idempotency_key 唯一；request_fingerprint 由服务端对订单 ID、不可变费用快照和补录原因做版本化规范编码后计算，同键同指纹返回原申请，同键不同指纹返回 FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT；
- 费用快照创建后不可变；
- PENDING 只能进入一个终态；
- WITHDRAWN 只能由 requested_by 在 PENDING 状态携带 expectedVersion 触发；审批、驳回和撤回统一锁定申请行并比较版本，竞争失败的一方返回状态冲突；
- 发起时在 Order 行锁内计算业务锁与提成净额财务锁，二者均不存在时拒绝；仅业务锁、仅财务锁和双锁分别固化为 BUSINESS、FINANCIAL、BOTH，普通无锁订单继续使用普通新增费用；
- 财务锁证据由服务端对当时参与净额的 CONFIRMED/PAID 提成行与调整行，按“组件类型 + 主键 + 方向 + 符号化 8 位金额”排序后使用 financial_lock_evidence_version 对应的规范编码计算；CONFIRMED 与 PAID 统一编码为 ACTIVE，不把二者之间的生命周期流转当作锁身份变化。参与记录进入/退出有效集合、方向、金额或主键变化才改变哈希；同时固化当时净额，禁止把客户端提供的摘要当作可信输入；
- 数据库 CHECK 保证 BUSINESS 必须仅保存大于零的 business_lock_generation，FINANCIAL 必须仅保存非空证据版本、证据哈希和大于零的净额快照，BOTH 必须同时保存两组依据；不得用客户端传入的锁类型、锁代次或财务证据覆盖服务端判定；
- 创建和审批均要求方向为 PAYABLE，锁内复核快照，不允许客户端把方向改成 RECEIVABLE；
- 审批时按申请固化的 lock_basis 和 financial_lock_evidence_version 复核原始锁依据：BUSINESS 要求同一业务锁代次仍然有效；FINANCIAL 要求当前财务净额仍大于零且规范化证据哈希与申请一致；BOTH 只要前述业务锁依据或财务锁依据至少一项仍匹配即可。原始锁依据全部失效时返回 LOCK_BASIS_CHANGED；若订单当前仍被另一项新锁锁定则提示重新发起补录，当前已无任何锁则提示改走普通费用新增。不得把旧申请自动嫁接到提交后新出现的业务锁代次或不同财务锁证据。
- 财务锁证据的提交计算与审批复核必须复用 financeCommissionLockNetAmount 的符号口径，并在同一事务、Order 行锁内读取参与事实。所有会改变财务锁净额或证据集合的提成/调整状态迁移必须先按订单 UUID 排序锁定受影响 Order，再修改提成或调整，统一保持 Order → 提成父单 → 调整的锁序；否则仅保存哈希仍会存在审批检查后的并发穿透。

### 2.2 复用 OrderFee

审批通过后创建普通订单费用：

- 状态直接为 CONFIRMED，补录审批承担费用真实性确认；
- 幂等键由补录申请 ID 确定性派生；
- 增加可空、唯一的 supplement_request_id 关联；
- 金额、税额、本位币金额与汇率快照复用现有计算；
- 创建后进入普通账单、对账、开票、收付和利润链路。
- 现有建账候选直接选择 CONFIRMED 且没有活动账单行的费用，业务锁/财务锁不额外屏蔽补录来源；建账转为 BILLED，符合现有取消条件的账单取消后恢复为 CONFIRMED。

普通费用仓储不增加 skipLock 或 force 参数。新增内部专用仓储命令，并在事务中验证补录申请，避免形成通用绕过入口。

### 2.3 补录费用作废边界

- APPROVED 申请保持终态，不增加“撤回已批准申请”迁移；生成费用使用现有 CANCELLED 状态及 cancelled_by、cancelled_at、cancellation_reason 保存作废事实；
- 专用作废命令只接受 source supplement、状态为 CONFIRMED、无活动 FinanceBillLine 的费用，并要求目标订单实时直接解锁资格、expectedVersion 和必填原因；普通 RemoveFee 仍受统一业务锁/财务锁门禁，不增加通用绕过；
- 按批准时间和 ID 判断同订单是否存在更晚的有效补录费用；存在时返回稳定冲突，要求按倒序逐笔作废，避免撤销早期成本后让后续边际冲减失真；
- 锁定该补录关联的提成父单及调整后，要求所有关联调整的 confirmed_at、paid_at 均为空且当前只处于 DRAFT 或 CANCELLED。DRAFT 在同一事务转为 CANCELLED，既有 CANCELLED 保持不变；任一调整曾进入 CONFIRMED 或 PAID 时拒绝作废，不允许通过状态洗白后绕过，也不自动生成 INCREASE 恢复；
- 费用为 BILLED 时先走现有账单取消。账单存在有效发票、核销或对冲时，继续由现有财务状态机阻止取消；先完成允许的下游撤销后，账单取消会把费用恢复为 CONFIRMED，再执行专用作废；
- 作废与新补录审批共用 Order 行锁，与账单创建通过费用行锁竞争：账单先提交则作废看到 BILLED 并拒绝，作废先提交则建账看到 CANCELLED 并拒绝，不产生部分成功。

### 2.4 复用 FinanceCommissionAdjustment

待处理冲减建议直接落为现有提成调整记录：

- direction = DECREASE；
- status = DRAFT；
- source_type 新增 LOCKED_FEE_SUPPLEMENT；
- source_fee_supplement_request_id 关联补录申请；
- commission_id、order_id、employee_id 继续使用现有字段；
- commission_id + order_id + source_type + source_fee_supplement_request_id 唯一。
- 数据库 CHECK 强制 LOCKED_FEE_SUPPLEMENT 与非空 source_fee_supplement_request_id 双向对应，其他来源必须为空；外键使用 RESTRICT，禁止删除申请造成孤儿调整；
- LOCKED_FEE_SUPPLEMENT 的 DRAFT 调整允许财务复用 CancelCommissionAdjustment 忽略，必填 cancellation_reason 并保留 cancelled_by/cancelled_at；TransitionAdjustment 必须按 source_type 增加状态门禁，LOCKED_FEE_SUPPLEMENT 的 CONFIRMED/PAID 即使直接调用通用取消接口也拒绝转为 CANCELLED；页面文案为【忽略建议】，不删除记录。

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
- 创建事务在同一目标订单与组织范围内解析当前有效 lock grant；没有任何可审批人员时返回 FEE_SUPPLEMENT_APPROVER_UNAVAILABLE，不写申请或通知。通知收件人只是在提交时形成的快照，不冻结未来审批资格。

### 3.2 审批资格

- 审批时实时复用目标订单业务类型的有效 lock grant 与数据范围；
- 不复制角色代码，不新增“主管”字段或负责人名单；
- 任一合格人员均可审批；
- 发起人本人合格时允许自行审批；
- 审批不会修改订单 locked_at、lock_generation 或解锁状态。
- 提交后当前审批人全部失效时申请继续保持 PENDING 并投影稳定阻断原因；发起人仍可撤回，后来新获得实时 lock grant 的人员可以审批或驳回。

### 3.3 补录费用作废资格

- CancelApprovedOrderFeeSupplement 使用目标订单 `lock` 操作的动态权限入口，与补录审批复用同一实时 lock grant 和组织范围，不依赖客户端传回的可作废标记；
- 该资格不等于普通 `fee.delete`，只允许调用专用命令处理与 APPROVED 补录申请一对一关联的费用；普通 RemoveFee 仍按原权限和锁门禁执行；
- 后端根据费用状态、活动账单行、后续有效补录和关联调整状态返回可作废能力及稳定阻断原因，前端只消费结果。

### 3.4 财务权限

- 财务工作台查看与确认冲减继续使用 system.finance.commission.read/manage；员工本人最小来源详情按第 3.5 节专用所有权授权处理；
- 前端只消费后端权限结果，不硬编码“财务角色”名称。

### 3.5 专用最小读取

- 订单页读取补录申请采用后端逐条授权：具备目标订单 fee.read，或 requested_by 等于当前用户，或实时具备该订单 lock grant，任一成立即可读取；本人读取与审批读取只返回申请和审批所需费用快照，不扩张到其他订单费用；
- 撤回仍要求 requested_by 等于当前用户、状态 PENDING、expectedVersion 匹配，不因持有 fee.read 或 lock grant 获得代撤回能力；
- 员工通知跳转到本人专属冲减来源详情，仓储查询同时限定 adjustment ID、employee_id = principal.UserID 和可见组织成员关系，只返回订单号、原提成号、补录费用摘要、建议金额及状态；不复用组织级 GetCommission，不授予 system.finance.commission.read，也不返回其他员工或父单其他订单行；
- 财务工作台继续使用组织级 commission.read/manage；本人专属读取不能确认、忽略或标记已扣回。

## 4. API 与页面

### 4.1 订单费用侧

订单费用契约新增：

- CreateOrderFeeSupplementRequest；
- ListOrderFeeSupplementRequests；
- ApproveOrderFeeSupplementRequest；
- RejectOrderFeeSupplementRequest；
- WithdrawOrderFeeSupplementRequest。
- CancelApprovedOrderFeeSupplement。
- GetMyFeeSupplementAdjustmentSource（员工本人最小来源详情）。

订单费用页面行为：

- 锁定后普通新增、编辑、删除继续禁用；
- 有 fee.create 权限时显示【补录费用】；
- 补录表单复用普通费用字段，额外要求补录原因；
- 页面展示申请状态、发起人、审批人和生成的费用；
- 发起人和审批人通过专用逐条授权读取申请；页面不得先以 fee.read 隐藏整个申请区域，再导致合法发起人或审批人无法操作；
- PENDING 且当前用户为发起人时显示【撤回申请】；撤回成功后状态为 WITHDRAWN，不再显示审批动作；
- APPROVED 申请展示生成费用当前状态；后端返回可作废能力时显示【作废补录费用】，强制填写原因并明确提示“已建账需先取消账单、冲减已确认后不能直接作废”；
- 后端返回可审批能力，前端据此显示通过或驳回；
- 审批候选收到通知并跳转订单费用页；MVP 不新增全局审批中心。

### 4.2 提成侧

财务契约增加服务端分页的调整列表，支持状态、来源、员工、组织以及订单号或提成号关键字过滤。

提成页面增加【待处理冲减】视图，默认条件为：

- direction = DECREASE；
- status = DRAFT；
- source_type = LOCKED_FEE_SUPPLEMENT。

该视图不依赖原提成月份，因此历史订单的草稿一直保留到处理。财务点击【确认冲减】复用现有确认接口；点击【忽略建议】复用现有取消调整接口并填写必填原因；实际少发后点击现有【标记已扣回】。

员工通知不跳转组织级提成工作台，统一跳转本人专属来源详情；服务端返回 404/无权限时不泄露调整是否属于其他员工。

若新增静态列表路由，必须声明在同前缀的 /{id} 参数路由之前。

## 5. 审批事务与并发

审批通过由 biz.Transactor.WithinTransaction 建立共享事务，仓储通过 Data.client(ctx) 使用同一事务客户端。

固定执行顺序：

1. FOR UPDATE 锁定订单；
2. FOR UPDATE 锁定与订单匹配的补录申请，校验 version、status，并按 lock_basis 复核 business_lock_generation 或重新计算财务锁证据哈希；审批、驳回、撤回均复用这条锁序，确保并发时只有一个 PENDING 终态迁移成功；
3. 实时校验审批人的 lock grant 与组织范围；
4. 在锁内重新解析并校验结算对象、费用项、币种和费用发生日汇率，先形成待创建费用快照但不落库；
5. 查询包含该订单的 CONFIRMED 或 PAID 提成父单，按父单 UUID 升序 FOR UPDATE，并锁定目标订单行及会影响订单行/父单余额的调整；
6. 按 calculation_version 判定成本敏感性：未知版本失败关闭；成本敏感行必须 snapshot_status = READY 且 base_currency 与待创建费用一致，否则分别返回 COMMISSION_SNAPSHOT_UNAVAILABLE 或 COMMISSION_BASE_CURRENCY_MISMATCH；REALIZED_REVENUE 等明确不受应付成本影响的行跳过分母与币种阻断；
7. 计算边际差额并按订单行、父单双重余额封顶，随后创建 CONFIRMED 费用和 DECREASE + DRAFT 调整；
8. 更新申请为 APPROVED，记录审批人和时间；生成费用通过 supplement_request_id 反向关联申请；
9. 创建确定性通知任务、明细和业务审计后提交。

任何一步失败全部回滚，不允许“费用成功但调整缺失”。驳回只写申请终态和审计。

幂等由以下约束共同保证：

- 申请组织级幂等键与版本化 request_fingerprint 语义比对；
- 申请到费用的一对一唯一键；
- 申请到每个原提成调整的一对一唯一键；
- 申请版本与状态机；
- 订单级串行锁和提成父单固定加锁顺序。

### 5.1 补录费用作废事务

专用作废使用 biz.Transactor.WithinTransaction，固定顺序为：Order → 补录申请/费用 → 按 UUID 排序的提成父单 → 关联调整。

1. Order FOR UPDATE 后实时复核直接解锁资格；这里只复用订单作为互斥点，不调用会禁止锁后专用动作的普通费用内容门禁；
2. 锁定 APPROVED 申请和生成费用，比较 expectedVersion，确认费用为 CONFIRMED、来源匹配且无活动账单行；
3. 查询同订单更晚批准且费用未 CANCELLED 的补录；存在时返回稳定冲突；
4. 按父单 UUID 锁定该申请关联的全部提成父单和调整，发现 confirmed_at/paid_at 非空或当前为 CONFIRMED/PAID 即整体拒绝；
5. 将关联 DRAFT 调整转为 CANCELLED，再将费用转为 CANCELLED；APPROVED 申请不改终态；
6. 写作废人、时间、原因、前一费用版本和审计后提交。

任何一步失败全部回滚。该事务不删除费用、申请或调整，也不把系统作废原因伪装成财务主动【忽略建议】。

## 6. 提成影响计算

只处理包含目标订单提成线且父单为 CONFIRMED 或 PAID 的提成。查询结果按父单 UUID 固定顺序锁定，但金额不按这个顺序分配：

- 每条原提成订单行是独立的历史 earning event；分别使用该行冻结的人员、计提口径、比例、规则版本、计算版本、已实现收入、历史总应收和历史总应付计算本次补录的边际差额；
- FinanceCommissionLine 增加可空的 total_receivable_snapshot、total_payable_snapshot（numeric(28,8)），以及 snapshot_status（READY、UNAVAILABLE）、snapshot_source（NATIVE、MIGRATED）和可空的 snapshot_backfill_version、snapshot_evidence_hash、snapshot_unavailable_reason_code；数据库 CHECK 保证 READY 必须有完整分母，READY + MIGRATED 必须有回填算法版本和证据哈希，UNAVAILABLE 不得伪填估算分母且必须有稳定原因码。新生成提成行固定写 READY + NATIVE；原提成主单的 rule_name、rule_version、calculation_version，以及订单行的 calculation_basis、rate_percent、realized_revenue、allocated_cost、realized_profit、commission_amount 与上述分母共同构成复算输入；
- 锁后复算调用按 calculation_version 路由的纯函数，禁止调用会根据 rule_id 查询 FinanceCommissionRule 的 calculateCommission/loadCommissionCalculationSource；当前规则修改、停用或删除不影响历史复算；
- 本次补录应付的本位币金额只增加历史 total_payable，不改变历史 total_receivable；原提成行确认时已纳入快照的费用不再叠加，禁止使用当前订单费用重建历史总应收；
- 仅对 calculation_version 明确声明 payable_cost_sensitive = true 的行执行成本复算；当前 REALIZED_PROFIT 为 true，REALIZED_REVENUE 为 false。未知 calculation_version 返回 COMMISSION_CALCULATION_VERSION_UNSUPPORTED，禁止通过字符串猜测或默认分支放行；
- 成本敏感行复算前要求补录费用 base_currency 与该 FinanceCommissionLine.base_currency 完全相同；不一致返回 COMMISSION_BASE_CURRENCY_MISMATCH 并整体回滚。本期不做当前本位币到历史本位币的二次转换；
- 对每条原提成行，按原快照分母及其确认后至本次审批前已生效且未作废的补录应付计算 before，再计入本笔费用计算 after；在逐次提成金额均按 8 位舍入后取非负差额。补录申请和订单费用的关联、批准顺序及费用是否 CANCELLED 作为边界，不根据冲减建议是否确认或忽略倒推成本事实；
- REALIZED_PROFIT 按现有部分实现公式比较前后金额，负差生成调整草稿；
- REALIZED_REVENUE 在实现收入未变化时不生成草稿；
- 差额为零不生成；
- 使用 decimal 与现有 8 位金额精度，禁止浮点数；
- 每个原提成父单分别创建调整，不合并不同父单，不采用 FIFO/LIFO，也不把一张父单的不足额顺延到另一张父单。

系统建议同时执行订单行和父单两层余额约束：

- 订单行余额以 FinanceCommissionLine.commission_amount 为起点，只汇总同 commission_id + order_id 的 CONFIRMED/PAID 符号化调整；父单余额以 FinanceCommission.commission_amount 为起点，汇总父单全部 CONFIRMED/PAID 符号化调整；
- 创建草稿时，两层余额均额外预留其他 DRAFT DECREASE，DRAFT INCREASE 不提前扩张余额；建议金额取理论边际差额、订单行可用余额、父单可用余额三者最小值；
- 财务确认时在 Order → 提成父单 → 调整固定锁序内重算不含其他 DRAFT 的两层有效余额；任一层确认后将小于零都返回 COMMISSION_ADJUSTMENT_EXCEEDS，不静默缩小建议；
- 现有通用 TransitionAdjustment 必须增加 LOCKED_FEE_SUPPLEMENT 的订单行余额检查，不能只沿用父单总额校验。同父单其他订单行的正余额不得替当前订单行兜底；
- 理论影响超出部分同时写入订单行和父单审计，不转扣其他订单行、其他原提成或其他员工，也不形成员工欠款。

如果建议生成后人工调整占用了订单行或父单额度，财务确认返回版本或金额冲突，要求刷新后处理，不静默修改建议金额。

## 7. 审计与通知

- 不抽象新的 outbox 表；直接复用现有一对一的 BackgroundTask + NotificationDelivery 通知模型，每名收件人对应一条 DINGTALK_NOTIFICATION 任务和一条投递明细，并在产生业务事实的同一事务内入队；
- 申请提交时向审批资格快照内人员发送 FEE_SUPPLEMENT_APPROVAL_PENDING 模板，resource_type = FEE_SUPPLEMENT_REQUEST、resource_id = 申请 ID；审批通过后向每条实际生成冲减建议的员工发送 COMMISSION_DECREASE_SUGGESTED 模板，resource_type = COMMISSION_ADJUSTMENT、resource_id = 调整 ID。未产生建议的员工不通知，通知参数只携带经长度限制的订单号、金额等最小摘要；
- 通知任务 ID 和 idempotency_key 分别按“申请 ID + 审批收件人”和“调整 ID + 员工收件人”确定性生成，BackgroundTask 与 NotificationDelivery 均使用唯一约束幂等收敛；审批资格后来变化不补发旧申请通知，实时审批资格仍以接口校验为准；
- 业务事务只保证通知任务可靠入队；真正的钉钉投递失败沿用 BackgroundTask 重试和 DEAD_LETTER 机制，不回滚已经提交的申请、费用或调整；任务/明细入库失败则回滚同一业务事务，避免业务已成功但通知事实永久缺失；
- 调整可追溯到补录申请、订单费用、原提成、审批人和财务确认人；
- 被忽略建议额外记录取消人、取消时间和必填原因；补录申请撤回记录发起人、时间、原版本和稳定审计动作；
- 业务审计记录申请快照、前后提成金额、建议金额及未形成员工欠款的超出金额；
- 补录费用作废额外记录作废人、时间、原因、费用版本、被系统取消的 DRAFT 调整 ID；
- 日志只记录 ID、状态、金额币种和稳定错误码，不记录完整敏感请求。

## 8. 迁移、兼容与回滚

- Ent Schema 是数据库真相源；新增表、字段、枚举、CHECK、外键和索引必须生成正式迁移；
- Proto 先行，生成 Go、OpenAPI 和前端客户端，禁止手改生成物；
- 项目尚未上线，不做旧数据双读或兼容分支；但本功能面向历史订单，新增历史分母字段前必须盘点现存 CONFIRMED/PAID 提成行；
- 回填以原提成行首次计算并写入 commission_amount 的“计算快照形成时点”为唯一截止时点，只能使用原提成行、其 calculation_version，以及该时点已生效且仍可追溯的不可变费用、账单行、核销和对冲事实，按历史版本纯计算器确定性还原。无法确定唯一截止时点，或依赖事实曾被原地修改且没有历史版本时，直接判为 UNAVAILABLE；不得读取迁移时点当前订单费用汇总，不得查询当前 FinanceCommissionRule，也不得用规则是否仍存在作为可回填判断；
- 可确定性还原的行写 READY + MIGRATED，并记录回填算法版本与证据集合哈希；回填后重新计算原 allocated_cost、realized_profit、commission_amount 并与已存值按 8 位精度校验，不一致时标记 UNAVAILABLE，不得覆盖原提成金额或填写猜测分母；
- 无法还原的成本敏感行写 UNAVAILABLE 并输出不含敏感报文的迁移报告。补录审批在创建费用前先按 calculation_version 判定是否受应付成本影响：已知非成本敏感版本（当前 REALIZED_REVENUE）不要求历史成本分母；成本敏感版本必须 snapshot_status = READY 且与本次费用 base_currency 一致，否则分别返回 COMMISSION_SNAPSHOT_UNAVAILABLE 或 COMMISSION_BASE_CURRENCY_MISMATCH；未知版本失败关闭。任一失败时费用、调整、申请终态、审计、BackgroundTask 和 NotificationDelivery 全部不写入，不得静默跳过该行转人工待办；
- 人工调整、核销冲减和对冲冲减行为保持不变；
- 上线后若关闭功能，只关闭创建和审批入口，保留申请、费用和调整的读取与审计；
- 不自动删除任何已批准补录或提成调整事实。

## 9. 自动业务锁定

### 9.1 触发与资格

- 应收核销创建生效或应收对冲确认生效后，收集本次有效分摊涉及的订单 ID，去重并按 UUID 排序，逐一发出自动锁定检查；一张跨订单账单不得只检查某个锚点订单。
- 费用草稿确认或作废后，只在该订单已经存在至少一笔有效应收结清事实时重试；纯成本、从未产生应收或仅有作废应收的订单不因“当前余额为零”进入自动锁定。
- 已确认应收的未结金额按订单关联的应收账单行减去有效核销分摊和已确认对冲分摊推导；未建账的已确认应收视为未结清。已反转核销、已反转对冲、已取消账单或失效分摊不计入已结清金额。
- 资格要求订单至少有一笔有效应收结清事实、当前未结清已确认应收为零、不存在费用草稿，并满足现有订单状态及 SE 快照校验。不得把整张跨订单账单余额代替订单余额。
- 非 SE 和只有一个活动成员的 SE 订单按单票检查。SE 当前 MBL 有多个活动成员时，把全部活动成员作为一个自动锁定组：已锁成员视为完成，所有尚未锁定成员必须分别满足上述资格；任一不满足时整组只记录原因，不锁任何剩余成员。纯成本或从未发生有效结清事实的成员会使该组继续依赖人工锁定兜底。

### 9.2 锁定来源与审计身份

- Order 与 OrderLockRecord 增加 lock_source：MANUAL / AUTO_SETTLEMENT；锁定状态响应向前端返回来源。
- 人工锁定保持 locked_by 必填，并展示实际锁定人；自动锁定的 locked_by 为空，页面固定展示【系统自动锁定】，不创建可登录的“系统用户”，也不复用触发人的权限或伪装 bootstrap admin。
- 自动锁定记录 trigger_type（VERIFICATION、NETTING、FEE_CONFIRM、FEE_CANCEL）、trigger_resource_id 与 triggered_by；triggered_by 只表示导致本次检查的业务操作人，不表示其执行或批准了锁单。
- 自动锁定为了固化 SE 快照而新建的 MBL、运输执行或 HBL 不可变版本，其 created_by 为空并保留 ORDER_LOCK 来源；触发操作人只记录在自动锁定触发审计中，不冒充版本创建人。复用既有版本时不改写原 created_by。
- 数据库 CHECK 保证 MANUAL 必须有 locked_by 且自动触发字段为空，AUTO_SETTLEMENT 必须无 locked_by 且触发类型、资源与操作人完整；历史人工锁定迁移统一标记 MANUAL。

### 9.3 执行与失败边界

- 自动与人工锁定共用一个不包含调用人授权判断的内部锁定核心，复用订单行锁、状态校验、SE 单证快照、锁代次、锁定记录与审计；人工命令在进入核心前继续校验调用人的 lock grant，自动命令只接受内部可信事件并执行自动资格重验。
- 核销、对冲或费用状态流转先按自身事务提交；随后自动锁定使用独立事务，因此锁定失败不得回滚或改变原业务事实。触发接口不得把锁定失败伪装成原业务失败。
- 本期不引入持久事件投递：若进程在原事务提交后、自动检查调用开始前中断，无法为这一次遗漏检查写审计；后续相关事件仍会重验，人工锁定保留兜底。若未来要求消除此窗口，应独立设计事务 outbox 与可靠消费，不得把锁定并回原财务事务。
- 单票自动事务先锁定 Order，再重验触发事实、应收结清、费用草稿、生命周期和当前锁状态；已经锁定时幂等结束。共享 MBL 组级事务先只读定位活动成员，按 Order UUID 升序锁定全部成员，再锁 MBL、按 ID 排序的全部活动 Link、共享运输执行和各成员 HBL，并重验活动成员集合；集合改变返回结构冲突且整组零写入。组内锁定记录复用同一 MBL/运输执行版本，各自保存所属 HBL 快照。
- 手动锁单继续使用现有单票语义；手动与组级自动锁定竞争时，组事务在取得全部 Order 锁后把已锁成员视为完成，并只在其他成员全部合格时原子锁定剩余成员。两个成员触发的组级自动检查因使用相同排序而串行收敛，每个成员最多增加一个锁代次和一份锁定记录。
- 普通费用写、补录审批和补录费用作废已经以 Order 为首锁。反核销、反对冲需先只读解析受影响订单，按 UUID 锁定全部 Order，再锁来源单并重验版本、状态和有效分摊，随后执行提成联动；取得 Order 锁只为与自动锁定线性化，不调用业务内容门禁。反转先提交时自动锁定重验后放弃；自动锁定先提交时反转仍可成功，但不得自动解锁。
- 每次自动检查均写成功、资格不满足或执行失败的结构化审计，包含稳定原因码、触发类型、触发资源和目标订单；失败不做全量定时扫描，仅由后续费用草稿确认/作废或有效应收核销/对冲事件再次检查，人工锁定保留兜底。
- 反核销、反对冲只恢复财务余额并沿用现有提成冲减联动，不触发自动业务解锁；自动锁定记录、锁代次和 SE 快照保持不可变，后续修改仍走现有显式解锁申请/直接解锁流程。
- 原补录费用、审批申请和提成冲减建议与自动锁定共用 Order 优先的互斥顺序；不得为自动锁定放宽普通订单写门禁或财务锁判定。
