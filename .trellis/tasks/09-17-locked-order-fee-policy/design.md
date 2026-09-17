# 设计：锁单后费用补录与提成冲减调整

## 1. 设计边界

本功能是订单普通费用写入口之外的一条受控追加命令，不改变现有业务锁和财务锁的普通门禁：

- 普通费用新增、修改、流转、删除继续受现有统一门禁保护；
- 新增“费用补录申请”聚合，只有审批通过才能创建锁后费用；
- 补录只新增正数费用，不修改或删除既有费用；
- 待处理冲减建议直接复用 FinanceCommissionAdjustment 的 DECREASE + DRAFT，不建设第二张建议表；
- 财务确认继续走 DRAFT → CONFIRMED，实际少发后继续走 CONFIRMED → PAID；
- 不建设员工负余额、月度发放批次、工资扣款、会计期间或总账。

现有契约继续作为真相源：

- 普通费用门禁位于 server/internal/data/order_fee.go；
- 财务锁遵循 .trellis/spec/server/backend/finance-commission-lock.md；
- 直接解锁资格遵循 .trellis/spec/server/backend/order-lock-and-document-version.md；
- 调整实体和状态机位于 server/internal/data/ent/schema/finance_commission_adjustment.go 与 server/internal/data/finance_commission.go；
- 外币费用遵循 .trellis/spec/server/backend/exchange-rate-single-rate.md。

## 2. 数据模型

### 2.1 新增 OrderFeeSupplementRequest

补录申请保存审批所需的不可变费用快照：

- organization_id、order_id、lock_generation；
- idempotency_key；
- 方向、费用项、结算对象、计费单位、数量、单价、税率、币种、费用发生日期和备注；
- reason、requested_by、requested_at；
- status：PENDING、APPROVED、REJECTED；
- version；
- decided_by、decided_at、decision_reason；
- 审批通过后唯一关联的 order_fee_id。

数据库约束：

- organization_id + idempotency_key 唯一；
- order_fee_id 可空且唯一；
- 费用快照创建后不可变；
- PENDING 只能进入一个终态；
- 发起时订单必须存在业务锁、财务锁或二者之一；普通未锁订单继续使用普通新增费用；
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
- RejectOrderFeeSupplementRequest。

订单费用页面行为：

- 锁定后普通新增、编辑、删除继续禁用；
- 有 fee.create 权限时显示【补录费用】；
- 补录表单复用普通费用字段，额外要求补录原因；
- 页面展示申请状态、发起人、审批人和生成的费用；
- 后端返回可审批能力，前端据此显示通过或驳回；
- 审批候选收到通知并跳转订单费用页；MVP 不新增全局审批中心。

### 4.2 提成侧

财务契约增加服务端分页的调整列表，支持状态、来源、员工、组织以及订单号或提成号关键字过滤。

提成页面增加【待处理冲减】视图，默认条件为：

- direction = DECREASE；
- status = DRAFT；
- source_type = LOCKED_FEE_SUPPLEMENT。

该视图不依赖原提成月份，因此历史订单的草稿一直保留到处理。财务点击【确认冲减】复用现有确认接口；实际少发后点击现有【标记已扣回】。

若新增静态列表路由，必须声明在同前缀的 /{id} 参数路由之前。

## 5. 审批事务与并发

审批通过由 biz.Transactor.WithinTransaction 建立共享事务，仓储通过 Data.client(ctx) 使用同一事务客户端。

固定执行顺序：

1. FOR UPDATE 锁定订单；
2. FOR UPDATE 锁定与订单匹配的补录申请，校验 version、status 和 lock_generation；
3. 实时校验审批人的 lock grant 与组织范围；
4. 在锁内校验结算对象、费用项、币种和汇率，创建 CONFIRMED 费用；
5. 查询包含该订单的 CONFIRMED 或 PAID 提成父单，按父单 UUID 升序 FOR UPDATE；
6. 计算并创建 DECREASE + DRAFT 调整；
7. 更新申请为 APPROVED，记录费用、审批人和时间；
8. 写审计和通知 outbox 后提交。

任何一步失败全部回滚，不允许“费用成功但调整缺失”。驳回只写申请终态和审计。

幂等由以下约束共同保证：

- 申请组织级幂等键；
- 申请到费用的一对一唯一键；
- 申请到每个原提成调整的一对一唯一键；
- 申请版本与状态机；
- 订单级串行锁和提成父单固定加锁顺序。

## 6. 提成影响计算

只处理包含目标订单提成线且父单为 CONFIRMED 或 PAID 的提成：

- 沿用原提成线冻结的人员、计提口径、比例、规则版本、计算版本与已实现收入；
- 使用本次费用加入前后的订单有效费用事实计算边际差额，防止多笔补录重复计算同一影响；
- REALIZED_PROFIT 按现有部分实现公式比较前后金额，负差生成调整草稿；
- REALIZED_REVENUE 在实现收入未变化时不生成草稿；
- 差额为零不生成；
- 使用 decimal 与现有 8 位金额精度，禁止浮点数；
- 每个原提成父单分别创建调整，不合并不同父单。

系统建议不得把某笔原提成的累计有效金额降到零以下。创建草稿时综合该父单现有非取消冲减和未处理系统建议，最多生成仍可冲减的金额；理论影响超出部分只写入审计，不形成员工欠款。

财务确认时继续执行现有有效金额校验。如果期间有人工调整占用了可冲减额度，返回版本或金额冲突，要求刷新后处理，不静默修改建议金额。

## 7. 审计与通知

- 向审批资格快照内人员发送待审批通知；
- 审批通过后向产生冲减建议的员工发送知情通知；
- 通知失败使用现有 outbox 重试，不回滚已提交业务事务；
- 调整可追溯到补录申请、订单费用、原提成、审批人和财务确认人；
- 业务审计记录申请快照、前后提成金额、建议金额及未形成员工欠款的超出金额；
- 日志只记录 ID、状态、金额币种和稳定错误码，不记录完整敏感请求。

## 8. 迁移、兼容与回滚

- Ent Schema 是数据库真相源；新增表、字段、枚举、CHECK、外键和索引必须生成正式迁移；
- Proto 先行，生成 Go、OpenAPI 和前端客户端，禁止手改生成物；
- 项目尚未上线，不做旧数据回填、双读或兼容分支；
- 人工调整、核销冲减和对冲冲减行为保持不变；
- 上线后若关闭功能，只关闭创建和审批入口，保留申请、费用和调整的读取与审计；
- 不自动删除任何已批准补录或提成调整事实。
