# 第一阶段验收记录

## 环境与范围

2026-10-13 用户批准规划后实施，分支 `codex/fee-exception-phase-one`。验证使用本任务独立创建的 PostgreSQL 数据库和专用测试角色，凭据只在仓库外私有临时目录中；每个测试使用 `getIntegrationData` 创建隔离 Schema 并执行正式 SQL 迁移。未重置开发数据库，未通过 Ent Schema.Create 初始化验收环境。

## 执行结果

- 专用数据库执行 `go -C server test -p 32 ./internal/data -run 'TestAutoOrderLock_' -count=1 -v`：8 个顶层测试组、30 个子场景全部 PASS，0 SKIP，17.815 秒。日志保存在本次会话仓库外 `/tmp/roncin-auto-lock-tests.log`。
- 触发场景：100+200 合账收100、100+100同额合账收100、共享与非共享订单均不误锁；全收后通过其它资格才锁；重复触发无重复锁记录；同票多行不重复结算。
- 相邻路径：多账单/多币种、超额异常、纯对冲/混合结清、无效分摊与父状态、草稿、取消、终止/结案、共享组原子性及反转线性化通过。
- 初轮有三个新增测试夹具遗漏正式 CHECK 要求的审计字段，补齐后重跑全部上述测试通过；未修改数据库约束或产品判断来规避失败。
- 独立 trellis-check 全范围静态审查通过，无须自修复；`go -C server vet ./internal/biz ./internal/data` 与 `git diff --check` 通过。
- 第一阶段最终 `pnpm run check:fast` 尚待结案子任务完成后执行；本子任务暂不归档。

## 交付边界

只覆盖本子任务 PRD；费用更正与提成联动纠错另有子任务，第一阶段不宣称该能力已经完成。
