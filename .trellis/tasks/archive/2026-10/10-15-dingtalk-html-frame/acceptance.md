# 验收记录

## 2026-10-15 开发入口
- 隔离测试服务：复用 AuthService.DingTalkLoginBootstrap 与原 API 注册，配置提供方输出本地测试 client_id 和当前入口回调；不连接数据库、不使用企业身份，不修改开发数据。
- Vite test 模式指向隔离服务，真实 Chromium 拦截钉钉授权页为测试文档。
- 阻塞 `/src/main.tsx`：iframe 已发起一次授权请求；配置 GET 为零；iframe 在监听安装前隐藏。
- 放行主程序：同一 DOM iframe 实例，授权请求仍为一次；viewport 缩至 390px 后区域位于视口内。
- 手动刷新销毁原节点，新配置与二维码各一次；密码切换移除 iframe，返回获取新配置。
- 旧来源消息不认证；同实例连续两次成功消息只换码一次。
- 隔离换码失败返回登录后获取新配置；无浏览器 pageerror。

## Go 同域生产入口
- 完整 `pnpm run build` 通过，生产资源经正式 stage 脚本嵌入 Go。
- 隔离 Go 服务同域提供静态资源与真实 AuthService 配置/换码入口。
- 与开发入口使用同一 Chromium 脚本：阻塞主入口 JS 时已有一次授权请求；配置 GET=0；放行后同节点且无第二次导航。
- 窄屏、刷新、密码切换、旧来源/重复成功与失败回调返回均通过；无 pageerror。
- Go 参数排序与浏览器授权 URL 不同顺序仍认领同一实例，未改 src。

## 自动检查
- 定向 Vitest：7 个文件，74 项通过。
- `go -C server test -p 32 ./internal/webassets ./internal/service`、前端 tsc、改动 TS/TSX Biome 检查通过。
- `pnpm run check:fast` 全部通过：前端 1278 通过 / 12 跳过；后端完整门禁通过。
- `pnpm run build` 与 `git diff --check` 通过。
- 独立审查通过，无确认缺陷或阻塞项；独立 Go webassets test/vet、前端类型与 Biome 通过，另行验证 3 个 Vitest 文件共 32 项通过。

## 限制
钉钉页面使用浏览器拦截的隔离响应；结果证明加载与认证时序，不代表真实企业扫码已通过，也不用于宣称减少固定毫秒数。
