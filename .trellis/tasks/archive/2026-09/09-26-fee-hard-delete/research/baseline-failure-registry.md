# 集成测试基线对照登记表（fee-hard-delete）

> 目的：区分任务回归与存量失败，为 A8 验收提供可复核证据。本表由主会话基于真实运行日志生成，日志留存于 `/tmp/baseline.log`、`/tmp/current.log`（回归修复完成后将归档到本目录）。

## 对照条件

| 项 | 基线 | 当前 |
|---|---|---|
| 代码 | commit `ce851158`（独立 worktree，未变动的 main 快照） | 工作区（本任务实施后） |
| 数据库 | `roncin_go_admin_integration_baseline`（新建空库，测试自动建表） | `roncin_go_admin_integration`（既有专用库，测试自动建表） |
| 命令 | `go test -p 32 ./internal/data/ ./internal/platform/migration/` | 同左 |
| 环境变量 | `RONCIN_INTEGRATION_DATABASE_SOURCE`（相同构造方式，仅库名不同） | 同左 |

失败计数：**基线（完整视角）21 项已知失败，当前定版 5 项（全部同名同因），0 任务回归。**

## §0 对照方法的修正记录（重要）

初版对照（基线 17 vs 当前 17）**是残缺运行得来的假阴性**：`TestFeeSupplementImpactAndCancelPostgres` 失败后其异步助手 goroutine 触发 `panic: Fail in goroutine after ... completed`，测试二进制中止，**排在它之后的所有测试（含 `TestSeaDocumentModeChangeBlockedByDownstreamFacts` 等 5+ 项）在基线与第二轮当前运行中都未执行**。两侧"恰好一致"掩盖了这一点。

修正方法：基线以 `go test -skip 'TestFeeSupplementImpactAndCancelPostgres'` 重跑（该测试已知失败+panic，跳过不影响其余结论），得到完整基线集合；当前侧在修复 panic 根因（夹具组织 kind）后自然完整。

## §0.1 完整基线集合（21 项）

初版 §1 的 17 项 + 因中止被隐藏的 4 项（`TestFeeSupplement{BillChain, EvidenceConcurrency, ListAuthorization, SensitivityRouting}Postgres`，同为组 A 根因）。

## §1 初版对照：17 项同名失败（历史记录）

> 初版两轮均为中止后的残缺运行；其中 12 项在根因修正（§2 意外收益）后已真实转绿，定版留存 5 项见 §7。表格保留作逐测试原因对照的证据。

| # | 测试 | 基线失败原因（摘要） | 当前失败原因 | 原因组 |
|---|---|---|---|---|
| 1 | TestCommissionAdjustmentMySourcePostgres | 发起补录申请 400 EXCHANGE_RATE_ORGANIZATION_INVALID | 同 | A |
| 2 | TestCommissionAdjustmentCancelGatePostgres | 子用例 LFS_DRAFT_可经通用取消接口忽略 失败 | 同 | A |
| 3 | TestFinanceBillBatchCreatePostgres | 子用例 不同幂等键并发抢占… 失败 | 同名子用例 | A |
| 4 | TestFinanceBillBatchNettingCreatePostgres | 子用例 对冲批次原子生成… 失败 | 同名子用例 | A |
| 5 | TestFinanceBillCreateSharedTransactionPostgres | 子用例 相同幂等键并发创建… 失败 | 同名子用例 | A |
| 6 | TestVerificationCreateSharedTransactionPostgres | 子用例 相同幂等键并发创建… 失败 | 同名子用例 | A |
| 7 | TestAutoOrderLock_SettlementTriggerPostgres | 子用例 核销用例创建生效后… 失败 | 同名子用例 | A |
| 8 | TestOrderFeeBulkMaintenancePostgres | 子用例 批量改费用时间… 失败 | 同名子用例 | A |
| 9-12 | TestFeeSupplement{CreateGate,Decision,FinancialEvidence,ImpactAndCancel}Postgres | 子用例 失败（补录流程深层断言） | 同名子用例 | A |
| 13 | TestRoleWorkspaceAnchorBackfillPostgres | organizations_workspace_parent_check 违反 | 同 | B |
| 14 | TestFeeCatalogCompanyTemplatesPostgres | 断言输出异常（结构指针 dump，123/1/200） | 同 | C |
| 15 | TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies | master_data_items 数量得到 18 期望 5 | 同 | C |
| 16 | TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots | order_fees 费用快照或其他字段被修改 | 同 | D |
| 17 | （TestCompanyBoundaryMigrationCopies* / PreservesRoots 两个在当前已通过） | — | — | — |

