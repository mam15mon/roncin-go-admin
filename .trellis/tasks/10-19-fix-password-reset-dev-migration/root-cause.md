# 缺陷复盘：热更新后未执行新增迁移

## 1. 根因分类

C（变更传播遗漏）与 E（隐含假设）：上期代码、正式迁移和隔离库测试已交付，当前长期运行的开发环境没有执行新迁移。Air 监听 SQL 文件只触发 Go 重建，不能同步 Schema。

## 2. 证据与验证路径

当前服务监听 8000；开发库最新版本仍为 20261016100000_retire_role_data_scope，挑战表与手机号投影均不存在。项目 scripts/dev.mjs 只在启动阶段调用 migrate:dev，server/.air.toml 的 build.cmd 仅 go build。业务接口查询缺失字段/表会进入统一内部错误响应，与截图吻合。正式迁移后以未知号码 HTTP 验证排除消息权限、收件配置等其他原因。

## 3. 之前交付遗漏

上期真实 PostgreSQL 验证使用独立测试库，证明了正式迁移和业务逻辑，但不能代表当前服务所连接开发库已更新。交付时没有完成当前开发库的升级；用户仍在运行的旧开发启动进程没有重新走启动迁移。

## 4. 预防机制

本次补迁移并核对身份与密码摘要不变；将当前环境 revision/表/列与无真实收件的 HTTP 烟测纳入移动登录规范。继续使用已存在的 migrate:dev/migrate:server，不新增自动建表、兼容分支、Air 迁移平台或业务回退。

## 5. 知识记录

已更新 .trellis/spec/server/backend/mobile-login-password-reset.md 的运行环境契约、错误矩阵、必要验证与错误/正确示例。本仓库没有 src/templates/markdown/spec 模板目录，模板同步不适用。
