# 财务账单固定币种边界

## 1. Scope / Trigger

本规范适用于普通账单的批量预览、批量创建、已确认账单更新、费用维护、账单行快照和前端建账工作台。
新增或修改费用分组、账单币种、自动创建日期汇率、结算账户或预计开票字段时必须遵循。

项目采用固定币种边界：不同费用币种永远拆成不同账单，不建设费用原币到可选目标账单币种的
转换子系统。这样避免跨币种金额被错误相加，也避免把预计开票汇率误用为账单余额汇率。

## 2. Signatures

`BillGroupingPolicy` 只包含模式与可选拆分维度：

```text
mode: NORMAL | NETTING
split_by_tax_rate: bool
split_by_order: bool
```

预览的叶子配置接收：

```text
group_key
settlement_account_id
estimated_invoice_currency
estimated_invoice_rate
```

创建时每个叶子附带预览返回的 `group_token` 和该组完整的 `fee_ids`。
`CreateBillRequest`、`UpdateBillRequest`、`BillBatchPreviewGroupConfigInput`、`CreateBillBatchGroupInput` 不接收 `bill_date`，原字段号及名称保留为 reserved。
`FinanceBill.bill_date` 与 `BillBatchPreviewGroup.bill_date` 是只读业务创建日（`YYYY-MM-DD`），前端展示名为「创建日期」；`created_at` 独立保留首次真实创建时间。
`CreateBillRequest`、`UpdateBillRequest` 与 `CreateBillBatchGroupInput` 接收可选的 `payment_terms_days`（0–3650）和 `due_date`（`YYYY-MM-DD`）。

普通账单头的币种关系固定为：

```text
bill.currency = leaf.fee_currency
bill.exchange_rate = rate(bill.currency -> organization.base_currency, bill.bill_date)
bill.base_currency_amount = bill.total_amount * bill.exchange_rate
```

## 3. Contracts

- 组织、收付方向、结算单位、费用币种和组织本位币始终参与普通账单分组；税率和订单是可选细分，建账工作台每次打开均默认启用，用户仍可关闭。
- API 不定义 `split_by_currency`，也不接受目标账单币种、费用原币到目标币种汇率或人工覆盖。
- `group_key` 必须包含费用币种；服务端创建日期跨日变化、修改账户或预计开票配置不得改变 `group_key`，但必须轮转预览令牌。
- 结算账户必须属于同一组织和结算单位、已启用、用途匹配且账户币种等于叶子费用币种。
- 新建与预览日期由服务端 `FinanceBusinessDate(now)` 统一取北京时间当天，客户端不计算、填写或暗送日期。初次预览即返回只读日期，账户未配置时仍为不完整分组，不签发创建令牌；配置齐全后按服务端日期解析账单到本位币汇率，币种相同时固定为 `1`。
- `bill_date`、`created_at` 和汇率快照在建账时确定。编辑和增减费用均沿用已保存的 `exchange_rate`、`exchange_rate_source`、`exchange_rate_date`、`exchange_rate_setting_id`。编辑保持账单头与明细的本位币金额，预计开票金额按原币合计和独立的预计开票汇率计算。
- 没有费用锁定事实时，增减费用在原事务内更新关联和金额，保留 `bill_date`、`due_date`、`payment_terms_days` 和汇率快照。所有有效行用账单汇率重算本位币，最后一行吸收舍入尾差；预计开票金额随原币合计重算。已到期的账单可以继续按相同规则增减费用。
- 建账和编辑可按账期天数推导到期日，也可以直接指定不早于创建日期的日期。两字段同时存在时，到期日须等于账单日期加账期。建账时两字段均省略则按已有默认账期规则带入。编辑按表单当前方式提交账期或指定日期，保存原有的指定日期时也须提交 `due_date`。
- 有效发票（含部分登记）、核销、对冲及免票继续沿用费用锁定事实，禁止增删。受保护账单允许编辑备注和抬头，无账期时还可单独修改到期日；原账期和其他受保护字段须保持原值。撤销相关事实后按当前事实与权限重新判定。
- 预览令牌保持 64 字符，内部前 8 位为 `YYYYMMDD`，后 56 位为配置/费用快照的 SHA-256 摘要片段；前端只原样传递。成功回放判定之后、新建汇率解析之前，先检查令牌日期，跨北京时间午夜的旧令牌明确返回预览过期，即使当天缺汇率也不覆盖过期错误。
- 批次请求摘要只反映客户端原始意图，不包含自动日期或自动推导到期日。已成功的原键跨日回放读取原账单，不能按当天重建、改日期或误报请求冲突；未成功组的旧令牌须重新预览。
- 单张 `CreateBill` 的原始意图摘要随 `finance.bill.create` 创建审计在同一事务写入 `details.create_request_hash`。仅 `GetByIdempotencyKey` 按公司、资源 ID、资源类型、创建动作及成功结果精确读取，避免后续修改日期、费用或默认条款改变重放判定；普通详情与列表不额外查询创建审计。缺少原摘要明确返回幂等冲突，不从当前可变内容猜测或添加历史回退。
- 单张建账锁内复核来源费用的原币金额、方向、币种、归属与税率；账单本币属于账单日派生值，不得要求它等于费用日的本币快照。
- 账单行币种与金额保存来源费用事实；账单头和行的本位币金额使用同一账单日汇率计算，最后一行
  吸收确定性舍入尾差。
