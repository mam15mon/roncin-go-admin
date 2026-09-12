# 汇率单一基准契约

## 1. Scope / Trigger

本规范适用于汇率主数据的 Schema、解析、写入、导入，以及所有按业务日期折算本位币的消费方
（订单费用、账单、资金流水、发票、提成人民币换算）。新增或修改任何汇率字段、解析入口或
消费方调用时必须遵循。

项目采用**单一折本币基准汇率**：全组织树由总部唯一定义与维护一套中间价，不区分单据类型
（无 `RateType`），不区分收付方向（无 `receivable_rate/payable_rate` 双列），不存在时间标准
与继承策略配置表。历史版本曾按 5 类 RateType × 收付双轨建模，已被 2026-09-11 重构推平
（迁移 `20260911160000_exchange_rate_simplification.sql`），禁止回潮。

## 2. Signatures

```go
// internal/biz/exchange_rate.go —— 解析结果携带来源（SYSTEM 直连 / DERIVED 套算）
ResolveRate(ctx context.Context, orgID uuid.UUID, currency string, targetDate string) (*ResolvedRate, error)
ResolveBaseRate(ctx context.Context, orgID uuid.UUID, fromCurrency, toCurrency, targetDate string) (*ResolvedRate, error)

// data 层交叉套算编排（直连优先 → pivot 两腿 → fail-closed）
resolveWithCross: (from → to) 直接行命中返回 SYSTEM；
  否则 (from→pivot) ÷ (to→pivot)，pivot = 树根总部本位币（ResolveContext.PivotCurrency），
  RoundBank(8) 返回 DERIVED
```

```text
Table: exchange_rate_settings
  organization_id  -- 恒为总部（树根）
  from_currency / to_currency
  effective_from / effective_to (nullable)
  rate numeric(18,8)
  is_active
  UNIQUE (organization_id, from_currency, to_currency, effective_from)
```

写入入口仅 `Create` / `Update` / `Disable`；`exchange_rate_custom_settings` 与
`exchange_rate_time_standards` 两张表已物理删除，对应 RPC（UpdateTimeStandards、
Get/UpdateCustomSetting）已从契约移除。

## 3. Contracts

- **归属与写入**：汇率行 `organization_id` 恒为组织树根（总部）。data 层写入前用
  `requireHeadquarters`（复用 `headquartersOrganizationID` 助手，同费用科目模式）校验调用组织
  即总部；**禁止**重新引入"分公司写入重定向到总部行"的逻辑——两者互斥，重定向即越权通道。
- **解析**：`currency == 组织基准币种` 直接返回 `1`；否则先查直连行（`from → to`，总部行，
  `effective_from <= t < effective_to` 且 `is_active`），未命中且 from/to 都非基准币时
  **交叉套算**：两腿 `(from→pivot)`、`(to→pivot)`（pivot=树根本位币，通常 CNY），
  `cross = legFrom ÷ legTo`（`RoundBank(8)`，to 腿非正数按缺失处理）。命中多行返回
  `ErrExchangeRateConflict`，任一腿未命中返回 `ErrExchangeRateMissing`，不做任何日期回退。
  **直连行永远优先于套算值**；财务显式维护的直连行不被推导覆盖。
- **各消费方的业务日期固定**（这是类型体系删除后"时间点"语义的唯一载体，不得混用）：

| 消费方 | 日期 |
| --- | --- |
| 订单费用 | `fee.ExpenseDate` |
| 账单 | `bill.BillDate`（头行同汇率，见 finance-bill-currency.md） |
| 资金流水 | `cashflow.TransactionDate`（手工覆盖走既有 override 机制不动） |
| 发票 | `invoice.InvoiceDate` |
| 提成人民币换算 | 提成生成日 |

- **导入**：模板版本 v2，列为「原币/本币/折本币汇率/生效开始/生效结束」；旧模板文件被版本
  校验拒绝；Preview/Confirm 均要求调用组织为总部。
- 汇率快照字段（`exchange_rate/exchange_rate_source/exchange_rate_date`）保留在费用/账单/
  流水/发票实体上，来源三态：`SYSTEM`（直连命中）/ `DERIVED`（交叉套算推导）/
  `MANUAL`（手工覆盖），由解析结果透传，禁止硬编码；系统解析不回填
  `exchange_rate_setting_id`。套算是纯推导：不落汇率表行、不缓存、不加录入界面。

## 4. Validation & Error Matrix

| 条件 | 服务端行为 |
| --- | --- |
| 非总部组织调用 Create/Update/Disable 或导入 | 拒绝（总部写防御），不得静默重定向 |
| 直连行存在 | 用直连值，来源 SYSTEM（即使套算也可行） |
| 直连缺失、两腿齐 | 套算 `RoundBank(8)`，来源 DERIVED |
| from 腿或 to 腿缺失 / to 腿非正 | `ErrExchangeRateMissing`（fail-closed） |
| 直连或任一腿命中多行 | `ErrExchangeRateConflict` |
| 日期格式非法 | 400 参数错误 |
| 消费方传入与自身业务日期不一致的日期 | 属契约违规，评审驳回（无运行时防护） |

## 5. Good / Base / Bad Cases

- Good：费用 USD 发生日 9/1、账单日 9/5 分别解析 9/1、9/5 的总部中间价，各自快照落库。
- Base：本位币组织的本位币费用解析恒为 `1`，不触发查询。
- Bad：核销按"核销日"再解析一次汇率——核销不是交易，不引入新汇率（见 finance-bill-currency.md
  核销口径一节）。
- Bad：为某个新单据类型新增"专属汇率类型"或时间标准配置——按上表固定业务日期即可。

## 6. Tests Required

- Biz：单一 rate 规范化与校验、基准币种返回 1、缺汇率/冲突/非法日期错误、Create 不重定向。
- Data：非总部写拒绝（Create/Update/Disable/导入四入口）、解析命中区间边界。
- 消费方：各领域按自身业务日期调用（order_fee/finance_bill/finance_cashflow/finance_invoice/
  finance_commission 的既有汇率测试已锚定日期来源）。

## 7. Wrong vs Correct

### Wrong

```go
// 给某类单据再发明一个汇率类型
rate := exchangeRate.Resolve(ctx, org, SomeNewRateType, direction, currency, dates)
```

### Correct

```go
// 单一基准汇率 + 该单据自己的业务日期
rate, err := uc.exchangeRate.ResolveRate(ctx, orgID, fee.Currency, fee.ExpenseDate)
```
