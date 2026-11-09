# 往来单位散客契约（Partner Casual Contract）

> 散客 = 单次合作往来单位。本契约由任务 `09-11-casual-partner-loop` 确立
> （commits `6632e77c` / `87654abf`）。触碰 `partner.is_casual`、散客账期、
> 散客出款或伙伴选择器散客标注的任何改动都必须遵循本文。

## 1. Scope / Trigger

- 触发条件：修改 `is_casual` 相关的 DB 列、proto 字段、导入逻辑、建账预览、
  现金流校验，或前端快捷建档 / 选择器 / 现金流 / 建账工作台的散客交互。
- 这是跨层契约（proto + 迁移 + biz + data + web），按 code-spec 深度维护。

## 2. Signatures

- **DB**：`partners.is_casual boolean NOT NULL DEFAULT false`
  （迁移 `20260911150000_partner_is_casual.sql`；ent 列序位于 `enabled` 与
  `search_keywords` 之间）。
- **proto `partner/v1/partner.proto`**：
  - `Partner.is_casual = 15`（非 optional bool，false 时 JSON 省略，前端消费必须
    用 truthy / `?? false`，禁止 `=== false` 死分支）；
  - `CreatePartnerRequest.is_casual = 10`、`UpdatePartnerRequest.is_casual = 11`（optional）；
  - `ListPartnersRequest.is_casual = 6`（optional 过滤，谓词 `partnerent.IsCasualEQ` 落 data 层）。
- **proto `finance/v1/settlement.proto`**：
  - `FinanceSettlementPartyOption.is_casual = 4`；
  - `BillBatchPreviewGroup.is_casual` + `optional int32 default_payment_terms_days`
    （散客应收组恒为 `0`）。
- **错误码**：`ERROR_REASON_FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED = 12`
  （BadRequest 400，文案「向散客供应商出款时，对方收款账户为必填项」）。
- **biz 行为**：
  - `biz/partner.go` Import：新建强制 `IsCasual = false`；upsert 更新**不触碰**该字段；
  - `biz/finance_bill.go` PreviewBatch：散客应收组 `DefaultPaymentTermsDays = &0`；
  - `biz/finance_cashflow.go` Create：`Payable` + 散客 + 对方账户空 → 拒绝。

## 3. Contracts

- **语义**：`is_casual` 是**伙伴级**属性，不按角色拆分；一个往来单位一个标识，
  行为按业务方向生效（客户卡账期、供应商卡出款账户）。
- **建档口径**：快捷新增默认散客（弹窗复选框可取消）；主档新建显式选择；
  批量导入新建一律正式；**导入 upsert 更新保留既有标识**（转正只能人工编辑档案）。
- **账期口径（边界红线）**：散客客户应收账单组在 PreviewBatch **硬编码**默认
  账期 0（到期日 = 账单日）；不因结算规则或余额计算改变这一散客默认值。正式客户账期带入与信用汇总已由
  `FinanceBillUsecase.enrichReceivableGroupCreditInfo` / `defaultFinanceBillPaymentTermsDays` 实现，
  与散客零天默认值分别判定。改大账期仅前端黄色预警，服务端不拦截（刻意的
  商业弹性：硬拒绝会激励摘掉散客标识再录账期，破坏标识真实性）。
  散客应付组只标记 `IsCasual` 不设默认账期；正式应收客户按启用结算规则带出默认账期，
  多规则取仓储信用摘要的聚合结果，无规则时为 `nil`。
- **出款口径**：`Direction = Payable` 且结算对方为散客 → `CounterpartyAccount`
  必填（服务端刚性拦截，唯一硬卡点）；收款方向与正式供应商不受影响。
- **选择器口径**：选项契约携带 `isCasual`，前端经 `optionRender` 动态渲染
  `[散客]` Tag；`label` 只拼 `legalName (code)`，**禁止把散客文本拼进 label**
  （防止单证 / 合同字符串污染，选中值不得出现多余前缀）。
- **快捷建档角色契约**：订单侧按触发字段静默写入单一角色；费用侧弹窗单选 +
  `defaultRole` 按当前费用方向预选（应收→客户、应付→供应商、方向未选默认
  客户）。同时勾选多角色属主档级维护决策，快捷弹窗不得承载。

## 4. Validation & Error Matrix

| 条件 | 结果 |
|-------|------|
| `Payable` + 对方散客 + 对方账户空 / 空白 | `FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED`（400） |
| 散客应收组账期 > 0 | 前端黄色预警，提交放行（无服务端错误） |
| Import 新建输入 `IsCasual = true` | biz 层强制改写为 `false` |
| Import upsert 更新既有散客档案 | 标识保留，不转正 |
| `ListPartners` 带 `is_casual` 过滤 | data 层 `partnerent.IsCasualEQ` 谓词 |

## 5. Good / Base / Bad Cases

- **Good**：录应付费用 → 快捷新增供应商，弹窗预选供应商角色 + 默认散客；
  日后付款必须先补对方收款账户。
- **Base**：正式客户建账按当前结算规则带出默认账期，无规则时不注入；不会因散客零天规则覆盖正式客户设置。
- **Bad（合法但被预警）**：散客应收账单改 30 天账期 → 黄色预警但可提交，
  事后可按「散客 + 账期 > 0」联查审计。

## 6. Tests Required

- `biz/partner_test.go`：创建透传与人工转正、Import 新建强制 `false`、
  `List` 过滤透传（三分支 true/false/nil）。
- `biz/finance_cashflow_test.go`：无账户拒绝 / 空白拒绝 / 有账户成功 /
  正式供应商不受影响 / 收款方向不受影响（五场景）。
- `biz/finance_bill_test.go`：散客应收组默认 0、散客应付组只置标记、
  正式应收组有规则时带出、无规则时为 `nil`。
- 前端 vitest：快捷弹窗勾选与角色单选、选择器散客 Tag、建账散客预警。

> **Common Mistake**：给 partner ent 表加列后，必须同步 `internal/data` 下
> 所有按 `partnerent.Columns` 顺序构造 `AddRow(...)` 的 sqlmock fixture，
> 否则 go-sqlmock 直接 panic 且会掩盖同包后续失败（本任务实测踩坑：
> `finance_netting_test.go`、`partner_attachment_transaction_test.go`）。

## 7. Wrong vs Correct

### Wrong 1：散客账期读规则 / 算余额

```go
// ❌ 错误：用正式客户规则或未核销余额覆盖散客零天默认值
terms := rule.PaymentTermsDays          // 不能覆盖散客默认值
balance := sumUnsettled(partnerID)      // 信用展示不改变散客账期
```

```go
// ✅ 正确：只看费用上的散客快照，硬编码 0
if item.SettlementPartyIsCasual && fee.Direction == OrderFeeReceivable {
    group.IsCasual = true
    group.DefaultPaymentTermsDays = &zero
}
```

### Wrong 2：导入 upsert 覆盖散客标识

```go
// ❌ 人工标记的散客被批量导入静默转正
updater.SetIsCasual(false)
```

```go
// ✅ 正确：新建路径显式 false，更新路径不 Set 该字段
if creating { updater.SetIsCasual(false) }
```

### Wrong 3：选择器 label 拼散客文本

```tsx
// ❌ 选中值与单证字符串被污染
label: `${legalName}（散客）`
```

```tsx
// ✅ 正确：契约字段 + optionRender 动态渲染
{ isCasual: option.isCasual, label: `${legalName} (${code})` }
// optionRender 中：data.isCasual && <Tag color="orange">散客</Tag>
```
