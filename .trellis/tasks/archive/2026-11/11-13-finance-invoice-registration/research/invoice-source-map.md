# 发票数据链与剩余差距

## 已有入口

- `server/internal/biz/finance_invoice.go`：FinanceInvoice、FinanceInvoiceBill、FinanceInvoiceLine；Create/Issue/Cancel/RedFlush、CreateScoped、buildFinanceInvoice。
- `server/internal/data/finance_invoice.go`：ListCreationBills、ListProfilesForBill、LoadBillsScoped、Create、Issue、lockAndReleaseInvoiceActiveBills、Cancel/RedFlush；均经Data.client/WithTx。
- `finance_invoice_economic.go`：invoiceableFinanceBill过滤有效INCREASE经济行，validateInvoiceEconomicSnapshot按整额快照校验，checkCorrectionInvoiceAncestors保护尚有原票的更正来源。
- `server/internal/service/settlement_invoice.go`：按各动作权限解析公司范围及DTO转换。
- `web/src/pages/finance/invoices`：现有创建草稿、开具/作废/红冲和台账；账单页尚无多记录分配录入能力。

## 已完成但尚待提交的本轮免票

新增FinanceInvoiceExemption实体及独立权限system.finance.invoice.exempt，SetBillInvoiceExemption/RevokeBillInvoiceExemption返回账单投影；历史从详情读取，姓名批量查询。fee_protected_fact、账单SQL/展示进度及取消依赖已接入。已验证的边界见exemption-and-filters-ui.md及finance_invoice_exemption_integration_test.go；不能把此部分视为逐费登记已完成。

## 需要替换的整额假设

1. CreateFinanceInvoiceInput只有BillIDs等字段；buildFinanceInvoice取整张invoiceable amount/net/tax、全部有效来源费，并按项目和税率归集。
2. 候选及抬头查询目前排除任何已有活动发票关联的账单，无法继续部分登记。
3. 仓储Create再次拒绝任何活动关联，validateInvoiceEconomicSnapshot假设账单关联等于全部当前可开票经济内容；仅改输入总额不能形成正确逐费快照。
4. FinanceInvoiceBill只有SourceFeeIDs与总额/税额，没有逐费金额，不能凭集合追溯部分登记。应新增不可变逐费事实，正式迁移补齐能确定的原整额快照；不运行期兼容。
5. 现有Create写DRAFT，Issue再记录号码/日期与汇率；新的账单多条已有发票登记须在单次原子命令保存全部信息，界面不能假装在线开票。
6. 取消/红冲按活动父关联释放并推进账单版本，新增逐费事实必须沿父关联有效性释放，不能重算当前来源费用覆盖历史。

## 相邻投影

`settlement.go.applyFeeLedgerBillProjection`与feeLedgerFinancialProgressPredicate当前以存在ISSUED关联当作开票信号；新增部分登记和免票后要同步，不能把部分登记当票齐、把免票伪装已开票，不能只改详情文案而不改筛选。

建账组合可复用已有Transactor/WithTx，不串联前端独立写入；完整响应提交后重读。核销差额输入已由同级任务提供，具体见该任务design。多人已有账单登记沿15秒/回焦检查，不推进提交基线；失败/迟到不关闭或覆盖新会话。

## 已确认金额

财务逐费用填本次含税金额和税额，0≤税额≤含税额，未税=含税−税额，逐层求和且不超实时余额；复用项目现有decimal精度，不自动按税率算税或补尾差，不反写费用。来源税率只保留来源追溯。只登记已有票据，直接修改范围待当前问题回复。

## 本轮只读开发前态盘点

2026-11-14仅按发票状态聚合查询，无写入：ISSUED 2张、DRAFT 0张，2张号码/日期均非空。未读取或记录真实号码、公司/客商或金额。正式迁移对无法成立的未完成登记仍须显式拒绝，不能以本次开发库无草稿推断所有目标环境无草稿。