- 预计开票币种、汇率和金额是独立预估快照。它们不改变账单币种、账单金额、本位币金额、余额、
  核销或实际发票汇率。
- 创建事务必须先按订单主键排序锁定完整订单集合，再按稳定顺序锁定费用、账户和汇率设置；
  选中费用须当前有效、需要建账、无有效账单占用且自身无待审更正。以同一纯分组内核重算并校验
  每组预览令牌；普通模式每组独立事务，NETTING整批原子。人工订单审核不是建账前置，无关费用与 ADD 待审不阻塞正式费用。公司归属加载不执行组级资格判断，否则一组待审会误阻整批。
- 新建账单直接CONFIRMED，无人工确认入口；更新、费用增删、免票、取消等已有账单操作统一先定位并按主键锁订单，再锁账单和账单行；
  需要费用行锁时最后锁费用。锁内重读来源归属，集合变化返回冲突。普通创建没有已有账单，
  直接走“订单 → 费用”；禁止任何“费用 → 订单”或“账单 → 订单”的反向路径。
- 人工审核锁固化当时整票费用内容版本、指纹与利润快照；账单关联及费用行 version 变化不改写该历史快照。
  锁定后的正式费用仍可建账；单笔、批量建账均在写事务内重新复核逐费准入，不能仅依赖候选列表 can_bill。
- `NETTING` 按同公司、同结算单位、同费用币种配对应收与应付；预览生成配对，同事务建立双方
  已确认原始账单及草稿对冲。对冲仍需另行确认，草稿对冲不提前抵减余额。

## 4. Validation & Error Matrix

| 条件 | 服务端行为 |
|---|---|
| 选中费用被替代、无需建账、已有有效账单占用或自身存在待审更正 | 预览或所属创建组拒绝；普通模式不回滚其他组；未人工审核本身不阻断 |
| 普通模式混入应收和应付 | 允许并按方向拆组，每张账单自身方向唯一 |
| `mode` 未指定或未知 | 显式拒绝，不默认采用普通或对冲模式 |
| `NETTING` 同单位同币种缺少应收或应付一侧 | 拒绝配对，不跨单位或币种拼凑 |
| 不同费用币种 | 必须形成不同叶子，不允许合单 |
| 请求试图提交或修改账单币种 | 正式契约不提供该字段；不得在 Service/Biz 增加旁路 |
| 账户币种不等于叶子费用币种 | 拒绝预览完成或创建 |
| 账单币种到本位币缺少账单日有效汇率 | 拒绝创建，不允许 `1`、`0` 或任意日期回退 |
| 未配置结算账户 | 返回含自动创建日期的固定币种叶子；配置不完整且不得签发可创建令牌 |
| 北京时间跨日后提交旧预览令牌 | 创建组返回预览过期，不生成账单；已成功原键重放仍读取原单 |
| 建账或编辑的显式到期日早于账单日期，或与同时提交的账期推导结果不一致 | 拒绝请求 |
| 预计开票币种不同但汇率为空、非正数或精度非法 | 拒绝该叶子配置 |
| 锁内费用版本、状态、金额、币种或分组事实变化 | 所属group_token冲突；普通模式只拒绝该组，NETTING整批失败 |

