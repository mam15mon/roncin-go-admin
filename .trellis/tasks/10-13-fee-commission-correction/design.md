# 费用更正及提成纠错实施设计

## 状态与范围

第一阶段已完成，用户要求继续完成本子任务。本设计覆盖 R2/R3，复用费用审核权限、普通资金操作和提成调整；不增加第二审批人、解锁开关或完整性清单。用户已确认 Q1：按真实已收金额及当前有效收入成本重算，例子为25元；“继续完成”的授权覆盖本期实现与验证。详细代码证据由 research/accounting-design.md、research/commission-design.md 承载。

## 1. 申请与界面

- 订单费用页提供追加费用、选择原费用发起更正两个入口；原补录生成的费用也可以被更正。每次申请针对一个订单、一笔原费用或一笔新增费用，显示完整前后快照、原因、利润变化、受影响账单及提成。
- 状态为 PENDING → APPROVED / REJECTED / WITHDRAWN。批准不可直接撤销删除，错误由后续关联更正处理。复用现有审核资格、组织范围和本人撤回规则；审批仍须显式操作。
- 同订单最多一个待审补录或更正，两个入口共用检查。待审阻止新建账、受影响提成确认/发放及结案；CLOSED 后可发起和批准，保留业务结案状态。
- 用户填目标费用内容，系统生成冲回与替代，不要求用户手填负数或自行选择经济正负号。列表默认展示当前有效费用，历史展开查看原费用、审批、冲回和替代关系。
- 申请列表以 ProTable request/actionRef 管理，编辑以 ModalForm.onFinish 管理；金额与时间复用公共格式函数，成功响应经校验后统一刷新费用审核、详情、台账、提成和财务待办。

## 2. 经济内容与账务内容

保留费用金额正数，显式增加经济效果 INCREASE/DECREASE、清偿方向与更正来源。经济冲回保留原费用的收入/成本属性，只有清偿方向反转。例如成本冲回不是营业收入。

原行一旦被批准更正，不能再次作为更正目标；后续更正选最新替代行。旧行、冲回行与新行之间形成可验证的一对一来源关系，不以时间排序猜测对应关系。旧行与冲回行净经济影响归零，当前费用汇总等于尚未被替代的正常费用；两种投影必须有一致性校验，不能重复计入。

| 原费用状态 | 批准结果 | 财务后续 |
| --- | --- | --- |
| 未建账 | 原行退出候选；经济冲回无需建账；新行正常待建账 | 只建新费用 |
| 已有草稿/确认账单 | 原账单及行快照不变；冲回行和替代行待建账 | 建反向调整账单及新费用账单，按真实余额对冲/收付 |
| 部分或全部核销/对冲 | 保留全部历史分摊 | 抵销未结部分；超过未结余额的部分形成待退款/追回 |
| 已开票 | 审批不改变发票状态；关联票据处理待办 | 财务按现有操作整张红冲，再按有效费用重开 |

同一结算对象、同一币种内才允许现有对冲；换对象采用旧对象全额冲回、新对象重立，不能跨对象抵销。换币种同样不直接相减。冲回经济本位币继承原费用快照，新费用按合法汇率规则固化，账单日汇率与汇差仍各自留存。

示例：供应商费用已付 1,000，正确成本 100。批准后经济成本为 `1,000−1,000+100=100`；原付款 1,000 不动，形成反向应收 1,000 与新应付 100。财务可对冲 100，剩余应收 900 待真实追回。未付款时，原应付 1,000 与反向应收 1,000 对冲后，仅剩新应付 100。

## 3. 建账、核销、对冲与开票

- 账单必须标明 NORMAL / CORRECTION_REVERSAL 用途并关联更正来源；分组键包括清偿方向和用途，禁止正常费用与反向调整混成语义不明的一张账单。
- 经济冲回是否需要建账取决于批准时原费用是否已被账单占用，由事务固化；不得通过后续金额或“当前无活动账单”推断。原账单被取消等后续动作须检查更正依赖，防止丢失旧债又保留冲回债权。
- 更正内部对冲须有明确来源链和用途：原未收账单与其冲回对冲不能生成收入；旧已收账单的冲回与替代应收对冲不能重复计提。来源候选和实际计提必须使用同一个资格判定。
- 普通收付款、核销、对冲的金额保持正数。审批不自动建现金流水、不自动确认核销、不改写实付事实。
- 开票候选按有效正常经济行计算；冲回账单不能当普通收入/成本开票。原合并账单 A1,000+B200，A 更正为 100：旧发票 1,200 由财务整张红冲后，旧账单可重开内容只剩 B200，新账单 A100，共 300。账单历史总额仍为 1,200，发票行与关联金额要支持此明确投影，不能强制等于原账单头总额。
- 不新增通用部分红字发票平台。已有未红冲发票时，展示待处理票据与具体动作；有效开票占用必须避免同一经济内容重复开票。

