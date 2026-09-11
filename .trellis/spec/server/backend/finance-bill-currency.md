# 财务账单固定币种边界

## 1. Scope / Trigger

本规范适用于普通账单的批量预览、批量创建、草稿更新、账单行快照和前端建账工作台。
新增或修改费用分组、账单币种、账单日期汇率、结算账户或预计开票字段时必须遵循。

项目采用固定币种边界：不同费用币种永远拆成不同账单，不建设费用原币到可选目标账单币种的
转换子系统。这样避免跨币种金额被错误相加，也避免把预计开票汇率误用为账单余额汇率。

## 2. Signatures

`BillGroupingPolicy` 只包含模式与可选拆分维度：

```text
mode: NORMAL | NETTING
split_by_tax_rate: bool
split_by_order: bool
```

预览和创建的叶子配置只接收：

```text
group_key
bill_date
settlement_account_id
estimated_invoice_currency
estimated_invoice_rate
```

普通账单头的币种关系固定为：

```text
bill.currency = leaf.fee_currency
bill.exchange_rate = rate(bill.currency -> organization.base_currency, bill.bill_date)
bill.base_currency_amount = bill.total_amount * bill.exchange_rate
```

## 3. Contracts

- 组织、结算单位、费用币种和组织本位币始终参与普通账单分组；税率和订单是可选维度。
- API 不定义 `split_by_currency`，也不接受目标账单币种、费用原币到目标币种汇率或人工覆盖。
- `group_key` 必须包含费用币种；修改账单日期、账户或预计开票配置不得改变 `group_key`，但必须
  轮转预览令牌。
- 结算账户必须属于同一组织和结算单位、已启用、用途匹配且账户币种等于叶子费用币种。
- 账单日期只解析账单币种到组织本位币的账单汇率；币种相同时汇率固定为 `1`。日期为空的初次
  预览只返回固定币种分组和原币金额，不查询临时日期汇率、不计算本位币金额、不签发创建令牌。
- 账单行币种与金额保存来源费用事实；账单头和行的本位币金额使用同一账单日汇率计算，最后一行
  吸收确定性舍入尾差。
- 预计开票币种、汇率和金额是独立预估快照。它们不改变账单币种、账单金额、本位币金额、余额、
  核销或实际发票汇率。
- 创建事务必须重读并按稳定顺序锁定费用、账户和汇率设置，以同一纯分组内核重新计算后校验
  预览令牌；整批成功或整批失败。
- 草稿更新、确认、取消等已有账单操作必须先锁账单。需要重算明细的更新随后按主键稳定顺序锁定
  费用，再校验账户和汇率；禁止“费用 → 账单”与其他路径的“账单 → 费用”形成反向等待。

## 4. Validation & Error Matrix

| 条件 | 服务端行为 |
|---|---|
| 普通模式混入应收和应付 | 拒绝，提示普通账单需分方向创建 |
| `mode` 未指定或为尚未实现的 `NETTING` | Fail-closed，返回参数或未支持错误 |
| 不同费用币种 | 必须形成不同叶子，不允许合单 |
| 请求试图提交或修改账单币种 | 正式契约不提供该字段；不得在 Service/Biz 增加旁路 |
| 账户币种不等于叶子费用币种 | 拒绝预览完成或创建 |
| 账单币种到本位币缺少账单日有效汇率 | 拒绝创建，不允许 `1`、`0` 或任意日期回退 |
| 账单日期为空 | 返回固定币种叶子，不查询临时汇率；配置不完整且不得签发可创建令牌 |
| 预计开票币种不同但汇率为空、非正数或精度非法 | 拒绝该叶子配置 |
| 锁内费用版本、状态、金额、币种或分组事实变化 | 预览令牌冲突，整批不创建 |

## 5. Good / Base / Bad Cases

- Good：同一单位同时有 USD、EUR 费用，服务端稳定返回两个叶子；各自选择同币种账户，按各自账单
  日期解析到组织本位币的汇率，一次事务生成两张账单。
- Base：账单币种等于组织本位币，账单汇率为 `1`；预计开票不填写时默认同账单币种、汇率为 `1`。
- Bad：隐藏前端“分币种”开关，但 API 仍保留 `split_by_currency=false` 或目标币种字段，让调用方
  可以绕过页面形成跨币种账单。
- Bad：使用预计开票汇率重写账单 `exchange_rate` 或本位币金额。

## 6. Tests Required

- Biz：不同费用币种一定生成不同 `group_key`；税率和订单开关只影响各自维度。
- Biz：`NORMAL` 单方向成功，混合方向、未指定模式和 `NETTING` fail-closed。
- Biz/Data：账单日期为空时返回叶子且不调用汇率解析；同币种汇率为 `1`；日期确定后外币缺账单日汇率明确失败。
- Biz/Data：账户错组织、错单位、错用途、停用和错币种均被拒绝，创建事务锁内再次校验。
- Biz：预计开票默认值、不同币种显式正汇率及预计金额计算正确，并断言不改变账单金额与本位币。
- Biz/Data：账单头本位币金额等于账单行本位币合计，末行吸收 Decimal 舍入尾差。
- 并发：两个批次争抢同一费用只有一个成功；费用或配置变化后旧令牌不能创建。
- 并发：草稿更新按“账单 → 费用 → 账户 → 汇率”处理，与取消账单的锁顺序一致。
- Web：没有分币种开关和账单币种编辑器；导航始终包含币种层；提交契约不携带目标币种。

## 7. Wrong vs Correct

### Wrong

```text
policy.split_by_currency = false
group.currency = user_selected_currency
line.amount = convert(fee.amount, manual_pair_rate)
```

该实现让同一账单混入多个费用币种，并额外制造默认币种、组合汇率、人工覆盖和审计分支。

### Correct

```text
raw_group_key += fee.currency
bill.currency = fee.currency
bill.exchange_rate = resolve_bill_date_rate(fee.currency, organization.base_currency)
```

账单币种由费用事实决定；用户只配置日期、同币种账户和独立的预计开票值。

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
`RoundBank(8)`。

**Why**：对冲是双边债务消灭，只披露应收侧会让应付端本位币无处安放、汇兑损益无字段承载
（2026-09 审计 B2）。

**Related**：提成计算的收入实现口径与分母聚合均以账单行 `BaseCurrencyAmount`（账单日汇率）
为基准；未建账费用分母回落费用自身快照（费用发生日），见 finance-commission-lock.md 与
提成分母相关测试。
