# 钉钉通知 Markdown 与 ActionCard 契约

## 1. 适用范围

全部实际企业机器人通知：订单人员分配、账号授权、周汇率提醒、费用待审核、费用审核结果、费用补录待审批、提成冲减及即时密码验证码。使用紧凑Markdown正文，存在合法本站动作时采用内置单按钮ActionCard；验证码等无动作内容采用Markdown。已退役模板不重新启用。

## 2. 发送协议

业务层交付纯Go消息内容，HTTP/JSON只在data编码，命名以实现中DingTalkMessage/SendMessage/PrepareMessage为准。动作由固定文案、受控根相对路径组成，准备阶段按已配置登录回调的scheme/host解析本站URL。

```text
POST /v1.0/robot/oToMessages/batchSend
x-acs-dingtalk-access-token: 企业访问令牌
robotCode: 当前应用客户端ID
userIds: [绑定的企业员工ID]
有动作：msgKey=sampleActionCard
        msgParam=JSON字符串 {title,text,singleTitle,singleURL}
无动作：msgKey=sampleMarkdown
        msgParam=JSON字符串 {title,text}
```

这是当前企业机器人内置消息格式，不能直接发送工作通知`msgtype=action_card`/`single_title`，也不能使用群Webhook的actionCard字段。参见[官方企业机器人消息类型](https://open.dingtalk.com/document/orgapp/types-of-messages-sent-by-robots)。不使用已停止新应用接入的StandardCard，不引入模板ID配置、个人DWS登录态或新增卡片权限。

## 3. 内容、动作与敏感边界

- 短标题、关键编号/对象、一句话说明和必要备注，使用固定少量状态图标；不全字段逐段加粗，不依赖font/HTML颜色或自定义彩色背景。动态值作为普通文本转义且压成一行，不能形成链接/标题/结构。
- 标题固定，不含验证码、令牌或敏感业务摘要；验证码保留6位前导零、5分钟有效和保密提示，仅即时正文出现。
- 合法动作仅限/welcome、/orders、/finance/exchange-rates、/orders/sea-export/{UUID}/fees、/commission-adjustments/{UUID}/my-supplement-source。订单分配和补录去明确标注的列表；补录ResourceID是RequestID，不能当订单ID。提成去本人最小来源页，不换成组织级财务入口。
- 费用入口继续严格匹配Parameter末尾由当前ResourceID构成的固定费用路径；缺合法目标则无动作，不猜测或纠错。姓名和摘要不能提供URL。
- 准备只做离线内容/本站URL解析，不申请token或发送；预览和发送复用。动作字段成对、目标限定本站路径，不能放任意外链。最终准备幂等，无正文重复追加链接。
- 根据是否有合法动作显式选型；请求失败时不发送第二条Markdown作为回退。响应必须有processQueryKey且目标不在invalidStaffIdList/flowControlledStaffIdList，否则视为未受理；processQueryKey只代表接口受理，不代表客户端实际收到或视觉一致。
- 普通通知保留后台领取、租约、退避和完成语义；验证码继续挑战事务提交后即时发送，激活/冷却规则不变，不进入通用通知表或后台重试。
- 审核通过/退回状态按生产端固定结果后缀判定，不可对动态姓名全文contains搜索；费用更正等其他合法结果按准确定义的信息态处理。

## 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 集成禁用 | 明确不可发送，不发起请求 |
| 收件人、标题或正文为空 | 拒绝，不获取令牌或发空消息 |
| 动作标题/目标不成对、URL或路径不在范围 | 明确失败，不伪造按钮或静默纠错 |
| 无动作的验证码 | sampleMarkdown，无占位按钮 |
| 动态内容含Markdown或换行 | 按普通单行内容展示，不生成结构 |
| 上游失败/缺processQueryKey/目标无效或被限流 | 保持发送失败语义，无协议降级重发 |
| 密码验证码发送失败 | 原挑战不能激活，保持冷却，不后台重试 |

## 5. 正常、基准与错误场景

- 正常：费用待审核展示状态图标、订单号和简短说明，合法费用URL成为独立按钮。
- 基准：审批、接收人、组织权限、业务状态和租约规则不因排版变化。
- 错误：用工作通知JSON调用机器人batchSend；使用font颜色承诺任意样式；以测试假UUID生成业务按钮；验证码进入标题/URL。

## 6. 必要验证

- 7业务模板+验证码逐类验证紧凑结构、关键内容、受控动作、中文与特殊字符、前导零、结果状态不被姓名干扰。
- HTTP替身验证sampleActionCard精确大小写、字符串msgParam、singleTitle/singleURL、无动作sampleMarkdown、绑定收件人及失败不降级。
- 预览离线零token，准备幂等，预览与发送的正文/动作一致；非法动作与站点配置在发送前拒绝。
- Worker完成/失败退避、密码挑战激活/失败与系统管理员测试权限相邻回归。真实库Skip与客户端收件不能用mock代替。

独立Admin与本人测试规则见[通知测试契约](./dingtalk-notification-test.md)。

## 7. 错误与正确示例

```text
错误：batchSend body={msgtype:"action_card",action_card:{single_title:"查看详情"}}
正确：msgKey=sampleActionCard，msgParam为包含title/text/singleTitle/singleURL的JSON字符串。

错误：ActionCard失败后再发sampleMarkdown；把RequestID拼为订单ID。
正确：失败按原链路外传；动作只使用可靠固定目标，列表按钮明确标为列表。
```