## 4. 提成净额与实际执行

- 已确认/已付提成按原员工、身份、计算版本、比例、来源及费用快照重算，不能套用当前方案。尚未计提的来源消费更正后的经济投影。
- 手工调整与费用系统调整独立：计算费用更正净差额时，不把手工奖励/扣减当作待纠正错误。
- 已执行提成和调整永不改写。尚未执行的旧费用系统调整由专用更正流程保留取消事实后按净差额重建，防止同时挂着应追回 100 和应补付 90，迫使财务先收再付。
- 原父单未发放时，新调整进入既有净额发放；父单已付时，新增调整由财务执行补付或追回。批准不是支付完成。
- 连续补录、曾经封顶归零、从亏损恢复盈利，都按当前目标与已执行费用系统净额核对；不能简单按本次费用差额乘比例，也不能直接反转原冲减全额。
- 来源回款份额必须能关联原费用及其替代链。部分回款按已确认的真实已收金额重算；全收后优惠的经济实现额必须不超过有效收入，退款、内部清偿与后续真实新回款不能重复贡献收入。具体来源分配和事件重算与账单行余额共用同一算法，完整合同见以下追加的最终算法；不能仅添加“排除反向账单”过滤后宣称正确。

## 5. 契约与事务边界

新增 OrderFeeCorrectionRequest（不可变前后快照、原批准/内容版本、理由、幂等键、申请人、决策与 version）及对应 Proto 的列表、提交、预览、批准、驳回、撤回。费用、账单、提成调整补充明确来源关联；提成计提快照补费用经济版本与参与计算的本位币金额。具体字段以追加的最终数据合同为准，源 Schema/Proto、正式迁移及生成物同组交付。

批准事务复用统一 Transactor/WithTx；授权 User/Membership/Role 父行在 Order 前，订单集合统一排序，随后申请、账单/来源、提成父单和调整按现有全局锁序合并。每条写入路径必须列实际锁序并检查相邻建账/发放/核销路径，不能仅凭此概述新增相反顺序。预览令牌包含经济版本、相关账务及提成事实，锁内重算不一致返回冲突；费用、更正结果、审批快照、提成调整、审计和通知同事务提交。

## 6. 影响范围与明确不做

必须同步：order_fee/review/supplement、费用台账、单笔/批量建账、账单历史保护、对冲/核销及反向动作、发票投影、提成候选/首次生成/确认/月申/发放/来源撤回、自动锁单、结案投影及对应前端。

更正待审或尚有应建未建调整时，不得触发新的自动锁单；已存在业务锁不自动解除，受控更正仍可使用。已结案状态不随财务待办改变。第五条人工完整性判断保持现状。

不做自动退款/追回、自动开票、跨单位跨币种抵销、多级审批、通用财务调整平台或历史兼容双读；不重置开发数据库。

## 7. 最终数据字段与接口

### A. 最少持久字段与不变量

`OrderFee`新增五字段，普通费用夹具继续有合法默认值：

| 字段 | Ent契约 | 语义 |
| --- | --- | --- |
| economic_effect | Enum(INCREASE,DECREASE), Default(INCREASE), Immutable | 金额正数；经济方向不变，INCREASE加/DECREASE减。 |
| source_fee_id | UUID Optional Nillable Immutable | 冲回/替代指向原费；普通/ADD为空；沿链找到空即根。 |
| correction_request_id | UUID Optional Nillable Immutable | 专用批准生成行的来源，旧普通/补录空。 |
| superseded_by_request_id | UUID Optional Nillable | 原行批准替代后写一次，不能再次作为目标。 |
| accounting_required | Bool Default(true) | 原无有效bill时原行及冲回false；已有bill原行及冲回true；ADD/替代恒true。 |

沿用原`version`；原fee批准后只变替代关联/是否需建账与version，不改金额、对象、方向等原值。角色由字段组合推导，免增role列：DECREASE是冲回；INCREASE+source非空是替代；INCREASE+source空是普通或ADD。CHECK要求DECREASE有source和correction，source非空有correction，DECREASE不能被替代，source不能等自身；同订单及无环由锁内校验。`(correction_request_id,economic_effect)`唯一（NULL普通多行合法）；`source_fee_id WHERE economic_effect='DECREASE'`唯一。来源FK NO ACTION，普通删除不得删除任何更正链参与行。

新增`OrderFeeCorrectionRequest`使用IDMixin/TimeMixin：