## 5. Good / Base / Bad Cases

- Good：同一单位同时有 USD、EUR 费用，服务端稳定返回两个叶子；各自选择同币种账户，按服务端自动创建日期解析到组织本位币的汇率，普通模式逐组事务生成两张账单。
- Base：账单币种等于组织本位币，账单汇率为 `1`；预计开票不填写时默认同账单币种、汇率为 `1`。
- Bad：隐藏前端“分币种”开关，但 API 仍保留 `split_by_currency=false` 或目标币种字段，让调用方
  可以绕过页面形成跨币种账单。
- Bad：使用预计开票汇率重写账单 `exchange_rate` 或本位币金额。

## 6. Tests Required

- Biz：不同费用币种一定生成不同 `group_key`；税率和订单开关只影响各自维度。
- Biz：`NORMAL` 单方向/混合方向均按组成功，未指定/未知模式拒绝；`NETTING` 合法双向配对成功，缺侧或错单位/币种拒绝。
- Biz/Service：所有创建和预览入口按北京时间确定创建日期，覆盖 UTC 与北京时间日界；编辑保持原创建日期。
- Biz/Data：初次未配账户预览返回自动日期且不可创建；同币种汇率为 `1`；外币缺创建日汇率明确失败。
- Biz/Data：跨日旧令牌拒绝新建，批量及单张原幂等键跨日重放不重复创建、不改变原日期；原单后续增减费用后仍须保持创建回放语义。
- Data：跨业务日增费、减费及同时增减保留账单日期、到期日、账期和汇率快照；当天缺少汇率仍按原账单汇率折算头和各有效行。版本冲突、最后一笔移除与锁定事实均零写入，部分发票也冻结增删。
- Data：改值或停用原汇率设置后编辑到期日、备注或预计开票，汇率四项及头、行本币金额保持原值。覆盖尾差归属和受保护账单的指定日期。
- Web：只读展示服务端创建日期，创建/编辑按当前到期方式只提交账期或指定日期；验证显式零、无约定、指定日期保存及重新打开后原样保存。列表/筛选/导出使用同一创建日期名称，已释放行不进入当前明细。
- Biz/Data：账户错组织、错单位、错用途、停用和错币种均被拒绝，创建事务锁内再次校验。
- Biz：预计开票默认值、不同币种显式正汇率及预计金额计算正确，并断言不改变账单金额与本位币。
- Biz/Data：账单头本位币金额等于账单行本位币合计，末行吸收 Decimal 舍入尾差。
- 并发：两个批次争抢同一费用只有一个成功；费用或配置变化后旧令牌不能创建。
- 并发：账单更新、费用增删、免票、取消统一按“订单 → 账单 → 账单行 → 费用”处理，与审核锁定/解锁、费用删除和补审串行。

## 建账附带登记与核销

`CreateBillBatchGroupInput.invoices[]/verifications[]` 只接受已有票据资料、来源费用与资金分配。
新账单 ID、版本、公司、方向、结算对象和币种由本组建账产物填入，调用方不能改挂。

