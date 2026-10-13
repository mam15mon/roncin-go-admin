# 研究：费用更正的经济内容、账务义务与开票边界

- 研究问题：已批准费用如何追加、改金额或换结算对象，同时保留原审批、账单、收付款与开票事实；选择最薄且不重复计算收入、成本及提成的模型。
- 范围：仓库内部；只读现有代码与规范，本文件为设计研究，未修改产品代码。
- 日期：2026-10-13

## 研究结论

### 1. 已确认范围与推荐结果

用户已确认业务结案与回款分开，结案后可以受控更正；经济更正由现有费用审核资格批准，实际补收、退款、补付或追回由财务执行。费用完整性继续人工判断。

首期建议每条申请只操作一张订单中的一条当前费用，或追加一条应收/应付。更改金额、结算对象等通过“原费用全额经济冲回 + 新费用重立”表达。原金额不改，账单和资金事实不改；原费用只能被替代一次，下次更正以替代后的新费用为目标。金额和计费表达继续正数，经济冲回必须携带显式来源和 `DECREASE` 语义，不能由普通录入入口任意输入负数。

推荐在既有 `OrderFee` 增加显式经济效果、来源和是否需要建账的字段，以不可变追加行表达更正。这比新增整套“当前费用版本 + 账单覆盖关系 + 独立债权台账”薄，且能够继续消费现有账单、核销、对冲及提成调整。代价是费用的全部经济汇总必须一次性接入同一效果语义；不可仅改补录入口。

### 2. 现有代码证据与文件地图

| 文件与定位 | 当前职责及对设计的约束 |
| --- | --- |
| `server/internal/data/ent/schema/order_fee.go:24` | 费用方向直接表示经济应收/应付，所有金额为正数口径；费用没有状态，原建账事实由关联行推导。 |
| `server/internal/data/ent/schema/finance_bill_line.go:26`、`:64` | 账单行持有费用快照；同一费用仅一条活动账单行，不能将现有费 ID 再次直接建新账。 |
| `server/internal/biz/order_fee.go:841` | 普通录入数量、单价、总额必须正；保留该规则，通过专用更正用例控制经济冲回。 |
| `server/internal/biz/order_fee_review.go:46` | 全票批准快照按费用 Direction 直接加总，必须改为显式经济效果或有效链投影。 |
| `server/internal/biz/finance_bill.go:367`、`:423` | 来源一致性直接比方向、结算对象、币种、金额；账单方向现直接取费用方向。更正冲回行要映射资金方向，并校验来源和账单用途。 |
| `server/internal/data/order_fee.go:496`、`:540`、`:619` | 普通改费只能按策略同步草稿账单；批准冻结仍应阻止普通入口。专用更正不通过临时解锁复用此更新。 |
| `server/internal/data/order_fee_review.go:390` | 有有效账单时撤销整票批准被挡；新更正批准应直接更新完整经济批准快照。 |
| `server/internal/data/order_fee_supplement_approval.go:90` | 补录批准已有 Order 后锁提成父单和调整、同事务生成费用/冲减/批准快照的范本。 |
| `server/internal/biz/finance_netting.go:116` | 正数账单剩余额及同组织/对象/币种对冲可复用；不支持跨对象、跨币种抵扣。 |
| `server/internal/data/finance_commission_calculation.go:54`、`:344`、`:527` | RECEIVABLE 资金账单作为计提来源，核销额/原账单额按行分摊；经济方向直接加总收入成本。必须同时处理更正排除与原来源份额继承。 |
| `server/internal/data/finance_commission_application_candidates.go:69`、`:92` | 月度来源发现只筛应收方向，未区分普通收入与成本追回等清偿调整。 |
| `server/internal/data/workbench.go:277`、`:302` | 工作台预计计提和候选另有同类发现谓词，必须与月度候选同步。 |
| `server/internal/biz/finance_invoice.go:339` | 目前开票整张账单，金额与分组行都来自全部 Bill/Line；更正后的合并账单不能原样重开。 |
| `server/internal/data/finance_invoice.go:158`、`:263`、`:423` | 候选为已确认且无活动开票关联的账单；整张红冲释放所有活动账单关联；当前没有部分红字能力。 |
| `server/internal/data/finance_bill_credit.go:73` | 客户信用债务来自正数应收账单剩余额；成本追回不能被当成新增客户营业应收。 |
| `server/internal/data/settlement.go:499` | 费用台账按方向和本位币 SQL SUM；需要同一经济效果，否则利润与审批不一致。 |
| `server/internal/data/order_auto_lock.go:570` | 任一方向未建账费用阻断，再逐张应收账单判断结清；非建账历史行必须明确排除。 |
| `server/internal/data/order_write_lifecycle.go:155`、`:195` | 结案“未建账费用”查询也必须排除无需建账的经济历史行，并新增更正待审。 |