- organization_id/order_id不可变UUID；operation ADD/REPLACE/VOID不可变；source_fee_id、source_fee_version可空不可变。ADD来源空；REPLACE/VOID来源有且version>0。
- approved_fee_review_event_id、approved_fee_content_version、approved_fee_review_version不可变，绑定提交时合法完整批准。
- requested_by/requested_at不可变；reason必填 MaxRuneLen(1000)；idempotency_key MaxLen128、request_fingerprint MaxLen128不可变。
- before_fee_snapshot、after_fee_snapshot为可空JSON不可变领域快照：ADD前空后有；REPLACE前后有；VOID前有后空。金额/汇率/税费快照按现有领域编码，不保存Ent/Proto对象。
- status PENDING/APPROVED/REJECTED/WITHDRAWN默认PENDING；version默认1；decided_by/decided_at/decision_reason可空，仅一次终态决策。
- approval_result_snapshot可空JSON，批准时固化经济、提成和相关财务事实。生成费用按其correction_request_id反查，免增结果费用ID持久列；申请详情返回derived IDs。
- 唯一(organization_id,idempotency_key)；部分唯一order_id WHERE status=PENDING；动作与快照完整性CHECK与Ent同源。跨补录表互斥必须Order锁内统一检查。

`FinanceBill`新增purpose NORMAL/CORRECTION_REVERSAL默认NORMAL；请求不用传purpose，后端按fee effect推导。NORMAL替代费也按根链单独分组，更正链不得与无关新费混成一张新bill。旧合并bill保留。BillLine继续既有OrderFeeID关联不可删除来源，不新增必填root。可按生成规范补经济来源快照，但不能建立另一套可写来源真相。

`FinanceNetting`新增nullable correction_request_id；内部清偿用途由服务端来源验证后写入，普通为空。`FinanceNettingAllocation`新增nullable source_fee_id，内部清偿明确指定本bill中的原/冲回/替代fee，普通空。避免给整bill贴标签后侵占无关行。`FinanceCommissionAdjustment`新增nullable source_fee_correction_request_id，具体提成计算字段见专项合同。

### B. Proto接口与准确载荷

`OrderFeeService`追加CreateOrderFeeCorrection、ListOrderFeeCorrectionRequests、GetOrderFeeCorrectionRequest、PreviewOrderFeeCorrectionApproval、ApproveOrderFeeCorrection、RejectOrderFeeCorrection、WithdrawOrderFeeCorrection。路由为`/api/v1/orders/{order_id}/fee-correction-requests`，详情`/{id}`，预览`/{id}/approval-preview`，终态操作`/{id}/approve|reject|withdraw`。响应沿用success/code/message/data/trace_id，分页沿用page>=1、1<=pageSize<=200。

所有RPC注解AUTHENTICATED，用例实际授权：ADD要求品类fee.create；REPLACE/VOID要求fee.update；审批依system.finance.fee.review及真实组织grant；本人仅撤回PENDING。详情/列表资格为该品类fee.read、申请本人或合法审核人，沿用补录逐行授权，不能只判断已登录。

精确创建载荷候选：

```proto
enum OrderFeeCorrectionOperation {
  ORDER_FEE_CORRECTION_OPERATION_UNSPECIFIED = 0;
  ORDER_FEE_CORRECTION_OPERATION_ADD = 1;
  ORDER_FEE_CORRECTION_OPERATION_REPLACE = 2;
  ORDER_FEE_CORRECTION_OPERATION_VOID = 3;
}
message OrderFeeCorrectionTarget {
  OrderFeeDirection direction = 1;
  string fee_setting_id = 2;
  string settlement_party_id = 3;
  string billing_unit_id = 4;
  string quantity = 5;
  string unit_price = 6;
  string currency = 7;
  string expense_date = 8;
  optional string note = 9;
  optional string exchange_rate_override = 10;
  optional bool tax_inclusive = 11;
}
message CreateOrderFeeCorrectionRequest {
  string order_id = 1;
  OrderFeeCorrectionOperation operation = 2;
  optional string source_fee_id = 3;
  optional uint64 source_fee_expected_version = 4;
  OrderFeeCorrectionTarget target_fee = 5;
  string reason = 6;
  string idempotency_key = 7;
}
message ListOrderFeeCorrectionRequestsRequest {
  string order_id = 1; int32 page = 2; int32 page_size = 3;
}
message GetOrderFeeCorrectionRequestRequest { string order_id = 1; string id = 2; }
message PreviewOrderFeeCorrectionApprovalRequest {
  string order_id = 1; string id = 2; uint64 expected_version = 3;
}
message ApproveOrderFeeCorrectionRequest {
  string order_id = 1; string id = 2; uint64 expected_version = 3;
  string preview_token = 4; string comment = 5;
}
message RejectOrderFeeCorrectionRequest {
  string order_id = 1; string id = 2; uint64 expected_version = 3; string reason = 4;
}
message WithdrawOrderFeeCorrectionRequest {
  string order_id = 1; string id = 2; uint64 expected_version = 3;
}
```

