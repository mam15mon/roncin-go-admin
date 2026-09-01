# 外币财务全链路验收报告

## 结论

2026-09-02 在提交基线 `07639177` 及本任务工作区变更上完成两次完整的一次性
PostgreSQL 双环境验收，最终修正版再次全量通过。独立 `trellis-check` 第六轮结论为
`PASS`，P0/P1/P2 均为 0。

本次证据支持以下结论：

- CNY 本位币基线中的真实 PostgreSQL 集成测试、应收、应付、页面核销和提成 CNY
  快照断言可以在全新数据库中连续通过。
- USD 本位币组织中，同一组真实业务 ID 可以从 EUR 系统汇率、订单费用、账单、
  发票、收款、核销连续传递到提成预览、创建与持久化详情重读。
- 外币链没有提交手工汇率覆盖值；五个业务阶段均命中各自系统汇率配置，提成 CNY
  汇率由 CNY→USD WRITE_OFF 配置倒数派生。
- 一次性环境编排器的正常、信号中断、超时、父流程失败、孤儿进程组及执行/清理
  双失败路径均有故障注入证据，最终没有遗留任务数据库、角色、固定端口或追踪进程。

本任务只新增和修正验收基础设施，没有修改生产财务、汇率或提成计算代码。验收发现
提成已实现收入采用账单明细保留的费用原始本位币快照，这是现有生产口径，不是本次
发现的生产缺陷。

## 执行环境

- 时间：2026-09-02 01:02 CST（Asia/Shanghai）
- 系统：本地 WSL，PostgreSQL `127.0.0.1:5432`
- 管理连接：通过环境变量显式注入；本报告不记录用户名、密码或完整连接串
- 临时资源：两套不同的一次性数据库与同名角色，均使用
  `roncin_acc_fin_` 固定任务前缀
- 固定服务端口：HTTP `8010`、gRPC `9010`、Web `8001`
- 验收账号：一次性数据库中由 `bootstrap-admin` 创建；本报告不记录密码

## 最终完整验收

执行入口：

```bash
node scripts/run-acceptance-finance-disposable.mjs
```

调用时显式提供 PostgreSQL 管理连接和 bootstrap 管理员所需环境变量。编排器拒绝
缺失变量、已占用固定端口、非任务前缀资源及不满足数据库 owner 约束的清理目标，
不会回退到开发数据库。

### Stage 1：CNY 基线

- 新库迁移：PASS
- 权限清单及订单主数据同步：PASS
- 真实 PostgreSQL 顶层测试：`PASS=10, SKIP=0, FAIL=0`
- 管理员初始化：PASS
- Go HTTP/gRPC 服务就绪：PASS
- Web 服务就绪：PASS
- 应收 API 验收：PASS
- 应付 API 验收：PASS
- Playwright 财务闭环：`1 passed (24.8s)`
- CNY 提成快照：`BASE_CURRENCY / 1.00000000`，本位币金额与 CNY 金额一致
- Stage 1 数据库及角色销毁后精确计数：0

### Stage 2：USD 本位币、EUR 业务币

新库迁移、管理员初始化和服务启动均为 PASS。外币连续 API 链路使用同一订单及其
下游业务对象，最终断言如下：

| 节点 | 业务币金额 | 系统汇率 | 本位币金额/结果 |
| --- | ---: | ---: | ---: |
| 费用 | 100.00000000 EUR | 1.10000000 | 110.00000000 USD |
| 账单 | 100.00000000 EUR | 1.20000000 | 120.00000000 USD |
| 发票 | 100.00000000 EUR | 1.22000000 | 122.00000000 USD |
| 收款流水 | 40.00000000 EUR | 1.25000000 | 50.00000000 USD |
| 核销 | 40.00000000 EUR | 1.30000000 | 52.00000000 USD |
| 账单本币分摊 | 40% | 账单汇率 | 48.00000000 USD |
| 流水本币分摊 | 100% | 流水汇率 | 50.00000000 USD |
| 应收汇兑收益 | — | — | 2.00000000 USD |
| 提成已实现收入/利润 | 费用原始本位币快照 × 40% | — | 44.00000000 USD |
| 10% 提成 | — | — | 4.40000000 USD |
| 提成 CNY 快照 | 4.40000000 USD | 7.14285714 | 31.42857142 CNY |

