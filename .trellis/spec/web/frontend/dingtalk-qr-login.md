# 钉钉扫码主登录契约

## 1. 适用范围

适用于 `/user/login` 的默认扫码、管理员密码切换与 `/user/login/dingtalk/callback`。复用钉钉 OAuth2 和人员状态机，不通过扫码创建或开通人员。

## 2. 签名

```text
GET /api/v1/auth/dingtalk/login-config
data: enabled, authorizeUrl?, expiresAt?
回调：/user/login/dingtalk/callback?authCode=...&state=...

GET /user/login（首次 HTML 导航）
响应头：Cache-Control: no-store、Pragma: no-cache
响应正文：<script type="application/json" id="roncin-dingtalk-login-config">...</script>
节点内容：复用 GetDingTalkLoginConfigResponse 的 ProtoJSON camelCase 信封
合法启用配置：在 #root 外输出 <iframe id="roncin-dingtalk-login-frame" ...>
iframe 属性：官方授权 src、data-expires-at（Unix 毫秒）、referrerpolicy=no-referrer
早期导航标记：固定 onload/onerror 仅写 data-navigation=loaded/failed，不拼接配置到脚本
配置生成失败：{"success":false,"code":500,"message":"钉钉登录配置加载失败，请稍后重试"}
```

```text
扫码端点：https://login.dingtalk.com/oauth2/auth
authorizeUrl 参数沿用服务端配置，追加 iframe=true
Query key：['auth', 'dingtalk-login-config']
```

## 3. 契约

- 默认展示真实钉钉 OAuth2 iframe，次级“管理员账号登录”切换密码，可返回扫码。保留品牌资源，配置失败不静默改成密码表单。
- HTML 入口通过 `preconnect` 和 `dns-prefetch` 提前准备 `https://login.dingtalk.com`
  连接；仅在本次动态登录 HTML 响应中输出带 state 的 iframe，构建产物不含授权 URL。
- 首次直接打开 `/user/login` 时，Go 服务复用认证 Service 创建本次配置和 Cookie，
  在同一个 HTML 响应中返回。每次导航独立生成，不缓存配置或共享 state；配置
  使用现有 Proto DTO 序列化，静态资源层只接收 JSON 和 Cookie，不做领域转换。
- HTML 内嵌配置节点是不可执行的 JSON；插入前必须转义 `<`、`>`、`&` 等字符。
  iframe 的 src 使用标准 URL 工具追加 `iframe=true` 并进行 HTML 属性转义；
  禁止把授权 URL、state 或配置拼入可执行脚本。只包含公开登录配置，
  不包含 clientSecret、会话令牌或换码令牌。
- Go 与 Vite 仅对合法、启用、未到期且回调同源的配置输出提前加载的 iframe，
  与内嵌 JSON 和 Cookie 对应同次响应。配置失败、禁用、非法或过期不提前导航。
- 提前加载的 iframe 保持在 `#root` 之外的 HTML 原位置，避免 `createRoot`
  清空节点或 DOM 重挂载重置浏览上下文。前端匹配授权参数与到期信息后认领原实例，
  不改 `src`、不改父节点，通过位置样式与扫码占位区对齐。
- iframe 在认证监听安装前隐藏，接管后显示；早期 `load/error` 通过固定 dataset
  标记传给前端，接管后沿用现有导航状态语义。不得把 `load` 标记解释成二维码就绪。
  几何对齐必须覆盖窄屏、窗口变化、卡片移动与文档滚动，且不持续进行无变化的 DOM 写入。
- 刷新、过期、管理员模式、认证开始和路由退出均清理提前加载的实例；StrictMode
  的模拟清理与重挂载不得销毁仍待认领的实例，真实卸载后不得遗留 iframe 或监听。
- 内嵌配置对应的 Cookie 与 HTML 必须同一次响应写入，保留 HttpOnly、Secure、
  SameSite、Path、有效期属性。登录页 GET 忽略条件缓存，不返回 304；HEAD、
  业务页、回调页、静态文件、API 与健康检查不生成登录配置或覆盖 Cookie。
- Vite 开发服务器只处理同一路径的 HTML GET，在服务器侧请求既有配置接口，
  将响应 Cookie 原样转发并把同一信封注入经 Vite 转换的 HTML；沿用当前开发/
  测试/预发代理目标，不新增浏览器配置请求，不把配置写回磁盘或构建产物。
