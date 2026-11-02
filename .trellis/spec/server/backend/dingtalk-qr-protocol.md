# 本站二维码与钉钉扫码网关

## 适用范围

普通浏览器扫码由本站绘制真实二维码并调用服务端轮询；前端与首屏契约见[扫码主登录](../../web/frontend/dingtalk-qr-login.md)。`data/dingtalk_qr.go` 集中封装上游页面内部协议，`biz/auth_dingtalk_qr.go` 绑定本站 state，传输层不直接访问上游。

## 上游请求

- 分析基准为 `h5-dingtalk-login/0.126.3/login.js`，原脚本 SHA-256：`efacc1d3e29fe331e79ed22a1b5a4b53681402ce4190f3a65711a61a9448120e`。生成与 waiting 已用当前应用公开配置实际验证，2026-11-02 用户在当前开发登录页实际扫码并确认成功进入后台。
- 基址固定 `https://login.dingtalk.com`。每次尝试先 GET `/oauth2/challenge.htm`，再 POST `/oauth2/generate_qrcode`；POST 使用 `application/x-www-form-urlencoded`。
- 生成及轮询完整保留本应用 OAuth query，包括 client_id、redirect_uri、response_type、scope、corpId、state、prompt，不借用研究示例应用。沿用旧集成的上游 `iframe=true` 协议模式，bootstrap、生成与轮询一致；该参数控制上游成功衔接，不代表本站还渲染 iframe。设备参数采用上游脚本已定义的设备模块不可用值；上游风控或错误直接失败，不实现验证绕过。
- 生成成功 `result` 是真实 `https://login.dingtalk.com/oauth2/qr_confirm.htm?code=...` 字符串。严格解析唯一非空扫码 code；绘制链接追加当前上游固定的 `dd_mini_app_id=5000000004998565` 与 `showmenu=false`。
- POST `/oauth2/login_with_qr` 继续携带同次参数及扫码 code、`exclusiveCorpId=''`、`stayLogin=false`。扫码 code 与最终 OAuth authCode 不是同一凭证。
- 轮询为上游长轮询，当前实测 waiting 约 5 秒；不得使用原 OAuth 请求的过短超时，也不通过降低全站权限或改变会话逻辑适配。

## 状态与一次性衔接

| 上游结果 | 本站结果 |
| --- | --- |
| success=false / 11021 | WAITING，允许下一轮 |
| success=false / 11041 | SCANNED，提示手机确认并继续 |
| success=false / 11019 | EXPIRED，终止旧尝试 |
| success=true / result.url 合法 | CONFIRMED，返回一次性 authCode，终止旧尝试 |
| 未知错误、风控、非法 JSON/链接 | 中文失败，停止，不静默按 waiting 或 iframe 处理 |

- `result.url` 必须与配置回调 scheme/host/path 相同，无 userinfo/fragment；query 使用严格 `url.ParseQuery`，非法转义拒绝，state 恰好一次且匹配当前尝试，authCode 恰好一次且非空。
- 只把通过校验的 authCode 传入原 `DingTalkLogin` 换码链路。授权身份、企业匹配、人员启用与工作台资格仍按原逻辑校验。
- 本地 state 为 5 分钟期限，配置 Cookie、DTO 到期与该期限对应。上游未返回 TTL，不能声称该期限来自钉钉；上游 EXPIRED 同样终止。
- 每次尝试独立客户端与 CookieJar，禁止共享登录 Cookie。网关只保留短期有界内存；过期清理、失败释放、终态删除，不新增数据库历史或兼容分支。
- 每次尝试至多一个在途上游轮询，取消必须释放串行控制。StrictMode 或网络取消后的新请求可在有界上下文内等待前次退出，不因取消交接窗口误判失败。
- HTTP 请求、响应与上游 URL 中的短期扫码标识不写日志；生成配置/轮询 no-store，不向前端返回上游 Cookie、设备头或任意代理能力。

## 验证

HTTP 替身验证：逐尝试 Cookie 隔离、请求参数与固定端点、四态、终态消费、非法/重复/错误转义参数、未知错误、取消与单尝试串行。业务及传输验证 state Cookie 不匹配时不访问上游。真实手机验收单独记录，本次用户已确认扫码成功进入后台；后续上游或应用配置变化仍须重新验收，不把静态成功分支或 waiting 实测当作完整登录通过。
