# 独立复查与补修报告

## 范围与结论

本次审查以 `e2910f2e` 为基线，包含前端组 `4aa7c46b` 和当前后端、迁移、规范及批量弹窗补修。逐项复查本任务七项验收标准，重点检查真实业务入口的调用顺序和错误返回，不重复旧任务已报告的问题或扩展无关模块。

七项实现均已完成。本轮发现的两处局部缺口（批量空结果仍回调、补录决策意见仍限 500）已经补修。复查中发现的来源订单子集追加全集死锁及方案分配外键隐式死锁，已交锁序实现代理修复并取得真实 PostgreSQL 回归结果。当前范围内没有需要产品决策的未修复发现。最终全量门禁由主代理执行，以下定向通过结果不替代全量验收。

## 已修复发现

### 1. 批量批准空结果误触发完成回调

- 文件：`web/src/pages/orders/components/fees/fee-review/BatchFeeReviewModal.tsx`、相邻组件测试。
- 问题：统一请求客户端消费失败后可能返回 undefined；旧 `response.data ?? []` 会把缺结果当成处理完成，调用 `onCompleted([])` 并提示「批准成功 0 票，失败 0 票」。
- 修复：仅在 `response.data` 为非空数组时消费逐票结果。批量契约没有顶层 success，不能强加单票响应的门禁；逐票成功继续由每行 success 判定。
- 回归：undefined、缺 data、空 data 均不触发完成、不关弹窗、不出现 0 成功提示；再次提交保留原预览、逐票意见和完全相同的版本、token、幂等键载荷。原有混合批准/失败处理仍通过。

### 2. 补录批准及退回意见没有贯通 1000 Unicode 字符

- 文件：`server/internal/biz/order_fee_supplement_usecase.go`、`server/internal/data/ent/schema/order_fee_supplement_request.go`、正式迁移及真实库回归。
- 问题：补录批准/退回共用 decision_reason，但领域仍限 500 个字符，Ent 仍用 MaxLen(500) 按字节校验，正式数据库列仍 varchar(500)。表单允许的合法中文长意见无法批准。
- 修复：领域两入口统一 1000 rune；Ent 使用 MaxRuneLen(1000)；重新生成 Ent。正式 Ent/Atlas Diff 生成扩容迁移，保留现有数据。
- 回归：1001 字拒绝且申请状态/版本不变；999 中文加 1 emoji 的批准和退回都成功并原样保存；批准事件意见相同；Ent 拒绝直接写入 1001 emoji。真实库校验列与 Schema 生成语义一致，并校验正式迁移记录的 SHA-256。

### 3. 完整来源订单集合及隐式外键锁（由锁序代理修复）

- 单笔确认先锁本员工提成行订单、再加载来源全部应收订单；两员工分别归属不同订单但来源相同，会互持子集后等待全集。现首次合并提成行与来源订单；批量确认、月申提交/重提一次合并全部来源并排序预锁，后续复核重入既有锁。
- 月申 User UPDATE → Membership，与方案分配 Membership → assignment 外键 User KEY SHARE 形成隐式环。独立临时 overlay 已实际复现 PostgreSQL SQLSTATE 40P01。现提成员工快照统一 NO KEY UPDATE，与 KEY SHARE 兼容，仍阻止账号实际更新及审核 SHARE。
- 已核对 `GetGenerationContext`、计算内核、单笔 Transition、月申 Submit/Resubmit/Approve 及整批确认辅助的外层预锁，未留下仅移动局部锁而外层仍反序的问题。详细证据和真实库结果见 `lock-order-validation.md`。

## 七项验收复核

| 验收点 | 实际复核结果 |
| --- | --- |
| 请求失败保留审核/付款弹窗 | 单票动作和付款显式验证成功实体；批量按真实 data 契约补空结果门禁；失败保留意见与幂等键 |
| 跨订单对冲计提集合 | 预锁只取有效 RECEIVABLE，与实际计提一致；PG 预览、创建、确认、月申候选/提交/批准通过 |
| 金额舍入 | 快照金额校验改用 RoundBank(8)，与录入一致；合法半数边界通过、篡改末位仍拒绝 |
| 授权及订单锁序 | 审核/补录 User SHARE→Membership SHARE→Role SHARE→Order；月申和提成入口员工 NO KEY UPDATE 在前，订单全集在后；实时停用资格仍拒绝 |
| 补录能力刷新 | 创建/批准/驳回/撤回/专用作废成功使审核查询失效；失败不伪成功；Antd confirm 保留错误状态供重试 |
| 1000 Unicode 意见 | 共用表单字符计数、领域、审核事件与补录 Ent 均按 Unicode；正式迁移解除补录旧列 500 上限 |
| 待审补录撤回重提 | 已批准基础与能否另建补录分别判断；本人可撤回并预填重提，待审时不允许另建第二份 |

## 正式 Schema 生成与迁移审阅

