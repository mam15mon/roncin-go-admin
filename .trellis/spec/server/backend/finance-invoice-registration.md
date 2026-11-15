# 发票逐费登记、修订与反向处理

## 1. 适用范围与触发

本契约覆盖独立发票台账、账单详情和批量建账附带的已有票据登记；修改 `finance_invoice*`、逐费分配 Schema、登记弹窗或发票进度投影时必须对照。只登记已有票据，不接在线税务；`CreateInvoice/IssueInvoice` 草稿→开具双步骤 RPC 与产品入口已退役，不保留双写或运行期旧流程适配。

唯一传输真相源为 `api/finance/v1/settlement.proto`。领域在 `biz/finance_invoice_registration.go`，持久化在 `data/finance_invoice{,_registration,_reverse,_progress}.go`；共用前端能力经 `features/finance/invoice-registration/index.ts` 公开。免票与账单保护同时遵循 [账单契约](./finance-bill-currency.md)，公司范围遵循 [当前工作区](../../web/frontend/finance-current-workspace.md)。

## 2. 签名与数据形状

| RPC / HTTP | 权限与用途 |
| --- | --- |
| `RegisterInvoices` / `POST /api/v1/finance/invoices/register` | `system.finance.invoice.create`，一次多票原子直登记 |
| `ReviseInvoiceRegistrations` / `POST /api/v1/finance/invoices/revise` | `system.finance.invoice.update`；含无ID新增项还须同公司 `invoice.create` |
| `GetInvoiceRevisionImpact` / `GET /api/v1/finance/invoices/{id}/revision-impact` | `invoice.update`，完整原分配、全部关联账单及有效结算依赖 |
| `ListInvoiceCreationBills` / `GET /api/v1/finance/invoices/creation-bills` | `invoice.create`，分页关键字候选，支持 `bill_id` 精确变动探测，不借 `bill.read` |
| `ListInvoiceProfilesForParty` / `GET /api/v1/finance/invoice-creation/party-profiles` | `invoice.create`，当前公司、指定结算单位的启用抬头，不借 `partner.read`；候选命名空间避开`invoices/{id}`动态记录路径 |
| `CancelInvoice` / `POST /api/v1/finance/invoices/{id}/cancel` | `invoice.update`，带版本与原因的显式作废 |
| `RedFlushInvoice` / `POST /api/v1/finance/invoices/{id}/red-flush` | `invoice.update`，带版本、原红字号码/日期与原因的显式红冲 |

登记的三层输入：

```text
RegisterInvoices = {
  idempotency_key,
  records: [{ invoice_profile_id, invoice_type, tax_invoice_no, invoice_date,
    total_amount, note?, allocations: [{ bill_id, expected_version, amount,
      fees: [{ order_fee_id, amount, tax_amount }] }] }]
}
ReviseInvoiceRegistrations = {
  idempotency_key, reason, bill_versions: [{ bill_id, expected_version }],
  entries: [{ id?, expected_version, remove, registration? }]
}
```

修订中旧项必须带 `id/expected_version`；删除项 `remove=true` 且不带 `registration`；新增项无ID、版本为0并带登记内容。一次最多100票、总来源费用最多5000项，每票最多500张账单；常规列表统一使用公共 `1..200` 分页校验。

## 3. 不变量与读写约定

### 三层金额与来源

