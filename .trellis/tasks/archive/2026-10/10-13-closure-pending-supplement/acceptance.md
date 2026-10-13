# 第一阶段验收记录

## 环境与范围

2026-10-13 用户批准规划后实施，分支 `codex/fee-exception-phase-one`。验证使用本任务独立创建的 PostgreSQL 数据库和专用测试角色，凭据只在仓库外私有临时目录中；每个测试使用 `getIntegrationData` 创建隔离 Schema 并执行正式 SQL 迁移。未重置开发数据库，未通过 Ent Schema.Create 初始化验收环境。

## 执行结果

- 专用 PostgreSQL 执行 `go -C server test -p 32 ./internal/data -run 'TestOrderClosurePendingSupplement|TestOrderLifecycleGatePostgres|TestFeeSupplementApprovalAdmissionPostgres' -count=1 -v`：5 个顶层测试组全部 PASS，0 SKIP，16.705 秒。包含补录先/结案先/批准先三个明确屏障控制的真实并发顺序。
- 服务端覆盖只读与锁内 PENDING 检查、拒绝时零状态/事件写入、清理后可结案、批准后未建账阻断、CLOSED 后提交/撤回/退回/批准及确认提成冲减、普通编辑仍被保护。
- biz `TestOrderClosure` 定向验证通过；独立 reviewer 的 `go -C server vet ./internal/biz ./internal/data` 通过。
- 前端三套：`detail-status-transition.test.tsx`、`order-detail-transitions.test.tsx`、`FeeSupplementSection.state.test.tsx`，共24例。初次23 PASS/1 FAIL为新增空拒绝值断言错误，按真实 Promise.reject(undefined) 改断言后失败所属套件7例全 PASS，另两套此前17例 PASS。
- `pnpm --dir web tsc`、修改文件 Biome、`git diff --check` 均通过。独立审查通过，无未解决产品/设计项。
- 日志在仓库外 `/tmp/roncin-closure-tests.log`、`/tmp/roncin-closure-web-tests.log`、`/tmp/roncin-closure-web-retest.log`。检查代理把新增空响应 mock 改为明确的生成函数参数 + 可空返回类型，去掉新增 any；随后7/7测试、tsc和Biome再次PASS，不改产品契约。

## 交付边界

只覆盖本子任务 PRD；费用更正与提成联动纠错另有子任务，第一阶段不宣称该能力已经完成。

## 第一阶段最终门禁

- `pnpm run check:fast`：PASS，130.1 秒；web 64.7秒，server 130.1秒。完整日志 `/tmp/roncin-fee-phase1-check-fast.log`。
- 前端 225 个文件、1,204 个测试 PASS；全量套件另有 1 文件/12 测试 SKIP，未作为本次验收证据。本次修改的定向测试无跳过。
- 后端全量测试、vet、Proto/分层/重复代码检查及漏洞扫描通过，扫描无未处理可达漏洞。全量命令没有注入专用数据库，真实数据库验收由上述定向测试独立完成，不能把全量命令里的数据库 SKIP 当作通过。
- 全量前端输出有 localhost:3000 连接拒绝/重置日志，但测试退出成功；本次定向套件无此噪音，未扩大到无关测试修复。
- 独立 trellis-check 最终全范围审查通过；最后的测试 mock 类型收窄之后又跑对应7例、tsc、Biome及差异检查，均通过。
- 未运行生产构建（无构建配置/依赖/生产入口变化），未部署、未推送。临时测试库只承载本任务验证，收尾删除本任务创建的库、角色与凭据。
