# Research: 钉钉通知前端免登与安全直达

- Query: 核对现有登录拦截、扫码、回调、组织选择及工作台隔离，给出钉钉内免登的最薄接入点、异步边界与验证清单。
- Scope: internal（受主会话派发限制，不访问外网、不调用真实服务）
- Date: 2026-10-19

## Findings

### 已读取规范

- `.trellis/workflow.md`：研究必须持久化，实施与检查由主会话协调。
- `.agents/skills/roncin-web-stack/SKILL.md`：React Router 集中路由、生成客户端、React Query；认证成功后为重建会话而整页导航是允许的例外。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：组织/工作台是有效数据范围，钉钉身份不代替组织授权。
- `.trellis/spec/web/frontend/index.md`、`dingtalk-qr-login.md`、`state-management.md`、`hook-guidelines.md`、`quality-guidelines.md`：扫码 state 与 Cookie 同批、串行配置请求、禁止隐藏回退、身份隔离、pnpm 验证。

### Files found

| 文件 | 职责 |
| --- | --- |
| `web/src/app/AppProvider.tsx` | 首次 `/auth/me`、公开认证路径、401 保存完整目标地址 |
| `web/src/app/AppLayout.tsx` | 业务布局未登录守卫、按身份挂载工作台 |
| `web/src/requestErrorConfig.ts` | 请求级 401 回登录、业务错误提取、写入防重 |
| `web/src/pages/user/login/index.tsx` | 默认扫码/密码切换、目标存储、密码认证后组织选择 |
| `web/src/pages/user/login/dingtalk-callback.tsx` | 一次性扫码换码、人员状态、认证后组织选择与整页导航 |
| `web/src/pages/user/login/components/organization-picker.tsx` | 组织候选转换、按选择换发会话 |
| `web/src/pages/user/login/components/dingtalk-qr-login.tsx` | iframe 生命周期、稳定回调、一次认证、过期退役 |
| `web/src/app/dingtalkLoginConfig.ts` | 公开扫码配置 Query、首屏预热、串行刷新 |
| `web/src/app/dingtalkLoginProtocol.ts` | 扫码端点/回调校验，无客户端 JSAPI SDK |
| `web/src/main.tsx` | React 渲染前预热扫码配置 |
| `web/config/dingtalkLoginBootstrap.ts` | Vite 登录 HTML 同批配置/Cookie/iframe 注入 |
| `server/internal/webassets/assets.go` | 生产登录 HTML 同批配置/Cookie/iframe 注入 |
| `web/src/components/layout/OrganizationWorkspace.tsx` | 用户+组织独立 QueryClient 与请求结果隔离 |
| `web/src/utils/requestClient.ts` | 公共认证协议豁免业务请求 scope、统一客户端 |
| `web/src/utils/queryClient.ts` | 全局默认禁自动重试、禁焦点/重连自动刷新 |
| `web/config/routes.ts` | 现有公开登录路由与扫码 callback |
| `web/src/hooks/useLatestAsync.ts` | 通用异步副作用隔离工具，不能盲用于一次性免登的 StrictMode 生命周期 |
| `server/internal/data/dingtalk.go` | Markdown 订单链接目前仅含订单 path |

### 当前行为与代码模式

