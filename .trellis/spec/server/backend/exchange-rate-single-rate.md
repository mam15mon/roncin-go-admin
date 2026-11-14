# 公司周汇率契约

## 1. 适用范围

本规范约束公司汇率的维护、导入、同步、解析，以及订单费用、账单、资金流水、发票等折本币调用。汇率归属公司，部门共享所属公司配置；系统工作台不维护业务汇率。旧公共基线行与单据快照保留为历史事实，不参与新业务取值。

## 2. 签名

```go
// internal/biz/exchange_rate.go
ResolveRate(ctx context.Context, organizationID uuid.UUID,
    direction OrderFeeDirection, currency, targetDate string) (ResolvedRate, error)
FetchExchangeRates(ctx context.Context, organizationID uuid.UUID,
    target ExchangeRateSyncTarget) (*ExchangeRateSyncPreview, error)
SyncExchangeRates(ctx context.Context, organizationID, actorID uuid.UUID,
    target ExchangeRateSyncTarget, rows []*ExchangeRateSyncRow) (int, string, string, error)

// internal/data/exchange_rate.go
ResolveContext(ctx context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error)
ResolveRate(ctx context.Context, companyID uuid.UUID, direction OrderFeeDirection,
    fromCurrency, toCurrency, rateDate string) (ResolvedRate, error)
```

`ExchangeRateContext` 只含 `OwnerOrganizationID` 与 `BaseCurrency`，不含公共 pivot 币种。业务用例先校验单据访问权，再按单据所属组织解析公司；不能把当前会话组织当成单据归属。事务内汇率读取使用 `Data.client(ctx)` 与 `ForShare()`。

## 3. 存储与接口契约

`exchange_rate_settings.organization_id` 对新写入必须是公司 ID；可空列和旧 NULL 行暂时保留，不新增公共行。`from_currency` 是业务原币，`to_currency` 是公司本币。`effective_from` 锚定业务日期所在自然周周一 00:00:00（Asia/Shanghai）。同公司、同币种对、同周重复导入或同步按现有幂等 Upsert 处理。`rate` 是基准价，`ar_rate`/`ap_rate` 分别用于应收/应付，均按既有精度规则保存。

**汇率精度口径固定 4 位小数**：人工输入、Excel 导入、牌价同步落库的校验正则，中间价与交叉盘推导（`RoundBank(4)`），账单/批量账单快照固化统一按 4 位收敛；存储列保持 `numeric(18,8)` 字符串透传，不做列迁移。折本币金额（`BaseCurrencyAmount` 等）仍按 8 位金额口径舍入，与本约束无关。

维护 API 从服务端 principal 取得组织并解析所属公司，列表只返回公司行；按记录 ID 编辑、停用、读取导入批次时仍需公司范围校验。`system.finance.exchange_rate.read` 用于查看，`create` 用于新建、导入和同步，`update` 用于编辑配置及单据显式手工输入，`disable` 用于停用。独立 `override` 权限退役，旧授权不能代替 `update`。系统工作台即使直接请求 API 也不得读写业务汇率。

新解析仅查目标自然周的本公司启用行。相同币种自动解析返回 `rate=1`、来源 `SYSTEM`；显式手工输入同币种汇率也必须等于 `1`，非 `1` 返回 `EXCHANGE_RATE_SAME_CURRENCY_INVALID`，不得静默纠正。该规则按单据所属公司的实际本币判断，不能硬编码 CNY；应收取 `ar_rate`、应付取 `ap_rate`，空值沿用 `rate`。命中行快照携带设置 ID，来源为 `WEEKLY` 或 `BOC_SYNC`。不存在上周继承、公共直连、公共历史回溯或公共交叉套算。外部牌价预览可计算交叉报价，它不读取数据库公共行。跨组织资金流仍按原币记账。