额外验证：

- 费用、账单、发票、流水与核销汇率来源均为 `SYSTEM`。
- 各阶段汇率日期均等于 Asia/Shanghai 当日，setting ID 精确指向对应的
  `BASE_CURRENCY`、`BILL`、`INVOICE`、`SETTLEMENT`、`WRITE_OFF` 配置。
- 提成 CNY 来源为 `DERIVED`，setting ID 精确指向 CNY→USD WRITE_OFF 配置；
  `round(1 / 0.14, 8) = 7.14285714`。
- 提成创建后通过详情 API 重读，持久化快照与创建响应一致。
- Stage 2 数据库及角色销毁后精确计数：0。

## 生命周期与失败安全验证

执行入口：

```bash
node scripts/run-acceptance-finance-disposable.mjs --self-test-lifecycle
```

最终修正版由 Codex 主会话亲自执行，以下场景全部 PASS：

1. Test 1A：only-role 正常清理。
2. Test 1B：角色真实创建后注入父流程错误，`finally` 精确回收。
3. Test 2：父进程创建资源、子进程验证 owner 后 adopt，收到 SIGTERM 后完整清理。
4. Test 3：child ready 后父流程注入错误，终止子进程组并回收数据库和角色。
5. Test 4：child adopt 后注入 ready timeout，父方 `finally` 精确兜底。
6. Test 5：leader 退出而孙进程组存活，追踪句柄不丢失并完整清组。
7. Test 6A：`runCommand` 的 leader 以 0 退出但进程组残留，必须拒绝假 PASS 并清组。
8. Test 6B：命令非零退出且进程组残留，错误同时保留退出码、脱敏 capture 输出和
   孤儿组证据，并完整清组。
9. Test 7：Stage 业务执行与资源清理同时失败时，两个原始错误均由
   `AggregateError` 保留。

结束断言：`activeChildProcesses.size = 0`。

## 静态与回归检查

以下命令均在最终代码快照上通过：

```bash
node --check scripts/run-acceptance-finance-disposable.mjs
node --check scripts/acceptance-finance-foreign-currency.mjs
node --check scripts/acceptance-finance-bill-batch.mjs
node --check scripts/acceptance-finance-payable.mjs
go -C server test ./...
go -C server vet ./...
pnpm --dir web lint
pnpm --dir web tsc
git diff --check
```

独立检查代理还实际复核了 Test 5、Test 6A、Test 6B 和 Test 7，确认测试确实创建并
观察目标失败状态，不是只匹配硬编码成功文案。

## 清理结果

最终完整验收和生命周期自测结束后再次查询：

```text
DATABASES_REMAINING=0
ROLES_REMAINING=0
FIXED_PORTS=FREE
```

其中计数仅针对本任务固定前缀；清理前验证精确名称、任务前缀和数据库 owner，清理后
验证精确 `COUNT(*) === 0`。没有执行模糊删除、`IF EXISTS` 假清理或未知进程终止。

## 独立质量复核

独立 `trellis-check` 对最终修正版逐层检查脚本接线、业务数据连续性、金额口径、进程
组生命周期、安全清理、错误聚合、静态检查和回归测试，最终结论：

```text
PASS
P0=0
P1=0
P2=0
Findings not fixed: 无
```

## 证据边界与剩余范围

这次验收提供了当前代码快照上强度较高的真实链路证据，但不能证明整个财务系统在所有
输入、权限、组织、并发和部署环境下绝对不存在缺陷。以下内容在 PRD 中明确不属于本次
范围，因此不能由本报告宣称已验证：

- 提成 `CONFIRMED → PAID`（`MarkPaid`）状态流转；
- 应付方向的外币连续链路；
- 低权限用户拒绝与双组织数据隔离；
- CI 自动创建 PostgreSQL 服务并强制执行这套一次性验收。
