# 方向身份组实施验证记录

日期：2026-11-16。本记录仅覆盖方向身份子任务；不代表进口、航空直单、结构化运单、PDF 或完整 CSV 已交付。

## 正式 PostgreSQL 回归入口

以下命令在仓库根目录实际执行。Node 只在子进程环境中传递本机数据库配置，不输出连接串。集成测试通过正式有序迁移建立随机隔离 Schema，测试后清理自己的 Schema；没有对开发库应用新迁移或清空数据。

```bash
node --env-file-if-exists=.env.local --input-type=module - <<'JS'
import { spawnSync } from 'node:child_process';
const pattern = '^Test(SeaMasterIdentity.*Postgres|AirMasterIdentity.*Postgres|AirExport.*Postgres|AirWeightConfirmationAndFeeSourcePostgres|OrderCreateTransactionPostgres|SeaMasterBillBatchRulesPostgres|SeaMasterBillCreatePreservesSharedContentPostgres|SeaOrderRestore.*Postgres|SeaDocumentChangePostgresFlows|SeaDocumentTerminatedHistoryPostgres|SeaSharedTransportPreservesIndependentDestinationsPostgres|ClosedOrderManualReviewPostgres|MapEntConstraintPostgres|SeaOrderSplitAndReassignment_PostgresIntegration|SeaMasterContentRejectsConcurrentMemberPostgres|SeaChangesRejectInvalidMasterMembersPostgres|OrderMasterDirection(Migration.*|DatabaseConstraints)Postgres)$';
const result = spawnSync('go', ['-C', 'server', 'test', '-p', '32', './internal/data', '-run', pattern, '-v', '-count=1'], { env: { ...process.env, RONCIN_INTEGRATION_DATABASE_SOURCE: process.env.DATABASE_SOURCE }, stdio: 'inherit' });
process.exit(result.status ?? 1);
JS
```

首轮耗时 **144.383 秒**，整体退出码为 1：以下两主项 FAIL，其余已选主项全部 PASS，**无 SKIP**。不能将该轮整体记为通过。

### 首轮两处失败与修复

1. `TestAirExportSharedWaybillPostgres`：跨公司读取原应返回 `ErrAirWaybillNotFound`，完整成员查询改造后在主单可见性校验前检查关联公司，导致返回冲突。`lockAirMasterMembers` 增加非锁定的公司/AE 主单定位，先保持不存在语义；原有 Order→MAWB→完整 Link 锁序和锁内身份复验保留。
2. `TestMapEntConstraintPostgres`：仍断言旧海运唯一索引截断名，而正式约束已变为 `sea_master_bill_business_no`。同步更新该测试及 `order_write_sea_master_bill.go` 两处、`sea_order_change_shared.go` 一处 `mapEntConstraint` 索引名，保持原业务错误映射。独立检查代理已静态复核上述两类修复。

### 首轮通过范围

- 新方向三项：`TestSeaMasterIdentityCandidatesAndDirectIDPostgres`、`TestSeaMasterIdentitySharedMembersAndRestorePostgres`、`TestAirMasterIdentityCandidatesWritesAndRestorePostgres`。覆盖同号跨方向独立、候选过滤、跨方向直接 ID 创建/预览/恢复拒绝、失败不部分写入、零活动成员身份及异常共享成员。
- 正式迁移三项：`TestOrderMasterDirectionMigrationPreservesHistoricalRowsPostgres`、`TestOrderMasterDirectionMigrationRejectsUnclassifiableRowsPostgres`、`TestOrderMasterDirectionDatabaseConstraintsPostgres`；含 13 个异常迁移子场景与海空两主单约束子场景。
- 航空相邻五项：`TestAirExportRequiredNumbersUniquenessAndRollbackPostgres`、`TestAirExportTerminationRestoreAndConcurrentIdentityPostgres`、`TestAirExportFlowVersionsAndSnapshotsPostgres`、`TestAirExportAuditRollbackAndOverduePostgres`、`TestAirExportMemberChangeAndLockRacePostgres`；另 `TestAirWeightConfirmationAndFeeSourcePostgres` 通过。
- `TestOrderCreateTransactionPostgres`：16 个创建、发号、引用、并发、主单共享与资料修改子场景。
- 海运共享：`TestSeaMasterBillBatchRulesPostgres`（5 子场景）、`TestSeaMasterBillCreatePreservesSharedContentPostgres`、`TestSeaMasterContentRejectsConcurrentMemberPostgres`、`TestSeaSharedTransportPreservesIndependentDestinationsPostgres`。
- 海运单证：`TestSeaDocumentChangePostgresFlows`（6 子场景）、`TestSeaDocumentTerminatedHistoryPostgres`。
- 拆票改配：`TestSeaOrderSplitAndReassignment_PostgresIntegration`（20 子场景）、`TestSeaChangesRejectInvalidMasterMembersPostgres`（4 子场景）。
- 恢复：`TestSeaOrderRestorePostgres`（DIRECT/HOUSE）、`TestSeaOrderRestoreBlockedPostgres`（9 阻断子场景）、`TestSeaOrderRestoreConcurrentPostgres`、`TestSeaOrderRestoreRepairPostgres`、`TestSeaOrderRestoreClosedUnlockedMemberPostgres`。
- 审核锁相邻路径：`TestClosedOrderManualReviewPostgres`（SI/SE）。

