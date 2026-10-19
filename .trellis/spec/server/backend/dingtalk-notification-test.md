# 钉钉通知预览与自发测试

## 1. 适用范围

独立Admin页面 `/admin/notification-test` 用于验收8类既有Markdown模板。它不在系统管理多页签中心内，不承担模板编辑、存储、业务审批或收件规则管理。

## 2. 接口与权限

```text
GET  /api/v1/admin/notification-tests/types    system.notification.preview
GET  /api/v1/admin/notification-tests/preview  system.notification.preview
POST /api/v1/admin/notification-tests/send     system.notification.test_send
```

预览权限依赖 `system.platform.access`，发送依赖预览；权限清单与前端权限键均按生成流程同步，不以管理员角色名硬编码。没有Schema变更，权限目录仍由标准迁移流程同步。

## 3. 数据契约

- 目录data：`types[{notificationType,label,supportsOrder}]`、`recipientName`、`recipientBound`、`integrationEnabled`、`canSend`、`sendUnavailableReason`、`canSelectOrder`、`orderSelectionUnavailableReason`。不输出钉钉员工ID。
- 预览query/发送body仅接受 `notificationType`、可选 `orderId`；字段使用camelCase，UUID及类型由服务端校验。预览data为 `notificationType/title/markdown/orderUrl`；发送data为 `accepted`，文案为「钉钉已受理，请在客户端确认」。
- 8类代码：`ORDER_PERSONNEL_ASSIGNED`、`USER_AUTHORIZED`、`EXCHANGE_RATE_WEEKLY_REMINDER`、`FEE_REVIEW_PENDING`、`FEE_REVIEW_RESULT`、`FEE_SUPPLEMENT_APPROVAL_PENDING`、`COMMISSION_DECREASE_SUGGESTED`、`PASSWORD_RESET_CODE`。
- 7类业务内容复用 `renderNotification`，验证码复用 `renderPasswordResetCode`，只使用无效示例且不创建认证挑战。全部标题/正文明确标识测试，不复制第二套模板。
- 收件人每次从当前会话UserID读取真实绑定，不信任客户端。费用真实订单只取本人现有读取范围内的编号与固定费用路径，其余均示例；无订单不生成虚假可点击链接。
- 预览与发送复用data中的无网络消息准备，含站点回调scheme/host及固定费用路径；不得在前端拼站点或令data发送时二次追加链接。预览不申请token、不发送。
- 测试同步调用现有机器人发送通道，不进入通知表/后台任务，不自动重试，不写审批、费用、提成或密码。运行日志不记完整通知载荷。

## 4. 校验与错误矩阵

| 情况 | 期望 |
| --- | --- |
| 无有效会话/无相应权限 | 在发送前拒绝，后端不可仅依赖前端隐藏 |
| 未知通知类型/非法UUID | 参数错误，上游零发送 |
| 非费用类型传orderId | 拒绝，不悄悄忽略不支持参数 |
| 集成禁用/本人未绑定 | 可看示例内容，不能发送，提供明确原因 |
| 未选订单 | 纯示例，无可点击假订单链接 |
| 越权订单/当前工作台无业务读取资格 | 拒绝真实订单模式，不自动扩大范围或切工作台 |
| 回调站点配置不合法 | 400 `DINGTALK_NOTIFICATION_SITE_INVALID`，不生成伪造链接 |
| 上游失败/未获得受理确认 | 502 `NOTIFICATION_TEST_SEND_FAILED`；先核对客户端，再决定重试，不断言未发送 |
| 接口受理 | 仅声明已受理，由用户客户端核对实际收件 |

## 5. 正常、基准与错误场景

- 正常：管理员选费用待审核，默认示例预览；有公司读取权限时选真实订单，点击发送给我，在钉钉检查排版/链接/免登。
- 基准：普通业务通知Worker与密码重置的正式模板、资格、租约/重试/挑战激活规则保持原流程。
- 错误：把示例验证码注册成有效挑战；通过输入钉钉ID指定收件人；预览触发真实发送；系统工作台按客户端公司ID绕过访问边界。

## 6. 必要验证

- biz：8类实际模板复用、测试标识/无效示例、可选订单范围、本人绑定重读、无业务写入。
- HTTP：实际路由与权限/必填/camelCase，未授权上游零调用；未知/越权输入明确失败。
- data：预览准备不取token，最终Markdown与发送JSON一致，费用绝对链接只追加一次，原正式发送与验证码相邻流程回归。
- web：独立路由/菜单/标签标题、加载不发、受控Markdown、只读/未绑定/禁用状态、发送失败、网络超时/缺accepted不伪成功与在途禁重、类型/订单切换迟到隔离；helper放web/tests。
- 验收：定向测试后全栈门禁与依赖相关生产构建；实际消息由用户显式发送验收，不把HTTP替身结果表述成真实送达。

## 7. 错误与正确示例

```text
错误：POST body={recipientUserId:"other",title:"...",markdown:"..."}
正确：POST body={notificationType:"FEE_REVIEW_PENDING",orderId:"本人可读订单UUID"}
      收件人从当前会话用户真实绑定获取，标题/正文由正式模板渲染并加测试标识。

错误：页面打开即发送8条示例，返回成功便显示「已送达」。
正确：打开仅预览，逐条显式点击，显示「钉钉已受理，请在客户端确认」。
```
