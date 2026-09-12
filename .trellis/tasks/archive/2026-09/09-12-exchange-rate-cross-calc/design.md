# 汇率交叉套算 Technical Design

## 1. Architecture & Design Principles

1. **单跳经基准币**：套算固定 `from→pivot ÷ to→pivot`，pivot = 树根总部本位币
   （汇率行的 to_currency），不做多跳寻优、不做币对图。
2. **解析层统一收口**：套算只在 `internal/data/exchange_rate.go` 仓储解析内实现，
   biz 的 `ResolveRate` / `ResolveBaseRate` 签名不变或最小扩展；所有消费方零改造。
3. **来源随值传播**：解析结果从"裸 decimal"升级为携带来源（SYSTEM/DERIVED），
   单据快照的 `exchange_rate_source` 如实记录，审计可区分录入值与推导值。

## 2. Core Modules & Data Models

### 2.1 pivot 解析（data/exchange_rate.go `ResolveContext`）

`ExchangeRateContext` 增加 `PivotCurrency`（树根总部本位币）：现有向上遍历到根的循环
在根节点处取 `root.BaseCurrency`（根必非空，沿用现有 fail-closed）；调用组织自身
`BaseCurrency` 字段语义不变。pivot 与 Rate 行的 `to_currency` 语义对齐（总部只维护
`X → pivot`）。

### 2.2 仓储解析（data/exchange_rate.go）

新增内部方法（或扩展现有 `ResolveRate` repo 方法）：

```go
// resolveWithCross：直连优先，缺失时经 pivot 交叉推导
// 1) SELECT 直连行 from→to（现行为不变）→ 命中返回 {rate, SYSTEM}
// 2) from == to → 1（调用方已处理，防御性保留）
// 3) 两腿：from→pivot、to→pivot（均在总部行上解析，含既有冲突检测）
//    任一缺失 → ErrExchangeRateMissing（fail-closed）
//    to 腿非正数 → ErrExchangeRateMissing（除零守卫，同缺失语义）
// 4) cross := legFrom.Div(legTo).RoundBank(8) → 返回 {cross, DERIVED}
```

- biz `ResolveRate(org, currency)`：to 为调用组织本位币，改走 `resolveWithCross`；
- biz `ResolveBaseRate(from, to)`：同样改走 `resolveWithCross`（提成 CAD→CNY 直连
  命中路径不变）；
- 返回值从 `decimal` 扩展为小型结构 `ResolvedRate{Rate decimal.Decimal; Source string}`
  （或 `Rate, Source` 双返回值，按仓库惯例择一），调用方记录快照来源时使用；
  现有把来源硬编码 SYSTEM 的调用点改为透传解析来源。

### 2.3 快照来源传播（消费方）

费用/账单/流水/发票创建与币种变更处的快照写入：`exchange_rate_source` 从固定
`SYSTEM` 改为解析结果的 Source；`MANUAL` 手工覆盖分支不变。`DERIVED` 枚举值在
`exchange_rate_source` 中已存在（存量合法值），**无 Schema/迁移/proto 变更**。

### 2.4 不做的事

- 不在汇率表落推导行、不加维护界面；
- 不缓存套算结果（每次解析即时推导，量级为两次索引点查）；
- 不改变冲突检测语义（任一腿命中多行仍报 `ErrExchangeRateConflict`）。

## 3. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 直连行存在 | 用直连值，来源 SYSTEM |
| 直连缺失、两腿齐 | 套算 `RoundBank(8)`，来源 DERIVED |
| from 腿或 to 腿缺失 / to 腿非正 | `ErrExchangeRateMissing`（fail-closed） |
| 任一腿命中多行 | `ErrExchangeRateConflict`（沿用） |
| from == to | 1（恒等，调用方语义不变） |

## 4. Tests Required

- biz：CAD 组织解析 USD 费用 = (USD→CNY)÷(CAD→CNY) 金额断言；CNY 费用 = 1÷(CAD→CNY)；
  直连行优先（加直连 USD→CAD 行后改用直连值且 SYSTEM）；腿缺失报错；来源标记断言
  （DERIVED/SYSTEM/MANUAL 三态）。
- data/集成（隔离 schema）：真实行上套算往返（费用创建 → 快照 source=DERIVED、
  rate 与手工推导一致）；CNY 组织全回归不变。
- 精度：RoundBank(8) 与除零守卫边界。

## 5. Wrong vs Correct

### Wrong

```go
// 给每个币对都要求财务录入直连行
rate := ResolveRate(org, "USD") // CAD 组织报 ErrExchangeRateMissing，费用录不进去
```

### Correct

```go
resolved := resolveWithCross(HQ, "USD", "CAD", date) // (USD→CNY) ÷ (CAD→CNY)
snapshot.ExchangeRate = resolved.Rate                 // RoundBank(8)
snapshot.ExchangeRateSource = resolved.Source         // "DERIVED"
```
