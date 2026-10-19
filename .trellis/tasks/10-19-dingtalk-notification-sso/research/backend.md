# Research: 钉钉通知点击免登的后端接入

- Query: 在现有钉钉扫码认证之上增加 H5 客户端免登入口，复用会话、账号状态与多工作台准入，定位最薄契约、生成与测试影响。
- Scope: mixed（内部源码为主，官方接口文档辅助）
- Date: 2026-10-19

## Findings

### 1. 当前认证链路与可复用部分

1. `server/api/auth/v1/auth.proto:31` / `:36`：扫码公开接口为 GET `/api/v1/auth/dingtalk/login-config` 与 POST `/api/v1/auth/dingtalk/login`，后者请求 `auth_code + state`，返回 `DingTalkLoginResponse`。
2. `server/internal/service/auth.go:78`：`prepareDingTalkLoginConfig(ctx)` 调 `StartDingTalkLogin()`，将 authorize_url 和 expires_at 返回浏览器，写入五分钟 state Cookie；`:94` 登录读取 Cookie，先以 Set-Cookie 清除再换码。
3. `server/internal/biz/auth.go:588`：`StartDingTalkLogin() (bool, string, string, time.Time, error)` 用 `newSessionToken()` 生成随机 state，交给 provider 构造扫码 URL。
4. `server/internal/biz/auth.go:603`：`LoginDingTalk(ctx context.Context, authCode, state, expectedState, userAgent string) (*DingTalkLoginResult, error)` 先 enabled 检查、非空检查及常量时间 state 比较，再 `ResolveIdentity`。
5. `server/internal/data/dingtalk.go:107`：当前 `ResolveIdentity` 是 OAuth 授权码专用：`/v1.0/oauth2/userAccessToken` → `/v1.0/contact/users/me` → 企业应用令牌 → `/topapi/user/getbyunionid`。`token.corpId` 必须与配置的企业匹配（`:124`）。H5 微应用码应有单独解析方法，不能猜测复用 OAuth 换码接口。
6. `server/internal/biz/auth.go:617` 到 `:641`：从可信 identity 查询已同步账号，然后依序处理未同步、待分配、停用，查询有效工作台，取稳定排序首项建立初始会话。建议抽为私有 `loginDingTalkIdentity(ctx, identity, userAgent)`，由扫码与客户端免登共同调用。这样不复制账号准入规则。
7. `server/internal/biz/auth.go:644`：`createSession(...)` 复用随机会话令牌、仅持久化哈希、TTL、UA、权限解析及审计；建议仍用现有 `auth.dingtalk.login` 审计，若确需区分渠道再明确增加来源，不额外改变现有契约。
8. `server/internal/data/auth.go:474`：`CreateSession` 已以 `WithTx` 在用户 ForShare 后复核账号 enabled / pending_assignment、工作台资格，再写 session 与 audit。无需新会话表、迁移或另一套事务实现。
9. `server/internal/service/auth.go:106`：结果 DTO 与正式 session Cookie 的写入可以提取私有 helper，两个登录入口共用。客户端免登继续返回现有 `DingTalkLoginResponse` 足够。

### 2. 人员绑定和组织权限不可绕过

- `server/internal/biz/auth.go:114`：`DingTalkIdentity` 含 `UnionID, UserID, CorpID, Name, Email, AvatarURL`。
- `server/internal/data/auth.go:126`：`FindDingTalkCredential(ctx, identity)` 要求 unionid 与 userid 都非空，按 unionid 找本地已同步人员，再严格核对 userid。不能只信任前端传 userid，不能按姓名/手机号自动绑定，也不应为免登修改现有查询规则。
- 主代理已核对并裁定：使用单一 `runtime.permission.requestAuthCode`，服务端固定调用 `/topapi/v2/user/getuserinfo`，响应 `unionid + userid + name` 全部必填校验后直接生成身份；不添加新旧 SDK 双协议、不加可选详情补读、不放宽为 userid 单匹配。
- `server/internal/data/auth.go:325`：工作台候选的唯一仓储为 `ListEnabledMembershipOrganizations`；`server/internal/biz/auth.go:726` 统一排序；`server/internal/data/auth.go:370` 从启用 membership/组织收集角色，再按所选工作台投影权限。
- `server/internal/biz/auth.go:138` 登录结果已有 Principal，其中 Organizations 是全部有效工作台；无需专为客户端入口新增 organization_choices 字段。前端可像现有钉钉登录一样派生选择候选。
- 多工作台登录必须沿用选择页契约。后台建立首项初始会话属于实现细节，不应把通知目标推断为授权工作台、不应暗中跳过选择或静默切换。选其他组织仍走 `SwitchOrganization` 会话轮转（`auth.go:144` service）。
- `server/internal/server/auth.go:57` 起：正常请求仍经过会话、权限和实体所属组织校验，通知链接不能作为授权凭据。现有订单详情可在授予的组织范围内临时定位目标订单，但保留 `WorkspaceOrganizationID`，不能把它当实际切换。

