# 钉钉通知预览与自发测试

## 1. 适用范围

独立Admin页面 `/admin/notification-test` 用于验收8类通知的紧凑Markdown与单按钮ActionCard。它不在系统管理多页签中心内，不承担模板编辑、存储、业务审批或收件规则管理。

## 2. 接口与权限

```text
GET  /api/v1/admin/notification-tests/types    system.notification.preview
GET  /api/v1/admin/notification-tests/preview  system.notification.preview
POST /api/v1/admin/notification-tests/send     system.notification.test_send
```

预览权限依赖 `system.platform.access`，发送依赖预览；两条权限仅在系统管理工作台（kind=system）且当前有效RoleGrants包含RoleCode=administrator时可用。系统管理员与公司管理员共用角色代码，不能只看code，也不能根据可变的中文名称或IsBootstrapAdmin标记放行。Principal统一计算有效权限，auth/me、HTTP鉴权、biz校验与前端access消费同一结果；前端不复制另一套角色判定。Manifest将两权限归系统管理，标准同步不向公司管理员挂载。没有Schema变更，权限目录仍由标准迁移流程同步。

## 3. 数据契约

- 目录data：`types[{notificationType,label,supportsOrder}]`、`recipientName`、`recipientBound`、`integrationEnabled`、`canSend`、`sendUnavailableReason`、`canSelectOrder`、`orderSelectionUnavailableReason`。不输出钉钉员工ID。
- 预览query/发送body仅接受 `notificationType`、可选 `orderId`；字段使用camelCase，UUID及类型由服务端校验。预览data为 `notificationType/title/markdown/actionTitle/actionUrl`（不再输出orderUrl）；发送data为 `accepted`，文案为「钉钉已受理，请在客户端确认」。
- 8类代码：`ORDER_PERSONNEL_ASSIGNED`、`USER_AUTHORIZED`、`EXCHANGE_RATE_WEEKLY_REMINDER`、`FEE_REVIEW_PENDING`、`FEE_REVIEW_RESULT`、`FEE_SUPPLEMENT_APPROVAL_PENDING`、`COMMISSION_DECREASE_SUGGESTED`、`PASSWORD_RESET_CODE`。
- 7类业务内容复用 `renderNotification`，验证码复用 `renderPasswordResetCode`，只使用无效示例且不创建认证挑战。全部标题/正文明确标识测试，不复制第二套模板。
- 收件人每次从当前会话UserID读取真实绑定，不信任客户端。当前页面仅系统管理员可用，系统工作台无经营读取资格，因此canSelectOrder=false，使用示例且不生成虚假业务链接。7类业务测试动作统一为真实存在的/welcome「打开系统首页」，正文明确不指向真实订单，用于验收ActionCard按钮；验证码无动作。既有可选orderId仍经过订单读取校验，系统管理员传入也被拒绝，不能为测试跨公司读订单。
- 预览与发送复用data中的无网络消息准备，含站点回调scheme/host及受控固定动作路径；不得在前端拼站点或令data发送时二次追加正文链接。预览不申请token、不发送。
- 测试同步调用现有机器人发送通道，不进入通知表/后台任务，不自动重试，不写审批、费用、提成或密码。运行日志不记完整通知载荷。

## 4. 校验与错误矩阵

| 情况 | 期望 |
| --- | --- |
| 无有效会话 | 401，上游零发送 |
| 系统普通角色持有通知权限、公司administrator、仅中文名为系统管理员、仅bootstrap标记 | 不投影两条通知权限，菜单/路由不可见，三个接口403且上游零发送 |
| 系统工作台有效administrator角色且持有所需权限 | 允许示例预览和显式本人发送 |
| 未知通知类型/非法UUID | 参数错误，上游零发送 |
| 非费用类型传orderId | 拒绝，不悄悄忽略不支持参数 |
| 集成禁用/本人未绑定 | 可看示例内容，不能发送，提供明确原因 |
| 未选订单 | 纯示例，业务测试仅真实首页演示按钮，不能标为查看订单；验证码无动作 |
| 越权订单/当前工作台无业务读取资格 | 拒绝真实订单模式，不自动扩大范围或切工作台 |
| 回调站点配置不合法 | 400 `DINGTALK_NOTIFICATION_SITE_INVALID`，不生成伪造链接 |
| 上游失败/未获得受理确认 | 502 `NOTIFICATION_TEST_SEND_FAILED`；先核对客户端，再决定重试，不断言未发送 |
| 接口受理 | 仅声明已受理，由用户客户端核对实际收件 |

## 5. 正常、基准与错误场景

- 正常：系统管理员进入系统管理工作台，选费用待审核预览示例，显式发送给自己，在钉钉核对排版。
- 边界：公司管理员和获授相同权限的系统普通角色均没有入口，直调接口也403；系统管理员不能借测试入口读取公司订单。
- 基准：普通业务通知Worker与密码重置的正式模板、资格、租约/重试/挑战激活规则保持原流程。
- 错误：把示例验证码注册成有效挑战；通过输入钉钉ID指定收件人；预览触发真实发送；系统工作台按客户端公司ID绕过访问边界。

## 6. 必要验证

- biz：两条权限的HasPermission/PermissionKeys/PermissionCapabilities一致，系统有效administrator允许、普通角色/公司administrator/同名/仅bootstrap拒绝；8类实际模板复用、测试标识/无效示例、订单范围、本人绑定重读、无业务写入。
- HTTP：实际路由与权限/必填/camelCase，未授权上游零调用；未知/越权输入明确失败。
- data：预览准备不取token，最终正文/动作与发送JSON一致，动作准备幂等且正文不重复追加链接，原正式发送与验证码相邻流程回归。
- web：独立路由/菜单/标签标题、加载不发、受控Markdown与精确允许路由按钮、禁止HTML/图片/任意正文链接、源码核对、只读/未绑定/禁用状态、发送失败、网络超时/缺accepted不伪成功与在途禁重、类型/订单切换迟到隔离；helper放web/tests。
- 验收：定向测试后全栈门禁与依赖相关生产构建；实际消息由用户显式发送验收，不把HTTP替身结果表述成真实送达。

## 7. 错误与正确示例

```text
错误：POST body={recipientUserId:"other",title:"...",markdown:"..."}
正确：POST body={notificationType:"FEE_REVIEW_PENDING"}
      系统工作台有效administrator与通知权限同时满足；收件人从当前会话用户真实绑定获取，标题/正文由正式模板渲染并加测试标识，业务动作替换为明确首页演示入口，不生成假订单链接。

错误：页面打开即发送8条示例，返回成功便显示「已送达」。
正确：打开仅预览，逐条显式点击，显示「钉钉已受理，请在客户端确认」。
```
