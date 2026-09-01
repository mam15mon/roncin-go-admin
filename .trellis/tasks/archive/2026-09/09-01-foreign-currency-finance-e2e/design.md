# 设计：外币财务连续验收

## 变更边界

主要交付物是可重复执行的验收脚本、根目录验收命令和本任务的真实运行报告。预计不
修改生产 API、领域规则、仓储或 Schema。现有应收验收脚本只做两项直接相关调整：
显式创建本轮客户，以及补充 CNY 提成快照断言。

若执行中发现生产响应与既定业务口径不符，先保留失败证据并停止，不把业务修复夹带
进本任务。

## 证据边界

```text
临时库 A（CNY 基线）
  ├─ migrate + bootstrap-admin
  ├─ 真实 PostgreSQL 集成测试（显式连接，禁止 SKIP 冒充 PASS）
  ├─ Go HTTP 服务 :8010
  ├─ Web 测试服务 :8001 → :8010
  └─ acceptance:finance（应收 + 应付 + Playwright）

临时库 B（USD 外币链）
  ├─ migrate + bootstrap-admin
  ├─ 业务数据写入前经 Admin API 设置 base_currency=USD
  ├─ Go HTTP 服务 :8010（与 A 串行，禁止共享进程/数据库）
  └─ acceptance:finance:foreign-currency（纯真实 HTTP API）
```

两个环境串行执行。编排器启动前确认 `8010/8001` 没有监听进程；端口被占用时直接
失败，不终止未知进程或另选端口。每次启动均显式设置数据库、API 和 Web 地址；脚本
缺少必需环境变量时立即失败，不读取开发环境默认值。清理由编排器使用经过固定前缀
校验的明确数据库名和角色名执行，Codex 主会话在结束后独立查询确认。

## 外币场景设计

组织本位币为 USD，业务币为 EUR，所有业务日期使用同一验收日。时间标准和汇率均由
API 创建：

| 环节 | rate_type | 时间标准 | 方向 | 系统汇率 |
| --- | --- | --- | --- | ---: |
| 订单费用 | BASE_CURRENCY | ORDER_CREATED_AT | EUR→USD RECEIVABLE | 1.10000000 |
| 账单 | BILL | BILL_DATE | EUR→USD RECEIVABLE | 1.20000000 |
| 开票 | INVOICE | INVOICE_DATE | EUR→USD RECEIVABLE | 1.22000000 |
| 收款 | SETTLEMENT | TRANSACTION_DATE | EUR→USD RECEIVABLE | 1.25000000 |
| 核销 | WRITE_OFF | WRITE_OFF_TIME | EUR→USD RECEIVABLE | 1.30000000 |
| 提成 CNY | WRITE_OFF | WRITE_OFF_TIME | CNY→USD RECEIVABLE | 0.14000000 |

每条配置同时填写合法的应收、应付汇率，但本任务只断言应收方向。`effectiveFrom` 统一
使用 Asia/Shanghai 当日零点、带时区且精确到秒的 RFC 3339 字符串，不传日期缩写或
纳秒。每个业务响应不仅检查
数值，还必须检查 `exchangeRateSource=SYSTEM`、验收日和创建时保存的 setting ID，
从而排除手工汇率或其他配置碰巧得到同一金额的假阳性。

## 金额与快照口径

外币链使用一笔 100 EUR 的应收费用和账单，开票 100 EUR，收款及核销 40 EUR：

```text
费用本币金额       = 100 × 1.10 = 110.00000000 USD
账单本币金额       = 100 × 1.20 = 120.00000000 USD
开票本币金额       = 100 × 1.22 = 122.00000000 USD
资金流水本币金额   =  40 × 1.25 =  50.00000000 USD
核销本币金额       =  40 × 1.30 =  52.00000000 USD
账单本币分摊       = 120 × 40%  =  48.00000000 USD
资金流水本币分摊   =  50 × 100% =  50.00000000 USD
应收汇兑损益       = 52 - 50    =   2.00000000 USD
提成已实现收入     = 110 × 40%  =  44.00000000 USD（基于账单明细保留的费用原始本位币快照）
10% 提成           = 44 × 10%   =   4.40000000 USD
CNY 派生汇率       = round(1 / 0.14, 8) = 7.14285714
CNY 提成           = round(4.4 × 7.14285714, 8) = 31.42857142
```

注意明确区分两种本币核销与分摊口径：
- 账单头核销分摊口径：使用账单自身汇率折算（120 USD × 40% = 48.00000000 USD）。
- 账单明细/费用快照提成口径：生产代码 `server/internal/data/finance_commission.go` 使用账单行保留的费用原始本位币快照计算已实现收入（110 USD × 40% = 44.00000000 USD），不直接取账单头折算金额。

提成规则使用 `REALIZED_PROFIT` 且该订单不创建应付费用，因此已实现利润等于已实现
收入。脚本应同时断言预览与创建详情，且详情中的 `cnyExchangeRateSettingId` 必须指向
CNY→USD 的 WRITE_OFF 配置，不得错误复用 EUR→USD 核销配置。

## 脚本结构

