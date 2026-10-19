# 交付进度

## 授权与分支

- 2026-10-19：用户在最新完整方案后批准实施，明确要求完成任务、review、合并回 main、安全删除分支后 push。
- 初始工作区干净，main 与任务分支起点均为 `a4714e1f`；任务分支 `codex/mobile-login-dingtalk-password`。规划提交此前已在 main，本次实现放在任务分支。
- 已 fetch origin；远端 main 为 `9f6aba8b`，本地 main 领先 65、落后 0。最终 push 会包含本地主分支此前尚未推送的已提交工作，不重写历史。

## 代理所有权

- implement_auth：server、正式迁移和所有生成物，包括 web 生成客户端；负责后端定向及真实库验证。
- implement_web：web 手写登录、密码找回组件/样式/测试；负责前端定向验证。
- 主会话：任务/规范、独立检查派发、全栈验收、Git 合并与清理交付。代理共享同一工作区，禁止覆盖他人修改。

## 隔离验证环境

已用本任务唯一命名创建 PostgreSQL 数据库/角色，数据库名前缀 `roncin_notify_verify_auth_`，public 安装 pg_trgm。0600 私有环境文件位于 `/tmp/roncin-mobile-auth-verification.env`，状态记录位于 `/tmp/roncin-mobile-auth-verification.json`。文档不记录连接串或凭据；验证结束按记录和所有权校验只清理本次资源，不能清理日常开发库。

环境同时开启集成数据库、迁移测试与真实时间后台任务开关，最终统计实际 PASS/FAIL/SKIP；不实际发送钉钉消息。

## 已确定接口

- `RequestPasswordResetCode`：`mobile` → 标准信封 + `data {challengeToken, expiresAt, resendAvailableAt, serverTime}`。
- `ConfirmPasswordReset`：`challengeToken, code, newPassword` → 标准成功信封，无 data、不新建会话。
- 429 metadata 提供 resendAvailableAt/serverTime；前端消费统一错误结构。

## 未完成

实现、review、针对性与全量验收、规范同步、归档、合并 main、删除分支、push 与远端一致性检查均待执行。此文件为过程记录，不代表验收通过。