1. 现有会话可用时，打开订单正常经 `/auth/me` 加载用户，不必再认证；首次 401 在 `AppProvider.tsx:68` 保存 `pathname + search + hash` 到 `/user/login?redirect=...`。请求级 401 同样在 `requestErrorConfig.ts:28` 保存完整目标。`AppLayout.tsx:53` 的补充守卫只跳裸登录页，会丢失目标；如果本次触及该路径应统一完整目标处理。
2. 公开认证页清单在 `AppProvider.tsx:19`，当前只有 `/user/login` 与 `/user/login/dingtalk/callback`。若另加免登路由，需同步 `web/config/routes.ts:9` 与此清单，否则免登入口会被重定向；现有 `/login` 别名未在公开清单内，不建议把它当新能力入口。
3. 密码登录页面 `index.tsx:30` 的 `safeRedirect` 检查根相对地址与 URL 解析后同源；成功后 `index.tsx:57` 用整页导航重建会话。扫码在 `index.tsx:98` 将目标存入标签页 sessionStorage，再站内进入 callback。
4. 扫码 callback 在 `dingtalk-callback.tsx:20` 消费 storage 目标，只验证 `/` 开头且非 `//`，比密码页面弱：例如反斜杠可改变浏览器 URL 解析语义。建议同一登录模块内共享完整同源解析函数，补 `try/catch`、拒绝登录/回调目标导致循环，返回 `/`；不要复制这段弱校验到免登。
5. callback 在 `dingtalk-callback.tsx:38` 用 `useMutation`，检查有效 data/status/currentUser，再更新初始状态；`handled` 在 `:92` 保证 StrictMode 单次 mutate。`onSuccess` 的组织选择和整页导航在 `:60`。现有组件没有完整的真实卸载后 UI 副作用门禁，新免登不要仅照抄这部分。
6. 人员状态在 `dingtalk-callback.tsx:105` 显示“人员尚未同步”“尚未分配组织”“账号已停用”；失败有明确页面与“返回登录”。认证失败不能隐式改账号密码，更不能自动循环申请授权码。
7. 组织候选转换在 `organization-picker.tsx:15`，优先服务端候选，否则 `currentUser.organizations`。当前组织直接进入，其他组织必须 `authServiceSwitchOrganization`，收到有效用户再进入（`:54`）。只要组织多于一个，现有扫码与密码都要求明确选择；免登应复用同一组件和语义。
8. `OrganizationWorkspace.tsx:9` 以 `userID:organizationID` 作为工作台身份；`:22` 身份不完整时不挂业务查询，`:37` 建立独立 QueryClient，layout cleanup 先停工作台请求 scope 再清缓存。`requestClient.ts:84` 明确把公开登录协议排除在业务 scope 外，新增免登配置/换码端点也应加入；认证后整页重建自然保留隔离。
9. `server/internal/data/dingtalk.go:204` 仅使用已校验 callback 的 origin 和订单 path 组成链接；没有组织 ID。不能从订单 URL 推断该订单归属并绕过组织授权。最薄方案维持多组织选择；如需求升级为无需选择准确进入通知组织，必须另行设计经后端校验的目标组织契约，不能在前端静默切公司。

### 扫码预热必须覆盖的三层

- `main.tsx:24` 在 React 首次渲染前对 `/user/login` 预热；`dingtalkLoginConfig.ts:72` 仅检查 pathname，没有钉钉环境检查。只在登录组件里切 UI 并不能阻止此请求。
- `web/config/dingtalkLoginBootstrap.ts:41` 在 Vite 收到登录 HTML 请求时就先向后端扫码配置接口请求，并注入 Cookie 和 iframe（`:56`、`:84`）；没有 User-Agent 判断。
- 生产 `server/internal/webassets/assets.go:62` 对登录 GET 执行 `serveLoginEntry`，`:80` 已生成配置 Cookie，`:94` 插入扫码 iframe。React 执行时授权页可能早已请求。
- 因此推荐按同一“是否钉钉客户端”规则，生产与 Vite 对钉钉 UA 跳过扫码 bootstrap，前端跳过扫码预热且不挂扫码组件。UA 只决定体验分流，服务端真实身份仍以换码结果为准。所有跳过路径仍需保留 no-store 登录 HTML；不能遗漏开发代理与生产行为一致性。
- 若使用专属 `/user/login/dingtalk/sso` 页面，直接通知入口的该路径自然无现有扫码预热；但从普通业务订单经 401 进入 `/user/login` 时仍需环境分流，而且直接打开 `/user/login` 仍会先触发服务器扫码 bootstrap。不要只测新专属路由便宣称钉钉内扫码零请求。

### SDK 与最薄实现建议