1. 修改 Ent 源字段为 `field.String("decision_reason").Optional().Nillable().MaxRuneLen(1000)`，执行：

   ```bash
   go -C server generate ./internal/data/ent
   ```

   实际成功；生成差异仅 `ent/migrate/schema.go` 对应字段 Size 500→1000。校验函数运行时由 Schema 构建，无手改生成文件。

2. 本项目没有此增量的现成 Diff 命令，使用临时 Go overlay：`/tmp/roncin-fee-supplement-comment-generator_test.go`，映射文件 `/tmp/roncin-fee-supplement-comment-generator.json`。调用 `getIntegrationDataBeforeMigration(t, "20261012110000")`，以正式 SQL 迁移创建独立 schema；`schema.NewMigrate` 配置 `migrate.WithDiffHook`，只保留目标表 decision_reason 的 ModifyColumn；`NamedDiff(..., "fee_supplement_decision_reason", entmigrate.Tables...)` 根据真实 Ent Schema 生成 SQL。

   ```bash
   node --env-file=/tmp/roncin-fee-approval-db.env scripts/run-with-env.mjs go -C server test -overlay=/tmp/roncin-fee-supplement-comment-generator.json -p32 -run '^TestGenerateFeeSupplementCommentMigration$' -count=1 -v ./internal/data
   ```

   实际 PASS，5.93s，日志 `/tmp/roncin-fee-supplement-comment-generator.log`。Atlas 产物目录 `/tmp/roncin-fee-supplement-comment-diff-v2`；原样取生成的 up SQL，按正式迁移顺序命名 `20261012110000_fee_supplement_decision_reason.sql`。没有手写或手改迁移 SQL。

3. 审阅实际生成结果：

   ```sql
   -- modify "order_fee_supplement_requests" table
   ALTER TABLE "order_fee_supplement_requests" ALTER COLUMN "decision_reason" TYPE character varying;
   ```

   Ent PostgreSQL 当前目标为不限制长度的 varchar，1000 字上限由领域及 Ent 验证；因此保留真实生成结果，不人为改写 varchar(1000)。只修改该列类型，没有删除、回填或其他表变更。旧 varchar(500) 数据保留。

4. 本项目正式执行器在 schema_migrations 按 SQL 文件记录 SHA-256，不维护仓库 atlas.sum。新文件 SHA-256：`9cd2aa7c90a12ae59ed8e47f6a2df66e6d0eff44a4bd3b13f8ac1525d4bf9254`。真实库回归已验证迁移记录与该文件完全一致。

## 验证

本审查直接执行：

- `pnpm --dir web exec vitest run src/pages/orders/components/fees/fee-review/BatchFeeReviewModal.test.tsx`：PASS，1 文件 4 例，11.96s；日志 `/tmp/roncin-fee-review-check-batch.log`。
- `pnpm --dir web exec biome check src/pages/orders/components/fees/fee-review/BatchFeeReviewModal.tsx src/pages/orders/components/fees/fee-review/BatchFeeReviewModal.test.tsx`：PASS。
- `pnpm --dir web tsc`：PASS；日志 `/tmp/roncin-fee-review-check-tsc.log`。
- `go -C server vet ./internal/biz ./internal/data ./internal/service`：PASS；日志 `/tmp/roncin-fee-review-check-vet.log`。
- `node --env-file=/tmp/roncin-fee-approval-db.env scripts/run-with-env.mjs go -C server test ./internal/data -run '^TestFeeSupplementUnicodeDecisionReasonPostgres$' -count=1 -v`：真实 PG PASS，批准/退回 2 子项，6.668s，无 SKIP；日志 `/tmp/roncin-fee-supplement-comment-final-check.log`。
- `git diff --check`：PASS。

协作代理实际执行并记录：

- 前端原组 8 文件 45 例、tsc/Biome：PASS，见 `frontend-validation.md`。
- 金额/对冲/审核事件 Unicode 真实库与相邻用例：PASS，见 `amount-contract-validation.md`。
- 最终并发组真实 PG：PASS，44.207s，包含外键、来源全集、审核/月申、补录/提成、申请头顺序；见 `lock-order-validation.md`。
- service/biz 全量：PASS，0.956s/1.714s，由锁序代理执行。

主代理已用浏览器验证真实 409 保留审批弹窗、1000 中文意见和相同幂等键重试 200 实际 APPROVED；最终 check:web 及全量 Go/真实库门禁由主代理补充。首次更大范围 Go data 测试因默认累计 10 分钟超时停止，不属于已通过结果；主代理使用更长测试时限重跑。

验证只写独立验收数据库的隔离 schema，不改日常开发数据，不发送实际钉钉通知，不输出数据库凭据。本审查未提交，改动交主代理分组提交。

## 未修复发现

主代理追加真实页面验收发现，撤回重提成功后的订单缓存刷新会卸载费用页子组件，导致预填表单丢失。已将费用页加载占位限定为无详情数据的首次加载，并补真实父页面刷新回归；主代理重新执行真实浏览器撤回重提验收通过，详见 browser-validation.md。其余已审查项无剩余具体发现。
