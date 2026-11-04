# 提成修复验证记录

## 实现范围

- `server/internal/biz/finance_commission_rules.go`：拆分创建专有日期准入和更新字段归一化。
- `server/internal/data/finance_commission_rules.go`：锁内重读分配员工集合；所有启用方案按真实实际区间复核；允许到期后非核心元数据编辑；保留核心参数与追溯失效拒绝。
- `server/internal/data/finance_commission_write.go`：逐笔与月申整批确认共用现有需建账费用谓词。
- `server/internal/data/finance_commission_rule_update_integration_test.go`：生效次日/到期方案元数据、版本/核心参数/追溯失败零写入、复制后延长区间拒绝、方案/分配两两相交但实际区间不相交、新身份冲突、Update 与 Assign 并发。
- `server/internal/data/finance_commission_correction_gate_integration_test.go`：真实取消成本账单、批准 REPLACE/VOID 后逐笔与月申确认；历史两行保留、金额正确；当前未建账替代费仍阻断且零部分写入。

## 已执行

- Biz：`GOCACHE=/tmp/finance-review-go-cache go -C server test ./internal/biz -run 'Test.*CommissionRule|TestCommissionClosedIntervalsOverlap' -count=1` 通过。
- `gofmt` 与 `git diff --check` 通过。
- Data 第一轮命中并行实施中的对冲 setter 缺失，第二轮命中并行账单测试 `ClearBatchID` 缺失，均在编译阶段结束，未计作数据库测试通过。等待对应组修正编译后重跑。

## 真实 PostgreSQL 结果

使用专用实例 38172、正式迁移和各场景独立 Schema，以下测试均实际执行且无 SKIP：

- 方案新回归：`TestCommissionRuleHistoricalMetadataUpdatePostgres`、`TestCommissionRuleUpdateActualIntervalsPostgres`、`TestCommissionRuleUpdateAndAssignConcurrentPostgres` 全部通过。
- 方案相邻回归：CreateAndAssign、IntervalUniqueness、EffectiveChange、LegacyAndCopy、MemberDisableGuard、ConcurrentOverlap 六个既有父测试全部通过。
- 门禁相邻回归：`TestCommissionApplicationApproveRollbackPostgres`、`TestCommissionConfirmationFullOrderSetPostgres`（confirm / batch-confirm）通过。
- 上述组合轮只有新增 R1 夹具失败：无海运单证的 SE 订单不能建立正式锁快照。按既有提成更正夹具切为 SI 并给真实 SI 审核权限后，定向重跑 `TestCommissionCorrectionHistoryConfirmationPostgres` 与 `TestCommissionCurrentUnbilledCorrectionStillBlocksPostgres`，两个父测试、六个子场景全部通过，包耗时 12.728 秒。前一轮组合日志 `/tmp/finance-commission-fixes-targeted.log`；R1 成功日志 `/tmp/finance-commission-correction-gate.log`。
- R1 成功验收金额：REPLACE 成本 400→200 后确认 80；VOID 成本后确认 100；逐笔/月申口径一致。当前未建账替代费继续阻断，原费及冲回历史两行保留。

## 独立审查结果

独立 reviewer 复现并修正历史员工停用后纯元数据修改仍被误拒绝的相邻问题；按“仅名称/备注且全部资格字段不变”的最小例外放行，成员加锁、集合复核与任何资格变更的有效性校验均保留。新增历史停用/启用拒绝验证。八个父测试在真实 PostgreSQL 独立重跑通过，包耗时 21.233 秒，无 SKIP；Go vet biz/data、gofmt、diff-check 通过。完整记录见父任务 `research/review-commission.md`。整体门禁与提交由主代理负责。
