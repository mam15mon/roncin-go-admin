# 实施验证记录（fee-status-review-fixes）

> 对照 prd.md A1—A6；代码 SHA、命令与结果随实施推进补齐。连接串等秘密不记录。

## R1/A3：已应用开发库只读核查（主会话执行，2026-09-27）

背景：迁移 `20260926100000` 已在开发库应用（旧版守卫允许 DRAFT 调整随费用删除）。校验和重录不会重跑 DML，因此对已应用环境做只读核查，不自动处置。

命令：对开发库执行只读 SQL（4 项计数检查），结果：

| 检查项 | 数量 |
|---|---|
| 孤立未取消调整（来源补录申请的生成费用已不存在） | **0** |
| 曾确认/已扣回调整且来源费用缺失 | **0** |
| 非取消调整总数（全库背景值） | **0** |
| 迁移归档审计条数（上次执行留痕） | 2 |

两条迁移删除的费用经归档审计复核均为非补录来源（`fee.supplement_request_id` 为空，fee.code 分别为 DOC/OF）。

**结论**：旧守卫的 DRAFT 漏洞在开发库**未留下任何孤立待处理调整**（全库无非取消调整），无需数据修复；新守卫收紧仅影响后续新环境的重放行为。

## 迁移修订策略（R1/A3 记录）

- 迁移文件未发布（仅仓库内），按 AGENTS.md 开发期迁移迭代规则**原地修订**，不新增后续迁移（排在删除之后的迁移无法拦住已发生的删除）。
- 开发库已应用旧版：修订文件后校验和变化，由 `pnpm run migrate:dev` 的 `-allow-checksum-repair` 重录为当前文件（仅元数据，不重跑任何 DML）——待实施完成后执行并记录输出。

## R2/A4：删除审计补全快照（已实施）

- 单条 `Remove` 与 `BulkRemove` 审计 Details 新增 `fee.name`（锁定行 `FeeName`，费用保存名）、`fee.settlement_party_id`、`fee.settlement_party_name`（删除事务内 `tx.Partner` 查询的当前法定名称，注释注明为删除时读取值）；批量按单位 ID 去重一次查询回填（复用 `order_fee_supplement_request.go` 既有范式，无 N+1）；审计与 DELETE 同事务、失败回滚；锁序/版本/组织边界/补录拒删未动。
- 审计展示：`web/src/features/audit/audit-presentation.ts` 补 3 条中文标签（费用名称/结算单位 ID/结算单位名称），管理端审计页未知键本就回退键名，属最薄映射。
- 测试：删除用例断言新键值（名称、单位 ID、法定名称）；新增「审计写入失败回滚删除」子用例（非法 result 枚举 → 校验失败 → 费用保留、无审计残留）；批量用例改为两条不同结算单位费用，覆盖去重回填不串行。
- 验证：`TestOrderFeeDeleteByBillOccupancyPostgres` 8/8、`TestOrderFeeBulkMaintenancePostgres` 7/7 PASS；`go -C server vet ./...` 通过；前端 audit 定向测试 PASS。

## R3/A5：自动锁定验收（已实施）

- 根因（实测复现）：夹具组织 `SetKind("system")`，核销链路 `ExchangeRateUsecase.BaseCurrency → ResolveContext` 对 system 组织显式拒绝（`exchange_rate.go:48-50`），与前序 `newFeeSupplementFixture` 修复同源。修法：`kind="company"` + `SetEnabled(true)`。
- 周锚核查结论：该夹具不创建汇率行，费用/账单/流水均直接写入预固化汇率快照，且核销不再解析汇率（由 base_amount 快照回填）——无需周锚改动。未关闭校验、未弱化断言、未补账单掩盖、业务规则零改动。
- 三个验证点实际执行（全部 PASS，无 SKIP）：
  1. 「核销用例创建生效后自动触发锁定」执行到最终锁定结果断言（LockedAt + LockSource=AUTO_SETTLEMENT）；
  2. 「未建账应付阻止自动锁定且删除阻断后重试锁定」：阻断（审计原因码 unbilled_fee）→ 删除 → FEE_CANCEL 重评触发锁定完整闭环；
  3. 「最后一笔未建账应付进入账单后重试自动锁定」PASS。
- `TestAutoOrderLock_SettlementTriggerPostgres` 7/7 子用例 PASS。

### 原任务登记表的验收更正（PRD R3 要求）

原任务 `baseline-failure-registry.md` §6 声称剩余失败「与本任务验收点无重叠」不成立：`TestAutoOrderLock_SettlementTriggerPostgres` 因汇率前置失败未执行到锁定断言，而该任务修改过自动锁定的未建账谓词——基线失败不能证明修改后路径已验证。本任务修正夹具后该用例真实 PASS，验收缺口已闭合。原归档保留不动，以本节为更正记录。

## 全量包级验证（R2+R3 代理执行）