ADD要求来源ID/version均空、target有；REPLACE要求来源ID/version与target均有；VOID要求来源ID/version有、target空。target所有数量单价正数，金额由服务端计算，零目标请使用VOID。方向错误允许替代的新direction不同；冲回保留旧direction。reason/comment最多1000Unicode；驳回reason必填，亏损批准仍意见必填。

RequestData返回id/order_id/organization_id/operation/status/version、source_fee_id/version、before_fee/after_fee只读OrderFee快照、reason、requested_by/name/at、decided_by/name/at/reason、derived reversal_fee_id/replacement_fee_id、can_approve/can_withdraw/approver_available、financial_status。ApprovalData返回request、base_currency/current与projected收入成本利润/profit_change/profit_rate可空、commission_impacts、按对象/币种/方向分组financial_impacts、preview_token。全部金额decimal string。服务端返回对象负责预览净额，不能依赖前端浮点自行做审批金额。

OrderFeeReviewData增加has_pending_correction及合并pending-change投影；OrderFee返回effect/source/correction/superseded/accounting_required和derived can_bill/can_correct。旧has_active_bill仅真实账务占用，不能把免建账历史强置为true骗原门禁。

## 8. 账务操作约束

### C. 三动作与财务阶段

- ADD：生成INCREASE正常费，source空、需建账。
- REPLACE：原fee必须当前INCREASE且未替代、version一致；写superseded_by，创建全额DECREASE冲回+一个INCREASE替代，source均指原费。原无有效bill则原/冲回免建账；原有DRAFT/CONFIRMED有效bill则原/冲回需建账。替代恒需建账。
- VOID：同REPLACE但没有新替代行，禁止创建金额0费用；原无账无需新账，原有账须建反向，原未结部分对冲、已付部分实际退款/追回。后续重新收费走ADD；不能撤销已APPROVED更正、解冻旧行或物理删除。

| 费用 | 经济direction | effect | 资金bill direction | purpose |
| --- | --- | --- | --- | --- |
| 正常应收/替代应收 | RECEIVABLE | INCREASE | RECEIVABLE | NORMAL |
| 正常成本/替代成本 | PAYABLE | INCREASE | PAYABLE | NORMAL |
| 应收全冲回 | RECEIVABLE | DECREASE | PAYABLE | CORRECTION_REVERSAL |
| 成本全冲回 | PAYABLE | DECREASE | RECEIVABLE | CORRECTION_REVERSAL |

普通修改/删除/拆票转移拒绝DECREASE、已替代及更正链参与行。单/批量建账共同检查需建账、是否可建、合法整票经济批准、无任何PENDING；用途由来源推导。已替代原行即使其原bill后续取消也永不能普通重建。

批准前原DRAFT可按原规则确认或取消，批准锁内决定原是否有效占用；批准后原DRAFT允许确认、拒绝取消。已CONFIRMED原账只允许沿正常资金/票据流程保留历史，原账取消一律被更正依赖拒绝。反向/替代账尚无有效核销/对冲/开票时可显式取消重建，经济批准不撤销、需建账不清零；新替代行再次被更正后，其bill成为该步原账，同样保护。

已有原invoice：批准保留其状态；财务按原权限与版本整张红冲/草稿取消；新有效经济内容不得在原票仍占用时重复开。发票候选按有效正常经济行，CORRECTION_REVERSAL不入普通票；红冲不取消受保护原账。

### D. 对冲端点服务端推导合法链

保留CreateFinanceNetting请求所选bill+expectedVersion+note+idempotencyKey，不加客户端可写purpose、skipCommission或任意指定source_fee_id。锁内读取选中账单行费来源，沿source找到根及对应APPROVED申请。每次内部清偿只涉及一个合法更正步骤：反向费可以对冲其source原费，或者同一申请的替代费。所有参与必须同组织、对象、币种，不能以同root为由跨步骤随意配债，也不能混入其他正常费用获得计提排除标签。

系统确定correction_request_id与每allocation的source_fee_id，只消费对应行额度。仅选原+反向或反向+替代均可，不强制三账同时选择。旧bill虽可含其他行，内部allocation只能明确指向当前合法源费；新反向/替代账单分组按链隔离。混入无关账单/多条更正步骤/不完整来源拒绝并提示分别对冲，不默认为更正或通过普通标签逃过计提。

