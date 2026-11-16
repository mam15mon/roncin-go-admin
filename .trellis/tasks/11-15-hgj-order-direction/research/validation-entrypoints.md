# 调研：方向隔离阶段定向验证入口与夹具影响

- 问题：海空主单业务类型新增后，哪些真实测试能验证创建、候选、共享、恢复、锁/版本与正式迁移；哪些直接Ent夹具需补显式身份？
- 范围：仓库内部只读代码；仅写本研究文件，未运行测试、生成、迁移或数据库操作，未读取环境文件/凭据。
- 日期：2026-11-16。
- 当前有实现代理并行修改。下列行号以调研时源码为准，夹具清单是修改前快照，不应把它当最终仍失败的文件清单。

## 可并行验证建议

1. 实现/生成期间只读核查测试名和上下文，不并发运行第二套生成或修改夹具。
2. Go源码/生成物稳定后，先跑biz/service纯逻辑、Ent迁移元数据和迁移运行器单元测试，再跑专用数据库下的方向新用例及相邻出口suite。
3. 前端经营字段代理可先独立跑adapter/模板定向Vitest和修改文件Biome；候选接口源/客户端生成后再跑海运候选和航空主单权限用例。
4. 所有PostgreSQL测试必须确认确实执行，未设置专用环境变量会SKIP。用例存在、退出码0但全SKIP均不代表数据库验证通过。

## 后端真实入口

| 入口 | 主要覆盖 |
| --- | --- |
| `server/internal/data/order_transaction_integration_test.go:54` `TestOrderCreateTransactionPostgres` | 正常建票、共享MBL/运输同步与事务失败；基础fixture在`:895` `newOrderPostgresFixture` |
| `server/internal/data/sea_master_bill_batch_integration_test.go:36` `TestSeaMasterBillBatchRulesPostgres` | 候选确认入批、过期版本、DIRECT阻断、批次分单排重及数据库兜底；T5迁移异常样本在`:322` |
| `server/internal/data/sea_master_bill_create_content_integration_test.go:22` `TestSeaMasterBillCreatePreservesSharedContentPostgres` | 加票不覆盖共享主单正文及既有成员状态 |
| `server/internal/data/air_export_integration_test.go:72` `TestAirExportSharedWaybillPostgres` | 共享MAWB及本票HAWB；夹具`:25` `newAirExportFixture`复用公共订单fixture |
| 同文件`:151` `TestAirExportRequiredNumbersUniquenessAndRollbackPostgres` | 必填号码、唯一性、失败回滚 |
| 同文件`:185` `TestAirExportTerminationRestoreAndConcurrentIdentityPostgres` | 终止恢复关联与并发身份 |
| 同文件`:246` `TestAirExportFlowVersionsAndSnapshotsPostgres` | 业务流程、单证版本与快照 |
| 同文件`:354` `TestAirExportAuditRollbackAndOverduePostgres` | 审计失败回滚及超时处理 |
| 同文件`:438` `TestAirExportMemberChangeAndLockRacePostgres` | 成员变化和审核锁竞争 |
| `server/internal/data/sea_order_restore_integration_test.go:49/:106/:162/:215/:282` | `TestSeaOrderRestorePostgres`、`TestSeaOrderRestoreBlockedPostgres`、`TestSeaOrderRestoreConcurrentPostgres`、`TestSeaOrderRestoreRepairPostgres`、`TestSeaOrderRestoreClosedUnlockedMemberPostgres`，恢复/阻断/并发/受控修复/已完结未锁成员 |
| `server/internal/data/sea_document_change_integration_test.go:384/:408/:452/:499/:758` | `TestSeaDocumentAmendmentAuditRollback`、`TestSeaDocumentAmendmentConcurrentPublish`、`TestSeaDocumentAmendmentConcurrentIdempotentReplay`、`TestSeaMasterBillMemberSetRevalidatedAfterLock`、`TestSeaDocumentChangeBusinessLockAndLifecycleGate` |
| `server/internal/data/sea_shared_business_gate_integration_test.go:21/:189` | `TestSeaSharedContainerMemberBusinessGatePostgres`、`TestSeaSharedContainerMemberChangeDuringUpdatePostgres`，共享箱成员业务门禁/集合变化 |
| `server/internal/data/sea_document_test.go:600` | `TestSeaDocument_ConcurrentOperationsNoDeadlock`；同文件创建/更新单证suite可做扩展回归 |
| `server/internal/data/order_lock_integration_test.go:32` | `TestOrderLock_PostgresFlows`，海空锁与快照组合 |
| `server/internal/data/air_weight_integration_test.go:19` | `TestAirWeightConfirmationAndFeeSourcePostgres`，方向增加不应破坏重量来源 |

纯逻辑/元数据入口：