`MANUAL` 只代表有显式手工输入权限者输入；不能静默生成公司周配置。保留既有单据的手工快照不等于新手工输入，更新其他字段时不得因此要求重新输入，也不得允许借旧来源改价。旧 `INHERITED_LAST_WEEK`、`DERIVED`、公共行 `SYSTEM` 来源仅供已有快照展示；不删除枚举或历史 SQL 约束，也不重算已保存金额。

**订单费用汇率的解析与手工覆盖口径**（2026-10 起）：

- 订单费用（应收与应付）折本币汇率按订单贸易方向取日期：**出口订单取 ETD、进口订单取 ETA**，换算为业务时区（Asia/Shanghai）自然日后定位所在周；解析日期不再是费用发生日期（发生日期字段已退役，录入时间以 `created_at` 为准）；费用快照的 `exchange_rate_date` 记录该业务日。取值日缺失或非法时分别返回 `ORDER_ETD_MISSING`（出口）/`ORDER_ETA_MISSING`（进口），前端提示先补对应日期；方向非法返回费用参数错误。统一入口 `biz.OrderRateDay`。
- 当周无本公司汇率时返回 `FEE_EXCHANGE_RATE_MISSING`（`ErrExchangeRateMissingForDate`，与 `ErrExchangeRateMissing` `errors.Is` 互通），元数据：`currency`（缺失币种，大写）、`week_start`（目标自然周周一，`YYYY-MM-DD`）、`direction`（费用方向）。
- **日期保存联动**（`internal/data/fee_rate_refresh.go`）：订单取汇率日（出口 ETD、进口 ETA 所在业务日）发生变化时，同一事务内按新取值日重取费用周汇率。触发入口：订单保存（`UpdateDraft`，含单票 SE 内联改航程日期）、共享海运航程更新、空运主单更新、改配、拆票（原票改配与新票改配）。只改非基准日期或同一业务日内改时分不触发；任一侧日期缺失视为变化，使新取值日不可用时显式失败。
  - 可更新费用：来源不是 `MANUAL`（`SYSTEM/WEEKLY/BOC_SYNC/DERIVED/INHERITED_LAST_WEEK` 均视为系统汇率）、当前有效（未被替代、非内部 `DECREASE` 冲回）、锁内无受保护财务事实（有效发票、核销、对冲，判定见 `fee_protected_fact.go` / `feeHasProtectedFinanceFact`）。已建账但无保护事实的费用同样更新，账单行汇率与折本币快照不动。更新汇率、取值日、来源、设置 ID、折本币金额，费用 `version+1`、订单 `fee_content_version+1`，写 `order.fee.exchange_rate_refresh` 审计，并经 `reconcileOrdinaryFeeEconomics` 重算提成（保留已付事实）。
  - 缺汇率整体失败：先解析全部目标汇率，任一缺失返回上述专用错误，调用方事务整体回滚，日期（订单列、TE、主单）与费用都不落库；不需更新的费用不要求汇率存在。
  - 共享航程/主单改期同步写回成员订单 `etd/eta` 列并令订单 `version+1`，再逐票判定是否联动（各票按自身方向取 ETD 或 ETA）。改配与拆票把目标 TE 的日期写入订单列，同样按 `orderRateDayChanged` 判定；拆票中新票费用先整行克隆（沿用来源汇率），再按新票目标日期联动。退关恢复不写订单日期（要求运输执行与订单一致），无需联动。
  - 锁范围：联动沿用调用方已按主键升序持有的订单锁，只再锁费用行（订单 → 费用行升序），不锁账单、发票、核销、对冲，不得补锁订单。提成经济重算会写入关联父单涉及的全部订单：若提成关联闭包超出调用方已锁订单集合，联动 fail-closed 返回 `ErrOrderStatusConflict`（409），不中途补锁、不放宽该检查；拆票/改配后订单集合变化较大时该情形可能更常见，需用户刷新后重试。
