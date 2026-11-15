# 设计：发票登记与修订易用性优化

## 边界

- R1–R7 修改前端共享能力 `web/src/features/finance/invoice-registration/`，不改服务端分配行为；R8 按文末方案扩展票种枚举、正式迁移和契约生成物。
- 提交载荷形状与 `invoiceRegistrationPayload` 的校验保持不变；自动带入只改变表单值，提交仍走现有三层校验与服务端事务。
- 共享编辑器同时服务登记/修订弹窗与建账附带登记（`BillGroupCard`），自动带入逻辑放在编辑器内，两处一致生效；"打开即建一张发票"只在登记弹窗（非修订、非建账）启用，因为建账中发票登记是可选的。

## 新增纯函数模块 `autofill.ts`

与 React 解耦、可单测：

```ts
type FieldSource = 'auto' | 'manual';
// 草稿行扩展（不进入载荷）：
// allocation.amountSource?, fee.amountSource?, fee.taxSource?

suggestTax(gross: string, rate?: string): string | undefined
  // rate 未设置 → undefined；否则 G×r/(100+r)，ROUND_HALF_UP 到 2 位，toFixed(2)
  // 例：1060 @6 → '60.00'；rate '0' → '0.00'；gross 非法 → undefined

availableFor(fee, records, selfIndex): Decimal
  // fee.invoiceableAmount ?? fee.totalAmount，减去本会话其他发票对同 bill:fee 的分配

registrableFees(bill): FeeSource[]   // 可登记余额 > 0 的费用

autoAllocate(total, bills, records, selfIndex): Allocation[]
  // 按账单顺序、费用顺序逐项填至可用余额；生成的金额/税额均标 auto；
  // 余额为0的行不生成；不补齐，剩余由差额显示

followTotal(record, bills): record
  // 仅当记录恰为1账单1费用时：账单分配(auto)←总额，费用金额(auto)←总额
  // manual 字段原样保留

followFee(fee, source): fee
  // taxSource 为 auto 时：tax ← suggestTax(fee.amount, rate)；建议为 undefined 则置空

followBill(allocation): allocation
  // amountSource 为 auto 且多费用时：账单分配 ← 费用金额合计

balance(record): { invoiceGap, billGaps[] }  // 发票待分配差额、各账单费用差额
```

金额计算一律 `Decimal`，不经 `Number`；用户输入不截断，8 位精度原样提交。

## 编辑器交互

- **来源标记**：隐藏 `Form.Item` 保存 `*Source`。Input 的 `onChange`（只在用户输入时触发）把对应字段标为 `manual`；程序写入用 `form.setFieldValue`，不触发 `onChange`。
- **联动入口**（事件驱动，不用 effect 同步，避免循环）：
  - 发票总额 onChange → `followTotal` → 对被改到的费用 `followFee`。
  - 费用金额 onChange → 标 manual → `followFee` → `followBill`。
  - 选择账单/费用 onChange → 现有清空逻辑 + 对新费用 `followFee`。
  - 添加发票：若会话只有1张账单且仅1条可登记费用，直接生成该账单分配与费用行（auto）。
  - 添加费用行：默认带入该账单剩余待分配金额（auto，≤可用余额）。
- **“按可登记余额分配”按钮**（每张发票卡片）：存在 manual 值或已有分配时先 `modal.confirm`，说明会覆盖本张发票全部分配与税额；确认后整体替换为 `autoAllocate` 结果。
- **展示**：
  - 发票卡片头部：总额 / 已分配 / 待分配差额（差额≠0 标红）。
  - 账单卡片：可登记余额、本次分配、费用合计差额。
  - 费用行：可登记余额（扣本会话其他占用）、来源税率（未设置 `-`，零 `0%`）、含税金额、税额；税额为 auto 且有值时下方显示“建议值，请以实际发票税额为准”；税率未设置时提示“来源税率未设置，请按发票填写税额”。
  - 超余额：费用金额校验器报错“超过可登记余额 X”，阻止提交。
- 顶部说明文案改为说明自动带入与手改保护，去掉“逐费用手填”措辞。

## 修订对照

- 修订弹窗已有原值基线（`baseline.invoice`）。新增只读组件 `InvoiceRevisionDiff`：按发票头（日期、号码、类型、总额、税额、备注）与逐账单/逐费用（金额、税额）列出“原值 → 修改后”，仅显示变化项，变化项高亮；新增/移除的分配单独标注。放在修订原因上方，随表单 watch 实时更新。
- 修订预填值全部标为 `manual`（来自真实登记），不会被总额变化自动改写；用户可主动点“按可登记余额分配”。
- 结算依赖阻断（`canRevise !== true`）保持现状：展示依赖并禁用保存。

## 规范更新

`finance-invoice-registration.md` §3 与 §7 现写“不自动算税”“错误：费用税率×本次含税额自动补税”。改为：前端可按来源税率预填**可编辑建议税额**（含税口径、2位、逐费计算、未设置不建议），提交值以财务确认后的表单值为准；服务端仍不计算/补税，不反写原费用。

## 取舍

- 来源标记存表单而非 ref：随 Form.List 增删自动对齐索引，不会错位。
- 不在发票层补尾差：混合税率逐费四舍五入后合计与票面税额可能差 0.01，由财务手改。
- 默认总额（R7，后续追加）：`newInvoiceRecord` 以 `remainingBalance` 带出所选账单扣除本会话占用后的余额合计并 `autoAllocate`；`totalSource` 区分默认与手改；账单勾选变化经 `rebuildUntouchedRecords` 先释放再按序重建未手改发票，手改与修订原票不动。

## 回滚

R1–R7 的前端辅助改动可通过 `git revert` 回滚；R8 含数据迁移，回滚须按文末约定另行审阅反向迁移，不能只回退代码。

## R8 发票类型细分（契约变更）

- 唯一票种表：biz `FinanceInvoiceType` 6 个常量 + `Valid()` / `Special()`；`partner_invoice_profile.go:101`、`finance_invoice_registration.go:310/407` 改用合法性/专票判定函数，不再比较两个字面量。
- Ent：`finance_invoice.invoice_type` 与 `partner_invoice_profile.default_invoice_type` 枚举改为 6 值（后者默认 `DIGITAL_NORMAL`）；新正式迁移：先删旧 CHECK，`UPDATE` 映射旧值，再加新 CHECK、改默认值。
- proto 仍为 string，只更新字段注释与生成物（OpenAPI/前端客户端）。
- 前端：`features/finance/invoice-registration` 新增 `invoiceTypes.ts`（选项、中文名、`isSpecialInvoiceType`），经 index 导出；登记编辑器、修订对照/历史、发票台账列与导出、抬头维护（pages/partners 只能经 features 公开导出使用）统一引用。
- 回滚：迁移+代码同一提交，`git revert` 后需反向迁移（开发库）。