- 同一票所有账单必须同当前公司、方向、结算对象、原币与本币，且是有效普通账单。费用必须来自该账单的当前有效普通经济行；不可拿别账单、已失效、重复或越公司来源凑额。
- `sum(费用含税分配)=账单分配`，`sum(账单分配)=票总额`。本次所有票对同一费用累计分配不得超过实时剩余额；同一次多票共用一个提交账单基线，不能要求第二张票预填第一张写后的版本。
- 金额复用 `numeric(28,8)`：含税额严格大于0，税额满足 `0<=tax<=gross`，未税额为二者相减；服务端不算税、不补尾差，也不反写原费用及提成税基。值域绝对值小于 `10^20`、小数最多8位；领域先对指数 `-8..19` 短路，再比较值域和生成JSON指纹，不能展开巨大指数。独立RPC输入拒科学记数文本。
- `FinanceInvoiceFeeAllocation.source_tax_rate` 为可空不可变来源快照；领域为指针、Schema为Optional/Nillable、Proto为optional。未设保持未设，显式0保持0；界面分别显示 `-` 与 `0%`，不能用 `COALESCE(...,0)` 或页面默认值抹掉存在性。

### 事务、版本与幂等

- 独立命令由 `biz.Transactor.WithinTransaction` 建立共享事务；仓储经 `Data.client(ctx)` 取得客户端。按主键固定顺序锁订单闭包→账单→发票；账单/发票悲观锁内复核 `expected_version`、状态、所有有效分配和结算事实。
- `FinanceInvoiceCommand` 以 `(organization_id,idempotency_key)` 唯一，持久保存命令种类、完整规范意图指纹、结果ID与完成标识。并发同键等待唯一索引和命令锁；完成后回放原身份；同键不同意图冲突。失败事务不能留下半条命令或局部票据。
- `RegisterWithinTransaction` 只返回写入身份，供建账组合使用；`RegisterScoped` 和修订只在提交后重读完整结果。提交后读取失败是结果未知，保留原意图与键确认，不能宣称已回滚或换键再建。
- 发票详情、列表完整汇总/关联及登记账单候选经 `Data.withReadSnapshot` 使用只读REPEATABLE READ；现有快照上下文复用，普通写事务内完整读取显式拒绝。头、当前版本行、有效/历史分配、修订历史及操作人不能混用两次修订。
- 列表主分页与汇总按发票实体计算，关联用独立预加载，不能因跨账单关联倍增数量或金额。普通台账加载轻量账单关联；账单内列表还加载逐费分配供追溯。当前金额只累计有效分配，旧修订关联不再累加；历史事实仍可展开。
- 候选的关键字过滤在COUNT/LIMIT前执行，覆盖账单号、源订单号、结算单位历史名及当前代码/中文名/英文名/别名/无声调全拼/首字母；复用 `financePartyKeywordPredicate`，不循环翻页拼全量。

### 直接修订与受控反向

- 只有全部所涉账单既无有效核销、也无已确认对冲时，才允许直接修改日期、号码、类型、金额、手填税额、分配或增删记录；原因必填；备注、作废/红冲原因与修订原因最多500个Unicode字符，Ent必须用MaxRuneLen而非按UTF-8字节缩小上限。缺任一关联账单版本、陈旧版本、超额或权限不足时整次回滚。
- 原票抬头ID及其原始抬头、税号、地址电话、银行快照不从当前主档覆盖。每张旧票用自己的快照，即使两票共用同一profile ID；新增项必须读取当前启用主档。专票仍要求完整地址、电话与银行信息，普通转专用也按该票原快照校验。
- 修订停用旧关联并追加新逐费事实；当前头及行指向新修订序号。不可变 `FinanceInvoiceRevision` 保存原币金额、号/日期/类型/备注、抬头、全部分配、来源税率以及汇率/来源/日期/本币/折本币金额的前后业务快照，另存操作人、时间、原因。分配为空输出 `[]`，不得输出 `null` 让合法移除被当成损坏历史。
- 有任一结算依赖先展示全部关联单号/账单/金额，再按既有授权完成必要反核销、反转或调整，之后重读。修订、作废、红冲均在写锁内再次检查，不隐式撤销资金或对冲，也不假设反向操作必然可执行。
- 反向成功仅停用原活动关联并推进账单版本；余额由活动事实派生，不能重复加回原金额。另一张票、有效核销/对冲或免票仍存在时保持费用保护；最后一项解除才恢复业务允许的增删及后续系统汇率更新资格，不解除人工订单审核锁。
- 前端登记辅助（`features/finance/invoice-registration/autofill.ts`）只改表单值，提交仍走完整三层校验：单账单仅一条可登记费用时自动建票并选好来源，发票总额经账单分配带入费用；费用来源税率明确时按含税口径 `G×r/(100+r)` 四舍五入两位**预填可编辑建议税额**并标注“建议值，请以实际发票税额为准”，税率未设置不建议（留空必填），显式0建议0；混合税率逐费计算，不补尾差。字段区分 `auto`/`manual` 来源，手改值不被后续联动覆盖，分配不平时显示发票与账单差额并阻止提交；“按可登记余额分配”须用户主动触发（已有分配先确认），按账单/费用顺序填至扣除本会话其他票占用后的余额，不补齐。修订预填全部视为手改，并展示原值→修改后对照。
- 前端已有单据每15秒前台探测、回焦立即探测；不以探测新版本替换提交基线。发现变动保留输入并持续提示，刷新覆盖先确认。关闭/切公司/卸载或在途编辑后旧响应不得关闭、覆盖新会话；结果未知保留同意图键。历史仅结构化显示业务内容，时间/金额/汇率使用统一格式化，不显示原始JSON或内部UUID/版本。