### 3. 建议契约（已按主代理单一旧版 H5 协议裁定修订）

新增独立公开 RPC，保留扫码协议单一职责，示例：

```proto
rpc GetDingTalkEmbeddedConfig(GetDingTalkEmbeddedConfigRequest)
    returns (GetDingTalkEmbeddedConfigResponse) {
  option (access.v1.rule) = {mode: ACCESS_MODE_PUBLIC};
  option (google.api.http) = {get: "/api/v1/auth/dingtalk/embedded-config"};
}
rpc DingTalkEmbeddedLogin(DingTalkEmbeddedLoginRequest) returns (DingTalkLoginResponse) {
  option (access.v1.rule) = {mode: ACCESS_MODE_PUBLIC};
  option (google.api.http) = {post: "/api/v1/auth/dingtalk/embedded-login" body: "*"};
}
message GetDingTalkEmbeddedConfigRequest {}
message DingTalkEmbeddedConfig {
  bool enabled = 1;
  optional string corp_id = 2;
  optional string state = 3;
  optional string expires_at = 4;
}
message GetDingTalkEmbeddedConfigResponse {
  bool success = 1;
  int32 code = 2;
  string message = 3;
  DingTalkEmbeddedConfig data = 4;
  string trace_id = 5;
}
message DingTalkEmbeddedLoginRequest {
  string auth_code = 1 [(google.api.field_behavior) = REQUIRED];
  string state = 2 [(google.api.field_behavior) = REQUIRED];
}
```

- 由新配置返回 state，是因为 JSAPI 不像扫码授权 URL 那样负责回传自定义 state；浏览器持有它，在同源 POST 时显式提交。
- 建议 provider 接口加 `EnterpriseID() string`、`ResolveEmbeddedIdentity(context.Context, string) (*DingTalkIdentity, error)`。`EnterpriseID()` 已在 `server/internal/data/dingtalk_directory_gateway.go:12` 实现，可直接复用；仅补新换码实现。所选旧版 SDK 只需 corpID，无需公开 ClientID。
- 域层增加 `DingTalkEmbeddedConfig` 纯 Go 配置对象、`StartDingTalkEmbeddedLogin()` 及 `LoginDingTalkEmbedded(...)`。也可采用其他一致命名，但不为此另造 provider 或 repo。
- 新配置和换码响应一律调用已有 `authNoStore(ctx)`（`server/internal/service/auth.go:254`）。
- 禁用返回 enabled=false，不发 state Cookie、不暴露其他配置；换码入口禁用时报 `ErrDingTalkDisabled`。
- 现有 `ErrDingTalkCodeInvalid`、`ErrDingTalkStateInvalid` 中文写“重新扫码”（`biz/auth.go:33` / `:35`）；若共享给客户端免登，建议改成“重新登录”，避免客户端失败提示要求扫码。
- 换码不接收 redirect、userid、unionid、corpID、organizationID；目标路径在前端按站内地址校验后保留，认证与业务导航各用现有协议。

### 4. state / CSRF 最薄复用与边界

- 复用 `newSessionToken`、常量时间比较与 `newCookie` 的 HttpOnly / Secure / SameSite / Path=/ 策略（`service/auth.go:172`）。
- 为客户端 state 使用独立名称，例如 `policy.CookieName + "_dingtalk_embedded_state"`，避免 `DingTalkLoginBootstrap` 首屏预热或扫码刷新覆写 H5 在途 state。
- 客户端每次成功取得新配置才请求 JSAPI；请求失败不悄悄改用 OAuth；重试应显式重新取配置与新码。换码提交时无论成功或失败，都只清除对应客户端 state Cookie，保持扫码 Cookie 不受影响。
- 现有机制不是数据库持久化挑战；五分钟有效期由浏览器 Cookie 寿命约束，上游临时码一次性由钉钉负责。不应把“清 Cookie”表述为服务端严格防并发重放；本期最薄复用无须另加挑战表。
- 本次定向源码未发现可复用的 Origin / Sec-Fetch-Site / 全局 CSRF 中间件。不要因此删除 state 防护或以 UA 为身份依据；UA 检测只决定前端是否尝试 SDK。
- `requestJSON`（`data/dingtalk.go:238`）已用五秒 HTTP client、1MiB 读上限、JSON 编解码与状态码处理，可直接复用。其非 2xx 错误不含完整响应，继续映射为领域错误，禁止输出 access token、auth_code 或 state。
- 企业 app token 的申请与缓存由 `enterpriseAccessToken(ctx)`（`:166`）提供，带 Mutex 与提前到期，可供新换码复用。

