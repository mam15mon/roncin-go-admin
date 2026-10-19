# 钉钉 Markdown 通知设计

## 边界与数据流

- 业务模板继续在 `server/internal/biz/notification.go` 渲染；返回包含固定标题和 Markdown 正文的纯 Go 消息值，领域层不依赖钉钉 HTTP 或 JSON 类型。
- `DingTalkNotificationSender` 改为明确的 `SendMarkdown`，一次性同步全部实现、调用方及测试替身，不保留 `SendText` 兼容分支。
- `server/internal/data/dingtalk.go` 使用现有单聊 `batchSend` 端点、令牌和响应处理；唯一载荷变化为 `msgKey=sampleMarkdown` 与字符串化 `msgParam={title,text}`。
- 密码验证码仍在 `PasswordResetUsecase.RequestCode` 事务外即时发送，只调整消息排版；保持60秒冷却、5分钟有效、发送激活和失败处理。
- 现有7类业务通知仍经原后台任务租约、重试和完成流程发送。不会触发历史消息重发。

## 模板规范

正文使用三级标题、加粗字段名或重点、空行分段；不使用表格、HTML、图片或交互卡片。固定标题不含验证码、令牌或业务摘要。动态姓名、编号和摘要统一作为普通文本转义 Markdown 控制符，避免动态内容生成标题或链接。现有业务信息、含义和处理提示完整保留。

费用待审核与审核结果实际入队时会在 Parameter 追加海运订单入口（`server/internal/data/order_fee_review_notifications.go:48`、`:64`、`:69`）。biz仅提取完全匹配当前ResourceID的固定费用路径，放入消息值的受控`OrderEntryPath`字段；data沿用现有redirectURI的scheme/host生成绝对地址，并追加可读的「查看订单费用」Markdown链接。其余动态字段作为普通文本转义。不引入新配置或通用链接系统。

## 契约与迁移

不修改HTTP接口、Proto、Ent、数据库、权限或前端，没有生成物更新及数据迁移。同步更新密码找回规范中的发送方法名称，避免领域接口文档漂移。

## 技术依据与限制

[钉钉官方组织SDK规范](https://github.com/DingTalk-Real-AI/dingtalk-channel-sdk-python/blob/main/SPEC.zh-CN.md) 第4节与4a节明确 `sampleMarkdown`、字符串化 `msgParam` 的 `title/text` 及主动单聊端点。只采用基础Markdown语法，真实字体与间距由钉钉客户端决定。HTTP mock 验证载荷受理协议，不能替代真实客户端视觉验收。

本期不主动向真实员工发测试消息；上线后新触发通知使用新格式，历史已发消息保持原样。回退通过正常代码回退处理，不实现自动文本回退。
