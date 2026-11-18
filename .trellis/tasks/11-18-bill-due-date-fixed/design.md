# 设计：账单到期日固定

## 1. 改动边界

| 层 | 文件 | 改动 |
|---|---|---|
| Biz | `server/internal/biz/finance_bill_fees.go` | `UpdateFees` 的刷新回调不再改日期、不再重取汇率 |
| Data | `server/internal/data/finance_bill_fees.go` | 原则上不改；`refresh` 参数是否保留见 §2.2 |
| Web | `web/src/features/finance/bill-creation/BillGroupCard.tsx`、`BillCreationWorkbench.tsx`、`billWorkbenchHelpers.ts` | 到期结果展示与「调整到期日」 |
| Web | `web/src/pages/finance/bills/components/BillEditModal.tsx`、`web/src/pages/finance/bills/index.tsx` | 编辑时展示、调整并如实提交到期设置 |
| Web | `web/src/pages/finance/bills/components/BillFeeMaintenance.tsx` | 补一句提示文案 |
| Spec | `.trellis/spec/server/backend/finance-bill-currency.md`、`.trellis/spec/domain/glossary.md`、`.trellis/spec/web/frontend/component-guidelines.md`（若提及） | 改写增减费用契约 |

**不改 Proto、不改 Ent Schema、不加迁移。** 后端已经接受 `due_date`；前端只是开始使用它。

## 2. 后端

### 2.1 `UpdateFees` 新口径

现状（第 214–224 行）：

```go
func(lockedCtx context.Context, bill *FinanceBill) error {
    bill.BillDate = uc.businessDate()
    if bill.PaymentTermsDays != nil {
        bill.DueDate = normalizedFinanceBillDueDate(bill.BillDate, nil, bill.PaymentTermsDays)
    }
    if !validFinanceBillTerms(...) { return errors.BadRequest("FINANCE_BILL_DUE_DATE_BEFORE_CREATION", ...) }
    return uc.applyBillExchangeRate(lockedCtx, bill.OrganizationID, bill)
}
```

改为：回调不改动账单头的日期、到期和汇率字段。仓储随后调用的
`RecomputeFinanceBillAmounts(bill, kept)` 本来就用 `bill.ExchangeRate` 给每一行（包括新增行）
写汇率、折算本币并分摊尾差，所以不需要额外的汇率逻辑。

仓储在 `refresh` 之后的写回语句（`SetBillDate(...).SetNillableDueDate(...).SetExchangeRate(...)...`）
写回的就是原值，不需要改动；保留它可以让写回范围一目了然。

### 2.2 `refresh` 参数的处理（二选一，倾向 A）

- **A（推荐）：删除 `refresh` 参数。** `FinanceBillRepo.UpdateFees` 的签名去掉
  `refresh func(context.Context, *FinanceBill) error`，同步修改接口定义、实现和所有测试替身。
  回调不再承担职责，保留空函数只会误导后来的人。
- B：保留参数，传一个只返回 `nil` 的回调。改动最小，但留下一个空壳。

按 AGENTS.md「不保留无用兼容层」的原则，选 A。

### 2.3 不改的部分

- 建账（单张 `Create`、批量 `CreateBatch`）仍在建账当天按业务日期取汇率，写入 `bill_date` 和 `exchange_rate_date`。
- `Update`（编辑账单）仍沿用当前 `bill_date`，不推进日期。
- 锁序、锁定事实（发票、核销、对冲、免票）拒绝增删、版本号、审计、提成经济重算不变。

### 2.4 错误目录

删除 `FINANCE_BILL_DUE_DATE_BEFORE_CREATION`（目前是在 biz 里内联的 `errors.BadRequest`，
没有进入错误目录；全仓检索确认前端没有按 reason 判断它）。

## 3. 前端

### 3.1 到期设置的表单模型

建账卡片和编辑弹窗共用同一套表单模型，避免两处各写一份：

```ts
type DueMode = 'terms' | 'date';
// 表单字段：dueMode、paymentTermsDays（dueMode=terms 时有效）、dueDate（dueMode=date 时有效，Dayjs）
```

- 初始化：
  - 账单或分组有 `paymentTermsDays` 时 → `dueMode='terms'`；
  - 没有账期但有 `dueDate` 时 → `dueMode='date'`；
  - 两者都没有 → `dueMode='terms'`、`paymentTermsDays` 为空，展示「未约定到期日」。