- React 首次渲染前的配置预热，优先一次性消费并移除 HTML 内嵌节点，通过已有
  QueryClient、固定 key 和查询定义缓存。内嵌成功配置的首屏配置 GET 次数为 0。
  节点不存在时（SPA 返回登录页、静态构建预览）仍通过生成客户端获取配置。
  节点存在但内容无效、配置生成失败或参数不合法时直接显示错误，不自动再 GET。
- 已完成和仍在途的预热不能在组件挂载后重复请求；失败不能借挂载隐式重试。
  其他业务页与钉钉回调入口不预取登录配置。内嵌节点只能消费一次，手动刷新、
  到期自动刷新或管理员返回扫码仍获取新配置。
- 配置 Cookie、state、expiresAt 来自同一次服务端响应，响应 Cache-Control=no-store。clientSecret 与换码令牌仅在服务端；前端不记录授权码或 state。
- React Query 固定 key 去重，配置请求不消费 AbortSignal，避免取消后的响应仍更新 Cookie 并覆盖新实例。禁止焦点/重连自动刷新及自动重试；管理员模式返回扫码时显式获取新配置，刷新串行 refetch(cancelRefetch:false)。
- 预取缓存保留 5 分钟，组件挂载默认复用本次配置；已经过期的配置必须重新获取，
  不能重新挂载过期二维码。管理员返回扫码的显式请求与组件挂载共用在途请求，
  不为绕过缓存另造查询 key。
- 钉钉回调开始换码时，将配置查询标记失效且 `refetchType: 'none'`，回调期间
  不获取新配置、不覆盖换码所需的 Cookie。后端换码会清除一次性 state Cookie，
  因此认证失败、人员待开通或多组织尚未选择后返回登录页，都必须重新获取配置；
  挂载检查同时识别缓存失效与配置到期，不能复用已消费配置。
- 手动刷新先同步卸载旧 iframe 和监听，再请求新配置。每个扫码实例以 state 为 key；父回调引用稳定，监听清理不能重置已处理实例造成重复认证。
- 配置必须是官方 OAuth2 端点，回调 origin 等于本站且路径精确匹配既有 callback。解析公开参数使用 URL/URLSearchParams，不能手工编码或生成普通授权 URL 二维码冒充扫码协议。
- message 只接受精确 origin=https://login.dingtalk.com、source=当前 iframe.contentWindow、正确结构及 success 标记。成功 redirectUrl 必须匹配当前回调 origin/path，只有一个相同 state 和一个非空 authCode。
- 只处理一次成功；卸载、退役、过期或旧 iframe 消息忽略。校验后构造本站固定回调 URL，不直接导航消息传来的外链。后端再次验证 Cookie state 与企业身份。
- 过期先退役旧实例，再自动获取新配置，不轮询；新配置立即过期时限流并显示
  “二维码已过期，请刷新”。切回前台和焦点恢复复核期限，管理员入口始终可达。
- iframe 的 load 仅表示导航事件结束，浏览器在网络失败时也可能触发，error 不能可靠
  检测跨源加载失败。现有官方协议没有二维码就绪事件，禁止把 load 当成二维码可用。
  扫码区域始终显示未显示/加载失败时检查网络、刷新或使用管理员入口的恢复提示。
  iframe 从挂载开始可见，加载提示放在二维码区域之外，不覆盖或隐藏已绘制内容；
  导航超过 15 秒则显示加载较慢提示，不据此退役仍可能成功的扫码实例；
  明确失败消息、error 和过期才退役。手动刷新仍先卸载旧实例。