- 新增 `scripts/run-acceptance-finance-disposable.mjs`：要求调用方显式提供 PostgreSQL
  管理连接，先确认固定验收端口 `8010/8001` 空闲，再串行创建临时库 A/B、迁移、
  初始化、启停服务、执行验收并在 `finally` 清理。数据库名和角色名必须带固定任务前缀并在删除前再次校验。
  - 所有追踪子命令在 Unix 环境下分配独立进程组（`detached: isUnix`），并通过 `stopProcess` 执行进程组整组终止，避免 `go run`、`pnpm` 孙进程孤儿残留；自定义选项（如 `capture`、`onSpawn`）不透传给 `spawn`；`registerChild` 在 leader exit 时仅当整组消亡才移除追踪，存活异常时保守保留；`runCommand` 在 leader 退出但组仍存活时强制拒绝并以 `AggregateError` 清理组，且在非零退出时同时保留原命令错误（含脱敏 capture 输出）与孤儿组错误，绝不假 PASS，error/exit 双事件受 `settled` 保护；`stopProcess` 成为严格可验证的终止原语（检查 `isTargetAlive`，SIGTERM 等待后升级 SIGKILL，再次等待并检查 `process.kill(-pid, 0)`，残留时主动抛错，仅 ESRCH/已退出视为成功）。
  - 建立单一串行清理队列（`runSerializedCleanup`），阶段 `finally` 与 `SIGINT/SIGTERM/SIGHUP` 信号清理串行互斥执行，彻底消除并发清理与双重 DROP 竞态；`runStageA`、`runStageB`、`cleanupStageResources` 与 `main()` 统一使用 `combineExecutionAndCleanupErrors` / `AggregateError` 保留所有业务执行与清理错误，杜绝 cleanup 错误吞掉原始 execution 错误。
  - 支持 `--self-test-lifecycle` 模式：所有自测执行前均运行 `querySQLCount01` 进行 `COUNT=0` Preflight 校验；持续捕获子进程单一输出缓冲 `waitForMarker` 杜绝多阶段握手丢信号；采用“父建子接管 (adopt)”模式，父进程受控创建并由其 pending 集合提供全生命周期安全保障，子进程核验 owner 后接管；覆盖 only-role 正常清理 (1A)、only-role 故障注入 (1B)、父建子接管 SIGTERM 中断 (2)、父流程主动模拟异常故障注入 (3)、就绪超时故障注入 (4)、进程组直接清理故障注入 (5)、runCommand 进程组安全防护双故障注入 (6A/6B) 以及 stage 阶段双错误保留故障注入 (7) 共八项自测（全部自测均内置失败安全 try/finally 与直接证据），末尾断言活跃追踪集合归零，且使用自定义 `SelfTestInjectedError` 精确代码断言，任何清理错误均以 `AggregateError` 严格暴露。
  - `execSQL` / `querySQLSingleValue` / `querySQLCount01` 在报错时对 SQL、stdout 与 stderr 中口令字符串统一执行脱敏（`PASSWORD '[REDACTED]'`），杜绝密码泄漏；`waitForHttpReady` 严格校验 HTTP 2xx。
- 调整 `scripts/acceptance-finance-bill-batch.mjs`：`--apply` 时创建名称带唯一时间戳
  的客户，不再选择数据库中的首个客户；收敛发票编号规则为精确数字枚举 `11`；完成提成调整测试后保持提成，在反核销时断言自动反冲至 `CANCELLED(4)` 并释放财务锁。
- 调整 `scripts/acceptance-finance-payable.mjs`：保存创建的专用 AP fee setting 并在费用创建时按其 ID 精确引用；`ConfirmFeeRequest` body 严格遵循契约字段 `id: draftFee.id`（而非 `feeId`），并精确断言确认后 `confirmedFee.status === 2`。
- 新增 `scripts/acceptance-finance-foreign-currency.mjs`：日历日期 `today` 与 `effectiveFrom` 显式基于 Asia/Shanghai 时区生成，避免跨日误差；状态断言收敛为精确数字枚举；顺序创建本场景所需数据。
- 在 `package.json` 增加单独命令 `acceptance:finance:foreign-currency`；不把外币链
  直接并入现有 `acceptance:finance`，避免已有开发验收命令隐式修改组织本位币。
- 在 `package.json` 增加一次性环境总入口；该入口缺少 PostgreSQL 管理连接或
  bootstrap 凭据时必须直接失败，禁止读取 `.env.local` 或连接默认开发库。
- 外币脚本要求调用方提供全新的空环境，并在启动时验证当前组织尚无业务数据且本位币
  仍为初始化值；验证失败直接终止，不提供回退或自动清库。

## Agy 审查后的拓扑取舍

Agy 独立审查确认：新建子组织后直接切换会因为该组织没有默认角色而得到 403。它建议
用数据库夹具预置 USD 子组织角色。本任务不采用该方案：临时库 B 本身就是单独的空
环境，bootstrap 后使用已有管理员权限通过 Admin API 更新当前总公司本位币即可，随后
才创建第一笔业务数据。这样保留纯 HTTP 业务链，也避免直接写角色、权限和组织表。

Agy 同时建议把提成支付、调整和反核销并入本链。它们不会增强“系统汇率至 CNY 快照”
的核心证据，且 `MarkPaid` 已被审计识别为独立缺口，因此本任务不扩张这些状态流转。

## 安全与失败处理

- 只使用本任务创建并记录的临时数据库、角色和进程；不终止未知进程，不删除未通过
  名称与连接归属校验的数据库或角色。
- 数据库与角色删除前执行存在性与归属验证，删除后执行 `COUNT(*) === 0` 验证；不使用 `IF EXISTS` 假清理。
- `execSQL` 与 `querySQLSingleValue` 执行失败时对涉及的 SQL 语句、stderr 和 stdout 统一脱敏输出，禁止密码出现在错误日志中。
- `waitForHttpReady` 仅接受预期的 HTTP 2xx 响应；其他任何 HTTP 错误码均重试至超时失败，防止未就绪或未授权状态被误判为 ready。
- 任一 HTTP 非 2xx、字段缺失、金额不一致、setting ID 不一致或清理失败都使验收
  失败。脚本不捕获后继续，也不把系统汇率失败改成手工汇率。
- 输出只包含必要业务编号、非敏感 ID、断言摘要和 PASS 状态；认证信息只保留内存。
