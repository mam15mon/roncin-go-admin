# 钉钉扫码主登录契约

## 1. 适用范围

普通浏览器 `/user/login` 默认使用本站绘制的钉钉扫码二维码；账号密码切换、回调换码、人员准入及工作台选择共用原有认证流程。钉钉客户端仍使用独立免登协议，见[客户端免登](../../server/backend/dingtalk-embedded-login.md)。不通过扫码创建或开通人员。

## 2. 接口与数据流

```text
GET /api/v1/auth/dingtalk/login-config
  data: enabled, qrCodeUrl?, state?, expiresAt?
POST /api/v1/auth/dingtalk/qr/poll
  body: {state}
  data: {status, authCode?}
status: WAITING=1 / SCANNED=2 / CONFIRMED=3 / EXPIRED=4
回调：/user/login/dingtalk/callback?authCode=...&state=...
POST /api/v1/auth/dingtalk/login
  继续由后端验证 state Cookie、兑换 OAuth 身份并执行人员准入。
```

- `qrCodeUrl` 是钉钉返回的真实 `https://login.dingtalk.com/oauth2/qr_confirm.htm` 扫码链接，带唯一非空 `code`。不得把普通 OAuth 授权 URL 编成二维码冒充扫码协议。
- 前端复用 Ant Design `QRCode` 本地绘制。二维码数据可用后直接显示真实码；不嵌入钉钉 iframe，不接收跨窗口登录消息，不使用固定延时猜测二维码就绪。
- 上游二维码生成与 `login_with_qr` 长轮询由服务端 gateway 执行，内部保留旧集成的 `iframe=true` 协议参数以沿用成功回调分支；页面不创建 iframe。每次尝试隔离 CookieJar、扫码标识与状态，秘密不放入日志、浏览器存储或 Git。
- state 随本次配置写入 HttpOnly Cookie；轮询请求先核对 body state 与 Cookie，再调用上游。所有配置、轮询及认证响应禁止缓存。
- 服务端只接受固定钉钉端点，完整沿用本应用的 OAuth 参数；不借用官方示例应用或回调，不让前端传入上游 URL、请求头或 Cookie。
- 上游成功返回的 URL 必须与本应用回调 origin/path 相同，无 userinfo，且仅有一个匹配当前尝试的 state、一个非空 authCode；不直接信任用户身份，也不导航上游提供的任意 URL。
- 前端收到有效确认结果后构造本站固定回调，原有换码接口继续执行 Cookie state、企业身份、启用人员、成员资格及权限校验。
- 上游协议按实际 `h5-dingtalk-login/0.126.3` 资源分析实现；这是页面内部协议，发生上游错误或风控明确显示失败，不自动回退 iframe 或伪造扫码成功。

## 3. 首屏 HTML 与配置预热

- Go 生产 HTML 与 Vite 开发 HTML 在普通浏览器 `GET /user/login` 时获取本次配置，在同一响应注入配置 JSON 与 Cookie。每次导航独立生成，不缓存、共享或复用他人的 state。
- JSON 节点固定为 `<script type="application/json" id="roncin-dingtalk-login-config">`，复用 `GetDingTalkLoginConfigResponse` 的 ProtoJSON camelCase 信封。
- 仅注入数据，不输出外部 iframe、临时执行脚本或二维码秘密到构建产物。标准 JSON 转义 `<`、`>`、`&` 等字符，防止闭合 script。
- 配置失败随 HTML 返回固定中文错误信封并保留账号入口，不自动发第二次请求覆盖本次校验状态。
- 钉钉 UA 在 Go、Vite 和前端预热三层均跳过扫码配置，仍返回 no-store 登录 HTML；UA 仅控制体验，身份由独立免登换码验证。
- HEAD、业务页、回调、资源、API、健康检查不生成扫码配置或覆盖 state Cookie；登录 HTML 不返回条件缓存 304。
- `preloadDingTalkLoginConfig` 一次性消费 HTML JSON 并移除节点。已有合法配置的首屏配置 GET 为零；无节点的 SPA 返回登录通过生成客户端获取配置。
- 保留 React Query 固定配置 key、5 分钟 gc、无重试/焦点刷新/重连刷新策略，避免挂载重复请求。配置请求不消费 AbortSignal，防止迟到响应仍覆盖 Cookie。
- 手动刷新串行 `refetch({cancelRefetch:false})`；先退役当前尝试，再取新配置。回调换码开始时只标记配置失效，使用 `refetchType:'none'`，不覆盖回调所需 Cookie。

## 4. 本地二维码与轮询生命周期

