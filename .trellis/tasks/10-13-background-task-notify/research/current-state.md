# 当前代码证据与方案选择

## 仓库证据

| 入口 | 当前行为/影响 |
| --- | --- |
| `server/internal/server/dingtalk_directory_worker.go:13`、`:46` | 固定 2 秒空闲检查，功能关闭挂起直到取消 |
| `server/internal/server/worker_poll.go:8`、`:30` | timer 等待，空闲最大 15 秒，未接信号 |
| `server/internal/server/notification_worker.go` | 处理成功立即继续，空闲退避 |
| `server/internal/server/object_deletion_worker.go` | 5 秒初始、最大 15 秒，租约 60 秒 |
| `server/internal/server/dingtalk_approval_worker.go` | 派发与 callbackEnabled inbox 共用循环 |
| `server/internal/data/background_task.go:29`、`:59` | 通用入队、PENDING/FAILED 到期和 RUNNING 租约过期领取 |
| `server/internal/data/notification.go:33`、`:68` | 独立创建 BackgroundTask，不能只改通用 Enqueue |
| `server/internal/data/object_storage_deletion.go:37` | 对象删除在既有事务中直接建任务 |
| `server/internal/data/dingtalk_directory.go:30` | 通讯录任务与同步记录在同一事务创建 |
| `server/internal/data/dingtalk_approval_repo.go:31` | 收件箱持久化独立于 BackgroundTask |
| `server/internal/data/dingtalk_approval_inbox.go:21` | RECEIVED/FAILED 到期和 PROCESSING 租约过期独立领取 |
| `server/internal/data/transaction.go` | 统一 WithTx/共享 txCtx，通知不能在外层提交前唤醒 |
| `server/cmd/server/main.go:49`、`server/internal/server/server.go` | Kratos server 生命周期和 Wire 注册位置 |
| `server/go.mod:12` | 已有 pgx/v5，无须新增客户端依赖 |

2026-10-13 源码检索未发现现有 LISTEN/NOTIFY 唤醒实现。当前会话已解释方案，
用户明确要求创建 Trellis 任务；本轮只规划。相关领域已读 `.trellis/spec/domain/`、
数据库/质量/钉钉专项规范；不再查询会话历史，因为讨论就在当前上下文。

## 取舍

- 仅改通讯录轮询不能覆盖用户指出的后台任务空转；本期统一四类真实消费者。
- 进程内 Go channel 不能跨进程恢复；增加消息队列不符合现阶段维护成本目标。
- 表触发器保证所有业务事务路径都能发唤醒，避免逐仓储遗漏；通知承载固定路由而非载荷。
- 永久只等通知会漏重试与租约到期；最近时间 timer 加 60 秒恢复扫描明确保留可靠性。
- 一条实例级专用连接减少四个 worker 分别占连接；局部分发仍按类别广播。
- 当前索引覆盖 status+next_run_at/lease_expires_at；实施时审查查询计划，只有证据
  表明需要才改索引，不提前增加索引或调度实体。

## 外部技术依据

PostgreSQL 官方 NOTIFY：事务提交后交付，回滚无事件，同事务同 payload 可以合并；
LISTEN：先监听提交再扫描状态，连接关闭时取消监听。

- https://www.postgresql.org/docs/current/sql-notify.html
- https://www.postgresql.org/docs/current/sql-listen.html
