# 账单创建与费用维护流程：实施计划

前置：[design.md](./design.md)、[prd.md](./prd.md) 经用户审阅，design.md 第 15 节已给出全部结论；
`task.py start` 后才能实施。步骤上的 `[结论 n]` 对应 design 第 15 节条目。
契约变更顺序：先改 `.proto` → `make -C server api` → `pnpm run generate:web-client`；修改 `access/manifest.go` 后
`pnpm run generate:permission-keys`；Ent Schema 变更后 `go -C server generate ./...`。生成物与源文件放在同一提交。
迁移文件新增在 `server/migrations/`，编号晚于链尾；先审阅 SQL；清库、重置开发库须另行取得用户明确授权。
每组改动后提交，Conventional Commits，中文说明。禁止 TDD：功能完成后按风险补验证。

## 阶段 0：核对与对齐（不改产品代码）

1. **复核静态结论**：用集成测试或最小复现核对 design 第 1 节的关键事实，结论写入 task 说明：
   - 发票创建、核销创建、对冲确认是否都先锁账单所属订单再锁账单行（3.2、3.3 的前提）。
   - 发票、核销、对冲仓储在 `WithinTransaction` 内可与建账共事务（1.5）；核销流水行锁序与新建账单行不冲突（7.3）。
   - 存量数据中 `DRAFT` 账单、`confirmed_by` 可回填性（迁移前只读查询，不修改数据）。
2. **对齐相邻任务契约**（评审会议或文档，不是代码）：发票任务确定无需发票标记模型与分配事实（9.2）；收付核销任务确定
   带符号分摊与组合输入（9.3）；结论回写 design 第 5、7、9 节。
3. 若第 1 步推翻 3.2 的锁序前提，停止，回到设计阶段修订，不用隐藏分支掩盖。

## 阶段 A：免确认（后端契约先行）

4. proto：删除 `ConfirmBill`、`ConfirmBillBatch` 及消息、`BillExpectedVersion`；`make -C server api`，检查
   `access_rules_gen.go` 差异。
5. `biz/finance_bill.go`、`finance_bill_batch.go`：建账（单张与批量）直接写 `CONFIRMED`、`confirmed_at/by`；
   删除 `Confirm/ConfirmBatch`；`UpdateBill` 前置条件改为 `CONFIRMED` 且未受保护 `[结论 4]`；
   `ListBillSettlementAccountUpdateCandidates` 同步。
6. `access/manifest.go` 删除 `FinanceBillConfirm`，`server/auth.go:261` 去除；`pnpm run generate:permission-keys`。
7. 迁移：存量 `DRAFT` → `CONFIRMED`（确认人/时间按创建人/创建时间回填）`[结论 3]`；`cmd/sync-dev` 去掉 `DRAFT` 种子。
8. 前端：移除确认按钮、批量确认、`canConfirmFinanceBills`；`billStatusMeta` 文案与筛选；更新相关测试。
9. 提交点：A 全部（后端契约＋前端移除）同组提交，避免前后端契约错位。

## 阶段 B：锁定谓词与完成状态投影

10. `data/fee_protected_fact.go`：新增 `billProtectedFinanceFacts/billHasProtectedFinanceFact`，
    `feeHasProtectedFinanceFact` 改为复用；`protectedFinanceFactPredicate` 预留无需发票条件，待发票任务标记表落地后接入
    `[结论 5、8]`。保持 `fee_rate_refresh` 行为不变并回归其测试。
11. 完成状态投影：`settlement_progress`、`invoice_progress`、`completed`、`fees_locked`、`fee_lock_facts`（设计 5.1–5.2）；
    `enrichVerificationAmounts` 批量聚合，禁止逐行 N+1；`billUnsettledPredicate` 与展示口径改为带符号比较，
    同步 `CalculateOverdueDays`、信用额度汇总；`ListBills` 新增进度／完成筛选。
12. 零额／负数承载 `[结论 1]`：本期只实现纯函数与表格测试，不放开 DB 约束与创建入口；放开（含 `amount_positive` 迁移及核销、对冲、信用、逾期回归）留待后续任务。

## 阶段 C：账单费用增删

13. proto：`ListBillFeeCandidates`、`UpdateBillFees`、错误原因；`make -C server api`、`generate:web-client`。
14. 提取 `validateBillFeeForAttach`（自 `buildFinanceBill`）供候选与提交共用；`ListBillFeeCandidates` 服务端关键字过滤分页
    （`ValidListPagination`，上限 200，补边界测试）。
15. `UpdateBillFees` 用例与仓储：设计 3.2 锁序、4.2 重算（账单日汇率、尾差、`fee_count`、预计开票金额）、
    `version+1`、费用 `version+1`、`reconcileBillEconomics`、审计；`ent/schema/finance_bill.go` 的 `fee_count` 改可变并生成。
    移空规则按 `[结论 2]`。
16. 前端详情抽屉「费用维护」：候选选择器、移除、锁定提示、`expectedVersion`、外部变动检测；成功后 `actionRef.reload()`。

## 阶段 D：批量建账逐组独立

17. 放开普通模式混合方向（保留各组单一方向）；`group_token` 按组摘要；`PreviewBillBatch` 响应与前端同步。
    拆分开关去留 `[结论 6]`，`NETTING` 去留 `[结论 7]`。
18. 批次头迁移与 Ent 同步（去汇总列，读取聚合）；批次头短事务＋逐组独立 `WithinTransaction`；逐组结果与错误结构；
    幂等重试：已成功组原样返回、失败组重试。
19. 前端结果页逐组展示、失败组单独重试、成功组不重复提交；去掉「确认本批」。

## 阶段 E：账户快建

