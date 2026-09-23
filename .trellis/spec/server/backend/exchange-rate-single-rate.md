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

维护 API 从服务端 principal 取得组织并解析所属公司，列表只返回公司行；按记录 ID 编辑、停用、读取导入批次时仍需公司范围校验。`system.finance.exchange_rate.read` 用于查看，`create` 用于新建、导入和同步，`update` 用于编辑配置及单据显式手工输入，`disable` 用于停用。独立 `override` 权限退役，旧授权不能代替 `update`。系统工作台即使直接请求 API 也不得读写业务汇率。

新解析仅查目标自然周的本公司启用行。相同币种返回 `rate=1`、来源 `SYSTEM`；应收取 `ar_rate`、应付取 `ap_rate`，空值沿用 `rate`。命中行快照携带设置 ID，来源为 `WEEKLY` 或 `BOC_SYNC`。不存在上周继承、公共直连、公共历史回溯或公共交叉套算。外部牌价预览可计算交叉报价，它不读取数据库公共行。跨组织资金流仍按原币记账。

`MANUAL` 只代表有 `update` 权限者显式输入；不能静默生成公司周配置。保留既有单据的手工快照不等于新手工输入，更新其他字段时不得因此要求重新输入，也不得允许借旧来源改价。旧 `INHERITED_LAST_WEEK`、`DERIVED`、公共行 `SYSTEM` 来源仅供已有快照展示；不删除枚举或历史 SQL 约束，也不重算已保存金额。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 系统节点、禁用公司或异常组织链解析汇率 | `ErrExchangeRateOrganizationInvalid` |
| 无维护权限或目标不属于本公司 | 拒绝读写；记录 ID、批次 ID、预览 token 均需校验 |
| 新写入的组织为空、非公司或批量混公司 | 拒绝写入，不产生 NULL 公共行 |
| 当周本公司有效行缺失，只有过去、未来或公共行 | `ErrExchangeRateMissing`，提示「请先维护汇率」，依赖自动折算的提交停止 |
| 当周同公司同币种对命中多条 | `ErrExchangeRateConflict`，不得任选一条 |
| 无 `update` 权限却提交手工汇率 | 服务端拒绝；仅有旧 `override` 权限同样拒绝 |
| 已固化单据汇率快照 | 保留原值与来源，不随周配置修改而重算 |

## 5. 正常、边界与错误场景

- 正常：两家公司在同一周配置不同报价，各自单据按归属公司取值。
- 边界：部门成员在获授权的公司业务范围使用公司汇率；历史业务按其业务日期所在周取值。
- 错误：目标周缺失但上周、未来周或旧 NULL 公共行有价，仍返回缺失，不继承、不套算。
- 错误：无 `update` 权限者伪造 `MANUAL`；只隐藏前端输入框不足以构成授权。

## 6. 必要验证

- 公司 A/B、部门和系统工作台覆盖查询、维护、导入批次与同步预览的范围隔离。
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
