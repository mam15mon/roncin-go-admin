# Research: 钉钉通知卡片代码地图与最小实现边界

- Query: 梳理八类通知的构造、发送、本人测试、权限与真实业务入口，提出一次性替换 Markdown 的最薄边界。
- Scope: internal
- Date: 2026-10-19

## Findings

### 约束与相关规范

已读根目录 `AGENTS.md`、`server/AGENTS.md`、`.trellis/workflow.md`、领域 glossary/architecture-map、server/backend 与 web/frontend 索引，重点专项：

- `.trellis/spec/server/backend/dingtalk-notification-markdown.md`：现有七类业务 + 即时密码验证码、敏感信息与工作任务边界；本次须整体更新为卡片契约，不保留自动 Markdown 回退。
- `.trellis/spec/server/backend/dingtalk-notification-test.md`：系统工作台有效 administrator only、本人绑定重读、八类测试、只预览无网络、示例不伪造订单链接。
- `.trellis/spec/server/backend/mobile-login-password-reset.md`：PENDING 提交后即时发送、受理后绑定仍有效才 READY、失败不重试、验证码不进入存储/任务/URL/日志。
- `.trellis/spec/server/backend/background-task-notify.md`：领取/租约/重试/等待语义不因消息协议变化而重做。

禁止产品代码修改、Git 操作，仅本研究文件落盘；协议研究由另一研究任务负责，本文不决定钉钉卡片 API。

### 当前完整消息路径

1. `server/internal/biz/notification.go:39` 的 `NotificationDelivery` 是 Worker 取数的最小快照，包含资源类型/ID、编号及摘要参数。
2. `server/internal/data/notification.go:113` 的 `FindByTaskID` 只读通知明细和当前收件人绑定，`data.client(ctx)` 取得客户端，不持有旧绑定快照。
3. `server/internal/biz/notification.go:101` 的 `ProcessNext` 领取 → 验证 lease token → 查明细 → `renderNotification` → sender → `Complete`；失败 `:126` 以原 lease token 走 `Fail`，退避为 `30s * 2^min(attempts,5)`。
4. `server/internal/biz/notification.go:57` 现为 `DingTalkMarkdownMessage{Title,Text,OrderEntryPath}`，sender 接口 `:63`，七类正式模板从 `:141` 起集中渲染。
5. `server/internal/data/dingtalk.go:233` 是最终发送适配器；`:270` 的 `PrepareMarkdown` 纯内容准备用于预览复用；`:277` 只将固定业务路径拼成回调配置 scheme/host 对应的本站绝对 URL。访问令牌只在 Send 中取得。
6. `server/internal/server/notification_worker.go:13` 保持 30 秒租约 / 15 秒处理 timeout；`:42` 使用现有 scheduler 等待，不应因卡片改造变更 Worker 或后台任务表。

### 八类通知与资源、入口映射

