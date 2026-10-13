# 钉钉扫码主登录：审查与验证记录

日期：2026-10-13。基线：`61a74a60`。工作区：`/home/vince/roncin-go-admin-worktrees/dingtalk-login-memberships`。

## 结论

当前实现与已批准 PRD、设计及 `dingtalk-qr-login.md` 契约一致，未发现剩余代码提交阻塞项，可按本轮不运行测试的约束提交。真实二维码显示、本人手机扫码和开发代理/生产同域回调尚未验证，不能将本记录作为企业登录联调完成证明。checker 未提交或归档。

## 行为核对

- 默认扫码界面使用钉钉 OAuth2 官方端点的真实 iframe，保留服务端公开授权参数并增加 `iframe=true`；没有生成普通授权 URL 二维码或改用跳转按钮。管理员密码为次级入口，可返回扫码，既有品牌和移动端 Logo 保留。
- 配置新增 `expires_at`，由同一次 `StartDingTalkLogin` 返回的期限格式化；state Cookie 使用同一期限。配置响应设置 `Cache-Control: no-store` 与 `Pragma: no-cache`。换码、Cookie state 验证、企业身份校验及未同步/待分配/停用状态机未改变，无秘密进入前端。
- 配置使用固定 React Query key、不消费 AbortSignal、明确禁自动重试及焦点/重连刷新。已发出的配置请求完成前保留去重，防止快速切换或 StrictMode 清理后产生两份在途 Cookie 响应。模式返回重新取配置；手动刷新先 flushSync 卸载旧 iframe/监听，再以 `cancelRefetch:false` 发起串行刷新。
- 消息要求精确官方 origin、非空且相同的 iframe window、对象结构、成功标记和正确 redirectUrl；回调 origin/path 必须与本站配置一致，state/authCode 各只有一个，state 相同、authCode 非空。合法消息只构造本站固定回调，不直接导航消息外链。
- 旧实例、退役、到期及重复成功消息不再认证。父回调稳定，监听和定时器可清理；到期不轮询，前台恢复/焦点事件复核期限。配置错误、禁用、iframe 失败或超时可见，刷新与管理员入口保留。
- 回调与密码认证仍沿用现有单工作台直进、多工作台选择，以及本次响应本地候选/组织 ID 的一致状态；扫码不自动创建、同步或开通人员。
- 无数据库 Schema 或依赖变化。Proto、Go 绑定、OpenAPI、Web 输入及客户端类型由生成器同步；既有登录测试仅同步 Provider、配置夹具和改变的交互预期，未增加用例。

## 审查发现与修复

1. `dingtalk-qr-login.tsx` 原消息 guard 未单独拒绝 iframe.contentWindow 为空；现在先获取当前 window 并要求非空，避免 event.source 为空时比较相等。
2. 原失败处理仅更新 React status，提交前迟到成功仍可能进入监听。现在以即时 retired 引用对失败、过期、加载超时、成功和卸载立即停用实例；onLoad 复核 retired，不复活已停用实例。实现代理完成，checker 已复核。
3. 新规范签名节最初写成 `/api/v1/auth/dingtalk/config`，已由主会话修为实际既有 `/api/v1/auth/dingtalk/login-config`。未改接口路径。

## 非测试验证

- checker 重新只读取得钉钉托管 [0.21.0 登录脚本](https://g.alicdn.com/dingding/h5-dingtalk-login/0.21.0/ddlogin.js)，与父研究记录一致：OAuth2 iframe、iframe=true、redirectUrl 中 authCode/state 和成功/失败消息协议。未执行脚本，也未调用企业接口。该源码证据不证明当前应用实测可用。
- checker 对本地 React Query 5.103.1 源码核对：不消费 signal 的请求在观察者卸载后继续；gcTime=0 不移除仍在请求的 Query；在途 refetch(cancelRefetch:false) 返回现有 Promise。
- 原始 `node web/scripts/check-architecture.mjs` 通过：432 个产品文件、1597 条本地依赖、4 条 features 能力依赖边，违规 0。未调用含测试的架构包装。
- checker 执行 `make -C server api` 与 `pnpm run generate:web-client` 成功；生成前后 API 目录、roncin 客户端、OpenAPI 及 Web 输入共 146 个文件内容指纹完全一致，生成结果可复现。
- 实现代理最终定向 Go build/service vet、Web `tsc --noEmit`、三个修改 TSX 的 Biome、两个产品组件 Ant Design lint 通过，Ant Design lint 零问题。
- checker 最终 `git diff --check` 通过。

## 验证限制

本轮明确禁止新增或运行任何测试，未执行 Go/前端业务测试、检查器自身测试或含测试的 `check:fast` / `check:architecture` / `check:layers:go` 包装。静态检查不能证明 StrictMode、快速模式切换、连点刷新、消息顺序、后台恢复和真实 Cookie 行为已在浏览器验证。

未运行实际数据库、读取环境秘密或调用企业换码接口。真实企业应用二维码显示、本人手机扫码、单/多工作台及人员状态登录、开发代理和生产同域回调、小屏布局与 iframe 网络错误浏览器体验均未实测。应保留这些验收项待完成，不使用 mock 或协议源码核验替代真实联调。
