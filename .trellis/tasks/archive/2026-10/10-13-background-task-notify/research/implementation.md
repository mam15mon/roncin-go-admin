# 实施记录

## 实际边界

本次仅替换现有后台消费者的空闲调度等待：两表通知触发器、纯 Go 调度接口、实例级
共享监听连接、最近到期聚合和四类 worker 接入。既有 ProcessNext、ClaimAny、锁、
租约 token、attempts 和业务幂等保持原样；汇率提醒周期、前端、Proto、权限和 Ent
字段/索引不变。注入变更使用 Wire 生成，未手改生成文件。

## 实现落点

- `server/migrations/20261013150000_background_task_notify.sql`：background_tasks
  与 ding_talk_approval_inbox_events 的精确 INSERT/调度 UPDATE 触发器。Schema、
  consumer 两字段固定路由，无业务载荷；其他类型不通知，无变化 UPDATE 不通知。
- `internal/biz/background_task_scheduler.go`：TaskConsumer 四类别和
  BackgroundTaskScheduler 生命周期、Subscribe、NextRun 接口。
- `internal/data/background_task_scheduler.go`：一个实例一个 pgx LISTEN 连接，
  按类别广播到容量 1 的每订阅缓冲；最后退订取消连接；重新订阅启动新会话；
  就绪/重连后广播，1～30 秒有界重连，Stop 关闭订阅且可取消等待。
- 最近时间通过 Ent 入口取得 `Data.client(ctx)`，SQL 聚合集中 data；后台任务
  PENDING/FAILED 看 next_run_at，RUNNING 看 lease_expires_at；仅回调开启时查询
  RECEIVED/FAILED/PROCESSING inbox 并合并最近时间。终态排除。
- `internal/server/worker_poll.go` 保留错误退避实现，新增统一等待器：领取前合并旧
  信号，领取及最近时间查询后信号保留；正常空闲等 60 秒，未来时间按定时器唤醒，
  过期但无可领取任务/处理失败/查询错误走 1～30 秒退避，Stop 关闭信号退出循环。
- 四类 worker 只替换订阅/等待，禁用时既不订阅也不扫描。
- `cmd/server/main.go` 加入 scheduler 生命周期；`Data.ProviderSet` 登记提供者；
  `go -C server generate ./cmd/server` 更新 Wire。

## 实际发现

复制 stdlib 活连接 pgx Config 时，`OnNotification` 已绑定旧连接的通知缓冲；若直接
用于新的专用连接，pgx 不再安装新连接自身的缓冲，真实 `WaitForNotification` 可返回
nil,nil。专用连接配置复制后必须清空 `OnNotification`，由新连接安装自己的回调。
真实提交通知集成测试覆盖此路径，不能只用伪造 Go channel 检查。

系统组织的正式当前 CHECK 要求 base_currency 非空，集成夹具按正式迁移设置 CNY。
监听断线测试精确使用受互斥锁保护的 listenerPID，避免终止其他隔离 Schema 的连接。

## 定向验证

- server 单元定向 PASS：等待中取消、关闭订阅停止、查询/等待竞态保留通知、未来到期
  无通知、过期时间/查询错误退避、四类禁用零访问、目录 59 秒无查询且 60 秒恢复。
- data 单元 PASS：Schema/类别路由、相同类别广播、重复信号合并、无订阅启动停止。
- 真实临时库 TestBackgroundTaskSchedulerPostgres PASS：真实业务写入口事务提交/
  回滚、通知延迟 <=1 秒、未来任务/租约/inbox 查询及领取、两个实例有效租约竞争、
  精确终止监听后重连扫描。
- 真实升级 TestBackgroundTaskSchedulerMigrationUpgrade PASS：正式前态升级、重复
  执行后仅两个通知触发器及一条 migration revision。
- 真实 TestBackgroundTaskSchedulerResubscribeAndStopPostgres PASS：最后退订后立即
  新订阅、新监听 PID 及 Stop 解除等待。
- 真实 TestBackgroundTaskSchedulerReconnectCancellationPostgres PASS：终止精确监听后
  Stop 无需等重连计时器，监听连接无残留。最后真实 suite 9.691 秒 PASS，零 SKIP。
- 定向 `go vet` 与 `git diff --check` PASS。
- race 检查无法运行：系统默认 cgo 关闭；显式 CGO_ENABLED=1 后环境缺少 gcc/cc/clang，
  因而 runtime/cgo 编译失败；未安装额外依赖，常规单元与真实 PostgreSQL 已通过。
- 主会话记录完整正式迁移链冷启动 PASS，后端完整门禁 PASS。
- 主会话真实时间实测：完全空闲 65.005 秒 Claim=1、NextRun=1；真实 pg_notify
  提交至 NotificationWorker 开始 Claim 为 1.536916ms；临时库已清理。

数据库回归使用主会话提供的 `/tmp/roncin-notify-verify.mjs`，始终创建并删除精确
`roncin_notify_verify_` 临时库；未清空或重建用户开发库。

## 交接审查重点

实现已交接，主会话负责规范和提交。独立审查可补真实跨 Schema 通知、调度 UPDATE
与无关 UPDATE、监听初始化/重连边界及错误通知风暴等场景；不修改既有业务重试规则。