- 普通模式每组执行建账→多票登记→多笔核销/差额登记，共用一个 `WithinTransaction`；任何一步失败整组回滚，其他组独立。NETTING 仍整批原子。
- 逐步校验发票登记、核销创建、资金创建/确认及显式资金汇率权限，不因缺附带权限降为只建账。
- 建账组合与独立差额核销共用全局顺序：完整经济订单闭包（按订单 UUID）→全部既有资金（按资金 UUID）→差额流水编号。独立 `CreateWithCashflow` 必须在创建流水前经 `VerificationRepo.LockBillOrders` 解析并锁齐账单订单闭包，复核集合未漂移，再经 `LockCashflows` 预锁全部非空资金；两个预锁入口只允许共享事务上下文，客户端来自 `Data.client(ctx)`。不能先持有流水编号再等订单或既有资金：同订单不同账单、不同订单共享同一资金都会形成跨入口反向环。F 已持有的订单/资金可重入原锁，不重新扩大范围。
- 内部 `*WithinTransaction` 命令只返回写入/回放身份，完整账单投影在提交后用普通上下文重读，同键回放也读取最新投影。
- 请求摘要包含附带意图，子步骤幂等键由批次、组、步骤、位置派生。响应未知或提交后读取失败保留原意图和键；禁止重新换键或宣称已回滚。
- 普通模式的附带金额解析错误属于组失败，按 `step/item_index/group_key` 定位；不能因其中一组非法金额在边界层提前拒绝其他合法组。非法原始意图也参与稳定摘要。F 边界在 Decimal 比较、转串及摘要前先约束指数和 `numeric(28,8)` 范围，拒绝超八位与巨大指数；非法原始金额只按原始字符串做 JSON 摘要，禁止展开。核销普通/差额路径的分配和合计均须为正且可无损保存为八位，禁止靠 `StringFixed(8)` 静默舍入。
- NETTING 整体失败用冻结提交的分组快照将 `group_key` 映射为结算单位、方向、币种，再显示步骤与记录序号；页面不显示内部组摘要。
- 前端复用共享发票编辑器，核销输入只作草稿；最终只调用批量创建 RPC。多行增删/改值沿用表单输入保护，搜索保留已选资金快照。
- 建账资料默认折叠预计开票、同次登记发票、同次核销三个可选分节。折叠保留字段、输入和校验，提交失败提示首个具体校验错误并切到对应账单。

定向验证覆盖多票多笔、无发票核销拒绝且无差额流水遗留、部分开票后核销、既有资金加差额、同键并发、第二步失败无遗留、缺权限、普通组隔离和 NETTING 整批回滚；另以真实数据库屏障验证同订单不同账单及不同订单共享资金的跨入口锁序，并验证九位金额整组回滚、合法八位原样保存。见 `finance_bill_composition*_integration_test.go`、`finance_verification_amount_test.go`、`settlement_bill_operations_test.go` 及 `BillVerificationFields.test.tsx`。
- Web：没有分币种开关和账单币种编辑器；导航始终包含币种层；提交契约不携带目标币种。

## 7. Wrong vs Correct

```text
错误：隐藏日期控件，但仍由浏览器 today 作为 billDate 提交；重试时把今天加入请求摘要。
正确：服务端按北京时间派生业务创建日，请求摘要中的日期信息来自客户显式提交的到期设置。

错误：增减费用按当天推进到期日并查询汇率，编辑备注时重新取账单日汇率。
正确：增减费用用已保存汇率在原事务内更新关联与头行金额，完成提成经济重算及审计。编辑沿用已保存汇率。
```

### Wrong

```text
policy.split_by_currency = false
group.currency = user_selected_currency
line.amount = convert(fee.amount, manual_pair_rate)
```

### Correct

```text
raw_group_key += fee.currency
bill.currency = fee.currency
bill.exchange_rate = resolve_bill_date_rate(fee.currency, organization.base_currency)
```

账单币种由费用事实决定，创建日期由服务端在建账时记录。用户可设置同币种账户、到期日及独立的预计开票值。

## 对冲汇总与余额的有效口径

### Convention: 对冲只计已确认，余额是派生值

**What**：财务汇总中的对冲抵销金额只统计 `status = CONFIRMED` 的对冲单；账单可用余额
= 总额 − 有效核销 − 有效对冲，全程实时派生，不落库冗余余额字段。

**Why**：草稿对冲未生效即扣减余额会提前阻塞正常核销；反转/取消通过对冲分摊
`active = false` 与状态翻转自然退出有效金额，无需双写恢复。列表汇总共享列表筛选谓词
（组织、单位、币种、关键词、状态），与「有效金额只统计有效记录」的口径一致。

**Example**：

```text
bill.available = bill.total
             - sum(verification_allocations where active)
             - sum(netting_allocations where active)
netting summary amount = sum(nettings where status = CONFIRMED)
```

**Related**：对冲预览与计划共用 `biz.MaxFinanceNettingBills` 上限（当前 500），
超限 fail-closed 返回 `ErrFinanceNettingTooManyBills`，不静默截断预览总额。

### 费用台账付款进度与对冲的边界

