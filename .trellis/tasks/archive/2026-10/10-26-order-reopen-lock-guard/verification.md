# 验证记录

## 已通过

- 反结案原因选填；空值、纯空格成功，超过 500 字仍拒绝。
- 服务端 CLOSED 已锁时不投影 REOPEN；订单行锁内再次校验人工审核锁。
- 前端入口与确认提交前校验锁状态，锁定、加载/同步、错误或缺失均不提交；只读已完结状态展示先解锁提示。
- 前端定向 4 个文件共 39 项测试；TypeScript、5 文件 Biome、Go 定向单元与 vet 通过。
- 真实 PostgreSQL 三组测试零 SKIP：`TestOrderReopenRequiresUnlockPostgres`、`TestOrderLifecycleGatePostgres`、`TestOrderClosurePendingCorrectionClosedWorkflowPostgres`。
- 并发验证：审核持有订单行锁，反结案等待后读取已提交的新锁状态并拒绝，订单、版本、生命周期事件、审计无副作用；正式解锁后空原因反结案成功。
- 全量前端复跑：`pnpm --dir web exec vitest run --maxWorkers 16`，307 文件通过、1 文件按既有配置跳过，1681 项通过、12 项跳过。

## 门禁说明

`check:fast` 的高并发前端测试中，验证码倒计时及共享主单继承两个无关测试超时；未修改测试规则或跳过用例，降低 Worker 数完整复跑后全部通过。生成物、架构、lint、类型检查通过。

后端完整 `pnpm run check:server` 通过，包含单元测试、vet、分层、Proto lint 与漏洞扫描；无未处理可达漏洞。工作提交：`1e378866`。