现有PlanFinanceNetting增加领域只读EligibleAmount，在真实bill剩余额基础上按本步骤目标行余额限额；bill.TotalAmount等历史字段不改。对冲金额min(本次两方合法额度)并按ID稳定分配。确认时锁内重算同一来源和额度，不相信草稿创建时快照。普通无更正对冲按原商业规则；一旦涉及更正反向账就必须经过来源闭合验证。

### E. 单一账单行结算投影（无新贡献表、无缺表回退）

全部账务余额与提成共同消费同一纯领域算法，输入为bill line初始正数金额、完整核销/对冲原事件及反转事实、内部allocation.source_fee_id、关联经济替代时间。Go建议对象：

```go
type FinanceBillLineSettlement struct {
  BillID, BillLineID, FeeID uuid.UUID
  Direction OrderFeeDirection // 资金方向
  TotalAmount, SettledAmount, RemainingAmount decimal.Decimal
  Contributions []FinanceSettlementContribution
}
type FinanceSettlementContribution struct {
  SourceType string // VERIFICATION / NETTING
  SourceID, AllocationID uuid.UUID
  EffectiveAt time.Time
  Amount decimal.Decimal
  IsCorrectionSettlement bool
  ReversedAt *time.Time
}
```

按有效发生时间+源ID+allocationID稳定重放。核销原事件时间created_at；对冲原事件必须confirmed_at，因为现有DRAFT创建时已有allocation但尚未生效，不能用allocation.created_at提前分摊。反转在parent.reversed_at产生单独撤销事件。只取当前active会丢历史：应包括ACTIVE/REVERSED核销原事件、CONFIRMED/REVERSED对冲原事件；DRAFT/CANCELLED未确认对冲不产生结算。

原事件将allocation Amount按当时合法行Remaining比例RoundBank(8)，bill line按ID升序，最后一行得到amount减已分配的尾差；不得超过该行余额，整体输入非法则业务错误不钳零。不拿前端float。内部更正allocation按指定source_fee_id定点消耗，不散到别行。

算法在内存缓存原allocation的精确line分摊。反转按该缓存原分摊逐行释放，标原Contribution.ReversedAt，不重新按当时比例分配撤销；随后既有事件的发生顺序不改。例：付款V1后付款V2，后来反V1，V2依旧保留当时A/B原归属；不能先剔除V1再让V2多分A。既有原金额/生效/反转时间及确定性规则是证据，免加持久贡献表。

更正批准后原混合bill中已替代费尚有未结余额时，普通核销/普通商业对冲拒绝该原bill并提示先完成指定费内部抵销；真实资金仍可以先登记cashflow留未核销，不伪造资金。内部抵销把旧目标费Remaining清至0后，原bill可继续普通核销其尚有效未结行。这条局部门禁使普通操作不用替用户选择被废费/退款链，代码成本最低；所有后来事件只分配合法Remaining>0普通行。

必验例：旧bill A1000+B200，实收400；普通分摊A333.33333333、B66.66666667。A更正100，原bill余额800，但原A可内部抵销只有666.66666667；不能把整800抵到A冲回。原bill随后剩B133.33333333；反向A剩333.33333333，与新A100抵销后应退233.33333333。后续B收133.33333333只归B，A不被二次分摊。整个内部对冲不生营业收入/提成。

注：反转先前事件导致后续已执行内部抵销额度大于该步骤复原后合法余额时必须阻断反转并给出依赖对冲，先显式反转后续依赖。不可重排历史或静默削减分摊。已有确认/已付提成来源保护继续依专项合同校验。

### F. 待审、结案、自动锁与事务

补录与更正创建均Order锁下跨表检查，最多一个PENDING。待审阻止新建账、相关提成确认/发放及结案；CLOSED后仍可提交/批准/驳回/撤回，既有业务锁不自动解除。accounting_required=false历史不算未建账；批准后反向/替代需建未建阻止新的自动锁。VOID不能因收入归零跳过至少有效结清事实的原自动锁条件。

经济内容、原fee版本/替代事实、批准基线、预览提成/财务事实不一致返回409并保留PENDING。授权父锁User→Membership→Role先于Order，随后申请/相关bill/fee/提成父与调整按统一顺序，动作实际锁序由主design记录。费用、更正申请终态、系统提成调整、完整新批准快照、审计/outbox一次提交；重复请求返回原结果，不重复冲回或计提。

## 9. 提成确定算法（Q1确认后的权威实施合同）

本章取代前文来源按全部贡献比例分额度的候选。用户已确认真实已收金额口径：R1000/K400/实收500/10%，优惠至R800后目标25。沿用全额经济冲回+新费用重立；不冻结原收入覆盖比例。

### 9.1 唯一账单行清偿投影

财务与提成共用 `biz.FinanceBillLineSettlement`，由会计worker实现；提成不另做静态原账单比例分摊。

