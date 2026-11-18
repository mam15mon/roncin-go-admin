# 服务端实施与验证记录

日期：2026-11-18（北京时间）。用户已批准方案，服务端实现、生成物及针对性验证完成；整包真实 PostgreSQL 扫描未通过，详见下节，不将其记为全量成功。

## 实现

- Proto 新增 FinanceBillBusinessStatus（0未指定、1未开票、2部分开票、3未核销、4部分核销、5已完成），ListBillsRequest.business_statuses=19，FinanceBill.business_status=61。finance ErrorReason 新增 FINANCE_VERIFICATION_INVOICE_REQUIRED=17。
- biz 统一五态读取解析及枚举校验；data 在原列表共同谓词中组合业务状态OR／其他筛选AND，列表、分页前matchedSummary、顶部指标共用。列表与详情返回相同业务状态投影。
- 核销候选在Count/LIMIT之前过滤完全未开票账单；新增核销在原有账单ForUpdate锁内重新读取有效发票／免票事实，缺前置返回HTTP409、“账单尚未开票，请先登记发票后核销”。未改变既有锁序、Schema、事务封装与幂等回放。
- 独立、多账单、差额流水、建账同次核销共用最终准入；部分开票、票齐及有效免票保留正常核销，反核销继续可办理。
- 正向核销夹具补现行ISSUED票据（号码、日期、登记人、登记时间、汇率、折本币及抬头快照）或明确免票事实；不以退役的登记草稿替代正向前置。更正结算夹具使用免票事实避免改变其被测更正账务路径。

## 已通过

1. `make -C server api`、`pnpm run generate:web-client`：生成Go绑定、OpenAPI及TS客户端／枚举／错误原因，无手改生成物。
2. `pnpm run lint:proto`；`go -C server vet ./internal/biz ./internal/service ./internal/data ./internal/server`。
3. `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'Test.*(FinanceBill|Verification|BillBatch|BillBusinessStatus|ListBillsBusinessStatuses)'`。biz/service/server全部PASS；覆盖五态及纯对冲／带符号／零额规则、非法枚举、真实ProtoJSON数字wire与省略0、生成HTTP重复参数camelCase／snake_case／枚举名称／Axios []形态绑定。
4. 真实PG专项（**最终ISSUED夹具版本**，31.014秒，PASS）：

```bash
go -C server test -p 32 ./internal/data -run 'Test(FinanceBillBusinessStatusFilter|VerificationInvoicePrerequisite|VerificationCreateSharedTransaction|VerificationShortfallAtomic|VerificationCandidatesSearchPagination|VerificationCreationCandidatesBeyondSettledLimit|FinanceBillComposition|SettlementDoesNotReviewOrLockOrders|SystemPinyinSearch).*Postgres' -count=1 -timeout=5m
```

5. `go -C server test -p 32 ./internal/data -run 'TestFinanceCorrection(MergedSettlement|SuccessiveCashCoverage)Postgres' -count=1 -timeout=5m`：补合法免票前置后的两条更正路径PASS，7.061秒前一轮因测试defer先关数据库导致cleanup报错，改为t.Cleanup后7.307秒PASS。
6. `git diff --check`。

PG运行通过进程环境注入 RONCIN_INTEGRATION_DATABASE_SOURCE，使用既有专用 roncin_go_admin_integration 库。每个测试通过getIntegrationData应用正式SQL迁移到自己创建的隔离Schema，只清理自身夹具；没有重置开发数据库、执行运行期Schema.Create或输出连接凭据。

## 新PG矩阵

- 五态逐行投影与SQL集合一致；单选、五态全选、两组OR、重复状态、其他条件AND、冲突条件空结果、跨页total和完整金额不变、取消及其他公司排除。
- 未开票、历史已结清未开票、部分票且部分核销、票齐未核销、纯部分对冲、部分实际核销、核销＋对冲结清、免票待核销、免票完成。
- 无票候选排除与直接提交拒绝；部分开票允许核销超过已登记票额（不新增另一额度）；票齐／免票成功；撤票／撤免票后旧候选失效；多账单混入一张无票整体回滚；成功后同键重放保留；反核销。
- 缺票的差额核销不留下新资金、核销、编号序列或审计；建账未登记票却同次核销整组失败；建账登记发票后可核销，原组合并发／锁序／8位金额路径正常；200条候选分页不在前端删行。

## 整包PG扫描：失败且超时，未算通过

曾执行 `go -C server test -p 32 ./internal/data -count=1 -timeout=15m`（同一专用隔离PG环境）。结果为FAIL，900.075秒，最终 `panic: test timed out after 15m0s`，当时正在 `TestOrderFeeSavePostgres/任一行失败整体零落库并带行级定位`。这次编译早于后续新专项及更正夹具修正，不据此否定已单独重跑成功的本任务路径，也不宣称全PG通过。

超时前还观察到以下范围外问题：

- `TestAirWaybillMigrationPreservesTextAndHistoryPostgres`、`TestAirWaybillMigrationStopsInvalidLegacyRelationsPostgres`：历史前态夹具cleanup尝试删除尚不存在的 `order_import_handling_events`，位置 `order_transaction_integration_test.go:1255`。
- `TestBillingUnitCatalogColdStartPostgres`：正式迁移单位目录的search_keywords空值与当前开发初始化目录／拼音检索预期不同，且重复迁移断言失败。
- 多条提成调整、结案待审更正、费用更正测试在共同旧夹具 `order_fee_supplement_transaction_integration_test.go:531` 写入人工审核锁时违反 `order_lock_records_business_type_document_refs_check`；包括 `TestCommissionAdjustmentCancelGatePostgres`、`TestOrderClosurePendingCorrectionOrderedConcurrencyPostgres`、`TestFeeCorrectionApprovalAcceptancePostgres`、`TestFeeCorrectionEconomicOperationsPostgres`、`TestFeeCorrectionRevenueAndPartyPostgres`、`TestFeeCorrectionPermissionsVersionsAndConcurrencyPostgres`、`TestFeeCorrectionApprovalRollbackPostgres`、`TestFeeCorrectionApprovalSerializesCommissionPaymentPostgres` 等。

本任务没有修改这些范围外模块或放松约束。整包超时中断可能使末次测试的隔离Schema未运行t.Cleanup，本轮没有按名称前缀批量清理，避免删除其他会话的测试Schema；没有改变开发业务库。完整任务门禁由主会话统一执行并另记结果。

## 正式约束限制

`20260826160000_finance_bills.sql` 保留 `finance_bills_amount_positive`（总额、本币额均>0），`20260826190000_finance_verifications.sql` 保留 `verification_allocation_amount_positive`（分摊额>0）。因此本期无法在真实PG构造零额账单、负总额账单或负核销额作为五态矩阵；这些边界由biz纯Go表格验证。没有为构造测试删除CHECK，未把纯Go验证称为真实PG成功。真实更正账单通过正金额与方向表达经济冲回。
