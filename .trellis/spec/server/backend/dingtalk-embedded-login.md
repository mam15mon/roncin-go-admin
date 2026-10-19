# 钉钉客户端免登契约

## 1. 适用范围与触发

钉钉内打开通知普通业务链接，无有效 Roncin 会话时自动识别当前钉钉员工并回原页面。
涉及公开认证接口、上游身份交换及开发/生产入口。普通浏览器扫码 OAuth2 保持独立。
身份及会话准入复用 `biz/auth.go`，组织边界见 [工作台与会话](./auth-session-org-switch.md)。

## 2. 签名

```text
GET /api/v1/auth/dingtalk/embedded-config   PUBLIC
信封 data: {enabled: boolean, corpId?: string, state?: string, expiresAt?: string}
POST /api/v1/auth/dingtalk/embedded-login   PUBLIC
请求: {authCode: string, state: string}
响应: DingTalkEmbeddedLoginResponse，data 共享 DingTalkLoginResult（status/message/displayName?/currentUser?）

SDK: dingtalk-jsapi/entry/union + api/runtime/permission/requestAuthCode
调用: runtime.permission.requestAuthCode({corpId}) -> Promise<{code}>
上游: POST https://oapi.dingtalk.com/topapi/v2/user/getuserinfo?access_token=<企业应用token>
JSON: {"code":"<客户端临时免登码>"}
响应必填: result.unionid, result.userid, result.name
```

## 3. 执行契约

- SDK 只按需用于钉钉客户端。前后端固定上述旧版微应用协议；扫码 OAuth code 仍走原
  `/v1.0/oauth2/userAccessToken`，不得混用或自动回退。此 JSAPI 不要求 `dd.config`。
- UA 包含 DingTalk（大小写不敏感）仅控制体验；真实身份始终通过企业应用 token 向钉钉换码核验。
  CorpID 从现有固定企业配置取得，不接受客户端提交 userid、unionid、corpId 或组织范围。
- 新配置成功时生成随机 state，返回 state 和五分钟到期时间，同时写独立
  `<session_cookie_name>_dingtalk_embedded_state` Cookie；HttpOnly、Path=/、Secure/SameSite
  复用现有会话策略。扫码 Cookie `<session_cookie_name>_dingtalk_state` 不受影响。
- 配置和换码均 `Cache-Control: no-store`、`Pragma: no-cache`。禁用配置不返回标识/state，
  不发新 Cookie；禁用换码明确拒绝。缺失、不相同 state 在上游换码前拒绝，常量时间比较。
- 换码时清除本通道 state Cookie；这是浏览器 Cookie 比对和上游临时码校验机制，
  不宣称是持久化服务端单次挑战，也不新增挑战表或权限平台。
- 企业 token 复用现有缓存与 HTTP 超时；上游业务错误、状态码、坏 JSON、空身份明确失败。
  三个身份字段非空且本地 unionID/userID 严格一致后，复用已有已同步/待分配/启停校验。
  禁止按手机号、姓名猜测绑定，禁止自动建人或授权。
- 单工作台直达，多工作台明确选择；后端稳定排序建立初始会话不代表默认工作台。
  选择其他组织仍走正常会话轮转。订单链接不参与身份核验，也不决定会话组织。
- 授权码/state 仅在当前尝试内存中使用，不写 URL、storage、日志或持久化。
  app token/clientSecret 仅服务端，不返回原始敏感上游错误正文。
- Go 生产 HTML、Vite 开发 HTML 及前端渲染前预热在钉钉 UA 下全部跳过扫码配置与 iframe，
  登录 HTML 仍 no-store。普通浏览器保留原扫码预热，不能只在 React 中隐藏 iframe。
- Proto 以独立 `DingTalkEmbeddedLoginResponse` 满足 buf 的响应唯一性和标准命名；信封与扫码同构，data 共用 `DingTalkLoginResult`。
- 本期无 Schema、权限码和配置字段变更；原生成流程同步 Proto、Go、OpenAPI、前端客户端。

## 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 认证禁用 | 配置 enabled=false，不公开其他标识；换码返回钉钉未启用错误 |
| state/Cookie 缺失或不匹配 | 拒绝，不调用上游，不建立会话 |
| 免登码无效、上游网络/权限/数据异常 | 映射中文领域错误；显式重试，不混用其他协议 |
| 本地未同步 / 待分配 / 停用 | 原状态响应，不发有效会话 Cookie |
| 没有有效组织资格 | 拒绝，不建立会话 |
| 双身份绑定冲突 | 复用仓储拒绝，不以 userid 单独匹配 |
| 合法账号 / 单工作台 | 正常会话 Cookie，前端回完整站内目标 |
| 合法账号 / 多工作台 | 明确工作台选择，再按已有成员资格轮转 |
| SDK 失败、空码或超时 | 当前页中文提示，重试/账号登录可达，无自动循环 |
| StrictMode 重订阅 / 离页 / 切账号 | 同一尝试不重复消费；旧结果不换码或导航 |

## 5. 正常、基准与错误场景

- 正常：钉钉订单费用通知 → 原费用路径401 → 客户端免登 → 多工作台选择 → 原 query/hash。
- 基准：已有有效会话直接读订单；Chrome 无会话使用既有扫码/账号入口，不加载客户端 SDK。
- 错误：把 UA 当身份；用 OAuth 接口消费微应用 code；读取 URL organizationId 自动切组织；
  隐藏扫码 UI 却仍在 HTML 导航 iframe；失败后自动循环申请临时码。

## 6. 必要测试与断言

- data HTTP 替身断言企业token申请、getuserinfo固定地址/JSON code和身份字段，无 OAuth 请求；
  覆盖上游错误/空结果/缓存与凭据不泄漏。
- biz 双通道相同准入、state在resolver前拒绝、单多组织与未同步/待分配/停用/零资格。
- 真实 Kratos HTTP 路由无会话可访问、camelCase DTO、no-store、独立 Cookie 属性/清除、成功才发会话。
- Go/Vite/main 三层钉钉UA零扫码预热、普通浏览器相邻协议及HEAD/业务路径不预热。
- web StrictMode 单次认证、显式重试新尝试、SDK/HTTP迟到和账号切换、单多工作台完整回跳、
  恶意/循环目标拒绝、公开请求scope豁免。构建核对 SDK 按需拆包。
- 真实钉钉免登单独记录。程序替身/浏览器模拟不能替代可信域名、应用可见范围和客户端真机验收。

## 7. 错误与正确写法

```text
错误：收到客户端 code 后，先试 OAuth；失败再改用企业 getuserinfo。
正确：embedded-login 固定消费 runtime.permission.requestAuthCode 的临时码。

错误：StrictMode cleanup 使尝试失效，但 handled=true 又阻止重新处理结果，永久 loading。
正确：尝试 Promise 去重；每次 effect 建立独立结果订阅，cleanup 只隔离失效订阅。

错误：window.location.replace(redirect) 直接使用外部输入。
正确：共享站内根相对路径校验，URL解析核对同源，拒绝登录循环，保留 query/hash。
```