### 3. 模型比较

| 模型 | 优点 | 本次不可忽略的成本/错误 |
| --- | --- | --- |
| 直接改已建账费用/取消核销再修改 | 表面代码少 | 原账务事实被破坏，仍需拆长财务链；不能满足当前纠错目标。 |
| 把成本减900登记成普通应收900 | 可以用现有资金流程 | 将成本追回误当收入，污染利润、开票和提成；必须拒绝。 |
| 有符号 TotalAmount、BillAmount 全链传递 | 经济公式直观 | 现有正账单、资金核销、余额、来源比例和发票都需大改；没必要将负数传播到资金层。 |
| 当前正数费用版本 + 独立账务调整 | 大部分费用总额汇总不用符号 | 新版本100不能再当未建账费普通建100，需要会计覆盖与来源映射；版本/覆盖/独立债权生命周期更多。 |
| **正数原行 + 显式全额经济冲回 + 正数替代行** | 原值保留；普通资金金额始终正；复用账单、对冲、资金；替代链可读 | 经济金额需统一按效果汇总；未建账原行冻结并免建账；更正内部清偿不触发计提；开票按有效经济行重建。 |

推荐最后一种。字段可以名为 `economic_effect` 而非存负数；领域 helper 输出 `SignedEconomicAmount`。`Direction` 保留经济方向，不将 PAYABLE 成本冲回改成经济收入；建账时由效果映射资金方向。资金层仍保存正数 Amount 与 RECEIVABLE/PAYABLE。

### 4. 建议 Ent 字段与数据库不变量

以下是设计候选，不是已落地字段；主会话应在 design 中统一命名。

**OrderFee：**

- `economic_effect`: INCREASE / DECREASE，既有普通与补录费为 INCREASE。
- `correction_request_id`: nullable UUID，专用批准生成的行才有来源，创建后不可变。
- `correction_role`: ORIGINAL / REVERSAL / REPLACEMENT（普通费 ORIGINAL；新更正追加可用 REPLACEMENT），也可用 effect 与来源组合推导而省字段，但不能从金额/方向猜。
- `source_fee_id`: 冲回与替代行指向被替代的原费用，不可变；原费用仍保存原金额和原补录来源。
- `superseded_by_request_id`: 原费用被批准替代的事实，最多写一次；当前经济行是 INCREASE 且未被替代。已替代费禁止普通操作、建账和再次作申请目标。
- `accounting_required`: bool 默认 true。未建账原费用被更正时，原行与对应冲回行都 false；替代新行 true。已有有效账单的原费用保持 true，对应冲回 true，新行 true。不是“已收款”标志。
- 沿用 `version`。金额、币种、对象与汇率等在批准后不能直接修改；更正原行只变替代关联/建账需求及版本，不改经济值。
- 冲回 `source_fee_id` 加部分唯一约束（每原费一条正式 REVERSAL）；替代行同申请一条；追加没有 source。正数校验与来源/角色/effect CHECK 需 Ent 注解和正式 SQL 同源。

**OrderFeeCorrectionRequest（新增）：**

- ID、order_id、organization_id、operation ADD/REPLACE；source_fee_id 可空，source_fee_version。
- submitted_by、提交时间、reason、PENDING/APPROVED/REJECTED/WITHDRAWN、version、idempotency_key/request_fingerprint。
- approved_fee_review_event_id、fee_content_version、fee_review_version 固化基线。
- before_snapshot、after_snapshot、economic_impact_snapshot、commission_impact_snapshot、financial_impact_snapshot；JSON 使用确定性编码，业务类型仍为纯 Go 对象。
- decided_by/at、decision_reason、新批准事件、reversal_fee_id/replacement_fee_id 等执行结果关联。
- 账务执行状态优先投影关联费用/账单/核销/对冲/票据，避免另建“已支付”可写镜像状态。需要待办时区分“调整未建账”“账单已建”“仍有未结余额”“票据待处理”。