### 原因组与已确认程度

- **组 A（汇率组织解析，约 12 个）**：夹具创建的组织未配置公司本币，触发 `EXCHANGE_RATE_ORGANIZATION_INVALID 当前组织无法解析公司本币`。已确认共因：汇率解析器要求公司级本币配置，测试夹具缺失该配置。**待确认**：组内各测试是否还有汇率之后的更深层失败被此错误提前遮蔽（修复夹具前无法看到）。
- **组 B（工位锚点 CHECK，1 个）**：夹具故意创建断链部门触发 `organizations_workspace_parent_check` 约束拒绝。已确认：约束拒绝先于测试意图。与费用/提成无相邻数据路径。
- **组 C（费用目录种子/断言漂移，2 个）**：种子行数与断言期望不一致（18 vs 5）；另一个断言输出异常。与 order_fees 无关，属 master_data 种子域。
- **组 D（公司边界迁移快照断言，1 个）**：`order_fees 费用快照或其他字段被修改`——两侧同因。**待确认**：需人工比对该公司边界迁移重写前后费用行差异是否为本任务意外引入（初步判断否：两侧代码不同但迁移链相同、断言相同；回归修复完成后复核）。

### 阻塞性评估（对 A1—A8 的验收覆盖影响）

- **A3（建账与编辑/删除竞争、取消与重新建账竞争）被组 A 直接挡住**：`TestFinanceBillCreateSharedTransactionPostgres`（相同幂等键并发创建返回同一账单）与批量建账并发用例正是 A3 的数据库级场景，当前在汇率配置步骤即失败，**无法作为 A3 已验证的证据**。处置：为 A3 补一组不依赖汇率解析或先行注入公司本币配置的针对性并发用例，或在验收报告中明确记录「A3 数据库级验证因存量失败未覆盖」。
- A4（生命周期门禁）、A5/A6（删除与撤销）的专用库用例为本任务新建，不依赖组 A 夹具，可独立执行。
- A7（迁移双向验证）：空库全链初始化已由 `internal/platform/migration` 包真实通过（修复 organization_id 引用后）；带旧数据升级路径由本任务新增的重放测试覆盖（当前回归修复中）。

## §2 任务回归（11 项）——已全部修复，且修复方式经独立审计

当前定版运行（panic-free 完整执行）中 11 项全部转绿。修复方式经只读核查代理对照基线代码逐项审计（结论存档于会话记录）：

- **无跳过测试、无放松守卫、无断言降级**；`t.Skip` 全 diff 零新增，测试函数数量与基线逐一对应；
- 提成 8 项：门禁为 (a)/(b) 类忠实翻译——原「存在未建账费用即阻止」改为「存在无有效账单关联费用即阻止」（方向未反转），原「排除 CANCELLED」类因 CANCELLED 集合在新模型不存在而自然消失，未引入「必须已建账」的新限制；夹具从「仅 SetStatus(BILLED) 的状态谎报」改为真实账单行关系（PRD R5 要求）；
- 迁移重放 3 项：真实重建旧结构（status/cancelled_* 列、双 CHECK、外键）+ 全要素断言（保留/删除/归档审计组织归属/占用异常整体回滚）；`min(uuid)` 改为 ORDER BY + LIMIT 1 子查询；
- `edge was not loaded`：`LoadBillableFeesScoped` 补真实装配，`orderFeeToBiz` 的「未装配关联显式失败」守卫全程保留；全部 10 个费用返回路径经横向审计无遗漏。

### 审计发现并已收口的问题

- **P1（已修复）**：`ErrOrderFeeSupplementDeleteForbidden`（补录费用拒绝普通删除，A5 验收点）原实现零测试覆盖，且有失实注释声称已覆盖。已补 `TestFeeSupplementGeneratedFeePlainDeleteForbiddenPostgres`（单条拒绝+行保留、混合批量整批失败零写入、全普通批量不受影响）并修正注释。
- **P1（已修复，修正对照方法时暴露）**：`TestSeaDocumentModeChangeBlockedByDownstreamFacts` 在基线通过、初版回归修复后失败——阶段 1 转换该夹具时只删 `SetStatus(BILLED)` 未补账单关联，测试前提（存在下游占用事实）在新模型下不再成立；且因 §0 所述中止从未在集成运行中暴露。已镜像姊妹测试补真实账单占用（草稿即占用），三姐妹测试全绿。
- **P2（记录）**：迁移重放夹具未覆盖带 `supplement_request_id` 的 CANCELLED 行（冲减冲突分支）；`finance_bill_write.go` Cancel 注释与实际顺序不符（由活动行唯一索引兜底）；两处 CANCELLED 场景测试删除属「状态不可构造」的合理裁剪；少量无关格式化扰动。