- 扫码模式采用紧凑二维码卡（200px 圆角卡、内部 180px 真码、220px 固定状态区），配置加载、刷新、已扫待确认、错误和真实二维码共用该区域；不因状态切换撑高卡片。二维码卡本身是可键盘操作的刷新入口，加载期间禁重复操作，不再提供底部独立刷新按钮与管理员说明。
- 加载时显示区域内居中的白底进度提示；数据就绪后直接绘制真实二维码，不显示钉钉灰色骨架。
- 轮询与配置、换码一样属于公开认证链路，豁免业务工作区展示 scope；工作区清理不得吞掉合法扫码响应。
- 轮询通过 React Query 管理。每个 state 独立查询身份，同一尝试至多一个在途轮询；长轮询等待不是页面自己模拟的扫码状态。
- WAITING 保持真实码且可点击刷新；SCANNED 提示手机确认并继续轮询；CONFIRMED 只消费一次；EXPIRED 停止旧尝试并在卡内提示点击刷新，不立即自动换码掩盖失效状态。其他错误停止并在卡内提供重试，不使用宽红色 Alert 替换二维码区，不静默改成等待。账号入口始终可达。
- 配置本地期限为 5 分钟，实际上游过期也以 EXPIRED 结果处理。不能把本地期限声称为钉钉返回的上游 TTL。
- 刷新、认证开始和真实卸载使旧尝试失效；旧请求即使迟到也不得导航、改变新尝试状态或替换提交基线。
- 密码模式保持扫码组件挂载并隐藏扫码区，暂停轮询，隐藏期间不消费或缓存迟到的授权结果。密码提交与工作台选择期间不接受扫码。
- 浏览器隐藏时暂停轮询，切回检查本地期限。切回有效且未消费的尝试复用配置；失效、已消费或配置错误则按显式生命周期重新创建，不新造配置 query key 绕过去重。
- StrictMode 重挂载不得重复创建上游尝试或重复消费授权码；取消轮询允许传 AbortSignal，因为该响应不写 Cookie，但仍必须检查 state 身份和 active 状态。
- 前端仅用 `qrCodeUrl` 绘制；不能读取或自行提交 clientSecret、上游 Cookie、设备头和轮询会话实现细节。

## 5. 两种登录方式的参考界面

- 扫码与账号页按实际钉钉参考页对齐：桌面卡 480×600、圆角 24；切换 pill 184×30、每项 88×26；图标 48；标题 20/30、副标题 16/30。Roncin 保留自己的品牌内容，不复制没有实际业务的注册、语言菜单等链接。
- `min(viewport width,height)<=710` 使用无外框 fullview；其他视口宽度达到 1040 时左右等分品牌区和登录区，否则单卡居中。734×870 时卡 x127/y115；1440×900 时右卡 x840/y130。
- 二维码卡 200、内部真码 180，位于卡内 top255；账号输入 318×56、圆角20，卡内 top264；主按钮 318×48、全圆角，top332。
- 账号首步只收集手机号/用户名并执行本地必填校验，“下一步”显示密码步，不额外探测账号存在性；修改账号后清除旧密码。密码、找回密码和组织选择仍走既有业务协议。
- “自动登录”与通行密钥选项按参考行距布置；前者真实提交 rememberMe，后者关联[本站通行密钥](../../server/backend/passkey-authentication.md)，当前 HTTP IP 不满足条件时明确禁用并说明。
- 页面内自有 semantic class 隔离全局 Input/按钮装饰；不修改全局规则，不为像素对齐降低占位文字的可读性要求。

## 6. 必需验证

- gateway HTTP 替身验证固定端点、表单参数、逐尝试 Cookie 隔离、等待/已扫/过期/成功解析、未知错误、非法 URL/state/authCode、超时与取消；不在普通测试访问真实钉钉网络。
- service/biz 验证 Cookie state 不匹配先拒绝、配置 no-store、生成失败不泄漏敏感内容、轮询成功只传授权码，人员准入仍由原换码流程执行。
- Go/Vite 验证同响应配置及 Cookie、JSON 安全转义、客户端 UA 跳过、相邻路径不预热、无外部 iframe。
- web 验证真实本地 QR 绘制、配置预热去重、刷新串行、旧请求隔离、扫码/密码切换、隐藏暂停、过期、失败与单次确认，以及原回调的多工作台流程。
- 真实浏览器测量加载/真实码/刷新状态的外卡和二维码卡几何一致，且不存在独立底部刷新按钮；DOM 无外部 iframe。
- 当前应用的自然生成与 waiting 请求已用真实公开配置验证；2026-11-02 用户在当前开发登录页完成手机扫码并确认已进入后台。该记录覆盖本次实际应用授权链路；后续协议或应用配置变化仍需重新真机验证，静态成功分支和 HTTP 替身不能代替该验收。

## 7. 错误与正确示例

```text
错误：将 authorizeUrl 直接编码成二维码，或 load/固定延时后声称二维码可用。
正确：使用 generate_qrcode 返回的真实扫码链接，由本站 QRCode 绘制并消费真实轮询结果。

错误：共享 http.Client CookieJar，导致不同浏览器的扫码会话串用。
正确：每 state 独立上游客户端与 CookieJar，只由本站 Cookie 绑定的请求访问。

错误：轮询成功就信任上游用户字段或直接导航任意 result.url。
正确：校验本站回调与 state/authCode，继续走原 OAuth 换码和人员准入。
```
