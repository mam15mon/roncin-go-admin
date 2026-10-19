# 钉钉通知 Markdown 契约

## 1. 适用范围

全部实际企业机器人通知采用 Markdown：订单人员分配、账号授权、周汇率提醒、费用待审核、费用审核结果、费用补录待审批、提成冲减及即时密码验证码。已退役注册/邀请模板不重新启用。

## 2. 发送协议

业务层通过以下接口交付消息；HTTP/JSON 仅在 `internal/data/dingtalk.go` 编码。

```go
type DingTalkMarkdownMessage struct {
    Title, Text string
    OrderEntryPath string
}
SendMarkdown(context.Context, string, DingTalkMarkdownMessage) error
```

```text
POST /v1.0/robot/oToMessages/batchSend
x-acs-dingtalk-access-token: 企业访问令牌
robotCode: 当前应用客户端ID
userIds: [绑定的企业员工ID]
msgKey: sampleMarkdown
msgParam: JSON字符串，内容为 {"title":"固定业务标题","text":"Markdown正文"}
```

## 3. 内容与发送边界

- 正文按三级标题、关键字段、处理说明组织，使用加粗及空行分段。姓名、编号、摘要等动态值作为普通文本转义，不能形成标题、链接或其他结构。
- 标题使用固定业务文案，不包含验证码、令牌或业务摘要。验证码保留6位数字和前导零，正文说明5分钟有效及保密提示。
- 普通通知保留原后台任务、租约、重试和完成语义；验证码继续在挑战事务提交后直接发送，不进入通知表、任务、URL或日志。
- 费用审核通知的既有入口来自入队Parameter：biz仅提取匹配当前ResourceID的固定`/orders/sea-export/{id}/fees`路径；data复用登录回调的scheme/host拼绝对地址，追加「查看订单费用」可读链接。动态姓名或摘要不能生成链接目标。
- 不增加新配置、外链、图片或HTML，不实现自动文本回退。新格式只作用于此后实际发送，不主动回放历史消息。
- 返回 `processQueryKey` 仅代表接口受理，不代表真实员工收到消息或客户端视觉验收成功。

## 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 集成禁用 | 保持既有禁用错误，不发起请求 |
| 收件人、标题或正文为空 | 拒绝发送，不获取令牌或发送空消息 |
| 动态内容含Markdown控制符或换行 | 按普通文本展示，不生成额外结构 |
| 令牌或发送请求失败、缺少受理键 | 原错误语义返回，由现有业务链路处理失败 |
| 密码验证码发送失败 | 保留原挑战激活/冷却规则，不转入后台重试 |

## 5. 正常、基准与错误场景

- 正常：费用待审核消息分段展示订单号、提交人和处理说明；验证码正文突出6位数字。
- 基准：发送资格、订单状态、审批流程与通知接收范围保持原规则。
- 错误：`sampleText` 正文内放Markdown；`msgParam` 传对象；验证码进入标题或通用通知持久化；因格式改动重发历史任务。

## 6. 必要验证

- 7类业务模板逐类断言固定标题、正文结构、关键业务字段与处理提示。
- 验证中文、前导零验证码、特殊Markdown字符及换行的纯文本处理。
- HTTP替身解码实际请求，断言 `sampleMarkdown`、字符串类型 `msgParam`、`title/text`、收件人及令牌；覆盖禁用、空值和失败相邻路径。
- 回归Worker完成/失败重试与验证码发送激活/失败流程。mock通过与真实钉钉渲染必须分别记录。

## 7. 错误与正确示例

```text
错误：msgKey=sampleText，content="### 费用审核\n**订单号**：..."
正确：msgKey=sampleMarkdown，msgParam为包含title/text的JSON字符串。

错误：把外部姓名直接插入Markdown，或把验证码加入消息标题。
正确：动态字段先转义；标题保持固定，验证码只在即时正文中出现。
```