- 当前 `web/package.json` 没有钉钉 JSAPI 依赖；仓库前端搜索未发现 `requestAuthCode`、`runtime.permission`、`dd` JSAPI 封装或钉钉环境检测。现有扫码完全是官方 OAuth iframe，不能复用它来获取客户端免登码。
- 前端环境判断建议放 `web/src/app/` 的小模块，供 main/config/Login 共用，不依赖页面模块；Vite 可复用无 DOM 依赖的纯 UA 判断。后端保持等价的传输层 UA 分流。
- 建议仍通过现有登录路由分流：钉钉中渲染独立小型免登组件，普通浏览器维持扫码/账号；若为了职责清晰采用专属免登页面，则注册一个公开路由并确保上述三层同时分流。不要把复杂 SDK 细节塞进 `AppProvider` 的全站状态逻辑。
- SDK 只在钉钉环境按需加载，固定可信脚本地址或经过核验的固定版本依赖。用单个模块级 in-flight Promise 去重脚本加载，处理已存在 SDK、load/error/超时；普通浏览器不得加载。JSAPI 名称、参数、桌面支持与配置要求需由主会话依据官方文档独立核验，本研究没有外网证据。
- 公开免登配置通过生成客户端+React Query 获取；换码用 `useMutation`，采用 `skipErrorHandler:true` 和 `meta:{silent:true}` 在当前页面显示明确结果；响应必须验证有效 data/status/currentUser，不能把 `undefined` 当成功。
- 成功复用人员状态、组织选择与整页 `window.location.replace(safeTarget)` 会话重建。目标只保存站内路由，不把授权码、令牌或 SDK 错误原始数据写 storage/日志/跳转 URL。普通浏览器保持原有登录入口。

### 单次认证、StrictMode 与异步过期

- “申请客户端授权码+后端换码”构成一次认证尝试，必须串行；自动发起只有一次，网络失败不自动重试，用户明确点击重试才创建新尝试和新授权码。不要重复消费同一个一次性 code。
- `dingtalk-callback.tsx:92` 的 ref 单次门禁可作参考，但若 SDK load/code Promise 在 effect cleanup 后被丢弃，StrictMode 第二次 effect 又被 handled 阻止，就会永远停在 loading。要把“尝试 Promise 去重”和“当前订阅是否仍可应用结果”分开：StrictMode 重订阅同一尝试 Promise；真实离开后忽略其 SDK/网络结果与导航，重新进入建立新尝试。
- 推荐数据查询使用 Query，认证尝试由稳定 ref 保存 Promise；effect 每次建立自己的订阅清理，恢复订阅时拿同一结果。SDK 回调只能 resolve/reject 该尝试，不能直接 mutate。发起后端换码前再次检查当前页/尝试是否仍有效。若用户已切账号或退出，迟到 SDK code 不得再发换码。
- 不要认为 React Query 的 hook 级 `onSuccess` 在组件卸载后不会运行；需要把导航/组织状态写入放到仍存活的观察者处理，或使用明确的活动尝试门禁。code、换码、组织切换的失败和成功都必须归属当前尝试。
- 现有 `useAsyncGuard` 在 `useLatestAsync.ts:52` cleanup 时 invalidate；若将它与一次 `handled` 组合，会让 StrictMode 首次尝试失效而第二次不启动。不要盲用，先写清上述尝试/订阅模型。
- `queryClient.ts:54` 已全局禁止自动重试、焦点重查、重连重查，调用点应沿用，不能为免登再加后台认证循环。

### 建议测试清单