`go -C server test -p 32 ./internal/data/ ./internal/biz/ ./internal/service/`（data 包以 `-timeout 40m` 完成，637.8s）：biz/service ok；data 仅 2 项失败——`TestRoleWorkspaceAnchorBackfillPostgres`、`TestFeeCatalogCompanyTemplatesPostgres`，与原登记表 #13/#14 同名同因（工位锚点 CHECK / 费用目录存量），属本任务「不在范围内」，仅登记。

## R1/A1-A3：迁移守卫收紧（已实施）

- 守卫条件 `(a."status" NOT IN ('DRAFT','CANCELLED') OR a.confirmed_at IS NOT NULL OR a.paid_at IS NOT NULL)` → `a."status" <> 'CANCELLED' OR a.confirmed_at IS NOT NULL OR a.paid_at IS NOT NULL`：DRAFT/CONFIRMED/PAID 均冲突；已取消但保留确认/扣回历史时间戳同样冲突。RAISE 文案与文件头注释同步修订。校验先于归档审计与 DELETE，整文件单事务，无自动修复分支。
- 夹具矩阵（`order_fee_status_hard_delete_migration_integration_test.go`）：`createReplaySupplementChain` 直构完整补录链路（APPROVED 申请 + 生成费用置 CANCELLED + 规则/提成快照 + 调整，幂等键沿用业务派生规则）。
  - **拒绝矩阵 5 场景**（表驱动，独立 Schema）：DRAFT / CONFIRMED / PAID / CANCELLED+仅 confirmed_at / CANCELLED+仅 paid_at → 迁移失败且报错含「需先处置待处理调整」；原子性断言：费用仍在且 supplement_request_id 不变、调整状态/版本/时间戳原样、申请保持 APPROVED、归档审计零新增、旧列仍存在；
  - **成功矩阵 2 场景**：仅关联已取消且无确认/扣回历史的调整 → 迁移成功（申请/调整保留、费用删除、归档审计组织归属正确）；无 CANCELLED 行 → 正常完成（费用保留、零审计、列删除）；
  - 场景 4/5 在当前业务流转下不可经通用取消产生，守卫对数据库层不变量失败关闭，测试注释已说明。
- 开发库校验和重录（实施后执行）：`pnpm run migrate:dev` 输出旧 `6a282187…` → 新 `21466b23…`，无迁移重放、权限/种子零变更——仅元数据重录，与 §R1/A3 只读核查结论（无孤立数据）一致。

## 独立复核与门禁（主会话执行，代码 SHA 见提交 50fec0bb/774b9aff/438b9450）

- 定向套件（implement.md 指定命令，专用集成库）：**8 顶层 + 32 子用例全部 PASS，0 FAIL**（TestOrderFeeStatusHardDeleteMigration×4、TestOrderFeeDeleteByBillOccupancyPostgres、TestOrderFeeBulkMaintenancePostgres、TestAutoOrderLock_SettlementTriggerPostgres、TestFeeSupplementGeneratedFeePlainDeleteForbiddenPostgres）。
- `go -C server vet ./...` 通过；`go test -p 32 ./internal/data/`（真实 PostgreSQL）仅剩 2 项存量失败（工位锚点 CHECK、费用目录断言），与原登记表 #13/#14 同名同因，本任务范围内路径全绿。
- `pnpm run check:fast`：WEB 60.2s / SERVER 59.8s 并行全绿。
- `git diff --check` 干净。

## A1—A6 对照结论

| 验收项 | 结论 | 依据 |
|---|---|---|
| A1/R1 拒绝矩阵 + 原子性 | 通过 | 5 场景独立 Schema 真实执行，断言费用/调整/申请/审计/列结构无部分变更 |
| A2/R1 成功路径 + 历史保全 | 通过 | 已取消调整场景迁移成功且申请/调整/历史账单快照保留；无关联/普通/空库路径既有用例覆盖并回归 PASS |
| A3/R1 修订策略与已应用环境 | 通过 | 本文档 §R1/A3（只读核查零异常）+ §校验和重录记录；未声称重录会重跑 DML，未自动处置 |
| A4/R2 审计快照 + 原子性 | 通过 | 单条/批量新键一致（名称/单位 ID/单位名），批量去重无 N+1，审计失败回滚子用例 PASS |
| A5/R3 自动锁定真实执行 | 通过 | 原失败用例 7/7 子用例 PASS 无 SKIP；未建账阻断与删除后重评闭环真实执行；已记录执行的子用例名 |
| A6 门禁与如实报告 | 通过（附存量清单） | 定向套件 0 FAIL、vet/check:fast 全绿；data 包完整集成仍剩 2 项存量失败（原登记表 #13/#14，同名同因），逐项列明不报告全绿 |

**结论**：本任务三项审阅发现全部修复并经真实 PostgreSQL 验证；完整集成测试仍存 2 项与验收路径无重叠的存量失败（工位锚点、费用目录），已登记原任务基线登记表，留待单独立项。
