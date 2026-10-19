# 现有认证与钉钉消息能力核验

核验日期：2026-10-19。仅检查代码和公开文档，未调用实际员工消息发送或读取私密配置。

## 可复用能力

- `server/internal/biz/auth.go:508`：用户名密码登录，先执行账号/IP 维度失败限流，随后验证密码、查询有效工作台并建立会话。手机号路径应复用工作台准入与会话创建规则。
- `server/internal/data/auth.go:27`：现有凭据查询只匹配 username。前端“用户名 / 邮箱”的提示并非现有真实契约。
- `server/internal/data/ent/schema/user.go`：手机号和密码均可为空；钉钉 UnionID、UserID 各有唯一约束，手机号当前无唯一约束。手机号只是可选通讯录快照，不能据此自动合并身份。
- `server/internal/biz/dingtalk_mobile.go`：现成国内手机号归一化和脱敏工具。
- `server/internal/data/dingtalk.go:194`：`SendText` 使用 `/v1.0/robot/oToMessages/batchSend`，`robotCode=clientID`、单个绑定 UserID、`sampleText`。取得 processQueryKey 表示请求受理，不能据此声称用户已收到。
- `server/internal/data/admin_user.go:466`：管理员重置要求启用账号、有管理范围内成员关系和用户名，写入密码同时撤销全部会话并记录审计。自助设置不应绕过待分配/停用控制，也不能调用此接口信任外部提供的目标 ID。

## 历史范围

`10-13-dingtalk-login-memberships/prd.md` 的 D3/R1 约定扫码优先，密码入口显示管理员。本次新需求主动扩展密码登录人群，需同步该规范口径。旧文档足以回答历史范围，未调用对话记忆 CLI。

## 公开文档与验证限制

官方页面：https://open.dingtalk.com/document/orgapp/chatbots-send-one-on-one-chat-messages-in-batches

浏览工具确认该官方页面标题为“批量发送人与机器人会话中机器人消息”，但动态页面未返回接口正文。本轮具体参数以仓库已有实现和测试为事实，不引用第三方教程作为契约依据，不将现有 mock 验证视为真实企业应用联调。

## 设计需要解决

1. 登录凭据输入空间：手机号与本地用户名可能重叠，必须有确定性语义，禁止依次试多个账号密码。
2. 手机号来自通讯录同步，不修改用户 ID、用户名、组织成员关系或角色。缺失与重复时不能猜测目标；号码更新后须使旧号码及旧验证码失效。
3. 服务端持久化发送冷却与验证码尝试状态，支持并发下仅一次有效消费；写密码、撤会话和写审计保持原子性。
4. 不能借公共发送接口泄漏账号存在性或向任意钉钉用户发消息；必要限流沿用当前最薄实现，不建设通用认证平台。
5. 通用通知系统可能保存正文；验证码应直接使用消息适配器，不将明文持久化到通知表或队列。发送结果区分受理、失败和不可确认，设计相应验证码状态。

## 补充核验

`server/internal/security/password/password.go:25` 要求原始密码长度至少 12 字节；`server/internal/biz/admin_user.go:191` 在管理员重置入口进一步检查去首尾空白后的长度至少 12。新接口沿用当前校验规则与 Argon2id 工具，无须新增密码策略系统。手机号登录审计需单独处理当前 username 审计路径，不能把手机号直接写入 `auth.login` 的 details。

## 规划收敛记录

- 用户先回复“好的啊”，确认员工手机号和本地用户名共用账号登录入口；后回复“好的”，确认首次设置与找回共用入口、6 位数字码、5 分钟有效、5 次错误作废、重发使旧码失效以及成功后退出所有设备。
- 已检查 `server/internal/data/dingtalk_directory.go` 的生产写入方：创建时 SetNillableDingtalkMobile，更新时 Set/ClearDingtalkMobile。规范化投影应在该路径同步，来源快照不能丢失；实施前仍需搜索其他实际种子与写入方。
- `LoginRateLimitBucket` 已有敏感 key_hash、窗口和计数及唯一索引，可给发送 IP 使用独立域前缀，不能混用 auth.login 失败次数或把未知号码作为失败登录审计。
- 使用单条当前挑战状态即可满足重发失效与频率门禁；新增实体属于原子状态流转，以行锁、唯一键和当前令牌条件更新，不是多人编辑业务表单，无须暴露 expectedVersion。
- 验证码摘要不能仅 SHA-256(6位码+库中盐)，否则数据库泄漏后可离线穷举；方案使用只在浏览器持有、数据库只存摘要的高熵 challenge_token 参与 code_hash，禁止将其原文记录到日志、URL 或持久通知。
- `make -C server api` 会运行 HTTP 访问规则生成；新增 PUBLIC RPC 必须通过生成流程更新相应元数据，不能只在手写路由放行。生成顺序与实际脚本见 implement.md。
- 规划产物 ready gate 为：prd/design/implement 完整、两个 jsonl 引用真实规范/研究文件、用户对最新完整方案另行批准。本轮只完成审阅稿，不提前启动任务或实际发送消息。
