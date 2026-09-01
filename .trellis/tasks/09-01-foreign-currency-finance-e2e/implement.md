# 实施计划：外币财务连续验收

## 1. Agy 实施与主会话职责

- 用户批准本计划后，由 agy 新建独立实施会话，固定使用
  `gemini-3.7-flash-high`；这是多币种财务口径和真实数据库验收，不使用 Medium。
- agy 从本任务目录开始，依次完整读取 `implement.jsonl` 及引用、`prd.md`、
  `design.md`、本文件和根 `AGENTS.md`，只执行本任务，不处理 `MarkPaid`、权限或 CI。
- agy 负责脚本实现和第一轮相关测试，禁止所有 Git 提交、推送、合并、变基、重置或
  覆盖工作区操作。
- Codex 主会话检查真实 Git 差异与完整 ID/汇率数据流，独立运行 `trellis-check`，
  在隔离环境复跑全部验收后才提交。

## 2. 让现有 CNY 验收可在空库运行

- 新增任务专用一次性环境编排器，显式接收 PostgreSQL 管理连接并确认现有验收配置的
  固定端口 `8010/8001` 空闲，串行管理临时库 A/B 的创建、迁移、初始化、健康检查、
  验收、进程停止和资源删除。
- 编排器对所有子进程在 Unix 下分配独立进程组并在 stopProcess 中执行可验证的完整进程组终止（检查 `isTargetAlive`，SIGTERM 等待后升级 SIGKILL，再次等待并检查 `process.kill(-pid, 0)`，残留时主动抛错）；通过单一串行清理所有权消除阶段 finally 与信号处理器的竞态；`runStageA` 与 `runStageB` 引入 `combineExecutionAndCleanupErrors`，业务执行错误与资源清理错误并存时使用 `AggregateError` 统一合并向上抛出；`registerChild` 在 leader 退出但组仍存活时保守保留追踪；`runCommand` 在 leader 退出但组残留时拒绝并强制终止组，且在非零退出时同时保留命令原错误与孤儿组错误，error/exit 受 `settled` 保护；通过 `--self-test-lifecycle` 自测 only-role 正常清理 (1A)、only-role 故障注入 (1B)、父建子接管 SIGTERM 信号中断 (2)、父流程主动模拟异常故障注入 (3)、就绪超时故障注入 (4)、进程组直接清理故障注入 (5)、runCommand 进程组安全防护双故障注入 (6A/6B) 及 stage 阶段双错误保留故障注入 (7) 共八项自测（所有自测均有严格 Preflight 校验、单一缓冲 `waitForMarker` 握手、直接真实创建证明、失败安全 try/finally 及 `finally` 归零回收，末尾断言活跃追踪集合归零，使用自定义 `SelfTestInjectedError` 严格代码断言）；由父进程预生成确定性资源名称并提供 pending 跟踪保障，外层受控 `try/finally` 确保全流程零泄漏；`execSQL`/`querySQLSingleValue`/`querySQLCount01` 在失败时对 SQL、stderr、stdout 统一脱敏输出口令，防止密码泄漏；`waitForHttpReady` 严格只认 HTTP 2xx；清理失败仍须返回非零；只能删除通过任务前缀和当前连接归属双重校验的库/角色，不提供 `IF EXISTS` 后静默成功的假清理。
- 调整应收批量账单脚本，在 `--apply` 模式显式创建专用客户和所需开票资料；移除对
  “数据库至少已有一个客户”的依赖；发票编号规则收敛为精确数字枚举 `11`；完成提成调整测试后保持提成，在反核销时断言自动反冲至 `CANCELLED(4)` 并释放财务锁。
- 调整应付验收脚本，保存创建的专用 AP fee setting 并在创建费用时按其 ID 精确引用；`ConfirmFeeRequest` body 修正为契约字段 `id: draftFee.id`（而非 `feeId`），并在确认后断言 `confirmedFee.status === 2`。
- 对提成预览、创建及详情补充 `BASE_CURRENCY/1` 和 CNY 金额断言。
- 先执行 Node 语法检查和脚本预检，再在临时库 A 完整运行原有验收。

## 3. 实现 USD 本位币下的 EUR 连续链路

- 新增外币验收脚本：登录、在无业务数据的新环境设置 USD 本位币、设置五类时间标准、
  创建六条系统汇率。
- 日历日期与汇率 `effectiveFrom` 使用 Asia/Shanghai 当日零点的秒级 RFC 3339；订单费用时间标准
  使用 `ORDER_CREATED_AT`，其余严格使用 `BILL_DATE`、`INVOICE_DATE`、
  `TRANSACTION_DATE`、`WRITE_OFF_TIME`。
- 复用现有 API 契约顺序创建客户、订单、人员归属、EUR 应收费用、EUR 账单、发票、
  EUR 收款、核销、提成规则及提成。
- 每一步保存创建返回的 ID 和汇率配置 ID；后续只能引用这些 ID，并按 `design.md`
  精确断言汇率、来源、日期、本币金额、汇兑损益与 CNY 快照。
- 资金流水和其他业务请求不得携带手工汇率字段。遇到生产口径差异，保留失败输出并
  返回主会话，不自行改生产代码。

## 4. 真实环境验证

编排器内部的临时库 A：

```bash
RONCIN_INTEGRATION_DATABASE_SOURCE='<临时连接>' go -C server test -v ./... -count=1
go -C server vet ./...
RONCIN_ACCEPTANCE_BASE_URL='http://127.0.0.1:8010' \
RONCIN_WEB_BASE_URL='http://127.0.0.1:8001' \
pnpm run acceptance:finance
```

编排器内部的临时库 B：

```bash
RONCIN_ACCEPTANCE_BASE_URL='http://127.0.0.1:8010' \
pnpm run acceptance:finance:foreign-currency
```

外部统一入口要求显式管理连接和 bootstrap 凭据，实际环境变量名以仓库现有命名风格
确定并写入脚本帮助信息；不得设置默认数据库。端口固定使用现有 acceptance 配置的
`8010/8001`，被占用即失败：

```bash
RONCIN_ACCEPTANCE_ADMIN_DATABASE_SOURCE='<仅用于创建临时库/角色的管理连接>' \
pnpm run acceptance:finance:disposable
```

通用检查：

```bash
node --check scripts/acceptance-finance-bill-batch.mjs
node --check scripts/acceptance-finance-foreign-currency.mjs
node --check scripts/run-acceptance-finance-disposable.mjs
pnpm --dir web tsc
go -C server test ./...
go -C server vet ./...
git diff --check
```

- 数据库迁移、管理员初始化、后端和 Web 服务均绑定显式临时数据库与端口。
- 记录每条命令的退出码、PostgreSQL 测试 PASS/SKIP 计数及两条验收链关键编号。
- 停止本轮进程，删除两个通过固定前缀校验的临时数据库和角色，并查询系统表确认不再
  存在。清理失败视为任务未完成。

## 5. 独立检查与提交

- `trellis-check` 重点检查：是否仍依赖历史客户、是否存在开发库回退、是否有手工汇率、
  是否用同一业务 ID 串联、六个 setting ID 是否逐层断言、金额舍入是否精确，以及
  真实 PostgreSQL 测试是否被 SKIP 冒充 PASS；另外检查编排器能否在中途失败时安全
  停进程并删除且仅删除本任务资源。
- 生成物预计无变化；若出现生成物差异，必须先解释源文件变更，否则不得提交。
- 全部通过后由 Codex 提交，建议提交信息：
  `test: 补齐外币财务全链路验收`。
- 随提交保存本任务验收报告；Trellis finish/archive 在提交和最终复核之后执行。