| 类型 | 实际构造 / 渲染定位 | 当前可靠资源 | 最薄卡片目标建议 |
| --- | --- | --- | --- |
| `ORDER_PERSONNEL_ASSIGNED` | `biz/order_personnel.go:89`、`biz/order_usecase.go:398` 创建意图；`data/notification.go:25` 入队；`biz/notification.go:149` 渲染 | `ResourceType=ORDER`，`ResourceID=orderID`，编号及角色 | 正式可定位订单，但快照没有业务类型。不能无条件把任意 ORDER 都当 SE；可用已注册通用订单列表 `/orders`（会跳 SE）需注明列表入口，若保证构造只为 SE 才用 `/orders/sea-export/{id}`。示例全部无按钮。 |
| `USER_AUTHORIZED` | `biz/admin_user.go:146`、`:156`；`data/notification.go:60`；`biz/notification.go:161` | `USER` + 员工 UUID；显示名称发送时读取 | `/welcome`，文案“进入工作台”；不拼员工管理详情。 |
| `EXCHANGE_RATE_WEEKLY_REMINDER` | `data/exchange_rate_reminder.go:120`、`:143`；`biz/notification.go:167` | `ORGANIZATION` + 公司 UUID，组织名称、本币与周范围 | 已注册 `/finance/exchange-rates`；收件规则 `data/exchange_rate_reminder.go:82` 包含有效成员/角色及汇率创建或更新权限。点击时仍由现有工作台和页面权限决定，不能自动扩权/切公司。 |
| `FEE_REVIEW_PENDING` | `data/order_fee_review_notifications.go:16`、`:21`、`:48`；`biz/notification.go:175` | `ORDER` + OrderID；Parameter 末尾由服务端附固定费用入口 | 复用被严格核对的 `/orders/sea-export/{ResourceID}/fees`；不能从任意动态摘要抽 URL。 |
| `FEE_REVIEW_RESULT` | `data/order_fee_review_notifications.go:16`、`:55`、`:59`；`biz/notification.go:177` | 同上；结果由“actor：已批准/已退回”文本组合 | 同上。颜色别用 `strings.Contains(parameter,"退回")` 判断，否则动态姓名可误导。可只给固定“审核结果”状态色；需要区分成功/驳回时，在去掉已验证路径后仅识别正式结果后缀，未知结果显式校验，勿猜。 |
| `FEE_SUPPLEMENT_APPROVAL_PENDING` | `data/order_fee_supplement_notifications.go:22`、`:54`；`biz/notification.go:179` | `FEE_SUPPLEMENT_REQUEST` + RequestID；订单号和摘要。**ResourceID不是OrderID** | 本期最薄可无 action，文案指引订单费用页。没有已注册的 requestID 独立页面；不可伪造 `/orders/.../{requestID}/fees`。不应为了视觉新增详情路由或额外数据库 lookup。若确需将来精确入口，`ent/schema/order_fee_supplement_request.go:42` 已有不可变 OrderID，但须显式 enrichment 并确认 BusinessType。 |
| `COMMISSION_DECREASE_SUGGESTED` | `data/order_fee_supplement_notifications.go:74`、`:110`；`biz/notification.go:187` | `COMMISSION_ADJUSTMENT` + AdjustmentID；收件人=EmployeeID | 现有本人专属 `/commission-adjustments/{id}/my-supplement-source` 可可靠直达，不能换 `/finance/commissions`（员工可能无组织级财务读权）。 |
| `PASSWORD_RESET_CODE` | `biz/password_reset.go:72`、`:103`、`:140` | 即时6位验证码，无通用通知、无任务资源 | 不显示按钮；重点验证码、5分钟有效、保密提示。测试固定012345明确无效，不创建挑战。 |

`web/config/routes.ts:140` 注册订单详情，`:147` 费用页，`:205` 财务提成管理，`:211` 汇率管理；`:229` 本人冲减页明确脱离 finance 路由权限，`:233` 仅要求已登录。

`web/src/pages/finance/commissions/my-supplement-source.tsx:12` 解释本人最小详情边界，`:33` 调用本人源接口；`server/internal/data/finance_commission_adjustment.go:103`明确 ID + employee_id + 来源约束。应保留服务端权威权限。

费用补录 UI 现状：`web/src/pages/orders/fees.tsx:568` / `:764` 将 `FeeSupplementSection` 放在具体订单费用页；`biz/order_fee_supplement_usecase.go:831` 的列表按 orderID 读取并逐行做费用权限/发起人/实时审批人校验。没有 standalone RequestID 路由，也没有全公司补录列表入口。

### 权限与测试必须守住的边界

