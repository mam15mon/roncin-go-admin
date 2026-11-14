# 收付核销闭环实施与验证（2026-11-14）

## 实施范围

- 在既有核销创建请求附带可选差额流水，空流水ID分摊引用该新流水。差额金额必须与对应分摊合计完全相等。
- `VerificationUsecase.CreateWithCashflow` 复用流水创建、确认和核销，外层共享事务；失败回滚流水、核销、分摊、审计和编号。独立流水仍保留草稿→确认流程。
- 授权沿用核销创建＋流水创建＋流水更新（确认）；显式手工汇率另校验汇率维护权限，所有写入同一公司。未创建差额时空流水ID仍拒绝。
- 子流水幂等键由主核销键派生；成功后重试、相同键并发均回放同一组合，不因汇率配置停用而失去重试能力。不同意图显式冲突。
- 核销工作台支持补差额资料、按已选余额填写金额、生成可编辑分配；取消无写入。失败保留输入与意图键；关闭、切换范围、继续编辑或卸载后旧响应不关闭/清空新输入。
- 收付、核销、对冲三个列表接入现有筛选项设置、页签查询恢复及选择范围；隐藏条件清值刷新，所有请求固定认证公司，统计/导出保持原官方采纳快照。
- 支付方式候选与名称因收付登记、核销差额出现真实跨模块使用，提取 `features/finance/cashflow-entry`；无页面互相深引。
- 保留现有同币种分配、本位币比例及汇兑损益公式和反向操作，未新增参考材料中尚未确认的手续费模型。

## 已通过验证

1. `go -C server test -p 32 ./internal/biz ./internal/service -run 'Verification|FinanceCashflow'`：PASS，含逐项权限拒绝和原核销路径。
2. 专用 PostgreSQL：`TestVerificationShortfallAtomicPostgres`（7场景）与 `TestVerificationCreateSharedTransactionPostgres`（2场景）PASS，验证成功及配置停用重试、审计失败原子回滚、差额合计、同键不同内容、并发余额、同键并发。专用连接由 `/tmp/roncin-finance-20261114-pg/run-integration.py` 注入，没有导入整个本地环境或清理开发库。
3. Vitest：工作台/分配/三个真实筛选组件/流水查询恢复及CSV共5文件25例PASS；流水表单、权限、查询恢复和核销/对冲当前公司共5文件22例PASS（查询恢复文件与上一组重叠）。单文件均小于20秒。
4. `pnpm --dir web tsc` PASS；改动文件Biome PASS；`antd lint VerificationWorkbench.tsx` 零问题；`pnpm run check:architecture` PASS；`git diff --check` PASS。

日志：`/tmp/cashflow-unit.log`、`/tmp/cashflow-shortfall-pg.log`、`/tmp/cashflow-shortfall-final-pg.log`、`/tmp/cashflow-final-web-tests.log`、`/tmp/cashflow-list-tests.log`、`/tmp/cashflow-tsc.log`。

## 后续集成

本组尚未提交，由根会话与账单/发票共享生成物一起分组提交。整体全量门禁与跨模块组合建账验收由根会话执行；本记录不宣称整套财务任务已完成。现有核销仍限定同币种，没有扩大为跨原币核销或添加手续费事实。
