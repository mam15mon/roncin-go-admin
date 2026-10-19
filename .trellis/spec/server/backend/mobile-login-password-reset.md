# 手机号账号登录与钉钉密码找回契约

## 1. 适用范围

适用于 AuthService 密码登录、钉钉员工首次设置密码和自助找回。与扫码共用 User、Membership、工作台准入和 Session；不创建/合并身份，不更改权限，不通过验证码直接登录。

## 2. 签名

```text
POST /api/v1/auth/login
  username（钉钉手机号或本地用户名）、password、可选 organizationId

POST /api/v1/auth/password-reset/code
  mobile
  data: challengeToken, expiresAt, resendAvailableAt, serverTime

POST /api/v1/auth/password-reset/confirm
  challengeToken, code, newPassword
  标准成功信封，无 data，不签发会话

RPC: RequestPasswordResetCode / ConfirmPasswordReset（PUBLIC）
biz: PasswordResetUsecase / PasswordResetRepo（password_reset.go）
DB: users.dingtalk_mobile_normalized、password_reset_challenges
```

时间均为 RFC3339，前端只展示相对倒计时。接口使用 no-store；成功重置清除本站会话 Cookie，服务端撤销状态仍是权威。

## 3. 契约

### 身份与资料

- username 输入联合解析本地用户名和规范化钉钉手机号；合法手机号同时将规范化号码作为本地用户名候选，避免 +86 或空白格式绕过数字用户名冲突。按用户 ID 去重后再验证密码。不同用户冲突必须拒绝，不以启停过滤猜测目标，不依次验证多人的密码。
- 钉钉员工无需 username 也能通过手机号登录；初始化管理员仍用本地用户名。CurrentUser.username 不为手机号登录临时填充完整号码。
- 保留 dingtalk_mobile 来源快照，dingtalk_mobile_normalized 只作敏感检索投影，使用非唯一索引。所有实际写入方同步维护；字段无授权、缺失或无效时清空投影。正式迁移回填需与 NormalizeDingTalkMobile/ValidDingTalkMobile 的 Unicode 空白、前缀及连字符规则等价。
- 重复号码不阻断通讯录同步和扫码，手机凭据路径必须拒绝。账号资料生效以同步提交时点为准，不自动推断未同步的钉钉资料变化。

### 发送与挑战

- 每规范化手机号至少间隔 60 秒，数据库取得挑战锁后以当前 UTC 时刻执行门禁并建立新一代冷却/有效期，不能用请求进入仓储前的旧时刻；来源 IP 独立发送计数每 10 分钟最多 20 次，使用 LoginRateLimitBucket 独立域前缀，不污染登录失败计数。
- 未知、重复、停用、待分配、未绑定钉钉、无有效工作台目标返回同形状不可用挑战，并执行相同冷却；不公开目标状态或收件人。合法目标只发送到已绑定企业 UserID，调用方不能传收件人 ID。
- challengeToken 使用至少 32 字节密码学随机值，只在发起页面内存保留，库中仅 token_hash。6 位数字验证码保留前导零，5 分钟有效；code_hash 使用域分隔 SHA-256(token + code)，不能仅使用库中盐加 6 位码。
- 每手机号只保留一条当前代，重发覆盖旧代。PENDING 事务提交后在事务外调用既有企业机器人 SendText；受理且绑定仍有效才条件转为 READY。失败、超时或崩溃遗留的 PENDING 不能重置，冷却仍保留；无后台自动重发。
- 机器人返回受理键仅表示受理，不表示本人已收到；真实企业收件验证与 mock 必须分别记录。
- AUTH_PASSWORD_RESET_COOLDOWN 的错误信封仅允许输出 metadata.serverTime/resendAvailableAt；不能通用透传全部领域错误 metadata，避免泄漏内部值。

### 消费与会话