1. 钉钉环境：获取配置→SDK→授权码→后端换码→单组织整页回原 `pathname/search/hash`；无扫码配置 GET、无 iframe、无扫码 state Cookie。
2. 普通浏览器：不加载 SDK、不调用免登接口，原扫码和密码照常；非登录业务页已有会话时不启动免登。
3. StrictMode 重挂 effect 与父层 rerender：SDK 只加载一次、授权码只申请一次、换码只调用一次且成功导航一次。
4. deferred SDK resolve/reject：切账号/离页后不发换码、不提示、不导航；deferred 换码 resolve/reject：离页后不应用 UI；显式重试新 code，旧尝试不能覆盖新结果。
5. SDK 未提供/脚本失败/超时、配置禁用/失败、换码网络失败、缺 data、未知/无状态、认证成功但缺 currentUser：均显示中文明确失败并可显式重试或切账号，不隐式循环。
6. 未同步、待组织分配、已停用三类人员状态：不导航、不创建人员；业务角色与权限仍由后端会话决定。
7. 多组织：候选均来自本次响应；选择当前组织不切换，选其他组织先换会话；切换失败留在当前页；目标含 search/hash 在选择后保持。
8. 安全回跳：站内路径允许；`https://evil`、`//evil`、反斜杠变体、解析失败、指向登录/回调的循环地址拒绝。扫码 storage 回跳与新免登共用同一验证，不只测新增组件。
9. 生产与 Vite bootstrap：钉钉 UA 登录 GET 不生成扫码配置/Cookie/iframe，普通 UA 保留现有同步预热；HEAD/回调/API/业务页不预热；响应 no-store。
10. 新公开认证端点继续豁免工作台 scope；相邻既有 `requestClient.workspace.test.tsx`、密码重置、扫码过期与模式切换测试保持通过。

可复用测试范本：`dingtalk-callback.test.tsx:126` 单组织、`:150` StrictMode 换码与 state 失效、`:222` 多组织；`index.test.tsx:184` 密码与扫码相互隔离、`:399` 回跳存储；`dingtalkLoginConfig.bootstrap.test.tsx` / `config/dingtalkLoginBootstrap.test.ts` / `server/internal/webassets/assets_test.go` 三层预热；`requestClient.workspace.test.tsx:247` 公开协议豁免。

建议先跑受影响定向 Vitest 与修改文件 Biome；涉及路由/契约/生成客户端再跑 `pnpm --dir web tsc`；最终按根 AGENTS 完整门禁。若改 Vite bootstrap、SDK 依赖或生产入口，构建验证有实际必要。

## External references

- 未访问外部网站或真实服务，按派发约束执行。不能把上述 SDK 方案当成已核验的钉钉 API 合约。
- 本地依赖：React `^19.2.7`、React Query `^5.103.1`、React Router `^8.4.0`、Vite `7.2.2`（见 `web/package.json`）。

### 主会话确认的实施合约（2026-10-19 补充）

- SDK 仅选择 `runtime.permission.requestAuthCode`；按需导入 `dingtalk-jsapi/entry/union` 与 `dingtalk-jsapi/api/runtime/permission/requestAuthCode`，参数使用 `corpId`。主会话确认配套后端企业 `getuserinfo`；不能混用新版 `requestAuthCode`（V2）或增加协议回退。
- 公开配置：`GET /api/v1/auth/dingtalk/embedded-config`，数据 `enabled, corpId, state, expiresAt`。
- 免登换码：`POST /api/v1/auth/dingtalk/embedded-login`，请求 `{authCode,state}`，复用既有 `DingTalkLoginResponse`。前端依旧只消费生成客户端。
- 多工作台沿用明确选择，普通浏览器行为保持，钉钉内允许用户明确点击“账号登录”。生产 HTML、Vite HTML 与前端预热一起完成环境分流。
- 此处是主会话传入的已选设计，本研究未独立查阅 SDK 外部文档；实施者按最终设计文档核对类型签名与包路径。

## Caveats / Not Found

- 未发现现有客户端免登 SDK、免登入口或钉钉环境检测；需要后端与官方文档研究提供真实免登配置/换码合约。
- 本研究没有更改产品代码、执行 Git 操作、读取私密配置或发送通知。
- 通知链接没有组织目标；多工作台明确选择属于现有授权边界，不能通过前端猜测补齐。
- 文件行号以研究时工作区为准，后续并发实施可能改变行号。
