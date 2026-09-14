# 汇率周汇率双轨点差与组织自治契约

## 1. Scope / Trigger

本规范适用于汇率主数据的 Schema、解析、写入、导入、同步抓取，以及所有按业务日期折算
本位币的消费方（订单费用、账单、资金流水、发票）。新增或修改任何汇率字段、解析入口或
消费方调用时必须遵循。

现行模型（2026-09-15 治理重构，迁移 `20260914231000` + `20260915093000`）：
**B 型存储 + 自然周窗口 + 应收/应付双轨点差 + 各核算组织自治**。单一基准价模型
（2026-09-11 曾把收付双轨推平为单 `rate`）已被再次演进：`receivable_rate`/`payable_rate`
的 proto reserved 已解除，正式定义为 `ar_rate`(9)/`ap_rate`(10)，`rate`(14) 降级为基准价
（中行折算价，审计口径）。

## 2. Signatures

```go
// internal/biz/exchange_rate.go
ResolveRate(ctx, orgID, currency, direction, targetDate) (ResolvedRate, error)
// direction: RECEIVABLE→ar_rate / PAYABLE→ap_rate；ResolvedRate 携带 Rate/Source/SettingID
// ResolveBaseRate 已退役——跨组织资金流按原币记账，禁止系统折算
FetchExchangeRates(ctx, orgID, target) (预览, error)  // target: 本周/预设下周
SyncExchangeRates(ctx, principal, source, inputs, audit) error  // 同周幂等 Upsert

// internal/data/exchange_rate.go —— 四级解析链（均不阻断单据保存）
// ① 本组织当周行 WEEKLY/BOC_SYNC
// ② 回溯最近一个有效历史自然周 INHERITED_LAST_WEEK
// ③ NULL 基线行直连 SYSTEM → 基线 pivot 交叉套算 DERIVED（仅兜底路径存在 DERIVED）
// ④ 现场手工 MANUAL；全链未命中才 ErrExchangeRateMissing
// 事务内汇率行读取必须 ForShare()（账单快照并发一致性依赖此锁）
```

## 3. Contracts

```text
Table: exchange_rate_settings（B 型，organization_id 可空）
  organization_id  -- NULL=总部兜底基线行；非 NULL=核算组织自维护行
  from_currency / to_currency  -- to_currency 恒为行归属组织的本币
  effective_from  -- 即当周周一 00:00:00（Asia/Shanghai）；区间由服务端派生，写契约不含 effective_to
  ar_rate / ap_rate  -- 中行现汇卖出价（收高）/ 现汇买入价（付低），numeric(18,8)
  rate  -- 中行折算价（基准/审计口径）；新浪 JSON 与直盘源无此价时取 (ar+ap)/2
  唯一性：(organization_id, from, to, effective_from)；同周二次同步=幂等覆盖，禁止唯一冲突报错
快照：order_fees/finance_bill_lines 落命中汇率值 + exchange_rate_setting_id + source
  （WEEKLY / INHERITED_LAST_WEEK / MANUAL / BOC_SYNC；DERIVED/BASE_CURRENCY 已退役）
同步：FetchExchangeRates 按组织本币路由——CNY：新浪中行专线 JSON 主源 + 中行牌价页 HTML
  兜底（/100）；非 CNY：国际直盘（Ask→ar、Bid→ap）首选 + 中行交叉盘备选（银行交叉商法
  ar=卖出腿÷买入腿，保证 ar>ap）；来源在预览中明示，fail-closed 引导手工录入。
督办：周一 10:00 起 24h 补发窗口检测当周未同步的 kind=company 组织，notification_delivery
  确定性 ID 幂等入队。
```

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 非总部写 NULL 基线行 / 无权限写 org 行 | 403（拦截器双校验） |
| 同组织同货币对同周重复 | 幂等 Upsert 覆盖（唯一索引兜底），不报唯一冲突 |
| 全链未命中且无手工覆盖 | `ErrExchangeRateMissing`（仅此时允许阻断） |
| 跨组织提成/往来 | 原币记账恒等快照（rate=1/BASE_CURRENCY），禁止二次折算 |
| 历史已存盘费用 | 快照绝对冻结，周中重同步不穿透 |

## 5. Good / Base / Bad Cases

- Good：香港分公司（本币 HKD）一键同步，直盘预填 USD→HKD 双轨价，财务微调后当周生效。
- Base：新组织当周未同步，费用录入自动继承上周（黄色 Tag「暂沿用上周汇率」），单据不卡死。
- Bad：跨组织结算用任一方本地汇率折算记账——必须原币对账。
- Bad：解析链跳过 ForShare——账单事务内并发改汇率会撕裂快照一致性。

## 6. Tests Required

- 真实库集成 `TestExchangeRateWeeklyDisasterChainPostgres`：四级链全分支、同周幂等、
  跨周预设、方向取列、快照行 ID。
- `TestFinanceBillCreateSharedTransactionPostgres`：并发修改汇率不改变事务内账单快照。
- 抓取解析单测：新浪 JSON（已归一化）/ 中行 HTML（/100）/ 直盘 Bid/Ask、交叉盘换算、
  fail-closed；`isBOCBankEntry` 精确匹配（bocom 不误中）。
- 迁移陷阱：改来源枚举值集必须同步重建四张财务表 `*_exchange_rate_source_check` 约束
  （阶段二实测踩坑）。

## 7. Wrong vs Correct

### Wrong

```go
// 跨组织折算 + 同步行硬编码归属
rate, _ := uc.ResolveBaseRate(ctx, orgID, from, to, date)      // 已退役
rows[i].OrganizationID = nil                                    // 分公司同步越权写基线行
```

### Correct

```go
// 组织内按方向解析；同步按组织身份落行（总部→基线，分公司→org 行）
resolved, _ := uc.ResolveRate(ctx, orgID, currency, direction, date)
scope := ensureOrganizationScope(principal) // 总部→RequireBaselineWrite+nil；分公司→org 行
```