- 订单费用中的手工汇率覆盖使用独立权限 `system.finance.order_fee.exchange_rate.override`（「调整订单费用汇率」），与维护公司周汇率的 `update` 权限分离；且仅当订单内应收费用币种集合与应付费用币种集合一致且均非空时允许（`orderFeeCurrencySetsMatch`），否则返回 `ORDER_FEE_EXCHANGE_RATE_OVERRIDE_MISMATCH`。
- 账单、资金流水等其他折本币调用的权限与日期口径不变（账单按账单日期、资金流水仍用 `update` 权限）。流水页面复用 `access.canUpdateExchangeRates`，不得误用费用专属 `canOverrideFeeExchangeRate`；服务端继续独立校验权限。
- 发票确认按单据方向解析周汇率：应收开票取 `ar_rate`，应付收票取 `ap_rate`；不能把全部发票视为应收。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 系统节点、禁用公司或异常组织链解析汇率 | `ErrExchangeRateOrganizationInvalid` |
| 无维护权限或目标不属于本公司 | 拒绝读写；记录 ID、批次 ID、预览 token 均需校验 |
| 新写入的组织为空、非公司或批量混公司 | 拒绝写入，不产生 NULL 公共行 |
| 当周本公司有效行缺失，只有过去、未来或公共行 | `ErrExchangeRateMissing`，提示「请先维护汇率」，依赖自动折算的提交停止 |
| 当周同公司同币种对命中多条 | `ErrExchangeRateConflict`，不得任选一条 |
| 账单、流水等调用无汇率维护 `update` 权限却提交手工汇率 | 服务端拒绝；旧通用 `override` 不替代 `update` |
| 订单费用缺专属汇率覆盖权限，或应收/应付币种集合不匹配 | 按费用专属权限与币种条件拒绝，维护 `update` 不替代费用覆盖授权 |
| 已固化单据汇率快照 | 保留原值与来源，不随周配置修改而重算 |

## 5. 正常、边界与错误场景

- 正常：两家公司在同一周配置不同报价，各自单据按归属公司取值。
- 边界：部门成员在获授权的公司业务范围使用公司汇率；历史业务按其业务日期所在周取值。
- 错误：目标周缺失但上周、未来周或旧 NULL 公共行有价，仍返回缺失，不继承、不套算。
- 错误：没有对应调用方所需权限仍伪造 `MANUAL`；订单费用检查专属覆盖权限，其他调用检查维护 `update`。只隐藏前端输入框不足以构成授权。

## 6. 必要验证

- 公司 A/B、部门和系统工作台覆盖查询、维护、导入批次与同步预览的范围隔离。
- 手工汇率：100 CNY（公司本币）填 7 必须拒绝，填 1 与合法外币率允许；非 CNY 公司同样验证。费用新增/修改复用同一解析校验，未改汇率的历史快照保留。
- 应付 100 USD，`ar_rate=7.2` / `ap_rate=7` 时收票本币为 700；应收为 720。流水入口覆盖仅有费用覆盖权限/仅有汇率维护权限的差异。
- 自然周精确命中、历史开口行不跨周、应收/应付方向、同币种恒等、非 CNY 公司。
- 当周缺失但存在前后周或公共直连/交叉价时，断言缺失错误及前端提示。
- 无权限伪造手工值、有权限显式输入、旧 `MANUAL` 快照保留、事务内并发改价。
- 导入和同步重复执行仍幂等；旧单据快照不被重算；跨组织结算继续原币记账。

## 7. 错误与正确写法

```go
// 错误：用会话组织取业务单据的汇率。
resolved, err := uc.ResolveRate(ctx, sessionOrganizationID, direction, currency, date)

// 正确：先验证单据访问权，使用其所属组织；用例解析到公司和公司本币。
resolved, err := uc.ResolveRate(ctx, document.OrganizationID, direction, currency, date)
```

写入时同样应以 principal 解析出的公司为归属，不接受客户端传来的公司 ID 直接决定目标。维护、导入和同步复用现有事务、锁和审计机制，不引入新的回退分支。