**FinanceBill/BillLine：**

- Bill `purpose`: NORMAL / CORRECTION_REVERSAL，不同用途不混入同一账单组；既有行正常默认 NORMAL。
- 更正反向 bill line 保留原经济方向/effect/source_fee_id/source_bill_line_id（可仅将部分元信息保存在不可变费用并关联读取；涉及来源指纹和历史删除时，行快照至少保留经济效果与来源标识）。账单 Direction 是**资金方向**。
- NORMAL 新替代费仍可普通建账；反向费专用来源校验允许资金方向与经济方向相反。
- 现有 active_fee_unique 不删除、不放宽；每次更正新的费用 UUID，自然满足唯一性。

**FinanceNetting：**

- 更正内部债务抵销需要明确 `purpose=CORRECTION_SETTLEMENT` 及 correction_request_id，普通对冲保持 NORMAL。关联必须限同组织、对象、币种与该更正链；不能让用户给任意普通对冲贴标签绕过计提。
- 不能仅新增 enum 后忽略计提。旧正常应收与反向 payable 抵销是债务取消，不能凭这笔抵销生出营业收入；原已收款与新正常应收重立后的份额转接同样不能再计提。

如果实现选择字段更少的推导方式，必须保持以上不变量，并在不可变来源与锁内查询上可证明；不能使用“缺字段就当普通”的运行时兼容分支。

### 5. 按财务阶段处理

| 原费用阶段 | 批准更正的经济动作 | 财务随后操作 |
| --- | --- | --- |
| 尚未建账 | 原值保留并替代；追加全额冲回、替代新行；原/冲回 accounting_required=false | 只给新费建账。不凭两个免建账行阻止结案/自动锁/提成。 |
| DRAFT 账单 | 默认视为已有账务占用，原快照保留，新增冲回和新行都需建账 | 可确认旧账及更正账后执行对冲；或者财务显式取消原草稿，再按专用流程重建。不能悄悄修改其他合并订单行。首期保留历史原账路线更统一。 |
| CONFIRMED、未付 | 原账保留；全额反向 + 新立 | 旧账与全额反向对冲，保留新正常应收/应付待收付。 |
| 部分已付 | 原流水/核销/对冲保留 | 先对冲旧账未结额，反向账剩余额代表原已付事实；与同对象新账抵销后形成真正差额。 |
| 全部已付 | 原核销不动，反向账全额剩余 | 同对象新账与反向账抵销，剩余退款/追回实际执行；换对象不得自动跨对象转。 |
| 已开票（可叠加任何收付阶段） | 批准可生效经济内容；保留原开票历史并显示待票据处理 | 由财务执行原整张发票红冲；红冲前不让更正后的相关有效内容重复开票。随后按有效经济行重开，所有同发票未改内容也需重新纳入。 |

草稿账单可被确认或取消的竞争必须在 Order+Bill 锁中判断，不能按提交时阶段固定。有效 bill 被取消后重建不能把已替代旧 fee 从禁建名单释放，否则会重复义务。

### 6. 数值守恒（所有表均原币，先不含汇率）

**成本1000→100，原同供应商。** 经济台账为 PAYABLE INCREASE1000、PAYABLE DECREASE1000、新PAYABLE INCREASE100，成本净100，利润提高900。资金方向分别 PAYABLE1000、RECEIVABLE1000、PAYABLE100。

| 原已付 | 旧账未结 | 与反向账对冲旧未结后，反向剩余 | 再抵新应付100 | 最终真实差额 |
| --- | --- | --- | --- | --- |
| 0 | 1000 | 0 | 新应付100不抵销 | 需支付100 |
| 50 | 950 | 50 | 抵销50，新应付剩50 | 需再支付50 |
| 400 | 600 | 400 | 抵销100，反向剩300 | 待供应商退款/抵未来款300 |
| 1000 | 0 | 1000 | 抵销100，反向剩900 | 待供应商退款/抵未来款900 |