### 5. 配置公开与敏感边界

- `server/internal/conf/v1/conf.proto:85`：现有 enabled / client_id / client_secret / redirect_uri / corp_id 足以支持本期，不必新增环境变量或 agent_id。
- `server/internal/data/dingtalk.go:70`：启用时强校验四项配置。新增入口复用同一个 provider 实例，不改变初始化契约。
- corp_id 是所选 JSAPI 的公开标识，新公开配置只返回 corpID、state、到期时间和 enabled。client_id 虽已出现在现有 `AuthorizeURL`（`:99` / `:101`），本期客户端流程不需要它，不额外返回。
- client_secret、企业 app token、用户 access token、event_token、event_aes_key 均不可进入 DTO、HTML、日志或错误。新配置只列所需公开字段，不序列化整个 conf。

### 6. 文件与生成影响

| 文件 | 职责 / 预计影响 |
| --- | --- |
| `server/api/auth/v1/auth.proto` | 新公开配置/客户端换码 RPC、请求与配置消息，复用登录响应 |
| `server/internal/biz/auth.go` | provider IF 扩展、配置 DO、客户端开始/换码方法、提取共享登录段 |
| `server/internal/data/dingtalk.go` | 复用 EnterpriseID、客户端换码方法、上游响应类型/URL及错误映射 |
| `server/internal/service/auth.go` | 新 handlers、独立 state Cookie、共享登录 DTO/cookie helper |
| `server/internal/data/auth.go` | 预计无需改：现有双身份绑定与 CreateSession 可直接复用 |
| `server/internal/data/data.go:25` | 已有构造与 `wire.Bind`，通常无需新增 ProviderSet/Bind |
| `server/internal/server/http.go:51` / `grpc.go:48` | 已注册整个 AuthService；新 RPC 生成后自动纳入，无需另手写路由 |
| `server/internal/server/access_rules_gen.go` | `make api` 从新 RPC 的 PUBLIC 注解自动生成，禁止手改 |
| `server/api/auth/v1/auth*.pb.go` | Go protobuf / HTTP / gRPC 生成物 |
| `server/openapi.yaml`、前端 OpenAPI 输入与 `web/src/services/roncin/*` | `pnpm run generate:web-client` 更新，交主代理串行生成，避免前后端代理同时改生成物 |

生成顺序：改 proto → `make -C server api`（buf + server access rules）→ `pnpm run generate:web-client`。如只扩展原 provider IF 且构造函数未变，一般不需 Wire 重新生成；若调整依赖签名则按官方生成流程更新，不能手改 wire_gen.go。

### 7. 测试替身与定向验证影响

全仓搜索 `AuthorizeURL` / `ResolveIdentity` 后，现有 provider interface 的直接实现只有真实 provider 和以下两处 stub：

- `server/internal/biz/auth_dingtalk_test.go:11`：`dingTalkProviderStub`，`:87` 有编译期接口断言。扩展 IF 时必须同步实现公开标识与新换码方法，建议记录两种 resolver 调用以断言通道不混用。
- `server/internal/service/auth_dingtalk_bootstrap_test.go:20`：`bootstrapDingTalkProvider`，同样补 IF 方法。已有首屏 bootstrap / API / state Cookie 一致性测试需继续通过，不改变原扫码 Cookie 与 HTML 预热语义。

可复用测试设施：

