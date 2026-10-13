# 钉钉扫码主登录契约

## 1. 适用范围

适用于 `/user/login` 的默认扫码、管理员密码切换与 `/user/login/dingtalk/callback`。复用钉钉 OAuth2 和人员状态机，不通过扫码创建或开通人员。

## 2. 签名

```text
GET /api/v1/auth/dingtalk/login-config
data: enabled, authorizeUrl?, expiresAt?
回调：/user/login/dingtalk/callback?authCode=...&state=...
```

```text
扫码端点：https://login.dingtalk.com/oauth2/auth
authorizeUrl 参数沿用服务端配置，追加 iframe=true
Query key：['auth', 'dingtalk-login-config']
```

## 3. 契约

- 默认展示真实钉钉 OAuth2 iframe，次级“管理员账号登录”切换密码，可返回扫码。保留品牌资源，配置失败不静默改成密码表单。
- 配置 Cookie、state、expiresAt 来自同一次服务端响应，响应 Cache-Control=no-store。clientSecret 与换码令牌仅在服务端；前端不记录授权码或 state。
- React Query 固定 key 去重，配置请求不消费 AbortSignal，避免取消后的响应仍更新 Cookie 并覆盖新实例。禁止焦点/重连自动刷新及自动重试；模式返回重新获取，刷新串行 refetch(cancelRefetch:false)。
- 手动刷新先同步卸载旧 iframe 和监听，再请求新配置。每个扫码实例以 state 为 key；父回调引用稳定，监听清理不能重置已处理实例造成重复认证。
- 配置必须是官方 OAuth2 端点，回调 origin 等于本站且路径精确匹配既有 callback。解析公开参数使用 URL/URLSearchParams，不能手工编码或生成普通授权 URL 二维码冒充扫码协议。
- message 只接受精确 origin=https://login.dingtalk.com、source=当前 iframe.contentWindow、正确结构及 success 标记。成功 redirectUrl 必须匹配当前回调 origin/path，只有一个相同 state 和一个非空 authCode。
- 只处理一次成功；卸载、退役、过期或旧 iframe 消息忽略。校验后构造本站固定回调 URL，不直接导航消息传来的外链。后端再次验证 Cookie state 与企业身份。
- 过期显示“二维码已过期，请刷新”，不轮询；切回前台和焦点恢复复核期限。iframe 加载失败或超时显式提示，可手动刷新；管理员入口始终可达。
- 回调仍保留未同步、待分配、停用提示；认证成功一个工作台直进，多个工作台明确选择，组织 ID 与候选来自本次响应的同一份本地状态。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 钉钉未启用 | 显示未启用，管理员入口可达 |
| 配置缺字段、回调不同源/错误路径 | 显示配置错误，不渲染可用扫码实例 |
| 配置或 iframe 加载失败/超时 | 显示失败并支持显式刷新 |
| 过期 | 禁止成功回调，显示刷新提示 |
| 非匹配 origin/source/state/回调路径或重复字段 | 忽略，不触发认证 |
| 旧实例或重复成功消息 | 忽略，不重复换码 |
| 人员未同步/待分配/停用 | 既有人员状态提示，不自动开通 |

## 5. 正常、基准与错误场景

- 正常：打开扫码、扫码成功后进入本系统回调；多工作台展示选择，再通过会话轮转进入目标。
- 基准：管理员切换密码并返回扫码；配置失败时仍可使用管理员入口。
- 错误：普通 URL 二维码替代钉钉iframe；刷新时旧消息仍触发导航；根据消息直接跳转任意 redirectUrl。

## 6. 必要验证

- StrictMode、快速切换、连续刷新、旧请求响应和 state Cookie 一致性。
- 非法消息、旧iframe、重复成功、到期与后台恢复、监听和定时器清理。
- 配置失败、加载失败、超时、切换模式、单/多工作台和人员各状态。
- 真实企业应用扫码及开发代理/生产同域回调，须单独记录实测，协议源码核验不能替代。

## 7. 错误与正确示例

```text
错误：反复调用没有公开销毁接口的全局SDK；将redirectUrl直接传location。
正确：就近iframe组件管理可清理监听；精确校验后构造本站callback。

错误：刷新先GET config再保留旧扫码实例；焦点自动GET覆盖state Cookie。
正确：先卸载旧实例，固定Query key串行刷新，焦点只复核期限。
```