审批后应展示成本100和“财务差额待处理”，绝不显示900已到账。

**客户优惠：应收1000→100，原已收400。** 经济收入100；资金原RECEIVABLE1000、反向PAYABLE1000、新RECEIVABLE100。旧余600与反向抵销600，反向剩400；新100抵销后剩PAYABLE300待退款。原400实收流水仍为400，实际退款300后净资金100。

**换结算对象：原供应商A成本1000→供应商B成本100。** A原账未付：A PAYABLE1000对A REVERSE RECEIVABLE1000；B另建PAYABLE100。A已付400：A旧余600抵反向600，A待追回400；B应付100。不能跨A/B自动抵成只需追回300。

**漏录应收100。** ADD仅生成正常RECEIVABLE100，收入+100；没有冲回，没有退款。更正后的已确认提成怎样补提按原规则与实现份额计算，不能直接100×当前比例。

**币种变更**即旧币种原方向全反向、新币种正常新立；不能相减金额，不能跨币种对冲。原冲回沿用旧费原币及经济汇率快照，避免平白出现费用利润汇差；反向账单按账单日资金折本币快照，汇兑损益继续财务现有口径。新费用按批准后实际新快照计算。

### 7. 开票的最薄完整改造

现有 `buildFinanceInvoice` 全额相加 Bill.TotalAmount 并遍历全部 Bill.Lines；`FinanceInvoiceBill.Amount` 是独立快照字段，Schema 没有要求该字段恒等 Bill 总额的数据库约束。可明确变更接口语义为：一次开票覆盖所选账单**当前所有可开票的有效经济行**，金额由服务端投影确定，仍不允许用户任意输入部分金额。

候选排除 CORRECTION_REVERSAL；普通旧 bill 只提供未被更正替代的 NORMAL 正向费用行。新替代费用进入其新正常 bill。投影金额为可用有效行的净/税/总和，零额账单不作候选。FinanceInvoiceBill.Amount 写本次实际票面来源快照，允许低于原财务账单总额。不是修改原 Bill 或老 Invoice。

例：旧合并账单 A1000+B200=1200，原发票1200已开；A改100。

1. 原账1200、原发票1200保留；A反向1000和新A100为新账务。
2. 财务将**原整张1200发票**红冲（含B），不能登记只红900冒充现有能力。
3. 旧账可开票有效内容仅B200，新账A100；选择两账单开新发票300。FinanceInvoiceBill对旧账保存Amount200，对新账Amount100。
4. 若原invoice还合并另张C500，红冲后C500需重开，可与B200/A100一起开800。UI必须提示整张票全部受影响；审批不能自动生成税控号码或宣称红冲已完成。

需要同步：ListCreationBills 查询前的有效行条件/分页总数、LoadBillsScoped、ListProfilesForBill、biz构建与校验、data.Create锁内重新计算行及金额、Issue前经济内容指纹复核。普通无更正账单开票行为仍是整额。红冲/取消旧关联按原现有事务。

**当前开票绑定是 Bill 级 active唯一，足以支持整张已确认更正后重开，尚不足以让同一个旧bill的有效行一部分继续被原invoice占用、一部分被新invoice占用。** 首期明确要求受影响原invoice整体先红冲或草稿取消，不开放任意部分开票；这是产品范围而非静默技术回退。

### 8. 提成：避免资金清偿被重复当成营业实现

两个必修点：

1. PAYABLE成本冲回资金方向RECEIVABLE，必须从月度候选、工作台预计与真正计提引擎同时排除 CORRECTION_REVERSAL，否则追回成本变成营业收入。
2. 旧正常应收与反向payable对冲属于更正内部清偿，不能因为 RECEIVABLE bill 为正常就计提。比如旧1000未收、改100，旧1000与反向1000对冲，若照旧计提会凭空把1000认成收入。原已收1000改100，新正常100和反向payable对冲100，也不能在原收款已承载该100之外再计提一次。

