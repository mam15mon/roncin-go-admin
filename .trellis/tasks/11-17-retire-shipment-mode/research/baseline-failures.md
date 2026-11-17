# 扩展 PostgreSQL 整包失败的基线复现

本次额外将全部真实 PostgreSQL／正式迁移／实时 worker 验收环境启用后运行 `pnpm run check:fast`。前端分支通过；后端 data 与 platform/migration 包失败，共22个顶层用例。没有包级超时，data整包执行1410.715秒。

## 本次修复的2项

- `TestBusinessTagRetirementUpgradePostgres`
- `TestOrderMilestoneRetirementUpgradePostgres`

原因：旧迁移的全行不变断言把后续合法删列也计为副作用。按照现行数据库质量规范改为正式目标前缀的首次／重复快照验证，再独立升级完整链；不删除原字段快照，不增加shipment_mode排除。当前权限同步在最新结构后检查，原授权断言保留。

修正后定向真实库结果：2 PASS、0 FAIL、0 SKIP，且通过独立只读复核。

## 修改前同样失败的20项

使用 `git archive 4c05c13a server` 创建未修改的实现前源码副本，使用相同隔离PostgreSQL环境和相同测试名称复现；没有改基线代码、迁移或断言。首组3项覆盖计费单位初始化、公共审核锁夹具和旧主单索引；剩余17项全部逐项执行，20项均再次失败，没有漏项或SKIP。

| 类别 | 实际现象 | 顶层用例数 |
| --- | --- | --- |
| 计费单位初始化 | 正式迁移的单位记录与开发初始化目录口径不一致 | 1 |
| 共享财务审核锁夹具 | 建立审核记录时违反既有 `order_lock_records_business_type_document_refs_check` | 18 |
| 历史主单索引断言 | `TestPostgresSeaShippingLineIdentityMigration` 查找的 MBL 唯一索引不存在 | 1 |

基线复现的完整顶层清单：

- `TestBillingUnitCatalogColdStartPostgres`
- `TestCommissionAdjustmentCancelGatePostgres`
- `TestCommissionAdjustmentMySourcePostgres`
- `TestFinanceBillFeeOverrideSearchPostgres`
- `TestFinanceEconomicOrderSetExpansionPostgres`
- `TestClosedOrderFeeContentEditableUntilReviewPostgres`
- `TestOrderClosurePendingCorrectionReadinessPostgres`
- `TestOrderClosurePendingCorrectionClosedWorkflowPostgres`
- `TestOrderClosurePendingCorrectionOrderedConcurrencyPostgres`
- `TestFeeCorrectionApprovalAcceptancePostgres`
- `TestFeeCorrectionEconomicOperationsPostgres`
- `TestFeeCorrectionRevenueAndPartyPostgres`
- `TestFeeCorrectionPermissionsVersionsAndConcurrencyPostgres`
- `TestFeeCorrectionApprovalRollbackPostgres`
- `TestFeeCorrectionApprovalSerializesCommissionPaymentPostgres`
- `TestOrderFeeSaveCommissionReconcileOncePostgres`
- `TestOrderFeeTaxRateReconcilesCommissionPostgres`
- `TestUnifiedFeeBillingAndCorrectionAdmissionPostgres`
- `TestOrdinaryFeeMutationReconcilesPaidCommissionPostgres`
- `TestPostgresSeaShippingLineIdentityMigration`

这些失败与本次模式字段／自拼汇总退役没有新增因果，不为让扩展整包显示通过而修改产品约束、删断言、加豁免或跳过。它们作为现有测试问题留档；不得把该次扩展整包执行记录为成功。

本机原始日志：`/tmp/roncin-retire-shipment-check-fast.log`、`/tmp/roncin-retire-baseline-failures.jsonl`、`/tmp/roncin-retire-baseline-remaining.jsonl`、`/tmp/retire-historical-upgrade-final.jsonl`。这些是本机证据位置，不是仓库提交物；本文件不含连接凭据。