- `FeeLedgerFinancialProgress` 表示关联整张账单的开票和实际收付款状态。付款事实
  取有效收付款核销（分摊 `active=true` 且父核销单 `ACTIVE`），对冲不能单独
  生成“部分付款”或“已付款”；开票只取活动关联且父发票 `ISSUED`。
- 有效核销为零时，无论对冲多少均为未付款；发生有效核销后，核销加有效对冲
  未覆盖账单总额时为部分付款，覆盖后为已付款。对冲金额继续独立展示。
- 例：账单300、对冲200、核销0 → 未付款且余额100；核销50 → 部分付款且
  余额50；核销100 → 已付款且余额0。账单200完全由对冲抵销、核销0时仍为未付款。
- `ResolveFeeLedgerFinancialProgress`、筛选 SQL、列表与详情投影须一次同步，
  共用只读快照；不得只改前端文案。金额列、余额和可核销候选继续沿用前述余额公式。
- 回归必须同时验证进度筛选/总数/完整筛选汇总、行投影与订单详情，覆盖纯对冲、
  混合核销、付款完成、反核销、反对冲及有效/失效发票，前端共享文案与导出同口径。

## 核销单头本位币与对冲两端本位币口径

### Convention: 核销不是交易，不引入核销日汇率

**What**：核销是流水与账单的配对分摊，不产生新汇率。行级三口径：
`cashflow_base_amount`（流水本位币按分摊比例）、`bill_base_amount`（账单本位币按比例）、
`exchange_gain_loss = cashflow_base − bill_base`（应收；应付取反）。单头
`base_amount = Σ 行级 cashflow_base_amount`。核销实体不携带任何汇率快照字段与行级
`write_off_base_amount`。

**Why**：核销日汇率是第三种无经济事实支撑的口径，会让 Σ核销 ≠ 所消耗流水的本位币合计，
报表体系出现不可解释差额（2026-09 审计 B1）。

**Example**：核销 100 USD 消耗某流水的一部分 → 单头本币 = 该流水本币 × (100/流水原币额)，
损益按两端差额沉淀在行级。

### Convention: 对冲单头双端披露，汇差=应付−应收

**What**：对冲单头三个本位币字段：`base_currency_amount`（**应收侧**抵销本位币，语义固定）、
`payable_base_amount`（应付侧，≥0 CHECK）、`exchange_gain_loss = payable − receivable`
（可负；同币种同汇率对冲为 0 属正常）。单笔计划与批量建账两路径同口径分侧累加，各自
`RoundBank(8)`。草稿确认须在账单锁内用当前账单快照重算每笔分摊、双端本币合计和汇差，
再激活分摊；账单按各自创建时确定的汇率折算。Ent 金额字段允许该次更新，已确认后的不可修改性
由状态门禁保障，重复确认不重算历史金额。

**回归**：AR/AP各100 USD、建账汇率均为7。草稿期间调整汇率设置后编辑应收账单，确认对冲仍为
700/700、汇差0。确认时校验当前余额、方向、单位、币种和版本，失败零写入。

**Why**：对冲是双边债务消灭，只披露应收侧会让应付端本位币无处安放、汇兑损益无字段承载
（2026-09 审计 B2）。

**Related**：普通费用的提成收入及分母使用账单行 `BaseCurrencyAmount`（账单日汇率）；未建账普通费用使用费用自身快照。受控补录/更正费用使用批准时的费用经济汇率快照，后续建账日汇率不得再次改变提成基础。收入归属须消费统一账单行清偿与经济来源投影，见 [费用更正契约](./order-fee-correction.md) 与提成分母测试。

### 对冲预览候选数量按未结账单计算

`financeNettingRepo.LoadPreview` 按公司、结算单位、账单币种和 CONFIRMED 筛选，并在
`LIMIT MaxFinanceNettingBills + 1` 之前应用 `billUnsettledPredicate()`。未结定义为
总金额不等于有效核销与有效对冲之和（候选仍只处理正数余额），不能用确认状态代替结清状态；500 张上限只限制可参与
候选，不能先装入已结清历史再报超限。真正 501 张未结仍返回 `ErrFinanceNettingTooManyBills`。