### 意外收益（根因修正，16 项转绿）

P1 收口时以 HEAD worktree 复现确认：组 A 的真实根因是**测试夹具组织 `kind` 配置错误**（`system` 应为 `company`，汇率解析器因此拒绝解析公司本币）+ 两处汇率周锚日期错位。修正后基线 21 项中 **16 项真实转绿**（含补录家族全部、建账/核销并发、批量维护、台账等），余 5 项留存（§1）。

## §3 迁移失败原子性的实证

开发库首次迁移执行在 `DROP CONSTRAINT order_fees_cancellation_consistency` 步骤失败（漂移见 §4）。失败后复查：`status` 列、CHECK、索引均仍存在，2 条 CANCELLED 费用仍在——**单事务回滚已被实证，无部分写入，无需手工清理**。修复漂移后重放成功。

## §4 开发库与迁移链的存量漂移（本次发现，已在迁移文件中兼容）

1. `order_fees_cancellation_consistency` CHECK：链文件 20260826150000 有、开发库无（该文件补入 CHECK 晚于开发库应用，migrate:dev 校验和修复容忍了差异）→ 迁移用 `DROP CONSTRAINT IF EXISTS`。
2. `cancelled_by` 外键命名：链为 `order_fees_users_cancelled_by`，开发库为 Ent 自动迁移期命名 `order_fees_users_cancelled_order_fees` → 双命名 `IF EXISTS`。
3. 审计组织归属：`order_fees` 无 organization_id 列（设计如此，归属在 orders/fee_settings），审计 INSERT 已改为 `LEFT JOIN orders`。

## §5 开发库迁移授权与执行记录

- 影响清单（授权前提供）：46 条费用，2 条 CANCELLED（无账单占用、无冲减冲突、非补录来源），44 条不受影响。
- 用户已授权；`pnpm run migrate:dev` 于本任务内执行成功。
- 执行后验证：4 列/CHECK 残留 0；费用 46→44；归档审计 2 条且快照完整（fee.code/amount/currency/direction/version/原撤销元数据）。

## §6 遗留待办（收口状态）

- [x] 回归修复验证：当前定版运行 panic-free 完整执行，失败集合 = 基线留存 5 项，同名测试失败原因逐条一致，无新签名。
- [x] 组 A 遮蔽问题：根因（夹具组织 kind）已修正，16 项转绿后不再存在遮蔽；A3 场景（幂等键并发建账、批量建账竞争）已真实执行并通过。
- [x] 组 D 复核：`internal/platform/migration` 整包基线与当前逐字节相同、夹具不触及费用状态列，与本任务无关，结案。
- [x] P1 补录拒删覆盖已补；A5 验收点成立。
- [x] 组 A 残留 1 项（TestAutoOrderLock_SettlementTriggerPostgres）与组 B/C 共 4 项留存，单独立项处理，不阻塞本任务验收（其测试路径与本任务验收点无重叠，详见 §1）。
- [x] 本表随最终验收报告提交；A8 验收结论按 PRD 原文表述。

## §7 定版结论（最终完整运行，panic 数 0）

- 当前失败：5 项（TestAutoOrderLock_SettlementTriggerPostgres、TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots、TestFeeCatalogCompanyTemplatesPostgres、TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies、TestRoleWorkspaceAnchorBackfillPostgres），全部为基线同名同因存量问题。
- 相对基线：21 项已知失败中 16 项转绿；0 项任务回归；2 项测试基建缺陷（迁移重放夹具、单证模式切换夹具）在本任务内修复。
- 验收表述：本任务引入的回归已全部修复，指定用例在真实 PostgreSQL 实际通过；完整集成测试仍有 5 项基线存量失败（详见 §1），完整集成验收未全绿，不构成「全量通过」。
