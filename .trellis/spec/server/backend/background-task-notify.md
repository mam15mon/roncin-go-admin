# 后台任务通知与到期调度

## 1. 适用范围与入口

- `biz.BackgroundTaskScheduler` 定义消费者订阅与最近到期查询；不暴露 pgx、Ent 或任务载荷。
- `data.NewBackgroundTaskScheduler` 为一个服务实例维护一条专用 PostgreSQL
  LISTEN 连接；`server` 的四类 worker 只消费接口并维持原有领取、处理和租约规则。
- 覆盖钉钉通知、对象删除、钉钉审批发件箱及独立审批 inbox、钉钉通讯录同步。
  周汇率提醒仍走原有 30 分钟定时检查。

## 2. 接口签名

```go
type BackgroundTaskScheduler interface {
    Start(context.Context) error
    Stop(context.Context) error
    Subscribe(TaskConsumer) (<-chan struct{}, func())
    NextRun(context.Context, TaskConsumer, bool) (*time.Time, error)
}
```

`NextRun` 返回空指针表示没有待调度行，返回 error 表示查询失败；第三个参数仅对
`TaskConsumerDingTalkApproval` 生效，控制是否合并 inbox。释放订阅函数可重复调用。
运行环境沿用 `DATABASE_SOURCE`，无新增运行配置；两表触发器分别为
`background_tasks_schedule_changed`、`dingtalk_approval_inbox_schedule_changed`。

## 3. 写入、监听与等待契约

- 两张任务表的通知触发器由正式迁移安装。新增任务及实际改变调度字段的更新
  在提交后通知，回滚不通知；不能仅在通用 Enqueue 中手动通知而遗漏直接写表路径。
- 通道固定为 `roncin_task_schedule_changed`，载荷只含 Schema 和固定消费者类别。
  监听方验证类别和 Schema，不包含组织、任务、人员、令牌、错误或连接配置。
- 专用连接不参加业务事务、不占用业务连接池来等待通知；它复用业务驱动配置，
  包括集成测试的 search_path。所有消费者禁用时不建立监听连接。
- 从已打开 pgx 连接复制配置时，必须清除绑定旧连接的 `OnNotification` 回调，
  由新连接重新安装默认通知缓冲。直接复制会使消息进入旧连接，新的
  `WaitForNotification` 返回 `(nil, nil)`，并在读取 Payload 时触发空指针。
- worker 在首次领取前订阅；信号按本地订阅者广播并以容量 1 合并。
  LISTEN 建立后广播扫描信号，晚注册订阅者也扫描，不能依赖 Kratos 启动顺序。
- 断线后按 1 秒起、最多 30 秒的有界退避重连。重建 LISTEN 后重新扫描数据库，
  恢复断线期间积压。取消等待与连接、释放最后订阅必须可以及时关闭。
- 日志记录监听启动、断线、恢复的状态变化；空闲检查不反复写日志，不输出原始连接错误。

### 等待与领取

- 通知只提示重新领取，数据库行仍是任务真相；原有锁、lease token、attempts、
  幂等和最大重试次数保持不变。多实例都可收到信号，但不能获得同一有效租约。
- 成功处理后继续领取直到无任务。空闲时查询最近到期，等待通知、到期时间、
  60 秒恢复检查或取消中的最早事件；禁止恢复原有 2～15 秒空轮询。
- 后台任务 `PENDING/FAILED` 取 `next_run_at`、`RUNNING` 取 `lease_expires_at`；
  inbox `RECEIVED/FAILED` 取 `next_run_at`、`PROCESSING` 取 `processing_expires_at`。
  终态排除，最近到期谓词必须与 Claim 保持一致；callbacks 禁用时不查询 inbox。
- 读取结束与开始等待之间的通知保留在缓冲中，不能清除。领取异常、查询错误或
  已过期却无任务必须有界退避，禁止过去时间导致忙循环。异常等待至少 1 秒，
  门槛内通知合并保留，门槛后可打断更长退避；取消或关闭订阅立即退出。
- 正常连接、空闲消费者的即时任务应在本机提交后 1 秒内开始领取；外部请求和
  已在执行的长任务不承诺完成时延。

## 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 写入事务回滚 | 不交付通知，不改变任务持久状态 |
| 未改变调度字段的 UPDATE | 不通知；业务更新不增加空闲检查 |
| payload 不合法、类别未知或 Schema 不符 | 忽略，不查询该消费者 |
| 全部消费者禁用 | 无任务扫描与监听连接 |
| 最近到期为空 | 最长等待 60 秒恢复检查，可被通知和取消打断 |
| 时间已过但领取为空、领取或查询异常 | 1 秒起至 30 秒有界退避；通知不能绕过最短 1 秒，取消/关闭立即退出 |
| 监听断线 | 记录一次中断，重连后广播一次恢复扫描 |
| Stop 或释放最后订阅 | 取消监听等待和连接，Stop 等待退出或返回调用方截止错误 |

## 5. 正常、基准与错误用例

- 正常：业务事务写入通知任务，提交触发消息，本地全部通知订阅者检查任务；
  成功领取者沿用 token 完成，其他实例继续等待。
- 基准：创建未来执行任务，通知促使重新计算下一次到期，到期不需要第二条通知。
- 错误：在等待前清除缓存通知，或者仅在 Enqueue 后手工发消息，造成竞态或遗漏写入口。

## 6. 验证要求

功能实现后补验证；数据库测试指向专用临时库并使用正式迁移，禁止 Schema.Create。
覆盖事务提交/回滚、直接写入、调度字段变化、类别/Schema 隔离、通知合并、启动竞态、
到期重试/过期租约/inbox、真实连接终止与重连、多实例竞争、禁用和取消。
可控时钟验证 60 秒空闲频率及查询到等待的竞态；交付时补一次真实时间观察，
不能只凭测试 SKIP 或短 sleep 判定成功。临时库验收完成后按精确名称销毁。

## 7. 错误与正确写法

```go
// 错误：领取结束后清除消息会丢掉查询期间刚提交任务的提示。
processNext(ctx)
drain(wake)
wait(ctx)

// 正确：每轮读取前合并旧信号，新信号保留到本轮等待。
drain(wake)
processNext(ctx)
waitForNotificationOrNextDue(ctx)
```
