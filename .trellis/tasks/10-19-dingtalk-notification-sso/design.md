# 钉钉内通知免登技术设计

## 授权与变更边界

用户明确授权「创建任务，然后你写好之后，review，合并，push」。按此授权完成规划后直接实施，不再次请求流程批准。现有差距是业务链接进入登录页后只能扫码/账号登录；行为所在位置为登录页、客户端 SDK、认证 provider 和开发/生产 HTML 预热。只增加客户端认证通道并复用现有会话及组织准入，无数据库、锁或权限模型变化。

## 数据流和协议

1. 有效会话照常打开业务目标；401 保存完整 pathname/search/hash 并进入登录页。
2. UA 包含 DingTalk（大小写不敏感）时，Go HTML、Vite HTML 和前端预热均跳过扫码配置及 iframe。UA 只决定体验，不是身份依据；登录 HTML 继续 no-store。
3. 钉钉内默认展示客户端免登面板，一次尝试先通过生成客户端 GET `/api/v1/auth/dingtalk/embedded-config`，数据 `enabled, corpId?, state?, expiresAt?`。禁用只返回 enabled=false。
4. 按需导入固定版本 dingtalk-jsapi 的 `entry/union` 与 `api/runtime/permission/requestAuthCode`，使用 `corpId`（以已核验 SDK 实际参数 `corpId`/`corpID` 类型为准）调用旧版 runtime.permission.requestAuthCode。只采用这一协议，不混用新版 dd.requestAuthCode 或 OAuth 授权码。
5. POST `/api/v1/auth/dingtalk/embedded-login` 请求 `{authCode,state}`，复用 DingTalkLoginResponse。服务端独立 state Cookie（五分钟、HttpOnly/Secure/SameSite/Path 与既有政策一致），换码时清除此 Cookie；两个接口 no-store。保留扫码 state Cookie 独立。
6. provider 复用企业 app token，POST `https://oapi.dingtalk.com/topapi/v2/user/getuserinfo?access_token=...`，JSON `{code:...}`。必须得到非空 unionid、userid、name；CorpID 来自固定企业 app token 配置，不信任客户端传身份。错误映射领域错误，不输出上游凭据或原始报文。
7. 扫码和客户端解析身份后进入共享私有准入函数：严格 unionID/userID 匹配已同步账号，复用未同步/待分配/停用状态、有效组织筛选、会话创建和审计。无自动建人、手机号绑定或授予权限。
8. 单工作台整页导航到安全站内目标；多工作台沿用 OrganizationPicker，明确选择和既有 SwitchOrganization，再回完整目标。不从订单 URL 推断组织。

## 前端生命周期

- 自动尝试一次；失败/禁用/SDK 空码/超时明确展示中文状态，提供显式重试和账号登录。普通浏览器不加载 SDK，保留扫码/密码流程。钉钉内账号模式不挂扫码组件。
- 尝试 Promise 与有效订阅分离，StrictMode 重订阅同一尝试，不重复取码/消费。真实卸载、切账号和新尝试使旧结果失效；发换码前复核仍有效，结果回调和组织选择后的导航同样检查归属。
- 配置 Query 使用稳定 key、无自动重试/焦点刷新，显式重试串行取得新配置；换码为 mutation。授权码/state 仅内存，禁止 storage/URL/log。
- 登录模块共用站内回跳校验：URL 解析后同源根相对地址，拒绝 //、反斜杠外跳及登录/回调循环，保留 query/hash。同步修复扫码 callback 现有较弱校验。
- 统一 requestClient 对新增两个端点精确公开豁免。组织切换仍为认证后操作，遵守正常工作台隔离。

## 文件与所有权

- 后端实施：server/api/auth/v1/auth.proto、biz/auth.go、data/dingtalk.go、service/auth.go、相关测试；生产 webassets 的 UA 分流和测试；所有 Proto/OpenAPI/前端客户端生成物。
- 前端实施：web/src/app UA 与预热、登录模块与安全回跳、requestClient 和测试、web/config HTML 分流和测试、SDK package/lock。
- 主会话：任务、规范同步、完整门禁与构建、review 协调、归档、合并和 push。
- 不变的扫码准入通过原测试复验；共享 helper 仅消除同一会话/账号准入重复。

## 生成、迁移与运行边界

先 proto，`make -C server api`，再 `pnpm run generate:web-client`。生成文件仅生成器更新；无 Schema 或配置迁移。现有应用必须开通网页应用能力、可信域名与可见范围，真实客户端运行另记；本任务不擅自修改开放平台、不发送消息。Cookie state 是与既有一致的浏览器校验，不宣称持久化一次消费挑战。

## 验证与交付

功能完成后补边界测试（禁止 TDD），定向 Go/Vitest、Biome/tsc 后独立 trellis-check。跨层认证契约与入口/依赖变化需要 `pnpm run check:fast` 及 `pnpm run build`。浏览器模拟验证钉钉内无扫码请求、单多组织回跳、显式失败/重试/账号切换。真实 SDK+真实用户免登未实测时明确标注限制。功能与生成物分组提交；main 快进包含前任务 Markdown，安全删除本次和已合并 Markdown 分支，push origin main，核对远端一致性。