```go
type FinanceBillLineSettlement struct {
    BillID, BillLineID, FeeID uuid.UUID
    Direction OrderFeeDirection // 资金方向
    TotalAmount, SettledAmount, RemainingAmount decimal.Decimal // 账单原币
    Contributions []FinanceSettlementContribution
}
type FinanceSettlementContribution struct {
    SourceType string // VERIFICATION / NETTING
    SourceID, AllocationID uuid.UUID
    EffectiveAt time.Time
    Amount decimal.Decimal // 本allocation归属本账单行的原币金额
    IsCorrectionSettlement bool
    ReversedAt *time.Time
}
```

实现对全部原发生和反转事件稳定重放：核销发生用created_at，商业/内部对冲发生用confirmed_at（不能用草稿allocation.created_at）；同一时刻按事件类型、来源UUID、allocation UUID排序，反转使用reversed_at作为后续释放事件。普通发生按**当时账单行剩余金额**比例分摊，经济更正对冲按source_fee_id定点清偿；原allocation→line归属在重放内存中固定，反转准确释放原分配，不使后来的既有事件重新分配。保留已REVERSED来源原发生及反转时间，不能查询时直接删掉所有历史事件。尾差采用会计统一的RoundBank(8)+稳定末行补差，禁止提成自己另建另一舍入口径。

原A1000+B200合并账收400，投影先分A333.33333333、B66.66666667；A更正100后内部对冲定点清偿旧A剩666.66666667。随后旧账收133.33333333全部归B，不能再次按1000:200分到已清偿A。历史400分摊不改。

更正旧账尚未完成必要定点清偿时，拒绝在该旧账新增普通核销，允许先登记未核销资金；完成后普通分摊只分给仍有合法剩余的普通经济行。此保护由会计投影负责，提成只消费已通过财务准入的行归属。

### 9.2 经济根、对象和币种桶

沿正常/替代费的source_fee_id链确定 `EconomicRootID`；链首正常费是根，冲回行不独立建根。新增费用是新根。链必须同Order且无环；更正可纠正经济方向，是否形成营业收入依据发生时原费用经济方向和当前有效叶子判断；Root查找结果可批量缓存，不新建通用根实体或事件平台。

经济应收根f的当前叶子给出 `CurrentFeeID/SettlementPartyID/Currency/Amount/BaseCurrency/CalculationBaseAmount`。桶键为(root,settlementParty,currency)，当前叶子匹配的桶才有当前应收经济额度，其余历史桶上限0。换对象/币种不能自动把旧对象旧币现金当新对象新币回款；旧退款与新真实回款分别执行。成本根无收入额度，追回成本即使账单资金方向RECEIVABLE也不入计提收入。

`CalculationBaseAmount`是明确的有效经济计提基数：普通原费用沿用其有效正常账单行的冻结本位币基数，未建账时费用快照；受控补录/更正生成的新经济内容使用提交/批准固化的经济基数，不因后来资金账单汇率改变而二次重算已批准费用影响。当前费用经济helper及初次计提必须统一这条规则。原行仍保存原R/K/率/计算版本；不能用当前市场汇率修改历史快照。

### 9.3 真实贡献、退款与配额

只从9.1行归属构造真实贡献，不从账单头Amount重新猜。

1. 正贡献：有效正常营业应收行上的真实应收核销，或purpose=NORMAL商业对冲的应收行归属。每份记录(source kind/id,allocation id,bill line id,root,party,currency,effectiveAt,native amount)。旧原费用虽已被替代，其真实正贡献仍跟随同根、同对象、同币桶保留。
2. 内部purpose=CORRECTION_SETTLEMENT全部贡献恒0；旧→反向→新清偿不生成第二份正贡献，不消耗或增加真实贡献。继承通过原根查询已有来源，不复制新source。
3. 退款贡献：营业应收根对应CORRECTION_REVERSAL应付行上的真实PAYABLE核销，或者NORMAL商业对冲清偿该应付行。按9.1归属追到同根/对象/币种；冲回成本的收款不当退款，也不当收入。
4. 真实正贡献、退款发生及它们的反转按同一稳定时间规则重放。每次退款在该桶中已有的未退正贡献之间按余额比例分配，使用同一8位尾差函数；缓存退款allocation→原正贡献扣减份额，退款反转恢复原份额，不把后来的新回款分给更早退款。先退款后新收款时只新收款增加新可用份额。
5. 原正来源反转移除该源原贡献；若尚有依赖它的有效退款，使该桶任一真实来源净余额负数或总退款超过真实入账，则拒绝本次反转并要求先反转关联退款核销。不制造负回款、不静默挪给别对象/币种。反转发生与拒绝都不受订单业务锁/CLOSED影响。
6. 当前根原币可计提收入 `Q=min(CurrentAmount,max(0,Σ同桶净真实贡献))`。对净贡献按(原发生EffectiveAt, SourceType, SourceID, AllocationID, BillLineID)升序依次占额：`q_i=min(net_i,remainingQ)`，扣减remainingQ，来源份额是其所有q_i之和。先来的真实来源先占额，过量后来的来源份额0；客户多付不能让旧提成被挪到新超额来源。
7. 原币q按当前有效经济快照折本币：`sourceBase=q/CurrentAmount*CalculationBaseAmount`；按稳定贡献顺序分摊根的整体base目标，统一尾差确保 ΣsourceBase=Q/CurrentAmount*CalculationBaseAmount。CurrentAmount=0时所有配额/base=0，禁止除零。跨币种只在各原币桶内分配，最后在同一组织本位币汇总，不把USD与CNY原币裸加。