- 发送激活与消费取得所需行锁后均须以当前 UTC 时刻判定过期，不能使用锁等待前的时刻。仅当前 READY、未过期、未使用且 failed_attempts < 5 的挑战有效；错误次数事务必须提交，第 5 次作废。返回给用例的验证码失败结果不能导致计数回滚。
- 最终事务复核当前绑定、号码唯一性、用户启用/非待分配及有效工作台；验证码申请后被停用或改手机号/钉钉身份不能继续重置。
- 按用户 → 组织/成员 → 挑战 → 会话顺序加锁，涉及多行按主键排序。首次状态创建由唯一键与原子冲突处理兜底；不能在已中止事务中吞约束错误继续写。
- 新密码沿用现有至少 12 位校验与 Argon2id 工具；公共未知/无效挑战不应在资格校验前触发昂贵散列。
- 密码写入、挑战消费、全部未撤销会话撤销与 auth.password.reset 审计同事务。并发正确提交只有一次成功，任一步失败整体回滚。
- 通讯录手机号投影或绑定变化在同事务主动废弃该用户未完成挑战，避免改号后改回使旧验证码复活。管理员重置同时废弃该用户找回挑战；密码登录最终建会话必须核对最初验证的哈希仍是当前值，防止重置后旧密码竞态新建会话。
- 验证码、密码、token 原文不进通用通知表、后台任务、URL、日志或审计。失败登录审计只存域分隔 account_hash，不存原始输入；手机号等价格式共用失败限流桶，成功审计不得含完整手机号。

## 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 登录账号命中不同用户 | 409 AUTH_ACCOUNT_CONFLICT，建议扫码或联系管理员 |
| 手机号格式无效 | 400 AUTH_PASSWORD_RESET_MOBILE |
| 钉钉未启用 | 503 AUTH_DINGTALK_DISABLED |
| 手机号或来源 IP 冷却 | 429 AUTH_PASSWORD_RESET_COOLDOWN，提供服务器时间与可重试时刻 |
| 未知或不合格手机号 | 同形状不可使用挑战，不发送，不公开原因 |
| 发送失败、超时、受理后不能激活当前代 | 503 AUTH_PASSWORD_RESET_SEND_FAILED，不声称送达 |
| 错误、失效、用过、旧代或绑定不符的验证码 | 400 AUTH_PASSWORD_RESET_INVALID |
| 新密码不符合现有规则 | 400 AUTH_PASSWORD_RESET_PASSWORD |
| 持久化或审计失败 | 整体回滚，不返回业务成功 |

## 5. 正常、基准与错误场景

- 正常：已开通但无 username/password 的员工用手机号申请，在本人钉钉拿码后设置；返回账号登录，输入手机号与新密码，进入原有工作台。
- 基准：本地管理员用户名登录、钉钉扫码及多工作台选择均正常；只有手机号缺失时仍可扫码，不自动绑定本地身份。
- 错误：重复手机号任取一人、把手机号写成 username、验证码正确即自动开通、事务内发送、消息失败仍 READY、只清 Cookie 不撤其他设备会话。

## 6. 必要验证

- 手机号归一化、同用户双命中、不同用户冲突、待分配/停用/无工作台、无 username 首次设置；同一手机号存在停用副本也不能靠启停过滤消歧。
- 正式迁移回填 + Ent CHECK/索引元数据 + Unicode 空白/国内前缀等价；保留用户、组织、角色和密码。
- 独立 PostgreSQL：并发发送一次、冷却重启保留、错误计数持久化与第 5 次失效、并发消费一次、重发/过期/绑定变化、发送未激活、管理员重置废弃挑战。
- 独立 PostgreSQL：全部会话失效、旧密码凭据竞争不能新建会话、审计故障密码/挑战/撤销整体回滚。
- HTTP：公开规则、no-store、Cookie 清理、冷却 metadata 白名单和敏感错误不泄漏。
- 前端：实际账号/密码载荷、首次/找回、倒计时及 429、手机号变化迟到响应、关闭清理、成功退出与扫码相邻路径。
- 验收区分真实 PASS/FAIL/SKIP，普通 go test 的 Skip 不能证明真实锁/迁移已经验证。

## 7. 错误与正确示例

```text
错误：按手机号查到第一条启用用户就发码，输入验证码后直接改密码。
正确：联合匹配并排除歧义，绑定固定 UserID + 钉钉身份；最终事务复核状态和当前手机号。

错误：验证码错误时在 WithTx 回调里返回领域错误，失败次数随事务回滚。
正确：事务提交错误次数；在事务外将未消费结果转换为验证码无效。

错误：只存 SHA-256(6位码+库中盐)，把验证码全文交给持久化通知系统。
正确：只在浏览器持有的高熵挑战 token 参与 code_hash；正文直接交给已有消息适配器。
```