## 修复后最小补跑

两处修复后实际执行以下命令，不重复已通过且未受此次修改影响的迁移等路径：

```bash
node --env-file-if-exists=.env.local --input-type=module - <<'JS'
import { spawnSync } from 'node:child_process';
const result = spawnSync('go', ['-C', 'server', 'test', '-p', '32', './internal/data', '-run', '^Test(AirExportSharedWaybillPostgres|MapEntConstraintPostgres|OrderCreateTransactionPostgres|SeaMasterBillBatchRulesPostgres|SeaOrderSplitAndReassignment_PostgresIntegration)$', '-v', '-count=1'], { env: { ...process.env, RONCIN_INTEGRATION_DATABASE_SOURCE: process.env.DATABASE_SOURCE }, stdio: 'inherit' });
process.exit(result.status ?? 1);
JS
```

结果：**PASS，16.608 秒，退出码 0，5 主项全部通过，0 FAIL、0 SKIP**。

- `TestAirExportSharedWaybillPostgres`：PASS，2.66 秒。
- `TestMapEntConstraintPostgres`：PASS，2.44 秒，3 子场景。
- `TestOrderCreateTransactionPostgres`：PASS，3.25 秒，16 子场景。
- `TestSeaMasterBillBatchRulesPostgres`：PASS，5.25 秒，5 子场景。
- `TestSeaOrderSplitAndReassignment_PostgresIntegration`：PASS，2.96 秒，20 子场景。

## 生成幂等与局部检查

实际使用 Python `hashlib.sha256` 对生成目录的文件内容建立生成前后指纹并比较文件集合（同时检查新增/删除）；生成命令失败会立即以失败退出。

- `go -C server generate ./internal/data/ent`：PASS。比较 `server/internal/data/ent/`，以及同次快照中的 API/OpenAPI/前端客户端路径，**变化文件数 0**。
- 顺序执行 `make -C server api`、`pnpm run generate:web-client`：PASS。比较 `server/api/`、`server/openapi.yaml`、`web/config/openapi.generated.json`、`web/src/services/roncin/` 与存在的 `web/src/constants/proto.generated.ts`，**变化文件数 0**。指纹比较不证明不存在的路径包含生成内容。
- `go -C server test ./internal/data ./internal/service ./internal/data/ent/migrate -run 'TestMaster(ContentHash|Candidates)|TestGeneratedMigrateTables_CheckConstraints'`：三个包 PASS。覆盖主单哈希身份、候选必须显式提交出口身份、三表生成 CHECK 完整表达式。
- 最后两类修复及补跑之后 `git diff --check`：PASS。

主会话和独立检查代理另有纯包、前端及审查记录，见 `direction-validation.md`、`direction-review.md`；本文件不把他人报告的检查冒充本代理重复执行。

## 限制与交接

- 开发库只读预检发现的孤立海运主单按批准合同阻断升级；本代理没有删除、猜测归类或修改该记录，没有放宽迁移。实际业务数据值不写入本记录。
- 本代理未运行父任务最终 `check:fast` 或构建，未提交 Git；主会话负责完整门禁、统一提交和后续子任务。
- 新测试早期调试中曾修复夹具的公司本位币、跨公司关联清理顺序及航空终止原因；最终上述三项方向测试均在正式 PostgreSQL 回归中通过。
