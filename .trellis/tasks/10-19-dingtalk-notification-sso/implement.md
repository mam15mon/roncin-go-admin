# 实施清单

- [x] 研究当前认证、组织、SDK 协议和三层扫码预热；PRD/设计收敛，按用户端到端授权实施。
- [x] 后端新增 embedded 配置/换码 RPC、独立 state Cookie、企业 getuserinfo 解析及共享准入。
- [x] 原生成器更新 Go/OpenAPI/前端客户端。
- [x] 生产 HTML 跳过钉钉 UA 扫码预热，保留 no-store。
- [x] 前端按需 SDK、一次认证与迟到隔离、显式错误/重试/账号登录。
- [x] 单多工作台完整回跳，公共安全校验，精确公开请求豁免；Vite/main 预热分流。
- [x] 完成实现后按验收标准增加针对性测试；定向测试、Biome、tsc。
- [x] 同步前后端规范；独立 review 并修复问题。
- [x] `pnpm run check:fast` 和 `pnpm run build`；真实浏览器协议模拟，记录真机验证边界。
- [x] 逐项验收与工作提交；整理归档、合并与推送核验流程（见 verification.md，实际Git结果在最终交付说明报告）。

## 验证命令

- `go -C server test ./internal/biz -run 'Auth.*DingTalk'`、`go -C server test ./internal/data -run 'DingTalk'`。
- `go -C server test ./internal/service ./internal/server ./internal/webassets`。
- `pnpm --dir web exec vitest run <本次相关 test 文件>`，修改文件 Biome 与 `pnpm --dir web tsc`。
- `git diff --check`、`pnpm run check:fast`、`pnpm run build`。

## 风险点

临时免登码必须配对旧版 JSAPI/getuserinfo；禁止协议回退。state Cookie 不能被扫码预热覆盖。StrictMode 不重复消费；离开页面结果不导航。返回路径不能外跳或回登录循环。测试不调用真实钉钉、不读取/打印秘密、不重置数据库。