- `server/internal/biz/auth.go:829` 是通知两项权限的工作台投影：必须 `principalIsSystemWorkspace`，且有 `RoleCode==administrator` 与 active DataScope。中文名称、bootstrap 标志、公司 administrator 或普通角色直接赋权限均不可放行。
- `internal/access/manifest.go:20` 权限常量、`:192` Requires、`:478` 公司管理员排除规则；本次无新增权限需求，不应修改权限语义。
- `server/api/admin/v1/admin.proto:165` 起三个接口带通知权限；数据结构从 `:692` 起。预览 `:720` 现为 type/title/markdown/order_url。
- `server/internal/service/admin_notification.go:27` 负责可选 orderId UUID 校验，`:37`/`:52` 分别转调 Preview / Send；`:50` 是 DTO 映射位置。
- `server/internal/biz/notification_test_usecase.go:82` Catalog 从本人查真实绑定，`:101` 读取当前工作台可选订单能力。**系统管理员当前永远没有公司订单读取范围，canSelectOrder=false**。
- `notification_test_usecase.go:116` 构造固定 `00000000-0000-4000-8000-000000000001` 假资源 UUID。新卡片不能因 UUID 非空就生成业务按钮！建议测试标记处理统一 `Action=nil`，包括 welcome/汇率这类通用入口，八类示例保持一致无 action。
- `:157` build 做权限/用户启停/可选订单授权；`:190` 验证码共用正式 renderer，但`:200` 添加无效提示。`:203` 离线 preparer 不能取 token 或发消息。
- `:213` Send 只走显式调用，`:224` 再次取当前账号，`:231` 绑定判定，`:234` 当前会话员工 ID 作为唯一收件人，绝不接受前端传收件人；失败返回“先核对客户端再决定重试”，不写业务表或重试队列。

### 密码验证码相邻语义

`biz/password_reset.go:96` 以 `%06d` 保留前导零；`:98` Reserve 成功后才即时 Send；`:106` 独立有界 finish context；`:107` 仅 sendErr==nil 才请求激活；finishErr、sendErr 或未 ready 均 `ErrPasswordResetSend`。本期只替换 render 返回类型及 Sender 方法，PENDING/READY/冷却/事务/密码哈希等不动。真实验证码不能纳入卡片标题、action URL、预览 API 或通用通知持久化；示例验证码无需资源来源查询。

### 最小卡片模型与预览契约建议

认可主会话提出的纯 Go 强类型内容模型：`tone/status/title/summary/facts[]/note/action?`，facts 为固定 label + 动态 value，action 为固定 label 与受控本站路径（准备后变绝对 URL）。模板 renderer 只构造该模型，data 专职编码实际钉钉 JSON；service 只做 DO ↔ DTO。动态图文不解释成 HTML，避免保留第二套 Markdown 文本作为隐式回退。

- `DingTalkMarkdownMessage` / `SendMarkdown` / `PrepareMarkdown` 整体替换为卡片名称与 typed model，正式 Worker、密码即时、Admin 预览同步；不保留旧字段或双协议。
- `NotificationTestPreview` / proto 改为类型 + typedCard + 实际卡片 JSON 源码，服务端输出与实际发送使用同一编码器；本地网页用 typedCard 渲染，源码作为核对，不反向用 HTML 注入实现预览。
- Action 的路径只从固定 route pattern + UUID 或固定页面常量构造，动态姓名/订单号/摘要永远不能成为 URL；data 延续配置回调 origin 拼接规则。
- 测试统一添加醒目测试状态或 note，再清除 Action；真实业务按钮在模板正常路径生成，不能由网页猜 URL，也不能让 Send 二次追加链接。
- 现有费用入口在 Parameter 尾部是已存在契约，当前 `biz/notification.go:226` 仅匹配 ResourceID 生成的固定后缀；换卡片可保留这一校验，不建设新的历史兼容。如果不顺便改业务入队 Parameter 契约，不需要迁移或历史回放。
- 测试 orderId 当前永远不可用，本期**建议保留现有显式拒绝与权限测试即可**，只把预览结构换卡片。删除 selector / orderId API 会扩大契约面、测试改动和 review 成本；不应该为了视觉开放跨公司订单读取。后续若清理可单独删死能力，但不得新增“系统管理员也能读所有订单”的旁路。
- `react-markdown` 当前唯一源码使用是 `web/src/pages/admin/notification-test.tsx:6`（web/package.json:50）。若卡片改版彻底不再需要，可正常 pnpm 移除依赖及锁文件，不手改生成物。

### 建议实施文件边界

后端实现者：

- `server/internal/biz/notification.go`：领域消息、sender 接口、七类 renderer、ProcessNext 调用点。
- `server/internal/biz/password_reset.go`：验证码 renderer 和即时发送方法名，原挑战逻辑不动。
- `server/internal/biz/notification_test_usecase.go`：预览模型、共享渲染、测试清除 action、离线 prepare。
- `server/internal/data/dingtalk.go`：共享离线卡片准备/编码与正式发送适配；可单独拆 `dingtalk_notification_card.go` 降低认证文件噪声。
- `server/api/admin/v1/admin.proto`、`server/internal/service/admin_notification.go` 及必要生成物：typed card DTO、源码字段。
- 全仓 24 处左右 SendMarkdown / DingTalkMarkdownPreparer 使用（包含 stub、transport fake）需同步，不只改正式 sender。

