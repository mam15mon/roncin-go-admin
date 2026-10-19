# Research: 钉钉通知卡片协议与最薄上线方案

- Query: 核实当前企业机器人单聊接口的内置 ActionCard、StandardCard、现行模板卡与 A2UI JSON 卡片能力，推荐支持真实状态颜色、简洁摘要与可选跳转按钮的上线方案。
- Scope: mixed
- Date: 2026-10-19

## Findings

### 本期最终决策（按用户最新明确选择）

**采用现有 `POST https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend` 的内置 `sampleActionCard` 单按钮方案。** `msgParam` 为 JSON 字符串，内层准确字段为 `title`、`text`、`singleTitle`、`singleURL`，不是 `markdown`、`single_title`、`single_url`，也不套用工作通知 `action_card` 或 Webhook `actionCard` JSON。

有合法处理动作时发送 ActionCard；即时验证码等没有动作的消息，按内容类型显式选择 `sampleMarkdown`（`title/text`）。这不是失败后的格式回退，ActionCard 失败直接按原错误流程处理，禁止捕获错误后再发 Markdown。

- 本期效果为紧凑摘要、状态图标和明确的单按钮。不宣称支持自定义背景、标题栏颜色或任意字体颜色；不使用官方未列明的 font/HTML 色彩。
- 七类业务测试消息统一使用真实通用站内 `ActionPath=/welcome`，按钮「打开系统首页」。清除任何示例业务资源动作，正文明确这是示例测试、该按钮不指向真实订单。
- OTP 测试使用无效示例验证码，仍不生成按钮、真实挑战、可用凭据或持久化验证码。
- 不新增模板平台、模板 ID 配置、卡片回调、Stream 或其他发送身份。

下文 StandardCard、新模板和 A2UI 是**已研究但本期不采用的备选资料**，不能作为本期要求添加外部模板配置。原始调查最初以真实状态颜色为目标，用户后续已明确收敛为 ActionCard 方案，以本节及任务 design.md 为准。

| 通道 | 模板依赖 | 颜色能力 | 本期结论 |
| --- | --- | --- | --- |
| batchSend + sampleActionCard | 官方内置 | 无独立颜色字段；Markdown 与单按钮 | **采用**；紧凑摘要、状态图标、合法真实动作 |
| sampleMarkdown | 官方内置 | 不承诺自定义颜色 | **按内容选择**，用于没有合法动作的消息与 OTP |
| StandardCard + 普通版 interactiveCards/send | 内置 JSON | 普通版组件 | 历史 API，不再支持新应用接入，不采用 |
| createAndDeliver + 企业模板 | 自有模板 ID | 模板支持真实颜色 | 需要额外外部配置，本期不采用 |
| A2UI JSON 构建 | 不需要模板 | Tag/Text 等真实颜色 | 当前公开发送通道为 DWS/MCP，非现有企业机器人链路，不采用 |

### 现有仓库定位

- `server/internal/data/dingtalk.go:27`：机器人 URL 为 `https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend`。
- `server/internal/data/dingtalk.go:233`：`SendMarkdown` 使用企业访问令牌、绑定员工 userId、robotCode=clientID。
- `server/internal/data/dingtalk.go:254`：msgParam 为 title/text 序列化后的 JSON **字符串**，msgKey=sampleMarkdown。
- `server/internal/data/dingtalk.go:274`：准备内容的统一路径由预览和正式发送复用。
- `.trellis/spec/server/backend/dingtalk-notification-markdown.md`：八类通知、验证码即时路径、后台任务与现有链接边界。
- `.trellis/spec/server/backend/dingtalk-notification-test.md`：系统管理员独立 Admin 测试页、只发本人、示例与真实订单权限规则。

### 1. batchSend 内置 ActionCard 的准确字段

