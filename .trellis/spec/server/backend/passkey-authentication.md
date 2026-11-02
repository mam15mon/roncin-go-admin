# 本站通行密钥认证

## 范围与入口

`PasskeyService` 提供本站 WebAuthn 注册、发现式登录和本人凭证管理，与钉钉自己的通行密钥无关。维护库 `github.com/go-webauthn/webauthn` 负责协议解析与验签；库类型、公钥数据与临时仪式会话停留在 data 层，biz 只消费领域身份和 JSON 协议载荷。

```text
GET    /api/v1/auth/passkey/config
POST   /api/v1/auth/passkey/login/begin
POST   /api/v1/auth/passkey/login/finish
POST   /api/v1/auth/passkey/registration/begin
POST   /api/v1/auth/passkey/registration/finish
GET    /api/v1/auth/passkeys
DELETE /api/v1/auth/passkeys/{id}
```

配置与登录公开；注册、本人列表、删除要求已有有效登录。注册/删除目标固定 `Principal.UserID`，不从请求体接收用户或组织 ID。

## RP 与部署约定

- 生产 RP ID 固定 `roncin.com.cn`，建议登录入口 `https://ops.roncin.com.cn`。同主域更换子域名时保留 RP ID，显式更新受信任 origin 与钉钉回调；不能把 RP ID 随 `request.Host` 自动变更。
- 环境变量：`PASSKEY_ENABLED`、`PASSKEY_RP_ID`、`PASSKEY_RP_NAME`、`PASSKEY_ORIGIN`。三套配置默认禁用；启用时必须明确 RP 与来源。示例不包含密码、私钥或其他凭据。
- origin 必须是无路径、查询、userinfo 的完整 HTTPS 来源，host 等于 RP ID 或其子域名；禁止 IP、公共后缀、任意来源及 related-origin 自动放宽。仅显式 `rp_id=localhost` 的开发验收允许 `http://localhost:<port>`。
- HTTP Host 与 origin 白名单同时匹配。Begin、Finish、删除要求精确 Origin 头；配置/本人列表允许无 Origin，但 Host 仍必须匹配。反向代理保留真实 Host；不从不可信请求头推导白名单。
- 当前 HTTP IP 登录继续使用扫码/密码；通行密钥配置明确不可用，不调用浏览器密钥 API，不展示虚假的成功开关。

## 注册与登录边界

- Begin 使用维护库生成随机挑战，响应 `optionsJson` 为标准 `publicKey` 内部对象，包含明确 `expiresAt`。服务器保留 session，浏览器只收到选项；登录/注册使用不同 HttpOnly challenge Cookie。
- 临时仪式保留 5 分钟且有容量上限；Finish 先从服务端缓存一次性取出，错误也不恢复已消费挑战。客户端不能指定 challenge、session、用户归属或公钥可信状态。
- 强制 user presence、user verification 与可发现凭证；不使用免交互自动升级注册，不接受缺失 UP/UV 的结果。
- 验证 RP ID、clientData origin、挑战、签名、凭证 ID 与 userHandle，错误统一业务失败，不把认证器结果直接当人员身份。
- 注册只能为当前已验证用户添加密钥。公钥凭证落 `passkey_credentials`，RP/credential 唯一索引兜底；完整库凭证 JSON 不对外输出、不写日志。
- 发现式登录依据验证过的凭证找回本站人员，继续检查启用、待分配及成员资格，复用原 AuthUsecase 的组织候选、会话创建与审计。通行密钥不能绕过企业人员准入。
- 认证 counter/凭证状态更新和创建本地登录会话在同一个 `biz.Transactor` 事务中完成；失败全部回滚。按用户 `ForShare` → 凭证 `ForUpdate` 固定顺序加锁，使用 `Data.client(ctx)`；不额外添加编辑版本字段。
- 注册和删除同样固定用户→凭证顺序。只能删除本人凭证；删除不额外撤销既有会话，停用人员仍按所有登录方式共用的禁用规则拒绝。
- 成功登录继续消费 `rememberMe` 的现有 Cookie 策略；注册失败不清理已通过扫码/密码建立的有效会话。

## 浏览器交互

- 前端使用 `@simplewebauthn/browser` 解析浏览器凭证，不手写 base64/原生协议编码。选项先由 React Query 准备，原生 `create/get` 在明确用户点击中启动，随后 mutation 提交结果。
- 登录页“开启免密登录(通行密钥)”只保存本次非敏感注册意图。扫码/密码成功后显示添加确认框；用户点击添加才调用原生注册，明确跳过则继续原登录，不静默假装已添加。
- 已有密钥的“免密登录”不依赖浏览器本地标记才能使用；跨设备/同 RP 新子域可由浏览器选择已有凭证。
- 登录与注册仪式按认证协议豁免业务工作区展示 scope，但服务端鉴权保持。本人列表/删除按用户隔离 query key，旧用户卸载后的结果不能改变新用户界面。
- 注册/认证进行中禁止其他登录入口交叠；组件卸载或用户跳过后，迟到结果不能提交或触发旧继续回调。挑战消费后重新准备选项，但不自动重新打开原生验证。
- 头像菜单提供本人通行密钥管理：查看名称/本地时间、添加、确认删除。接口不暴露原始 credential ID、公钥或签名。

## 正式迁移与验证

- Ent Schema 是凭证表真相源，正式迁移 `20261102090000_passkey_credentials.sql`，禁止运行期 `Schema.Create`。
- 真实 PostgreSQL fixture 生成 P-256 测试认证器并走库的真实注册/ES256 验签，覆盖错误 origin/RPID/challenge/signature/userHandle、UP/UV 缺失、重放、跨用户维护、账号停用、唯一性与相同 counter 并发。
- 事务验证必须证明认证失败时 counter 与新会话一起回滚，不能只断言返回错误。
- 浏览器原生端到端验收使用 Chromium 虚拟认证器、显式独立数据库与 localhost RP：密码登录→添加→退出→免密登录→挑战重放拒绝→本人删除。2026-11-02 此完整链路实际 PASS，临时服务、数据库和角色已清理。不得将测试指向日常开发或生产库。
- 虚拟认证器验收只证明协议与浏览器链路；正式域名、证书及真实设备生物识别体验启用后仍需现场验证。