例如两来源各收500，应收优惠至900：按来源先后分配500、400，总实现900；同为10%且成本100，总提成80。原收1000优惠至900，再多收100：旧源仍900，新超额源0。原R1000/K400/收500优惠R800：q仍500，成本分摊250，利润250，提成25。优惠审批降低经济上限；退款实际执行扣净贡献，已全收1000→优惠900→退款100时实现额始终900，不重复减少。

### 9.4 初次计提与已存在父提成

对每个真实商业来源、员工与岗位，r为9.3给该source在所属订单全部应收根的base配额。R/K取当前有效经济费用的统一基数。计算仍按原V3逐步骤Round(8)，原CONFIRMED/PAID使用各自原rate/basis/employee/role，不读当前方案；尚未计提才按既有来源日期方案解析。`r=0`或`R=0`的目标直接0；亏损按零；收入模式目标r×原率，不受K影响。

新来源r=0不生成空壳提成候选；r>0而利润0可沿既有零金额提成规则。初次行固化经济版本、R/K、配额证据，后续补录/更正从当前经济投影计算目标，不再把全部历史补录累加到已经包含补录的R/K中。

现有DRAFT只允许通过显式重新生成得到新原快照：经济指纹改变后不得确认旧金额；月申仍按已提交快照拒绝批准，先退回后显式重提并重新生成未确认来源。原CONFIRMED/PAID行不可原地重算，使用下节调整。

### 9.5 一次净调整计划

复用本文件4/5节公式，但费用系统集合统一包括LOCKED_FEE_SUPPLEMENT、FEE_CORRECTION、ECONOMIC_REALLOCATION。有效源的新理论T来自9.4；C0是原行。人工M独立保留，负目标按订单行和父单预算封顶，DRAFT减提仅预留预算。目标系统累计A*与**已实际系统累计净额**对账，绝不恢复此前未扣掉的封顶额。

审批或真实来源变化若目标发生变化：专用取消本行所有未执行系统调整（保留确认/取消历史与本动作来源），不动PAID调整；新CONFIRMED调整金额=abs(A*−F_paid)，符号按差额。相同目标与当前系统累计一致时零写调整，只保留审批/动作证据。父CONFIRMED净发放，父PAID真实补付/追回。

**真实来源反转是同一个计划，不能再执行旧reconcileCommissionsForSourceReversal之后重复重算：** 被反转源未付父单，取消其未执行调整与父单，保留原来源与月申历史；被反转源PAID父单保持PAID，真实目标0，取消其所有未执行调整，按原行付款证据 `lineCash=C0+Σ该行PAID有符号调整` 创建待追回lineCash（0不创建）。此处来源失效同时失去该来源未执行人工加减资格；人工已实际执行也进入真实现金追回，但不改历史。校验各行lineCash≥0且ΣlineCash等于commissionCashExposure；不猜缺付款证据。其他仍有效源因配额变化，继续按原率和人工分离公式重算。统一锁内计划计算全部旧/新系统调整，既有反转helper改为调用此计划，避免两套追回叠加。

退款等导致有效源份额变0而源未反转：父单仍存在且原人工调整保留，按费用系统目标处理，不把它误当来源反转的整单取消。

### 9.6 共享事务与幂等触发

不新建事件平台。现有核销Create/Reverse、商业/内部对冲Confirm/Reverse在自己的共享事务中完成资金分摊，再直接调用同一 `ReconcileCommissionEconomics(txCtx, org, actor, orderIDs, trigger)`；来源反转事件先写active/reversed_at，再在提交前按新的完整投影生成一次计划，失败整体回滚。CORRECTION_SETTLEMENT通常目标不变，但仍复核投影；真实退款PAYABLE核销也会触发，不仅RECEIVABLE核销。

