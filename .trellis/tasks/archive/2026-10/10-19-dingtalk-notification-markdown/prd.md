# 所有钉钉通知统一 Markdown 排版

## 用户目标

用户截图显示账号授权和验证码均为难读纯文本，要求改成 Markdown 格式化输出；已同意建立任务，并明确补充“不光这两类，所有钉钉通知都改”。

## 当前证据

- biz/notification.go DingTalkNotificationSender.SendText 统一发送；data/dingtalk.go:194使用企业机器人单聊batchSend、msgKey=sampleText、msgParam={content}。
- 现行7种业务模板：订单人员分配、账号授权、周汇率提醒、费用待审核、费用审核结果、费用补录待审批、提成冲减。验证码在biz/password_reset.go独立即时发送，合计8类。
- Schema中还保留3个历史注册/邀请枚举，但当前renderNotification没有支持、没有新发送入口。任务覆盖所有实际发送路径，不复活已退役流程。
- 官方DingTalk-Real-AI/dingtalk-channel-sdk-python SPEC.zh-CN.md明确企业机器人sampleMarkdown载荷title/text与msgParam JSON字符串；官方open.dingtalk.com/document/orgapp/robot-message-types-and-data-format为动态页，抓取无正文。
- 费用待审核与审核结果的Parameter实际附带海运订单费用页入口，入队证据为`server/internal/data/order_fee_review_notifications.go:48`、`:64`、`:69`；转换后保留为「查看订单费用」可读链接，站点沿用现有登录回调来源。

## 需求与关键决策（已批准）

统一改用Markdown发送，标题与正文分离：标题不含验证码等秘密，正文以三级标题、加粗字段与自然段/少量列表组织。所有模板保留现业务信息及行动说明，动态姓名/订单号/摘要按Markdown文本转义，不引入HTML、图片或新外链；现有实际链接保留为可读链接。验证码突出显示六位码，分开有效期与保密提示，不进通知持久化。

本次统一覆盖验证码、账号授权、订单人员分配、周汇率提醒、费用待审核、费用审核结果、费用补录待审批、提成冲减8类实际发送场景。消息列表预览标题使用简洁的固定业务标题，正文按照「标题 → 关键字段 → 处理说明」排版。

验证码示例（仅用于说明）：

```markdown
### Roncin 密码验证码

您的设置 / 重置密码验证码：

**123456**

有效期：**5 分钟**。

请勿向他人透露。如非本人操作，请忽略本消息。
```

## 验收

1. 全部实际发送路径均使用sampleMarkdown，title/text与JSON字符串格式正确；无遗留实际SendText调用。
2. 7业务模板及即时验证码均有清晰标题、重点与分段；保留业务字段、发送资格、时机、重试和敏感边界。
3. 名称/摘要中的特殊Markdown字符不会破坏布局或注入链接，UTF-8中文正常。
4. 定向渲染/HTTP适配器/验证码相邻测试通过，生成物若无契约变化不动；按风险执行后端门禁。
5. 不主动给员工补发或群发历史消息；已发送历史消息不会被修改。真实钉钉渲染需指定收件人授权后再验证。

## 范围外

新通知类型、权限/工作流/重试变更、Schema变更、消息编辑撤回、交互卡片平台、图片和HTML风格。

## 规划状态

用户已回复“好的”，批准全部8类通知的统一样式与范围。实现及针对性验证完成，独立审阅和后端门禁见`verification.md`。真实钉钉客户端视觉验收未执行。
