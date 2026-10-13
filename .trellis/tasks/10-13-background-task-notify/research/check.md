# 独立审查记录

## 审查范围与结论

按 PRD、design、implement、check.jsonl 和后端数据库/质量/分层/钉钉专项规范，
审查正式触发器迁移、biz 接口、共享 data 监听器、四类 worker、Wire 与配套测试。
最近到期查询与既有 Claim 条件一致，业务领取、租约 token、attempts 和幂等规则未改。
通知提交/回滚、类别广播与合并、真实 search_path 隔离、监听先就绪再扫描、晚订阅、
最后退订再订阅、重连和取消路径均有实际代码与验证证据。修复后没有未解决的产品问题。

## 已修复发现

1. `internal/server/worker_poll.go` 原错误退避 timer 可以被任意已缓存或新通知立即
   打断。领取/失败状态更新自身也发通知，可能令错误处理路径绕过最短退避快速重试。
   增加异常等待最短 1 秒门槛，门槛内合并保留通知，门槛后可提前结束更长退避；
   context 取消、关闭订阅始终立即退出。不改变业务重试次数和调度时间。
   `worker_schedule_test.go` 增加处理错误/查询错误/过期时间三种通知风暴场景和
   两种停止场景；synctest 证明每轮至少等待 1 秒且停止不必等待计时器。
   design.md 与后台能力规范同步最终等待契约。
2. 原真实调度测试只查询 RUNNING 最近租约时间，没有实际等到租约到期后再领取。
   增补 FAILED 延迟重试实际 Claim、RUNNING 过期再 Claim、租约前不可领取、attempts
   及 token 更换断言；inbox 过期领取也断言 token 更换。
3. 原 Schema 过滤与 UPDATE 过滤只有部分单元证据。新增两个正式迁移隔离 Schema、
   两条真实 LISTEN 连接，验证写入只唤醒匹配 Schema；无关字段和调度字段同值更新
   不通知；实际 next_run_at、RUNNING 租约和终态更新都通知。

## 未修复发现与范围限制

- `.github/workflows/ci.yml` 的既有 Go Job 只运行 `go test ./...`，没有 PostgreSQL
  service 或专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`。新增真实通知/迁移用例会在
  该常规 CI 环境跳过，现有真实数据库用例亦有同样缺口。补 CI 数据库 Job 涉及工作流
  和环境配置，留给后续任务；本次不扩大产品实施范围。建议后续为 PG Job 显式提供
  专用库，并执行本次 scheduler suite 与冷启动链，不能以 SKIP 视为数据库验证通过。
- race 检查受环境限制：默认 cgo 关闭，启用后无 gcc/cc/clang，实施阶段已确认
  runtime/cgo 编译失败。没有关闭规则或安装额外依赖来掩盖结果。

## 验证证据

- 定向单元：`go -C server test ./internal/server ./internal/biz ./internal/data
  ./internal/platform/migration -run 'TestWorkerSchedule|TestTaskSchedule|
  TestDingTalkDirectoryWorker|TestAllDisabledWorkers|TestWorkerPoll' -count=1` PASS。
  纯 Go 单元覆盖通知保留、到期 timer、60 秒频率、禁用零访问与停止。
- 定向静态检查：`go -C server vet ./internal/server ./internal/biz ./internal/data`
  PASS；`git diff --check` PASS。
- 真实 PostgreSQL 18.6：使用 `/tmp/roncin-notify-verify.mjs` 运行
  `./internal/data ./internal/platform/migration -run
  'TestBackgroundTaskScheduler|TestPostgresColdStartMigration' -count=1 -v -timeout=180s`。
  5 组 scheduler suite 共 16.398 秒 PASS，完整正式迁移冷启动 3.756 秒 PASS，零 SKIP。
  包括事务、四个业务写入口、独立审批 inbox、竞争、断线恢复、重复迁移、退订重订、
  Stop 和真实跨 Schema/更新过滤。精确临时库
  `roncin_notify_verify_1791877402349` 已由 finally 销毁，未操作开发数据。
- Wire：重新执行 `go -C server generate ./cmd/server`，生成前后 SHA-256 均为
  `e7903a76bbbf8e30a426c5fc882da609e9be7eb18cdad66b8defcfdddc629ff7`，幂等 PASS。
- 主会话先前真实时钟验收 65.005 秒空闲 Claim=1、NextRun=1，真实通知至 worker
  领取 1.536916ms。此次修复仅错误等待分支，正常空闲计时语义不变，未重复长等待。
- 最终 `pnpm run check:server` 在产品冻结后由主会话执行，exit 0 PASS；包含
  Go 分层与重复代码、Proto lint、全部 Go 单元/编译、完整 vet 和漏洞检查。
  证据日志为 `/tmp/roncin-notify-check-server.log`；未配置真实库的普通门禁
  数据库用例跳过，与上述显式真实数据库 PASS 分开计。

无前端、Proto、权限或 Ent Schema 字段变更；未手工编辑生成物。审查期间保留其他
会话任务及其提交，没有提交或覆盖无关改动。