更正内部对冲要有来源闭合的 CORRECTION_SETTLEMENT，并对该对冲禁止新计提机会。更正审批需按原来源及原方案快照调整已有提成；未来新现金核销仍可产生应得新增提成。旧实收已实现份额应明确继承到替代费，账务债务迁移与实现来源迁移分开。

例如收入1000、旧实收400，改收入900：保留400实际收款来源份额，新900账与反向账内部抵销400后尚需收500；内部抵销不新计提，后来真正收500可计提。改收入100：旧来源能承载的收入最多100，退款300不伪造新来源；已有提成应减至对应100的合法计算。多笔来源则需要确定性分摊及最后一项尾差规则，不能各笔都独立 `min(原来源金额,新收入额)` 导致多次占满100。

**该份额继承算法与已确认/已付提成差额不是本文件声称已解决的现成功能。** 主会话应结合提成专项研究给出可执行设计：同一更正链，原合法实现份额 + 后续真实新增实现份额不得超过当前经济收入；内部清偿不能重计；更正前已未计提旧来源在后续候选中必须按更正后有效经济内容复算。仅把原已确认提成补一笔调整，而遗漏未计提旧来源，同样不完整。

### 9. 必须一次性同步的调用方清单

本清单由非生成生产文件检索得到；代码行数会随实施变化。

| 功能 | 文件 | 处理要求 |
| --- | --- | --- |
| 订单费用列表/汇总 | `data/order_fee.go:64,119,984`、`biz/order_fee.go` | 当前有效费用默认列表；历史展开能看原费/冲回/替代；服务端金额按统一经济效果，JSON返回当前金额与源关系。 |
| 整票审批快照/列表 | `biz/order_fee_review.go:46`、`data/order_fee_review.go:105,181,495,625,730` | 指纹涵盖经济效果、不可变来源与替代事实，批准刷新全票快照；建账关联变化不使经济批准失效。不要用费用通用version替代经济指纹。 |
| 补录及其更正 | `data/order_fee_supplement_approval.go:98`、`order_fee_supplement_cancel.go:78,108,191` | 原补录行被替代后不可直接删除；原补录及冲减保留；新更正关联原费/原补录。此前补录合计不能把旧1000与替代100重复叠加。 |
| 费用台账汇总 | `data/settlement.go:98,323,499,506` | 按经济方向+效果汇总；默认当前行与历史筛选语义明确；将资金方向另列。 |
| 普通/批量建账 | `biz/finance_bill.go:367,423`、`finance_bill_batch.go`、`data/finance_bill.go:367,499`、`finance_bill_write.go:34,273`、`finance_bill_batch.go:127` | 禁止已替代旧行及accounting_required=false；用途分组；反向方向映射，锁内校验。 |
| 未建账门禁 | `data/order_auto_lock.go:570`、`order_write_lifecycle.go:155,195`、`finance_commission_write.go:155,286`、`workbench.go:543` | 只统计需要建账的费用；已批准未建更正调整阻止新自动锁/相关提成门禁；待审更正与补录互斥并阻止结案/新建账，已锁不自动解锁。 |
| 提成计算/候选 | `data/finance_commission_calculation.go:54,344,392,409,527`、`finance_commission_application_candidates.go:69,92`、`workbench.go:277,302` | 经济总额有效链/符号口径一致，资金调整不计收入，更正内部对冲不计提，继承原来源份额与避免未计提漏算。 |
| 客户信用 | `data/finance_bill_credit.go:73` | 不把供应商成本追回看成普通客户收入；客户原债务和新债务在抵销前后展示真实往来，不以审批假称已经清偿。 |
| 发票候选/创建/签发 | `biz/finance_invoice.go:339`、`data/finance_invoice.go:158,263,423` | 来源有效经济行投影；全票红冲/取消后重开；历史和资金账不改。 |
| 海运改单/拆票 | `data/sea_order_change.go:135`、`sea_order_change_split.go:175`、`sea_order_change_split_execute.go:562`、`sea_document_change_shared.go:120` | 历史调整行不得作为普通未建账费转移/删除；当前批准费用既有冻结仍生效。只改相关谓词，勿扩大订单变更能力。 |
| 标签/审计 | `data/business_tag.go:307`、各批准审计 | 确认历史可读授权及现行行不可普通改写；费用变化、账务状态和资金执行分别记录。 |
| 前端订单 | `web/src/pages/orders/fees.tsx:158,614`、`components/fees/OrderFeeTableTabs.tsx`、`FeeSupplementSection.tsx`、`features/orders/fee-review` | 默认有效费用正数列表，汇总消费服务端；更正入口/原新对照/影响预览/审批历史；不让前端按资金方向算经济利润。 |
| 前端财务 | `pages/finance/fees`、`pages/finance/bills`、`pages/finance/invoices`、`features/finance/bill-creation` | 账单用途和实际资金方向显式；更正费用建账待办；开票可用额不同于原账额时同时展示，必须使用后端投影。 |