- 提交载荷：
  - `terms` → 只带 `paymentTermsDays`（可为空）；
  - `date` → 只带 `dueDate: 'YYYY-MM-DD'`，**不带** `paymentTermsDays`。
- 展示文案由一个纯函数生成，传入 `billDate`、`dueMode`、`paymentTermsDays`、`dueDate`，输出：
  `YYYY-MM-DD 到期 · 账期 N 天` / `当天到期 · 现结` / `YYYY-MM-DD 到期 · 指定日期` / `未约定到期日`。
  日期按纯日期字符串处理，不经过时区换算（AGENTS.md 时间规范对纯日期字段不做限制）。

**放置位置：** 编辑弹窗在 `pages/finance/bills`，建账卡片在 `features/finance/bill-creation`。
把共享的纯函数和小组件放进 `features/finance/bill-creation/` 并从它的 `index.ts` 导出，
`pages` 从 feature 引用，符合依赖方向（pages → features）。新增导出后，同步更新
`.trellis/spec/web/frontend/capability-navigation.md`。实现前先查这份导航，确认没有现成的同类能力。

### 3.2 建账卡片 `BillGroupCard`

- 「账期（天）」那一列换成「到期日」展示块：大号文字显示到期结果，旁边一个「调整到期日」链接按钮。
- 点开后，在卡片内展开一个 `Radio.Group`（按账期天数 / 指定日期），下面跟对应的输入
  （`InputNumber` 0–3650 或 `DatePicker`，禁选早于 `group.billDate` 的日期）。不用弹窗，避免多层浮层。
- 默认收起；打开时保留输入。
- `BillCreationWorkbench` 合并分组表单值时（约 620–660 行），新增 `dueMode`、`dueDate` 字段，
  已有输入优先于默认值（沿用现有合并逻辑）。提交时（约 495–510 行）按 §3.1 组装 `paymentTermsDays` 或 `dueDate`。
- 服务端兜底规则不变：请求里账期和到期日都没有时，按客户默认账期注入。所以当客户有约定时，
  界面不要提供「清空成未约定」的操作，否则提交后又会被服务端带回默认值，造成误解。

### 3.3 编辑弹窗 `BillEditModal` + `bills/index.tsx`

- 「账期（天，可选）」换成与建账卡片相同的到期展示块和「调整到期日」。
- `index.tsx` 的 `submitBill` 按 §3.1 组装载荷。这同时修复了「保存后到期日被清空」的问题。
- 受保护账单（`fieldsProtected`）：
  - `dueMode='terms'` 且有账期 → 整个调整入口禁用（账期不能改，日期又由账期决定）；
  - 没有账期 → 允许改为指定日期。服务端允许受保护账单修改到期日。
  - 现有黄色提示文案保持不变。
- 「创建日期」提示改为「建账时由系统记录，之后不再变化」。

### 3.4 `BillFeeMaintenance`

在增减费用区域加一行次要说明：「增减费用不会改变到期日和账单汇率」。

## 4. 测试设计

### 后端
- `server/internal/biz/finance_bill_automatic_date_test.go` 及其他断言「增减费用刷新日期」的单测：
  改为断言日期、到期日、汇率字段全部不变。
- `server/internal/data/finance_bill_fees_integration_test.go`（真实 PostgreSQL）：
  - 增加费用：头部的 `bill_date`、`due_date`、`exchange_rate`、`exchange_rate_date`、`exchange_rate_setting_id`
    不变；新行 `exchange_rate` 等于账单汇率；头部本币金额等于各行本币之和。
  - 外币账单在「当天无汇率」时，增减费用仍成功。
  - 移除费用、同时增删费用，结论相同。
  - 原有的锁定事实、版本冲突、最后一笔、零写入断言保留。
- `finance_bill_correction_scope_integration_test.go` 如果调用了带 `refresh` 的仓储签名，同步调整。

### 前端
- 到期展示纯函数：四种显示各一例，另覆盖账期为 0 和缺少账单日期的情况。
- `BillGroupCard` / 建账工作台：默认带出账期时显示「X 到期 · 账期 N 天」；
  切换为指定日期后，提交载荷只有 `dueDate`。
- `BillEditModal`：
  - 「指定日期、无账期」的账单不做修改直接保存，载荷带原 `dueDate`；
  - 受保护且有账期 → 调整入口禁用；受保护且无账期 → 可以改日期。
- 现有测试中的文案断言（「增减费用会更新创建日期」）同步更新。

## 5. 回滚

后端和前端分两次提交，可以分别 revert。没有 Schema 和契约变化，不涉及数据回滚。