20. `BillGroupCard` 内新增结算账户快建（设计第 8 节）：复用 `QuickCreateModal` 与既有创建接口；创建后仅重载当前叶子候选并
    写入该叶子账户，其余配置不丢失；权限与公司范围；更新 `capability-navigation.md`。

## 阶段 F：组合发票／核销（依赖阶段 0.2 的契约）

21. 信封与编排：`CreateBillBatchGroupInput` 增 `invoice/verifications`；组事务内建账→发票→核销，逐步鉴权，
    逐项错误；派生幂等键。发票、核销的载荷、分摊与税额精度按相邻任务落地，本任务验收原子性与错误结构。
22. 前端按组载入位与失败回显；组合区块的表单由发票／核销任务提供。

## 阶段 G：规范与文档同步

23. 更新 `.trellis/spec/server/backend/finance-bill-currency.md`（去确认、逐组独立、批量响应）、`database-guidelines.md`
    （如新增并发范本）、`web/frontend/capability-navigation.md`、`domain/glossary.md` 账单状态说明；本任务 PRD 的验收场景回写。

## 验证

**后端（真实 PostgreSQL，SKIP 不算验收；只传 `RONCIN_INTEGRATION_DATABASE_SOURCE`，不要把整个 `.env.local` 导入环境）**

- 免确认：单张与批量新建即 `CONFIRMED`；发票、核销、对冲、提成、信用额度、逾期在新账单上无需确认即可进行；存量 DRAFT 迁移。
- 增删费用：候选只含同公司／同方向／同币种／同结算对象的未建账费用；篡改请求带不同币种、不同对象、跨公司、已建账、已占用
  费用均被拒；移除后原费用金额、税率、结算对象不变且可重新建账；两账单并发抢占同一费用只有一方成功；
  金额与本位币尾差正确，`fee_count` 与账单行数一致；版本冲突 409。
- 锁定：仅发票（含部分）、仅核销（无发票号）、仅对冲、仅无需发票标记各自禁止增删；两张发票只作废一张仍锁定；
  发票全作废但仍有核销仍锁定；最后一项撤销成功后恢复；撤销失败不提前解锁；撤销与新登记并发按最新事实。
- 并发矩阵（设计 3.3）逐项：增删×发票登记、增删×核销、增删×对冲确认、增删×标记、增删×取消、增删×汇率联动。
- 完成状态：正数需票账单「仅结清」「仅票齐」均不完成；无需发票标记后结清即完成；零额创建即完成；
  负数账单带符号结清后完成、发票仍按需票规则；展示与 SQL 筛选同源。
- 批量：按方向／币种／结算对象分组；应收应付并存；某组失败其他组仍成功；同批次键重试已成功组原样返回、失败组重试；
  组内陈旧令牌只影响该组；批次头聚合按本位币分组。
- 组合：建账＋发票＋核销任一步失败整组回滚、无孤立账单／发票关联／核销／流水；逐项错误结构；缺权限整组拒绝；
  同键重试回放；他组不受影响。
- 迁移：空库冷启动与已有版本升级两种方式；`go -C server test -p 32 ./internal/biz ./internal/service`；
  `go -C server test -p 32 ./internal/data`（约 15 分钟，放后台，超时设 40 分钟）；`go -C server vet ./...`。

**前端**

- `pnpm --dir web exec vitest run <相关测试>`：覆盖真实 wire 形态（省略默认值、显式零、非零、可选未设置、必填缺失），
  批量响应逐组缺 `bill` 或 `error` 的形态，不只用字段齐全的 mock。
- 账户快建：权限显隐、预填与只读、成功后只更新当前叶子、其余配置保留、取消不创建、失败保留输入、迟到响应不污染。
- 费用维护：锁定态置灰与事实提示、版本冲突提示与输入保护、候选分页与关键字服务端过滤、多选提交。
- 列表：进度列与筛选、汇总与导出沿用快照；确认入口不再出现。
- `pnpm run check:architecture`；多个 `*.test.tsx` 按文件拆分，避免单文件超过 20 秒。

**提交前**：`git diff --check`、`pnpm --dir web tsc`、修改文件的 Biome 检查；任务整体完成后 `pnpm run check:fast`
（放后台）。

## 评审门与回滚点

- 评审门 1：design 第 15 节结论已记录，方案获批准后才 `task.py start`。
- 评审门 2：阶段 0 核对结论若推翻 3.2 或 7.3，回设计阶段。
- 评审门 3：阶段 A 合并前独立 review（`/code-review`），重点：下游 `CONFIRMED` 判定、迁移、权限删除；确认问题后再合并。
- 评审门 4：阶段 C、D 合并前独立 review，重点：锁序、并发矩阵测试、批量逐组语义与幂等。
- 阶段 F 须待发票、核销任务契约已批准且阶段 B/C 已合并再启动；契约未定不提前接入。
- 回滚：阶段 A 为前后端同组契约变更，同组回退，迁移按反向迁移文件处理而非改写已应用迁移；阶段 C、D 的接口与前端同组回退；
  不增设运行期兼容分支或自动回退。
- 任何要清空或重建开发数据库的步骤，先向用户取得明确授权。

## 2026-11-14 接续检查点

- A/B/C用户实现提交已核对；E账户快建已提交f1e73b9d。
- D逐组独立、免票消费、真实候选检索及独立审查修复已提交bd6afafa。证据见research/batch-completion.md、full-review.md和fee-candidate-search-completion.md。
- F仍待发票多记录/逐费用手工税额契约实施。组合将使用内部只写入口，不能直接在外层txCtx调用会重读完整响应的独立核销命令；外层提交后才组装账单/发票/核销响应。
- G主要账单规范已同步；待逐费开票与费用台账投影集成后再完成最终统一验收。不归档整任务，不提前声称所有并发矩阵/组合已通过。