### 10. Proto/UI 最小契约建议

在 `server/api/order/v1/order_fee.proto` 追加更正创建、分页列表、详情/批准预览、批准、驳回、本人撤回接口；业务要求用同一现有费用审核权限，组织范围在用例中复核。创建 ADD 走现有fee.create资格；REPLACE走对应fee.update资格。不能单凭“已登录”准许更正。

请求：order_id、operation、source_fee_id/source_fee_expected_version（REPLACE）、after_fee正数DTO、reason、idempotency_key；订单经济批准基线由服务端固化。批准请求带request expected_version、预览指纹/经济与财务事实版本、opinion、idempotency_key；真正操作仍锁内复核，不信任浏览器计算差额。

预览返回：原费用和拟新费用、按本位币收入/成本/利润前后值、相关账单及开票状态、原已收付事实、预计账务操作和差额（币种分别列）、逐人提成影响、权限与阻断原因。不要将“预计对冲/退款”标成已完成。审批后返回当前批准快照及更正结果引用，前端只在实际成功响应时刷新并提示成功。

订单详情显示“有效费用100”，历史可展开“原1000/冲回1000/替代100/理由/审批人/时间”；财务待办按金额+对象+币种列“需建反向账”“需对冲”“待退款/追回”“待红冲重开”。无需另建复杂可配置工作流。

### 11. 事务与固定锁序

复用 `biz.Transactor.WithinTransaction` 和 `Data.client(ctx)`。创建/批准更正以现有授权父锁 User→Membership→Role 为入口，再 Order；多个Order按UUID排序（若关联账单需全组读锁，先解析并一次排序，不可已锁单票后递增锁更多Order）。在Order锁下重读授权、批准基线、当前费用、互斥申请。

更正批准自身不修改现金、核销、对冲、发票，因此无需拿这些可变父单锁做资金动作。确需把账单阶段/金额固化到“批准结果”时，建议 Order→CorrectionRequest→相关Bill按UUID→BillLine按UUID→Fee按UUID→Commission父按UUID→CommissionLine按UUID→Adjustment按UUID，与现有建账/确认/取消及提成用例保持Order在前。请求、费与提成局部顺序应由主设计对照现有补录锁序统一，不能硬插一个与其他动作相反的新序。

更正提交阻止同Order PENDING补录/更正；后者也需查询前者，事务内Order锁串行化，跨两表无法单靠各自部分唯一索引保证互斥。approve重复请求只能返回同一次既有结果，不能再生成第二冲回/新费/冲减。所有变化和审计/通知outbox一次提交。

**发票重要并发缺口：** 现有开票data.Create先Profile后Bill，不先Order；更正若改OrderFee替代关系、开票只锁Bill，则更正可能在开票锁内计算以后改变经济来源。需要开票创建/签发也参加Order串行化：读取受影响源行解析Order集合→按UUID锁Order→再沿既定开票锁序锁实体，并锁内重算可开票经济行/指纹。不能一边更正Order→Bill、一边开票Bill→Order。redflush仅改原票据关联，可以复用其现行Invoice→Bill锁序，只要更正不持Bill后去锁Invoice；新预览不对Invoice获取FOR UPDATE。若批准阶段需要把发票事实纳入指纹，应做该部分完整锁序审查，不能因读到ISSUED就自动处理。