正式迁移回归覆盖 500 张已清历史加两张未清、真正 500/501 未清边界、核销/对冲/混合结清、
撤销分摊后恢复候选及公司/单位/币种隔离。预览公开参数无日期或账单 ID，不得把历史耗尽上限
归因于用户没有继续缩小筛选范围。

### Convention: 费用删除按账单占用判定，账单行快照独立于来源费用

**What**：当前有效订单费用的普通删除是物理删除；订单须 ACTIVE、未人工审核锁定，并通过删除权限及版本校验。
账单占用以「存在未取消账单（含草稿）的活动账单行」为唯一占用判定（`orderFeeRepo.Remove` 事务内复核，冲突码 `ORDER_FEE_BILL_OCCUPIED`）；已取消账单
的历史行不阻断。`finance_bill_lines.order_fee_id` 可空，来源费用删除时由外键
`ON DELETE SET NULL` 置空，行上费用代码/名称/数量/单价/税额/币种快照独立支撑历史展示；
业务标签及其费用关联已[正式退役](./business-tag-retirement.md)；费用删除不再参与标签关联清理。历史迁移 20260922130000 保留，不作为当前写路径契约。

**Why**：用户澄清「取消账单后费用可删，不能因曾建过账永久禁删」（2026-09-22）；同时历史
账单及其金额快照必须原样保留，故来源引用改可空而非级联删行。

**Example**：费用已入账单 → 先按实际依赖与权限回退并取消有效账单；订单 ACTIVE 且未人工审核锁定时，
当前有效费用可按权限删除。仍锁定时先解锁或走费用补审，不恢复独立费用批准或提成财务锁。
账单行保留快照且 `order_fee_id` 为空。删除与建账并发在订单及费用行
`ForUpdate` 上串行，不会出现账单引用已删除费用（集成测试
`order_fee_delete_integration_test.go`）。

被替代的原费及内部 `DECREASE` 冲回属于受保护历史，不允许普通修改/删除。未被再次替代的当前有效替代费用，
在解锁、未建账且通过普通权限/版本门禁后可修改或删除；更正来源本身不形成永久冻结。普通删除同事务执行提成经济重算，保留已发事实。

## 场景：免确认、逐组建账与费用维护

### 1. 适用范围

新建、编辑、增删费用、免票与批量建账。已确认是内部有效状态，不要求业务人员执行确认操作。

### 2. 签名

```text
PreviewBillBatch.data[].group_token
CreateBillBatch.groups[] = { group_key, group_token, fee_ids, ...叶子配置 }
CreateBillBatch.results[] = { group_key, success, bill? | error? }
UpdateBillFees = { id, expected_version, add_fee_ids, remove_fee_ids }
SetBillInvoiceExemption / RevokeBillInvoiceExemption = { id, expected_version, reason }
FinanceBill = { settlement_progress, invoice_progress, completed, business_status, fees_locked, fee_lock_facts, invoice_exemptions[] }
```

实际字段以Proto为准；批次头仅持有身份，费用数、账单数及base_totals按已有账单读取聚合。

### 3. 契约

- HTTP 200与顶层success只表示请求已处理，普通批量必须逐组读取results；成功含bill、失败含error，严格覆盖本次所有组。失败组可单独重试，成功组同键回放；结果未知时冻结原意图同键重试，禁止编辑后冒充重试。
- 普通组不同方向可共批；逐组令牌只覆盖该组费用与配置，其他组变动不得误伤。NETTING保留整批原子。续做失败组的确认人与审计取本次实际actor，不能借批次创建人。
- 费用增删在订单→账单→账单行→费用锁内再次校验同公司/方向/币种/对象、当前来源及未占用，至少留一条。更正归属由当前活动行的来源费用重建；普通编辑也只校验活动行，停用历史行可能已无来源费用，不能参与本次写入。
- 发票、核销、已确认对冲和有效免票属于受保护事实；草稿对冲也会阻止费用增删与整账取消。免票与有效发票互斥，独立权限system.finance.invoice.exempt，原因必填，撤销保留历史并推进账单版本。
- 账单列表与详情按当前页账单 ID 批量投影 `can_cancel`。状态允许取消、存在有效明细、有效明细的来源费用均存在且未被更正替代，并且无活动发票关联、活动核销分摊、草稿或活动对冲、有效免票时才返回 true。写入继续锁内核对同一组条件并返回具体回退提示。
- 结算净额＝有效核销＋有效对冲（带符号）；普通需票完成＝结清且票齐，免票完成＝结清。SQL筛选与ResolveFinanceBillProgress同源。零额/负数本期只验证纯判断，不放开创建约束。
- 决定的revoked_at/by/reason要么全空，要么全有；CHECK必须显式写revocation_reason IS NOT NULL，不能靠length(NULL)>0阻止缺失。