官方[企业机器人发送消息的消息类型](https://open.dingtalk.com/document/orgapp/types-of-messages-sent-by-robots)明确描述企业内部应用机器人消息，不能与 Webhook actionCard 或 corpconversation 工作通知混用。

```json
{
  "robotCode": "企业应用clientID",
  "userIds": ["绑定的企业员工userId"],
  "msgKey": "sampleActionCard",
  "msgParam": "{\"title\":\"固定通知标题\",\"text\":\"正文\",\"singleTitle\":\"查看详情\",\"singleURL\":\"https://example.com\"}"
}
```

- `sampleActionCard`：title、text、singleTitle、singleURL；一个按钮。
- `sampleActionCard2`：title、text、actionTitle1、actionURL1、actionTitle2、actionURL2；纵向两个按钮。
- `sampleActionCard3/4/5`：相同 actionTitleN/actionURLN，分别纵向三个、四个、五个按钮。
- `sampleActionCard6`：title、text、buttonTitle1、buttonUrl1、buttonTitle2、buttonUrl2；横向两个按钮。**注意 URL 字段大小写和前五类不一致。**
- 官方消息类型列表没有 sampleActionCard7；不要假设存在。
- 官方明确 sampleActionCard 与 2 的 text 支持 Markdown。支持语法列出标题、引用、加粗、斜体、链接、图片、列表和换行，**没有 font/color/HTML 的正式支持承诺**。没有状态色或 headerColor 字段；不能仅凭社区 font 示例宣称可上彩色。
- 官方样例使用 HTTPS URL；本任务可继续限定自身 HTTP(S) 站点地址。不能把小程序页面路径直接当 URL。
- 未发现这一消息类型正文精确长度上限的官方列明值，不猜上限；当前短业务摘要足够，本地可按业务约束限长。

### 1.1 当前官方批量接口补充：响应与长度证据

当前官方[批量发送人与机器人会话中机器人消息](https://open.dingtalk.com/document/development/chatbots-send-one-on-one-chat-messages-in-batches)与[消息发送与接收类型](https://open.dingtalk.com/document/development/robot-message-type)直接关联，确认此处内置模板就是现有单聊 API 可用类型。调用权限仍为「企业内机器人发送消息权限」，不要求新增互动卡片实例写权限。

响应形状：

```json
{
  "processQueryKey": "受理消息ID",
  "invalidStaffIdList": [],
  "flowControlledStaffIdList": []
}
```

除了非空 processQueryKey，发送时应拒绝收件人出现在 invalidStaffIdList 或 flowControlledStaffIdList 的结果；对单收件人而言不能把该情况报告为成功。受理不等于客户端收到或渲染验收。

现行正文写每批最多20人，错误码还残留100人文案；本仓库每次只发一个绑定用户，不受矛盾影响。文档仅有 `invalidParameter.msgParam.tooLong` 错误，没有列出 sampleActionCard 的「1000字」固定上限，也没有 singleURL 最大长度。**不能把工作通知 action_card 的1000字符约束直接声称为此API硬限制。** 若实现选择1000作为本地简洁消息边界，应在说明中称业务约束，并验证中文与转义后长度；不得无说明截断验证码或关键业务值。

URL 当前官方样例为 https；本期复用服务端从自身站点来源构造的 HTTP(S) 绝对URL，只允许设计列明站内动作，外部姓名与摘要不能成为目标。无合法动作就按内容明确选 Markdown，不传空单按钮或伪造业务资源URL。

### 2. StandardCard 是历史 API，不能按旧教程直接选型

[普通版发送接口](https://open.dingtalk.com/document/orgapp/robots-send-interactive-cards)正文重要提示：2024-12-31 迁入历史文档，不支持新应用接入，已接入应用可继续，接口不再新增能力。官方旧[Go 教程](https://open-dingtalk.github.io/developerpedia/docs/explore/tutorials/stream/bot/go/send-streaming-card/)仍有可直接复制的 StandardCard 示例，但不能忽略现行 API 文档弃用限制。

旧协议准确为：

```text
POST /v1.0/im/v1.0/robot/interactiveCards/send
cardTemplateId = StandardCard
singleChatReceiver = JSON字符串 {userId: ...}
cardBizId = 唯一外部卡片ID（最多100字符，建议64以内）
robotCode = 企业clientID
cardData = 整张普通版卡片结构的JSON字符串
callbackUrl = 可选，不传则无需回调
pullStrategy = false
响应 {processQueryKey: ...}
```

普通版的数据结构是 config/header/contents；新模板版是 cardData.cardParamMap；A2UI 则是 createSurface/updateComponents 消息数组。三者不可混用。本仓库仅接入普通消息 batchSend，并不能据此声称已获历史卡片接口准入。

### 3. 未采用备选：现行模板卡准确协议与外部前置

依据[创建并投放卡片](https://open.dingtalk.com/document/orgapp/create-and-deliver-cards)及官方[Go 卡片教程](https://open-dingtalk.github.io/developerpedia/docs/explore/tutorials/stream/bot/go/card-callback/)，机器人单聊可用以下薄请求：

```json
{
  "cardTemplateId": "企业自己的模板ID",
  "outTrackId": "应用生成的唯一投递ID",
  "userIdType": 1,
  "openSpaceId": "dtv1.card//IM_ROBOT.绑定的企业员工userId",
  "cardData": {
    "cardParamMap": {
      "title": "费用待审核",
      "status": "待处理",
      "statusColor": "orange",
      "summary": "有一票费用需要你审核",
      "referenceCode": "示例订单号",
      "actionLabel": "查看订单费用",
      "actionUrl": "https://自身站点/固定路径",
      "showAction": "true"
    }
  },
  "imRobotOpenSpaceModel": {
    "supportForward": false,
    "lastMessageI18n": {"zh_CN": "费用待审核"}
  },
  "imRobotOpenDeliverModel": {
    "spaceType": "IM_ROBOT",
    "robotCode": "企业clientID"
  }
}
```

上例 `title/status/...` 是**建议自定义的模板变量契约，不是 API 内建字段**；应与实际模板 JSON 一次性对齐，Map 值按官方接口使用字符串（布尔为 `"true"`/`"false"`，数组等序列化为字符串）。没有模板 ID 就不能发送；不得传 StandardCard 到此接口冒充现代模板。

API 使用同样 `x-acs-dingtalk-access-token` 企业令牌。新增权限为「互动卡片实例写权限」，现有「企业内机器人发送消息权限」不代表它已开通。

响应示意：

```json
{
  "success": true,
  "result": {
    "outTrackId": "本次唯一投递ID",
    "deliverResults": [{
      "spaceType": "IM_ROBOT",
      "spaceId": "绑定员工userId",
      "success": true,
      "carrierId": "投递结果id",
      "errorMsg": ""
    }]
  }
}
```

不能只检查 HTTP 200 或顶层 success；应检查外部 ID 与目标场域匹配、对应 deliverResults.success。carrierId 文档表示 IM 场域的 processQueryKey。受理/投递返回均不证明客户端颜色、按钮或布局真实可见。

`callbackType/callbackRouteKey` 可省略；文档列为可选。只做模板 URL 链接跳转无需 HTTP 回调或 Stream 常驻连接；不要照抄包含审批按钮的教程额外搭建审批回调。空或示例动作 URL 应在模板内隐藏按钮，不能生成假入口。

模板准备依据[普通卡片模板](https://open.dingtalk.com/document/development/card-template-building-and-publishing)、[高级版编辑器](https://open.dingtalk.com/document/development/interactive-card-editor)：后台创建模板、可导入导出的模板 JSON、保存后获取 ID；发布后不可编辑，需谨慎。官方[卡片示例仓库](https://github.com/open-dingtalk/dingtalk-card-examples) README 明确要求先在自己的卡片平台创建模板，再导入 JSON；未保证可长期依赖示例公共模板 ID。

若未来单独选择模板卡路线，才需要一个自有通用通知模板、模板ID环境配置与权限开通。本期不执行这些配置。真实状态色不能通过把颜色字符串放正文伪造。

### 4. A2UI 是真实免模板方案，但当前公开身份通道不同

[JSON 构建卡片](https://open.dingtalk.com/document/development/json-build-card)与[FAQ](https://open.dingtalk.com/document/development/json-card-building-faq)明确：无需预建模板，可由普通业务代码生成 JSON，不强制 Agent/Skill。消息 envelope `version: "v1.0"`；catalogId 为 `https://dingtalk.com/card/a2ui/catalogs/public/catalog.json`；创建顺序为 createSurface → updateDataModel → updateComponents。

[基本组件](https://open.dingtalk.com/document/development/json-card-basic-components)有 Tag.theme=black/gray/red/orange/green/blue，以及 Text/Icon colorToken 与 customLightColor/customDarkColor；能够真正做彩色卡片。

但[发送与更新](https://open.dingtalk.com/document/development/json-card-dws-send-and-update)只给 DWS 命令，单聊目标为 openDingTalkId；[官方 CLI 源码](https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli/blob/main/internal/helpers/chat.go)中 send-a2ui-card 调用 `callMCPToolOnServer("im", "create_and_send_a2ui_card", ...)`，契约 Mode=mcp，内容 a2uiMessages、个人接收者 receiverOpenDingTalkId，没有 robotCode 或现有企业 userId 对接契约。创建默认 PROCESSING，后续需 update-a2ui-card 推至 FINISH。不能把这个 MCP 工具名自行拼成企业 OpenAPI endpoint。

[交互事件](https://open.dingtalk.com/document/development/json-card-listen-to-interaction-events)明确为当前登录用户的个人订阅和长连接，服务端 HTTP 回调未开放。[版本说明](https://open.dingtalk.com/document/development/json-card-client-version-and-fallback)未列各组件最低客户端版本，只能真机验证。

因此该免模板能力可用于个人 Agent/DWS 场景，但没有足够证据支持本任务把生产后台通知切成个人登录态通道。若未来官方给出企业机器人发送 A2UI 的稳定服务端接口，再单独评估。

### 外部文档读取方式及可复核来源

open.dingtalk.com 动态页面在浏览工具中常只显示目录；本次除通过浏览搜索/打开官方页面，还读取其前端实际使用的官方静态 HTML 源。地址规律：

```text
https://icms-document.oss-cn-beijing.aliyuncs.com/zh-CN/dingtalk/{orgapp|development}/topics/{slug}.html
```

例如普通版弃用原文可在 `orgapp/topics/robots-send-interactive-cards.html` 复核。当前 JSON 导航是官方 `GET /api/docCenter/getDocInfoList?tabCode=lHGhOeTVdR`，并非猜测第三方文档。所有结论均以钉钉官方文档、官方 GitHub 教程/示例/CLI 为依据，搜索所得 CSDN/第三方 SDK 不作为结论证据。

## Related specs

- `.trellis/spec/domain/glossary.md`
- `.trellis/spec/domain/architecture-map.md`
- `.trellis/spec/server/backend/dingtalk-notification-markdown.md`
- `.trellis/spec/server/backend/dingtalk-notification-test.md`

## Caveats / Not Found

- 未实际调用任何外部发送、创建模板、发布模板或修改钉钉后台；研究不触发通知。
- 未找到面向企业后台 accessToken 的稳定公开 A2UI 发送 endpoint；未找到创建/发布业务卡片模板的公开服务端 API。结论为「本次未找到」，不是断言平台永远不存在。
- 官方旧教程与新文档准入规则存在时间差，必须以 API 页弃用提示为准。
- sampleActionCard 不含颜色字段，font/HTML 颜色未获该协议官方语法支持；有按钮不等于彩色互动卡片。
- 本期不需要模板导入、模板ID或新权限。替代路线若以后采用，必须另行准备外部配置。
- 本期不增加任何自动降级或历史格式兼容。

### 最后决策与实施交接

以用户最后明确选择为准：现有 batchSend 内置 `sampleActionCard`，参数 `title/text/singleTitle/singleURL`；无动作内容（尤其 OTP）显式 `sampleMarkdown`；七类业务测试按钮只打开真实通用 `/welcome` 首页并说明不指向真实订单；发送失败无格式回退。本期不引入 StandardCard、createAndDeliver、A2UI 或后台模板配置。实施与校验以当前任务 design.md/implement.md 为准。