现有核销Create主要锁Cashflow→Bill，反向核销/对冲既有Order锁机制；更正审批不执行资金操作。财务执行更正内部对冲若要参加Order锁，必须先Order再进入本次专用对冲，不以Bill锁后追加Order。待确认主设计应逐条证明新增共享锁和业务动作无环。

### 12. 迁移、生成与验证路径

这是财务跨域改造，主实现阶段必须维护Ent Schema→正式SQL→Go生成→Proto API→OpenAPI→Web client一致；权限码若不新增仍要验证现有授权映射。不存在生产历史兼容需求，默认值属于正式现有数据一次性迁移，不是双读回退。不得清空开发数据库。

建议正式迁移加入费用经济字段/来源FK/一次替代唯一、申请表、账单purpose和对冲purpose；原全部正常行设 INCREASE/accounting_required=true/NORMAL。CHECK与FK删除策略在Ent owning edge上声明并校验生成元数据。更正历史来源须NO ACTION，不能随着普通旧费用删除被SET NULL失去追溯；普通费用删除入口也应拒绝已参与更正的行。

实施后回归应覆盖：未建账1000→100且原补录已冲减；草稿/确认/部分已付/全付各原账不变；换对象无跨抵；币种分开；合并账单及合并发票全红重开；提现退款未发生前无新资金事实；已替代旧费取消账单后仍禁建；逆向成本收款不进收入候选；内部对冲不新计提；更正前来源尚未计提与更正后实际增收；二次更正新替代费；审批与建账/发放/结案/开票的真实并发；失败审计/费用/提成/批准快照全回滚。

真实PostgreSQL使用隔离DB+正式迁移，区分PASS与SKIP；全部研究期间未运行测试。本文件只列实施验证建议，最终测试执行依主会话授权和当前规则决定。

## 相关规范

- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：费用、账单、开票与提成实体地图。
- `.trellis/spec/server/backend/order-fee-review.md`：整票批准、授权父锁、补录与结案契约。
- `.trellis/spec/server/backend/finance-bill-currency.md`：正数账务、按币种分组及账单汇率快照。
- `.trellis/spec/server/backend/finance-commission-lock.md`：确认/发放净额与系统调整边界。
- `.trellis/spec/server/backend/database-guidelines.md`：事务、实体版本、正式CHECK与FK/迁移同源。
- `.trellis/spec/server/backend/quality-guidelines.md`：生成物与真实DB验收边界。

## 外部资料

- 无外部材料；本结论仅为仓库当前软件能力与建议契约，未声称对实际税务规则做法律判断。外部票据的真实红冲/开具由财务登记。
- 本次未查询生产/远程数据库、未改代码或Schema、未运行测试。

## 限制与未决项

1. 推荐模型是一条最薄路线，但全额重立会增加临时债权债务及财务操作；用户只确认审批与实际资金职责分开，尚未逐条确认全额冲回策略。设计应直观展示旧/反/新与最终差额，不能把原已付1000同时显示成未付。
2. 合并已开发票的更正需要整票红冲后重开；现有能力不支持部分红字。若用户现实业务必须保留原发票只红差额，应另选部分红字契约，不能用现有整红字段假装支持。默认首期选择完整整红/重开并在设计明确。
3. 原实收来源在替代链中的实现份额继承、混合普通与更正对冲、尚未计提旧来源的重算必须由提成专项设计完成。本文明确风险并给守恒要求，没有把新enum误当算法已经存在。
4. 对应整张开票范围中的其他订单必须在数据范围内才能实际整体红冲重开；审批人可批准当前单经济变化，财务实际操作仍须拥有全部相关范围。不能让单票权限自动获得别单经济明细。
5. 待审互斥、已批准未建调整阻断自动锁、CLOSED继续可更正及已锁不自动解锁均为本次确认边界。提成未发/已发更正效果不同，由原合法方案快照联动调整，不采用当前新比例。
6. 真正需要产品选择的是：是否接受首期全额冲回+新重立及整张发票红冲重开；是否需要更正为零/撤销整条费用（当前正数替代只覆盖本次已述金额更改）；缺提成计算快照时可否先经济批准并标明待人工处理，还是应阻断（不能静默吞提成影响）。不应向用户询问已有明确批准的“业务/回款分离”“第五条人工判断”。