- `server/internal/data/dingtalk_test.go:19` 的 `roundTripFunc` / `jsonResponse` 可截获全部上游请求，无需真实钉钉或数据库；新增客户端 resolver 的 payload、应用令牌、成功身份、错误码、空身份、非2xx、坏 JSON、token 缓存与超时/网络失败覆盖。
- `server/internal/biz/auth_dingtalk_test.go:37` 的未同步/待分配/停用表驱动及 `:63` 已授权建会话测试应扩为两种认证入口；加 state 空/缺 Cookie/不匹配确保 resolver 未调用、多工作台初始排序、零工作台拒绝、身份仓储冲突不建会话。
- `server/internal/service/auth_dingtalk_bootstrap_test.go:67` 与 `auth_organization_switch_test.go:17` 提供 service / 真实 Kratos HTTP 测试模式。新测试覆盖独立 Cookie、清理策略、no-store、disabled不发状态、成功才发 session Cookie、PUBLIC endpoint无需会话但缺state拒绝、schema Required 校验。
- `server/internal/server/auth_test.go` / `auth_retirement_test.go` 涉及中间件公开面，执行相邻认证测试检查新增 RPC 确实走 PUBLIC 而非漏登记。
- 建议初步定向：`go -C server test ./internal/biz -run 'Auth.*DingTalk'`、`go -C server test ./internal/data -run 'DingTalk'`、`go -C server test ./internal/service ./internal/server`。最后由主代理按跨层契约风险跑完整门禁。

### 8. Related specs

已读取 `.trellis/workflow.md`、`server/AGENTS.md`、领域 glossary / architecture-map、server backend index；直接相关：

- `.trellis/spec/server/backend/auth-session-org-switch.md`：多工作台、成员资格、会话轮转、锁序和不可绕过权限。
- `.trellis/spec/server/backend/mobile-login-password-reset.md`：现有认证相邻流程、密码设置/找回与会话撤销，不应受新增入口影响。
- `.trellis/spec/web/frontend/dingtalk-qr-login.md`：独立扫码 state、首屏预热、StrictMode / 刷新在途状态与多工作台相邻路径。
- `.trellis/spec/server/backend/dingtalk-registration-approval.md`：已同步人员与待分配开通边界，禁止登录即自动创建账号。
- `.trellis/spec/server/backend/dingtalk-notification-markdown.md`：机器人消息现有 OrderEntryPath，免登只提升点击认证，不把链接变成权限凭据。

### 9. External references / versions

- 本期选定官方旧版 `dd.runtime.permission.requestAuthCode({corpId, onSuccess})`，npm SDK 路径 `dingtalk-jsapi/api/runtime/permission/requestAuthCode`；服务端搭配 `/topapi/v2/user/getuserinfo`。主代理已通过 [open-dingtalk/corp_demo_php](https://github.com/open-dingtalk/corp_demo_php) 与 Aliyun PDS 集成文档核对链路；前者为钉钉官方源码，包含无需 dd.config 的旧版客户端调用示例。
- 官方 JSAPI Explorer [requestAuthCode](https://open.dingtalk.com/tools/explorer/jsapi?id=11723) 展示另一套带 clientId 的新接口。本期不采用该协议，其版本门槛不得用来描述本期旧版 SDK。
- 官方服务端页：[通过免登码获取用户信息](https://open.dingtalk.com/document/development/obtain-the-userid-of-a-user-by-using-the-log-free)，本次工具只能取得标题、正文为空；主代理已通过其他 primary 来源补齐所选流程证据。请求 JSON `{"code": "..."}`、URL 带企业 app token，响应 `result.unionid/userid/name` 必填校验；具体错误码映射仍须按官方资料/目标用例落实。
- 现有仓库调用自行封装 HTTP，无需为本期引入服务端 SDK 依赖。

## Caveats / Not Found

- 未调用真实钉钉、未连接数据库、未读取本地私密配置、未执行 Git 操作、未修改实现或 spec。
- 用户看到的“点击免登”最终还依赖应用网页能力、可信域名、可见范围、可访问部署域名和真实客户端；这些不能由单元测试或代码读取得出已通过。
- 不混用新的 `dd.requestAuthCode`、旧的 `runtime.permission.requestAuthCode` 和 OAuth 扫码码；本期只支持主代理选定的旧版 H5 → v2 getuserinfo 路径，不加猜测性 fallback。
- 本研究中的新接口与方法名称是建议，最终以任务 design.md 和主代理裁定为准。

### 实施修订：独立响应消息

buf 强制每 RPC 使用独立且按 RPC 命名的响应。`DingTalkEmbeddedLogin` 实际返回 `DingTalkEmbeddedLoginResponse`，信封同构、data 共享 `DingTalkLoginResult`；此前建议直接复用响应消息不符合此规则。HTTP JSON 字段不变，原生成器更新绑定和客户端，不关闭 lint。
