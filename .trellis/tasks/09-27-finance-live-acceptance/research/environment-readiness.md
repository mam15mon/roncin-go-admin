# 执行前环境与契约核查（2026-09-27）

## 环境

- 用户在本轮要求「完成计划」，按上一轮最终审阅范围进入实施。
- `.env.local` 有应用数据库与 bootstrap 配置，但无显式集成/验收管理员连接变量。未输出任何值。
- 应用数据库账号无 CREATEDB / CREATEROLE；本机 `sudo -n -u postgres psql` 可用。
- 主会话已创建任务专属临时管理员和集成库，连接变量放在 0600 的仓库外环境文件；主会话统一清理。不使用既有开发库执行集成测试。
- pnpm、psql、Playwright 依赖和浏览器缓存存在。Chromium 启动实测缺少 libatk-1.0.so.0，主会话已通过 Playwright install-deps 安装官方运行依赖，复测 Chromium 151.0.7922.34 启动成功。
- 8010 / 9010 空闲，8001 已被既有服务占用，不终止该进程；验收服务需使用明确的独立端口。现有编排写死 8001，实施时应做最小配置调整并继续保持冲突检查。

## 必须处理的契约漂移

1. `server/cmd/bootstrap-admin/main.go:122` 创建 system 工作台；默认四家公司虽存在，但 bootstrap 账号只在 system 有成员资格。现有财务脚本登录后直接使用 currentOrganization，不切换到有授权的 company。实施时需要通过正式 API 准备真实公司成员/角色并选择公司工作台，不放开系统账号经营权限。
2. 一次性编排 Stage A 先跑 data 中 `Postgres$` 测试，再初始化系统管理员、启动 Go/Vite、调用三段 acceptance:finance。当前没有 -p 32 和较长 timeout；不能只按 Postgres 后缀取代本任务要求的完整 data/migration 运行。
3. Stage B bootstrap 后直接运行外币脚本；外币脚本假设 currentOrganization 可以维护 baseCurrency 并开展经营，需按当前系统管理与公司边界准备主体。
4. 现有 UI 用例使用脚本生成的 ACC-FIN 订单，再另外选择已有提成。必须补同一订单关键财务步骤的 UI 操作；HTTP 样本与 UI 样本独立，避免已核销余额或已建账费用耗尽。

## 证据状态

本文件是前置核查，未宣称 HTTP、浏览器或人工验收通过。临时环境变量文件的内容不得提交；日志与 trace 必须脱敏。
