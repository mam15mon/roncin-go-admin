# 金额、计提集合与意见字符契约修复验证

## 实施范围

- `server/internal/biz/order_fee_review.go`：本位币金额校验使用 `RoundBank(8)`，与正常录入 `calculateAmounts`、批量日期更新一致。
- `server/internal/data/finance_commission_calculation.go`：对冲预锁集合仅取有效 RECEIVABLE 分配，和实际计提集合一致；`calculateCommission` 先锁员工 User，再加载订单来源。`GetGenerationContext` 新增 employeeID，事务内先锁 User 再锁订单及来源，接口与调用由锁序代理同步。
- `server/internal/data/ent/schema/order_fee_review_event.go`：意见 `MaxRuneLen(1000)`，统一领域已有 Unicode 字符计数。
- 新增三份回归文件：`biz/order_fee_review_contract_test.go`、`data/finance_commission_cross_order_netting_integration_test.go`、`data/order_fee_review_comment_integration_test.go`。

## 生成与数据库

执行 `go -C server generate ./internal/data/ent` 两次均成功。MaxLen 与 MaxRuneLen 的字段描述 Size 均为 1000，区别仅为运行时校验函数；生成文件无差异。正式 SQL 中 comment 为无限长 `character varying`，本次无 DDL 变化，无需新增迁移。没有手工修改生成物。

所有真实 PostgreSQL 用例使用 `/tmp/roncin-fee-approval-db.env` 对应的独立测试环境与正式迁移。未修改日常开发库、未处理既有补录记录、未实际发送钉钉消息。

## 命令与结果

1. `go -C server test ./internal/biz -run 'TestFeeReview(SnapshotAcceptsRecordedBankRounding|CommentUnicodeBoundary)$' -count=1`：PASS，0.011s。
   - 正常 normalizeOrderFee → calculateAmounts 得到总额 100.01030003、本位币 750.07725022，审核接受。
   - 篡改为 750.07725023 仍拒绝。
   - 1000 个中文 / emoji 字符接受，1001 拒绝。
2. `node --env-file=/tmp/roncin-fee-approval-db.env scripts/run-with-env.mjs go -C server test ./internal/data -run '^(TestCommissionCrossOrderNettingPostgres|TestOrderFeeReviewUnicodeCommentPostgres)$' -count=1 -v`：PASS，6.931s，无 SKIP。
   - 跨票应付订单不进入应收计提集合；只读预览、创建和确认均为两票收入 800、提成 80，指纹一致。
   - 月度候选、提交受控创建及整单批准成功。
   - 1001 字意见返回领域参数错误；999 个中文加 1 个 emoji 原样落库并批准成功；Ent 直接校验也拒绝 1001 个 emoji。
3. `go -C server test ./internal/biz -run '^(TestFeeReview|TestCalculateOrderFeeAmounts)' -count=1`：PASS，0.011s。
4. `node --env-file=/tmp/roncin-fee-approval-db.env scripts/run-with-env.mjs go -C server test ./internal/data -run '^(TestCommissionNettingSourcePostgres|TestOrderFeeReviewLifecyclePostgres|TestOrderFeeReviewBatchPermissionAndLossPostgres)$' -count=1 -v`：PASS，19.102s，无 SKIP。
   - 原有纯对冲计提、去重、来源反转取消、已付冲减、不同来源指纹均通过。
   - 自审、审核状态机、冻结、撤销、批量权限及亏损意见相邻路径均通过。
5. `git diff --check`：PASS。

在锁序代理同步 GetGenerationContext 新参数期间曾出现预期的临时编译失败；接口与调用同步后上述全部真实库验证通过。

## 交接

当前工作未提交，由主代理统一提交。整体验收仍由主代理与 check 代理负责，包含锁序并发、前端及全量门禁。