## 4. 错误矩阵

| 触发 | 结果 |
| --- | --- |
| 非法字段、0含税额、负税、税额大于含税、精度/指数越界、三层不守恒或重复来源 | `FINANCE_INVOICE_INVALID_ARGUMENT`，整次拒绝；不分摊纠错 |
| 账单/票据陈旧版本 | 对应账单/发票版本冲突，HTTP409；保留原输入与基线 |
| 跨公司/方向/对象/币种/本币、非有效普通账单 | 领域非法来源/不匹配错误；不改挂来源 |
| 逐费累计超实时余额 | `FINANCE_INVOICE_ALLOCATION_EXCEEDED`，HTTP409 |
| 同键不同意图 | `FINANCE_INVOICE_IDEMPOTENCY_CONFLICT`，HTTP409 |
| 抬头不属来源客商、停用、不完整或试图换原票抬头 | `FINANCE_INVOICE_PROFILE_REQUIRED`；不刷新旧快照掩盖 |
| 号码唯一冲突 | `FINANCE_INVOICE_TAX_NO_EXISTS` / `FINANCE_INVOICE_RED_NO_EXISTS`，数据库唯一索引兜底 |
| 有有效核销或已确认对冲时修订/作废/红冲 | `FINANCE_INVOICE_SETTLEMENT_DEPENDENCY`，HTTP409，提示先核对并处理依赖 |
| 有有效免票时登记，或有有效票时标记免票 | 互斥冲突，整次回滚 |
| 任一新增记录无create权限，或当前公司范围不允许 | 权限拒绝，不退化成只更新/只建账 |
| 提交已完成但完整响应读取失败 | 明确结果未确认；同键重试，不另建财务事实 |
| 正式迁移遇到缺票号/日期/真实登记人时或无法核对原逐费来源 | 显式“迁移受阻”，整个迁移回滚；不删除数据或伪造历史 |

## 5. 正常、基准与错误样例

- 正常：来源费用100，先登记30/税3.14159265，未税26.85840735，余额70；再登记20/税0，累计50、余额50。两张票只作废一张，另一张仍保护整张账单费用。
- 基准：一张票明确向两张同单位USD账单分配30与20，形成一张50的票及两个真实来源关联；台账计数仍为1。再次按原键提交回放同一票据。
- 错误：第一张票60、第二张50同时分到余额100，第二张超额时全部回滚，不能留下第一张、编号/命令或账单版本变化。
- 修订：原40改70，旧40分配留为失效历史，新有效额70，当前余额30；移除登记后余额100且历史仍可读，不把40+70当当前110。
- 受控更正：票跨A/B两张账单，B已有核销时不能只在A弹窗更改整票或直接作废；须先呈现B依赖，按原权限完成反向处理后重新读取全部基线。