前端实现者：

- `web/src/pages/admin/notification-test.tsx`：保持 React Query 与显式 mutation，替换 Markdown 显示为 typedCard，状态色与密度适配。
- 页面私有卡片展示文件可放 `pages/admin/` 相邻；不为了一个页面提取 features/global 平台。
- `web/tests/notification-tests/support.tsx` mock payload 与八类 page tests，生成客户端等待后端生成，不手修 typings。
- 路由 / access / 设置入口无需改变。

### 必要验证定位

- `server/internal/biz/notification_test.go:38` Worker 成功完成、`:103` 失败退避、`:134` 七类模板（替换为卡片字段断言）；保留 lease token 断言。
- `server/internal/biz/notification_test_usecase_test.go:79` 八类预览/本人发送、`:114` 系统工作台拒真实订单、`:164` 权限/禁用/缺绑定拒绝，`:209` 系统范围。
- `server/internal/biz/auth_notification_permissions_test.go` 与 `server/internal/server/admin_notification_transport_test.go:142` 保留 system administrator only + auth/me 一致性；真实 HTTP routes 在 `:60`，校验权限在 `:86`。
- `server/internal/data/dingtalk_test.go:84` 实际出站 HTTP 参数，`:143` 空值/禁用、`:170` 上游失败；卡片改造需断言真实消息 JSON 与 action，不能仅测试网页。
- `server/internal/data/dingtalk_preview_test.go:15` 离线 prepare 零 token，预览与发送相同；`:54` 禁用集成仍可准备内容。
- `server/internal/biz/password_reset_test.go:49` 绑定消息与随机挑战、`:71` 未知/发送失败、`:98` 边界、`:117` 前导零；仅替换消息模型断言。
- `server/internal/data/password_reset_integration_test.go:353` 即时发送确在事务之外、`:243` 发送失败未激活、`:115` 一次消费，作为相邻风险回归；正式 SQL 迁移测试条件不具备时明确 SKIP。
- `web/src/pages/admin/notification-test.test.tsx` 八类只预览/迟到响应/无权限；`notification-test.send.test.tsx` 显式本人发送、在途禁重、受理/未确认失败；`notification-test.access.test.tsx`、`.navigation.test.ts` 权限/菜单保持。
- 新增有效的正式 action 映射、恶意动态字符串、测试一律无 action、源码与预览内容相同的断言；无需为每个静态色值机械补测试。

## External References

本文未研究外部协议；当前仓库钉钉 renderer/接口以本次源码为准，最终彩色卡片 API、msgKey 与 JSON 格式应使用另一研究产物的官方文档证据。

## Caveats / Not Found

- 未发现费用补录 RequestID 独立网页或全公司补录列表，不能把现有 API endpoint 当成网页按钮。
- 订单分配通知有 OrderID，没有 BusinessType 快照。`biz/order_usecase.go:344` 当前订单创建仅允许 SE + EXPORT，因此本期正常新建业务均为 SE；但 `data/order_personnel.go:59` 赋员读取任意同组织订单，未显式检查 BusinessType。若卡片精确直达 SE，最小可靠方案是在现有读取处显式传业务入口或在快照读取时确认类型，不能仅凭 ORDER 名称猜。无新增查询方案则使用现有列表并明确标签“查看订单列表”。
- 当前费用审核 Parameter 256 字节截断可能截掉尾部入口；既有 renderer 因严格后缀匹配而不给按钮。视觉改造不应对不完整入口做猜测补正。
- 本人冲减页适用 LOCKED_FEE_SUPPLEMENT 来源，现有模板构造是这一路径；将来若同一模板扩大其他调整来源，需重新确认可落地来源。
- 本地预览只能验证内容组织与请求一致，真实钉钉端彩色、按钮和移动布局必须保留实际客户端验收边界，不宣称 mock 已送达。