### 4. 错误矩阵

| 触发 | 结果 |
| --- | --- |
| 陈旧expected_version | 409，保留输入，不借探测的新版本重提 |
| 费用已有任一锁定事实 | 拒绝增删，提示对应事实；不隐式撤销 |
| 移除最后一条 | 拒绝，释放全部请取消账单 |
| 有有效发票时设免票，或有效免票时开票 | 整次拒绝，不留下互斥事实 |
| 撤销免票但仍有核销/对冲 | 保留其他事实保护 |
| 普通一组失败 | 其他成功组保留；NETTING同情形整批失败 |
| 确认/审计任一步失败 | 所属事务整体回滚，版本与关联不提前变化 |

### 5. 正常／基准／错误场景

正常：同批应收USD成功、应付CNY失败，响应两组明细；基准：单组按旧键重试回放同一账单；错误：把HTTP 200当全成功后关闭失败输入，或为普通重试更换全部幂等键。

### 6. 验证

真实PG覆盖逐组/NETTING/正式前态升级、并发争用、更正来源重读、历史费用移除并删除后编辑、免票标记/撤销/互斥/审计回滚/空值CHECK；前端覆盖部分成功、逐组缺字段、未知结果冻结、旧会话预览与写入迟到、取消确认保留输入。对应实现与用例见finance_bill_batch*、finance_bill_correction_scope_integration_test.go、finance_invoice_exemption_integration_test.go及BillCreationWorkbench*。

### 7. 错误与正确

```text
错误：await CreateBillBatch(); 直接清全部输入；confirmed_by = batch.created_by
正确：校验results逐组结果；只处理已成功组；confirmed_by = 本组实际actor

错误：编辑账单遍历全部历史Lines加载源费；CHECK length(revocation_reason)>0
正确：仅活动Lines参与写入；撤销分支同时要求revocation_reason IS NOT NULL且非空
```

该实现让同一账单混入多个费用币种，并额外制造默认币种、组合汇率、人工覆盖和审计分支。

## 场景：账单五态筛选与核销开票前置

### 1. 适用范围

账单列表／详情／CSV、核销账单候选及新增核销（独立、差额资金、建账同次）。2026-11-18 用户明确要求限制完全未开票核销，替代此前允许先核销后开票的规则；不追溯改写既有账务，不限制普通对冲的办理顺序。

### 2. 签名

```proto
// settlement.proto；具体字段编号和生成类型以源文件为准。
enum FinanceBillBusinessStatus {
  FINANCE_BILL_BUSINESS_STATUS_UNSPECIFIED = 0;
  FINANCE_BILL_BUSINESS_STATUS_UNINVOICED = 1;
  FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED = 2;
  FINANCE_BILL_BUSINESS_STATUS_UNVERIFIED = 3;
  FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_VERIFIED = 4;
  FINANCE_BILL_BUSINESS_STATUS_COMPLETED = 5;
}
// ListBillsRequest追加 repeated FinanceBillBusinessStatus business_statuses。
// FinanceBill追加 FinanceBillBusinessStatus business_status，只读派生，不落库。
```

HTTP查询重复传 `businessStatuses`（例如 `businessStatuses=1&businessStatuses=2`）；前端消费生成客户端的数组属性，不手拼服务主机。

### 3. 契约