## 6. 生成、迁移与验证入口

Schema为真相源，新增 `FinanceInvoiceFeeAllocation` / `FinanceInvoiceCommand` / `FinanceInvoiceRevision`，票据及行增加修订序号；正式迁移为 `server/migrations/20261114140000_finance_invoice_registration.sql`。同一账单可多张部分登记，移除旧 `finance_invoice_bills_active_bill_unique`，逐费事实用 `invoice_bill_id+bill_line_id` 唯一及含税/未税/税额CHECK。旧记录只从当时账单行快照补齐，来源计数/金额/税额必须核对一致，未设来源税率保持NULL；无法确定的草稿或已删除来源显式拒绝。不上线运行期双流程、不清库。

按 `.proto` / Ent Schema 修改后依次执行API/Ent生成、`pnpm run generate:web-client`；权限键变动执行 `pnpm run generate:permission-keys`，源与生成物同组提交。不得手修生成文件。

定向入口（PG须显式配置隔离 `RONCIN_INTEGRATION_DATABASE_SOURCE`，SKIP不算通过）：

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'TestInvoiceRegistration|TestNormalizeInvoice|TestInvoiceRevision|TestInvoiceAllocation|TestInvoicePartyProfilesUseCreatePermissionAndCurrentCompany'
go -C server test -p 32 ./internal/server -run '^TestInvoiceCreationPartyProfilesHTTPRoute$'
go -C server test -p 32 ./internal/data -run '^TestFinanceInvoice(Registration|BillListAllocationProjection|CreationBillKeyword|RevisionReadSnapshot|Terminal|Exemption)' -count=1 -timeout=5m
pnpm --dir web exec vitest run src/features/finance/invoice-registration/autofill.test.ts src/features/finance/invoice-registration/InvoiceRegistrationModal.autofill.test.tsx src/features/finance/invoice-registration/InvoiceRegistrationModal.test.tsx src/features/finance/invoice-registration/InvoiceRevisionHistory.test.tsx src/pages/finance/bills/components/BillInvoiceRegistrationSection.test.tsx
```

其他现有测试：`finance_invoice_text_length_integration_test.go` 验证登记/修订备注、作废/红冲原因的500中文接受及501拒绝；`finance_invoice_registration_migration_integration_test.go` 验证正式前态/升级回滚/NULL与0；`finance_invoice_review_integration_test.go` 在读头后另一连接提交真实修订，校验本请求旧快照、下请求新事实；`settlement_invoice_registration_test.go` 验证新增权限与wire存在性；共享编辑器/载荷/会话测试覆盖手税、三层、旧快照、失败/迟到、未知结果原键及当前公司。

## 7. 错误与正确对照

```text
错误：服务端或前端静默按费用税率补税、补尾差后直接提交；部分登记即视整账单已开票。
正确：前端仅预填可编辑、带“建议值”标注的税额，提交以财务确认的表单值为准，未税=含税−税；进度按有效逐费分配累计。

错误：按profile ID合并旧票快照；修订覆盖旧分配；只存号码和原币头。
正确：按原票身份保留各自抬头；旧分配失效并追加；保存完整经济前后快照。

错误：list用关联JOIN后直接LIMIT；无links仍把金额显示0、票据显示已释放。
正确：以发票主实体分页/汇总，独立预加载真实关联；历史与当前分配分开表达。

错误：头SELECT后并发修订，再取新分配；结果未知换键再建；空分配编码null。
正确：整次只读快照；保持同意图键确认；合法空分配输出[]。

错误：把新静态候选放到已注册的invoices/{id}后，HTTP被当成详情ID。
正确：创建候选使用独立invoice-creation命名空间，以真实生成HTTP注册测试验证方法与Operation分发，同时验证正常详情不变。
```
