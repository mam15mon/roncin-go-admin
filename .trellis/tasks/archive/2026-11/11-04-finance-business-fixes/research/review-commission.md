# 提成组独立复审

## 范围与结论

基线 `a847e193`，审查 R1 / R6 / R7，包括产品函数、两份实施回归、专项规范及本次补充回归。发现一项 R6 相邻缺口，已复现、最小修正并验证；当前无未处理阻断问题。

## 已修正问题

- 文件：`server/internal/data/finance_commission_rules.go`。
- 触发：员工分配终止后，经正式成员停用入口合法停用；随后只修改原方案名称/备注仍返回 `FINANCE_COMMISSION_RULE_EMPLOYEE_INVALID`。已在隔离 PostgreSQL 实际复现，不能把此场景算作方案核心参数修改。
- 修正：拆分 Membership 父行加锁与有效员工校验。UpdateRule 始终维持 Membership→Rule 锁序、成员集合重验与版本检查；只有名称/备注变化且启用状态、身份、口径、比例、起止日全部不变时，允许历史成员或账号已停用。任何资格字段变化，以及创建、分配、移除和复制仍使用严格有效性校验，不补取反序新行锁。
- 补充：`finance_commission_rule_metadata_integration_test.go` 覆盖已到期方案历史 Membership / User 停用后名称备注可改、旧版本冲突、资格日期修改拒绝且零写入；未启用未来方案的元数据例外不能绕过启用资格校验。设计和 `commission-scheme-assignment.md` 同步最小边界。

## 其他核对

- R1：逐笔 Transition CONFIRMED 与月申整批确认两个权威入口均复用 `accountingOrderFeePredicate`；普通未建账费用、当前替代费用及需建账反向行仍参与，已被更正原费与不需建账冲回历史退出。待审更正仍由 `ensureCommissionOrdersReady` 阻断。整批批准共用同一确认辅助，失败由外层事务回滚。
- R6：创建启用方案保留不得追溯起始日；更新只做字段归一化，核心参数及追溯终止限制在锁内判定，已到期原结束日不变允许非核心修改。
- R7：先求各自“方案∩分配”真实交集，再比较实际区间；按输入新身份检查，排除本方案既有分配段。先锁完整预读成员集合，再锁方案，重读集合变化返回冲突；并发 Update / Assign 不会留下重叠资格。

## 独立验证

使用专用 PostgreSQL 实例、正式 SQL 迁移和独立 Schema，以下 8 个父测试全部执行且无 SKIP：

- `TestCommissionRuleHistoricalMetadataInactiveEmployeesPostgres`
- `TestCommissionRuleInactiveEmployeeMetadataCannotEnablePostgres`
- `TestCommissionRuleHistoricalMetadataUpdatePostgres`
- `TestCommissionRuleUpdateActualIntervalsPostgres`
- `TestCommissionRuleUpdateAndAssignConcurrentPostgres`
- `TestCommissionCorrectionHistoryConfirmationPostgres`
- `TestCommissionCurrentUnbilledCorrectionStillBlocksPostgres`
- `TestCommissionApplicationApproveRollbackPostgres`

包耗时 21.233 秒；REPLACE / VOID 的逐笔与月申金额、历史保留、未建账失败零写入均通过。日志位于仓库外 `/tmp/finance-commission-review-fixed.log`。

`go test ./internal/biz -run '(CommissionRule|CommissionClosed|WorkbenchActual|Commission.*Confirm)' -count=1`、`go vet ./internal/biz ./internal/data`、修改 Go 文件 gofmt 与 `git diff --check` 通过。无前端产品变更，TypeCheck 由 Go 编译覆盖；前端 tsc 已在同轮汇率复审通过。未执行漏洞扫描或全量门禁，整体检查由父任务负责。