- 有效账单的当前待办以 `biz.ResolveFinanceBillBusinessStatus` 为纯Go真相：已完成优先；否则未开票／部分开票优先于核销待办；票齐或免票且未结清时，实际核销额为零显示未核销，结算为部分且实际核销非零显示部分核销。纯部分对冲不能冒充核销。非有效账单或未经完整进度投影返回UNSPECIFIED。
- `business_status` 为读取投影；页面／导出只查共享元数据。原实体status、双进度和completed各自保持语义，不能以主状态替代原财务事实或权限。
- 多选五态内部OR，与其他条件AND；未知或UNSPECIFIED输入显式拒绝。SQL在Count、LIMIT与聚合之前应用同源谓词，页面固定有效账单，公司范围不受筛选扩大。total、matchedSummary、顶部指标共享谓词；导出取官方onLoad采纳页，不自行循环全量。
- 核销前置：开票进度PARTIAL、FULL或NOT_REQUIRED才具备办理资格。完全未开票且无有效免票时，候选查询在分页前排除，实际写入在现有账单行锁内复核最新有效发票分配／免票事实，返回409 `FINANCE_VERIFICATION_INVOICE_REQUIRED`：“账单尚未开票，请先登记发票后核销”。部分开票可核销，不新增按已开票额限制核销额度的旁路。
- 既有事务与锁序继续适用；多账单任一无票整次回滚。差额流水与核销同事务失败不残留资金；建账同次先登记发票再核销，无票组失败不得留下部分账务。普通组独立和NETTING整批原子性保持。
- 反核销用于解除分配，不因新前置阻断旧事实回退；幂等已成功同键回放不重复写入。撤销发票或免票后提交旧候选必须依据当前事实拒绝，不用旧投影绕过。
- 建账表单无有效发票登记时明确提示并阻止同次核销提交；删除发票行仍保留核销草稿，不能静默丢弃用户输入或自动补发票。无开票权限仍能单纯建账。

### 4. 校验与错误矩阵

| 场景 | 必须结果 |
| --- | --- |
| 未开票＋部分开票多选 | 并集，与单位／方向等其他条件取交集，汇总同源 |
| 非法或UNSPECIFIED筛选值 | 显式参数错误，不静默当全部 |
| 票齐＋仅部分对冲＋无实际核销 | 主状态未核销，仍保留真实对冲金额 |
| 完全未开票直接核销／混合多账单 | 409开票前置错误，所有分配和资金变化回滚 |
| 部分开票、票齐、有效免票＋有剩余余额 | 在原有公司／权限／币种／余额条件下允许核销 |
| 建账＋登记发票＋同次核销 | 按先票后核销顺序提交，任一步失败整体回滚 |
| 无票建账＋差额资金核销 | 所属组失败，账单、资金、分配、占用无部分遗留 |
| 旧候选对应发票／免票已撤销 | 保留表单输入，提交依据最新事实拒绝 |

### 5. 正常／基准／错误场景

- 正常：100元账单先登记20元发票，再核销40元，主状态仍为部分开票；核销额度仍按账单余额校验，不钳至20元。
- 基准：无状态条件列全部有效账单；票齐并结清显示已完成，免票结清同样完成。
- 错误：删除全部登记行后静默去掉核销草稿，或页面隐藏无票核销但直接API仍接受。

### 6. 必需验证

biz五态优先级与核销资格，service重复枚举HTTP绑定及ProtoJSON投影；真实PG对照投影与SQL匹配集合，覆盖单态、多态OR、其他条件AND、分页／汇总／公司／取消隔离、纯对冲与带符号核销。核销覆盖无票拒绝、部分票／票齐／免票成功、旧候选失效、多账单原子性、差额资金及同次建账失败回滚、幂等和反核销。前端覆盖多选请求、隐藏清值／恢复／重置、CSV服务端状态、建账删除发票后输入保护及提交阻断；相关入口是finance_bill_progress、finance_verification、finance_bill_composition的定向用例。隔离PG使用正式迁移，SKIP不算通过。

### 7. 错误与正确对照

```text
错误：ProTable拿到一页后再filter业务状态，汇总仍来自未筛选全集。
正确：businessStatuses传入ListBills，在共用SQL谓词中筛选，返回分页结果及完整同条件汇总。

错误：根据TS可选businessStatus缺失回退前端重算，或仅前端判断可核销。
正确：只映射服务端投影；候选服务端过滤，最终核销在原有账单锁内复核有效开票／免票。
```
