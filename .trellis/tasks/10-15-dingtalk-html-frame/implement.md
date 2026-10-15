# 实施顺序

- [x] 实现 Go 与 Vite 登录 HTML 提前输出 iframe，验证安全转义、配置合法性和路径边界。
- [x] 实现前端一次性认领、位置对齐和清理，复用现有消息验证逻辑。
- [x] 功能实现后补充定向测试：同实例、StrictMode、刷新销毁、禁用/坏配置、早期 load 与卸载。
- [x] 运行 Go webassets/service 定向测试、相关 Vitest、改动文件 Biome 和类型检查。
- [x] 独立检查 agent 审阅全链路并修复；主会话浏览器阻塞主程序证明提前请求与接管不重复加载，覆盖开发和 Go 同域生产。
- [x] 更新扫码规范；运行 pnpm run check:fast 和生产构建，记录真实企业扫码验证限制。
- [ ] 中文 Conventional Commit 提交，归档本任务并记录日志。

风险文件：server/internal/webassets/assets.go、web/config/dingtalkLoginBootstrap.ts、web/src/app/dingtalkLoginConfig.ts、扫码组件与其接管模块。只修改本任务范围，完整回退本组提交为回滚点。
