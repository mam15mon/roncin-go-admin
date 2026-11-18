# 当前实现、来源与变更边界

## 用户决定

2026-11-18，用户在本聊天批准创建任务并规划，并明确选择“也限制未开票账单核销”。该决定允许改变此前先核销后开票的业务行为，尚未完成最终规划评审批准。

## 本系统源码入口

- web/src/pages/finance/bills/index.tsx:65：BillSearchParams及筛选设置；:593固定CONFIRMED，:605只传独立进度，尚无五态多选。
- web/src/pages/finance/bills/components/billProgressStatus.ts:8：前端按双进度、completed及核销额重算五态；CSV复用，需统一消费服务端投影。
- server/api/finance/v1/settlement.proto:499：ListBillsRequest当前18号字段为completed；:599 FinanceBill读取投影到60号invoice_exemptions。新增字段应追加，生成器决定派生物。
- server/internal/service/settlement_bill.go:18：ListBills边界转换。
- server/internal/biz/finance_bill_progress.go:79：ResolveFinanceBillProgress；server/internal/biz/finance_bill.go:59列表校验。
- server/internal/data/finance_bill.go:40：List全部谓词共用于matchedSummary、total与summary，分页后再enrich投影；finance_bill_credit.go现有开票／结算／完成SQL定义可复用。
- server/internal/data/finance_verification.go:150：候选仅CONFIRMED+OnlyUnsettled；:224 Create现有订单→资金→账单加锁，目前没有开票准入。
- server/internal/biz/finance_bill_batch_operations.go:196：组内先RegisterWithinTransaction，再CreateWithinTransaction或CreateWithCashflowWithinTransaction，必须保留原子语义。
- web/src/features/finance/bill-creation/BillGroupCard.tsx:271：发票编辑器；:301无条件挂载BillVerificationFields。后者当前按核销权限加载资金候选，尚未依赖本组发票登记前置。
- web/src/pages/finance/verifications/VerificationWorkbench.tsx:272：服务端账单候选加载，保留分页与跨页选择；verification-candidates.ts校验失败与合法空列表，不能在这里过滤当前页冒充服务端准入。
- web/src/components/ui/search-filter-template/types.ts：fieldProps现成支持透传Select多选，不需要扩建模板。

## 海管家只读来源

本轮本机原始证据与脱敏结论在 /home/vince/.local/share/rea/hgj/bill-status-20261118/，不复制原始业务值／账号／报文进Git。可复核本机核对结论.md及session.evidence.json。

- 状态选项和查询数组已实测；近两年未开票116张全取样（含4张对冲类型），核销额均0；部分核销4张全取样均票齐但有余额；已完成7281张仅抽查40张，均票齐且结清；部分开票查询0张。
- 当前Bill脚本只映射服务端state，不能从前端读出后台完整公式。EditBill普通账单核销入口要求state=[1,2,4]；已有未开票详情不显示核销，未核销／部分核销详情显示。
- 对冲类型样本票齐未核销仍显示未核销；没有验证实际已完成对冲的全部算法。
- 海管家invoiceAmount不是本系统“有效已登记额”的同义字段，未开票时也可能有值；不能照搬金额字段推导。
- 未写外站单据。免票／部分开票交叉规则未证实，采用本任务PRD明确的规则，不声称完全复刻。

## 约束与已有工作

任务前Git干净，本轮产品代码未修改。根AGENTS与server/AGENTS有效；禁止TDD，功能完成后补验证；正式迁移初始化隔离测试，不对开发库重置。没有Schema、权限、依赖或构建配置变化。