- 回调仍保留未同步、待分配、停用提示；认证成功一个工作台直进，多个工作台明确选择，组织 ID 与候选来自本次响应的同一份本地状态。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 钉钉未启用 | 显示未启用，管理员入口可达 |
| 首次登录 HTML 携带合法内嵌配置 | 消费一次并移除配置节点，认领 HTML 已请求的同一 iframe，配置 GET 为 0 |
| 前端主程序被阻塞 | HTML iframe 已请求授权页；认证监听安装前隐藏 |
| 前端接管提前加载的 iframe | 同一 DOM 节点、同一浏览上下文、仅一次授权导航，不改 src 或父节点 |
| 提前 iframe 已触发 load/error | 接管读取 dataset 标记，不再次等待 load；error 按既有失败恢复处理 |
| 切换密码模式或离开登录页 | 旧实例退役并移除，不残留遮挡层、监听或定时器 |
| 内嵌配置失败、非法 JSON 或参数不合法 | 显示配置错误，不自动 GET，保留管理员入口 |
| 连续两次 HTML 导航 | 两组不同 state，分别与各自响应 Cookie 一致，均为 no-store |
| 登录 HTML 附带条件缓存请求头 | 返回新 HTML 与本次 Cookie，不返回 304 |
| 首屏配置预取在途或已完成 | 挂载复用同一请求或结果，只写入一次本次 state Cookie |
| 首屏配置预取失败 | 挂载展示失败，用户显式刷新或切回扫码后才重试 |
| 已缓存配置到期后挂载 | 获取新配置，不展示过期二维码 |
| 换码后返回登录页 | 换码时只标记缓存失效，返回后获取新配置，不能复用已消费 state |
| 配置缺字段、回调不同源/错误路径 | 显示配置错误，不渲染可用扫码实例 |
| 配置失败、iframe 明确失败消息/error | 显示失败并支持显式刷新 |
| iframe 网络失败仍触发 load | 保留可见的未显示/加载失败恢复提示，不声称二维码就绪 |
| iframe 导航超过 15 秒 | 显示加载较慢提示，保留扫码与刷新能力，到期后才失效 |
| 过期 | 禁止旧实例成功回调并自动刷新；新配置立即过期时显示手动刷新提示 |
| 非匹配 origin/source/state/回调路径或重复字段 | 忽略，不触发认证 |
| 旧实例或重复成功消息 | 忽略，不重复换码 |
| 人员未同步/待分配/停用 | 既有人员状态提示，不自动开通 |

## 5. 正常、基准与错误场景

- 正常：打开扫码、扫码成功后进入本系统回调；多工作台展示选择，再通过会话轮转进入目标。
- 基准：管理员切换密码并返回扫码；配置失败时仍可使用管理员入口。
- 错误：普通 URL 二维码替代钉钉iframe；刷新时旧消息仍触发导航；根据消息直接跳转任意 redirectUrl。

## 6. 必要验证

- StrictMode、快速切换、连续刷新、旧请求响应和 state Cookie 一致性。
- Go HTML 的配置与 Cookie 对应、连续请求隔离、条件缓存与 HEAD/非登录路径边界。
- 开发服务器注入、Cookie 透传与代理目标；恶意闭合 script 的字符正确转义。
- 内嵌配置成功/禁用/错误/坏 JSON/坏参数、StrictMode 首屏零 GET 与配置节点只消费一次。
- 阻塞主程序的真实浏览器验证：授权请求先发出、接管复用原节点与浏览上下文、无第二次授权导航。
- Go 和 Vite 使用相同属性、转义及参数判断；参数顺序或编码差异不导致错误重建。
- 提前 load/error、窄屏与滚动对齐、刷新/密码切换/路由退出清理，不残留外部 iframe。
- 首屏预取在途、早于挂载完成、失败、缓存到期和管理员返回扫码时的请求去重。
- 已预热配置进入换码回调时不请求新配置，认证失败后返回登录页获取新 state。
- iframe 的 load 事件触发前已可见，加载提示与二维码区域互不遮挡。
- 非法消息、旧iframe、重复成功、到期与后台恢复、监听和定时器清理。
- 配置失败、加载失败、超时、切换模式、单/多工作台和人员各状态。
- 浏览器阻断 iframe 后触发 load，仍能看到恢复提示；慢加载后有效成功消息仍可处理。
- 真实企业应用扫码及开发代理/生产同域回调，须单独记录实测，协议源码核验不能替代。

## 7. 错误与正确示例

```text
错误：反复调用没有公开销毁接口的全局SDK；将redirectUrl直接传location。
正确：就近iframe组件管理可清理监听；精确校验后构造本站callback。

错误：刷新先GET config再保留旧扫码实例；焦点自动GET覆盖state Cookie。
正确：先卸载旧实例，固定Query key串行刷新，焦点只复核期限。

错误：把已加载 iframe appendChild 到 React 容器，或改写其 src 以对齐字符串。
正确：保留 #root 外原节点和 src，校验同次配置后仅更新定位样式，刷新时才销毁。
```