- `server/internal/biz/air_export_test.go:11/:33/:74`：`TestAirMasterAndHouseNumberContract`、`TestAirMasterTimeAndAirportValidation`、`TestAirExportPaginationBounds`。
- `server/internal/service/air_export_test.go:14`：`TestAirRestoreLinkRequiresLifecyclePermission`。
- `server/internal/data/ent/migrate/schema_test.go:9`：`TestGeneratedMigrateTables_CheckConstraints`，检查生成注解而非只看Go字段。
- `server/internal/data/migration_manifest_test.go:21`：`TestMigrationManifestCoversEntSchema`，只验证表创建/引用集合，不证明列、索引或回填正确。
- `server/internal/platform/migration/runner_test.go:40/:135/:220`：`TestApplyExecutesNewMigrationInTransaction`、`TestApplyRejectsChangedMigration`、`TestApplyRollsBackFailedMigration`。

## 建议定向命令（未执行）

先在已经配置 `RONCIN_INTEGRATION_DATABASE_SOURCE` 的测试进程运行；不要把连接串直接写进命令、日志或仓库。以下不会为调用者注入凭据。

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'TestAir(MasterAndHouseNumberContract|MasterTimeAndAirportValidation|ExportPaginationBounds|RestoreLinkRequiresLifecyclePermission)$' -count=1
go -C server test -p 32 ./internal/data/ent/migrate ./internal/platform/migration -run 'Test(GeneratedMigrateTables_CheckConstraints|ApplyExecutesNewMigrationInTransaction|ApplyRejectsChangedMigration|ApplyRollsBackFailedMigration)$' -count=1
go -C server test -p 32 ./internal/data -run 'Test(AirExport.*Postgres|SeaMasterBillBatchRulesPostgres|SeaMasterBillCreatePreservesSharedContentPostgres|SeaOrderRestore.*Postgres|OrderCreateTransactionPostgres|SeaMasterBillMemberSetRevalidatedAfterLock|SeaDocumentAmendmentAuditRollback|SeaDocumentAmendmentConcurrentPublish|SeaSharedContainerMemberChangeDuringUpdatePostgres|MigrationManifestCoversEntSchema)$' -count=1 -v
```

新方向专用测试名称由实现确定，应在此基础上另外明确运行，不以原出口suite代替新行为验收。若索引错误映射变化，追加 `TestMapEntConstraintPostgres`；若扩大到锁快照，追加 `TestOrderLock_PostgresFlows`。

## 正式迁移与隔离模式

`server/internal/data/integration_test_helper_test.go:21/:28`：

- `getIntegrationData`调用`getIntegrationDataBeforeMigration(t, "")`。
- 专用环境键为 `RONCIN_INTEGRATION_DATABASE_SOURCE`，缺失在`:30`起SKIP。
- 随机 `roncin_test_iso_<UUID>` Schema于`:52`创建；测试cleanup仅删除自己Schema（`:57`起），不重建/清空开发数据库。
- pgx配置将search_path设为隔离Schema加public（`:72`起）。pg_trgm扩展公共安装与Schema各自索引语义须保留。
- `beforeVersion`非空时复制文件名小于该版本的正式SQL到临时目录（`:84`起），然后通过正式`migration.Apply`执行，`AutoMigrate:false`（`:117`）。不调用`Schema.Create`。

推荐新方向迁移测试：

1. 空Schema完整迁移成功，检查两主单business_type为NOT NULL、有效值CHECK、方向唯一/查询索引与旧索引确已移除。
2. 新迁移前态用原生SQL插入缺少新列的合法出口主单/历史关系，运行完整后续迁移；断言按真实关联回填，号码/正文/版本/时间/活动及已终止关系保留，不做默认假归类。
3. 独立前态分别构造未知订单类型、混进出口关联、跨组织引用等已批准的异常；Apply返回明确预检错误；断言目标迁移revision不存在、回填/列/索引无半完成。海/空各覆盖，不能只造一个成功例。
4. 按新索引验证同公司同号进出口独立、同公司同方向重复失败、跨公司隔离；再验证按ID绑定错方向的业务拒绝及共享成员集合复验。
5. 重复Apply只记录一次revision、不再改写业务资料；Ent注解/生成schema与正式CHECK、索引完全对齐。

可借鉴 `server/internal/data/sea_house_bill_draft_version_integration_test.go:197` 的前态→Apply两次→CHECK/revision断言；`company_name_en_integration_test.go:19` 的前态数据保留；迁移运行器 `server/internal/platform/migration/runner.go:209` 将一个文件的SQL和revision写入同事务，失败rollback。

不要照搬 `sea_master_bill_batch_integration_test.go:322` 中先造最新Schema再手动删索引的旧测试方法作为新迁移主路径：本项目当前规范优先正式SQL前态，新增方向迁移可直接用beforeVersion，无需删除现约束强塞数据。

## 直接Ent主单夹具受影响清单

初次只读扫描：12文件、20处直接主单Create链均未显式SetBusinessType。以下是修改前行号；实现代理已同步收到。共享fixture通过真实用例创建的主单不在此直接链清单中。

| 文件/原行号 | 数量/实体 | 用途 |
| --- | --- | --- |
| `server/internal/data/sea_document_test.go:113/:183/:666/:808` | 4 Sea | DIRECT/HOUSE、并发操作、订单单证校验 |
| `server/internal/data/sea_order_change_integration_test.go:171/:1098/:1242` | 3 Sea | 拆票/改配基础与候选MBL |
| `server/internal/data/order_lock_integration_test.go:278/:497/:1357` | 2 Sea+1 Air | 锁单、航空快照及单证组合 |
| `server/internal/data/fee_ledger_master_no_integration_test.go:33/:53` | 1 Sea+1 Air | 财务主单号码投影/搜索 |
| `server/internal/data/sea_document_terminated_history_integration_test.go:45` | 1 Sea | 终止后历史只读 |
| `server/internal/data/finance_report_date_integration_test.go:21` | 1 Air | 权威航空日期和进口报表日期 |
| `server/internal/data/ent_constraint_integration_test.go:94` | 1 Sea | 数据库索引错误映射，旧索引名也须同步核查 |
| `server/internal/data/sea_cargo_allocation_test.go:110` | 1 Sea | 共享箱与货量分配 |
| `server/internal/data/order_lock_closed_integration_test.go:45` | 1 Sea | 已完结未锁订单的人工审核 |
| `server/internal/data/order_settlement_review_lock_integration_test.go:110` | 1 Sea | 结算与人工审核锁分离 |
| `server/internal/data/sea_document_change_integration_test.go:87` | 1 Sea | 改单/作废/模式切换基础 |
| `server/internal/data/air_weight_integration_test.go:34` | 1 Air | 重量来源/共享成员 |

应给正常已有夹具显式SE/AE，不为照顾旧测试给新字段自动默认值。真正迁移前态测试应继续用旧SQL形状，不为了新Ent编译强加新列。扫描时实现者已经在部分链添加SetBusinessType，故最终以最新源码及真正测试结果核对；本研究未修改这些夹具。

## 前端定向入口

```bash
pnpm --dir web exec vitest run src/pages/orders/order-kinds/air-export/form-adapter.test.ts src/pages/orders/order-kinds/sea-export/form-adapter.test.ts
pnpm --dir web exec vitest run src/pages/orders/templates/sea-create-candidate.test.tsx src/pages/orders/templates/components/sea/HouseBillNumberField.candidate.test.tsx src/pages/orders/air-master.permissions.test.tsx
pnpm --dir web exec vitest run src/pages/orders/detail-port-save-master.test.tsx src/pages/orders/detail-restore-refresh.test.tsx src/pages/orders/templates/components/sea/SeaFinalDestinationField.shared.test.tsx
```

- `sea-create-candidate.test.tsx:83`覆盖候选确认、直单阻断、批次排重；`HouseBillNumberField.candidate.test.tsx:56`覆盖身份未齐、候选迟到、换船司和失败重试。
- `air-master.permissions.test.tsx:35`覆盖编辑与恢复关联权限不同；方向新增仍需验证传入真实业务类型、不借AE授权。
- `detail-port-save-master.test.tsx:14`覆盖主单动作不自动采纳外部版本，`detail-restore-refresh.test.tsx:44`覆盖本人恢复与更晚外部变化。
- `air-export/form-adapter.test.ts:186`起已出现并行实现代理新增经营字段/清空测试，本研究没有执行或修改。后续应增加/运行新方向候选参数的真实wire测试。
- Biome只针对修改文件；类型影响跑`pnpm --dir web tsc`。禁止用`pnpm --dir web test -- <file>`冒充定向执行。

## 相关规范与外部参考

规范：根AGENTS.md、server/AGENTS.md、`.trellis/spec/server/backend/database-guidelines.md`、`air-export-contract.md`、`sea-export-document-contract.md`、`sea-shared-write-consistency.md`及前端`type-safety.md`。

本研究只检查仓库当前代码，无外部网页/库版本事实，不需要外部引用。

## 限制与未覆盖

- 未运行任何命令中的测试、生成或数据库迁移，不能据本报告宣称通过。
- 原有SE/AE测试只证明原行为，新增同号跨方向/跨方向ID拒绝/主单零成员身份/迁移异常回滚必须补专用断言；SI/AI完整创建仍属后续组，不在方向基础阶段用假单绕过门禁。
- 正则式直接Create扫描不覆盖其他包、SQL动态字符串、间接helper或未来新增链；不是全仓夹具穷尽证明。
- 单元元数据/manifest测试不能证明真实数据库已落约束，前端候选mock也不证明最终写门禁。