```go
type CommissionEconomicTrigger struct {
    Kind string // FEE_CORRECTION / VERIFICATION_CREATE / VERIFICATION_REVERSE /
                // NETTING_CONFIRM / NETTING_REVERSE
    ID uuid.UUID
    Version uint64 // 已完成该动作后的实体version
}
type CommissionSourceQuota struct {
    SourceType string
    SourceID, AllocationID, BillLineID, EconomicRootID uuid.UUID
    SettlementPartyID uuid.UUID
    Currency, BaseCurrency string
    EffectiveAt time.Time
    NativeNetAmount, NativeQuota, BaseQuota decimal.Decimal
}
type CommissionEconomicOrderProjection struct {
    OrderID uuid.UUID
    FeeContentVersion uint64
    BaseCurrency string
    Receivable, Payable decimal.Decimal
    Quotas []CommissionSourceQuota
    Fingerprint string
}
```

核心已定义的 `FeeCorrectionCommissionRepo.PreviewCorrectionImpact(ctx,org,order,request,currentFees,projectedFees)` 返回逐行target/current/difference及Fingerprint；`ReconcileCorrectionImpact(ctx,org,order,actor,request,impact)` 在共享事务锁内重算后执行净计划。提成worker给普通补录一个同算法adapter，不再保留旧仅成本边际算法。内部方法建议 `LoadCommissionEconomicProjection`（调用唯一行清偿投影）、`PlanCommissionEconomicReconciliation`（纯Go）、`ApplyCommissionEconomicReconciliation`（Ent/审计）。

新调整FEE_CORRECTION关联申请；ECONOMIC_REALLOCATION关联触发的source_verification_id或source_netting_id，含真实退款PAYABLE核销。幂等键为sha256(kind+id+version+commissionID+orderID)加短前缀，不溢出128字符；来源字段数据库CHECK与真实类型对应。普通动作重试先复用既有幂等资金事实，不能重复执行改变目标；最终调整本身唯一键兜底。旧“parent/order/source_verification_id唯一”仅对原反核销来源适用，新事件可发生create与reverse不同版本，必须修改为按新幂等键去重。

锁序：需员工资格快照的初次生成/确认先User；费用审批先授权User→Membership→Role。随后先解析受影响全部Order，包括本资金事件账单的订单及同根来源涉及的父单所有订单，按UUID一次FOR UPDATE；再申请（费用更正时）、Cashflow（核销时）、当前来源、Bill按UUID、BillLine/费用按UUID、Commission父按UUID、CommissionLine按UUID、Adjustment按UUID。不得在持Bill/Cashflow/Commission后追加Order，也不得在持Order后为重算原快照再锁员工User。现有核销Create的Cashflow→Bill入口必须移到先Order；NettingConfirm不能先锁来源再回头锁Order。确认提成已有Order→来源→Bill路径沿用相同序。授权与经营范围锁内重读，所有repo经Data.client(txCtx)。

使用Order锁稳定整个根及共享父单目标，不需新增可写配额状态/version镜像。预读定位与锁内源集合不一致即409重新开始，不能增量追加未排序订单。费用、请求、整票批准、调整作废/新建、来源状态、审计/outbox在同一事务，失败不留半笔资金或半笔提成。

### 9.7 最薄持久字段和实施回归

沿用core新增字段：adjustment的source_fee_correction_request_id/source_netting_id/superseded_by_fee_correction_request_id及FEE_CORRECTION、ECONOMIC_REALLOCATION；line的fee_content_version_snapshot、realization_snapshot JSON。realization_snapshot保存本次9.3全部参与配额证据与经济根/币种/原币/base/排序/指纹；原JSON不用于猜当前费用版本。更正申请commission_impact_snapshot固化计划及封顶。真实清偿投影用既有allocation、confirmed/created/reversed_at重放，不新建通用事件或分摊表。

正式迁移从原费用/账单/有效与反转分摊确定性生成经济根和原证据；只生成可证明数据，缺失则明确UNAVAILABLE及原因码，禁运行时“缺字段按当前比例猜”。开发样例同步新字段，不重置库。初次计提/工作台/月申所有调用共用新投影和原规则解析，来源候选不把内部对冲、成本追回当新提成。

必须验收Q1=25、全收优惠90→80、共享账A更正后后续收款仅归B、退款及退款反转恢复、原来源有退款依赖时反转拒绝、两来源额度500+400、后续超额源0、换对象/币种旧贡献不跨桶、连续补录跨零/封顶恢复仅实际差、CONFIRMED/PAID/OFFSET_WITH_PARENT四种执行证据、手工-80不抹除、源反转单一追回、来源回款后旧父单净调整、同刻稳定顺序/8位尾差及共享事务竞态。
