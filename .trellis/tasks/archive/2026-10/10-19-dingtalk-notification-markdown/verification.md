# 钉钉 Markdown 通知验收

## 实施结果

- 全部8类实际通知统一使用`DingTalkMarkdownMessage`与`SendMarkdown`；7类业务模板和即时验证码均使用固定标题、三级正文标题、重点加粗及自然分段。
- data继续使用既有企业机器人单聊端点，载荷为`sampleMarkdown`，`msgParam`为包含`title/text`的JSON字符串。
- 动态字段转义Markdown/HTML字符并折叠为单行，保留中文与业务信息。验证码保留6位数字和前导零，只在即时正文出现。
- 两类费用审核通知保留实际订单费用页入口；仅提取与当前ResourceID匹配的固定后缀，由data沿用现有回调站点生成「查看订单费用」链接。
- 后台任务租约/重试、发送资格、验证码60秒冷却/5分钟有效、挑战激活与失败语义保持原规则。无Schema、API、权限、前端或生成物变更。

## 定向验证（全部通过）

```bash
go -C server test -count=1 ./internal/biz -run 'Test(Notification|Render.*Notification|RenderExchangeRateWeeklyReminder|FeeSupplementNotificationRendering|FeeReviewMarkdown|PasswordReset)'
go -C server test -count=1 ./internal/data -run '^Test(DingTalk|FeeReviewNotificationOrderEntry)'
go -C server test -count=1 ./internal/server -run '^TestPasswordReset'
go -C server vet ./internal/biz ./internal/data ./internal/server
git diff --check
```

覆盖全部业务模板、动态特殊字符/换行、受控/不匹配/伪造链接、验证码前导零/有效期/固定标题、实际JSON载荷、发送失败/禁用/空值、worker重试及密码重置相邻流程。

## 独立检查与最终门禁

独立Trellis检查代理审阅产品代码、测试、规范及任务文档；未发现产品代码问题。文档中的旧规划状态已同步更新。

`pnpm run check:server`通过，包含Go重复检测、分层检查、Proto lint、默认范围的`go test -p 32 ./...`、全量`go vet ./...`及漏洞扫描。分层违规0处，未发现未处理的可达漏洞。完整日志位于`/tmp/roncin-markdown-check-server.log`。

主代理只读验证当前开发服务`GET /health/ready`返回HTTP 200、`{"status":"ok"}`。没有发送消息或写业务数据。

## 验证边界

本次没有提供真实数据库集成测试环境开关；默认Go测试会跳过这些用例，本记录不将SKIP记为真实库验证通过。数据库、事务和Schema未调整，相关测试替身只同步了新发送接口。

没有向真实员工发送测试消息，没有回放历史通知。HTTP替身只证明协议载荷和失败行为，真实钉钉客户端的字体、间距及收件效果尚未验收；此后新触发消息会使用新格式，历史消息不会更新。
