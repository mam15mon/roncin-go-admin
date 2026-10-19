# 开发环境验证码申请恢复验证

验证日期：2026-10-19（北京时间）。本次只应用仓库既有正式迁移，不修改产品代码、生成物或依赖，不清库、不重置密码、不额外执行开发业务种子同步。

## 前态与执行范围

- 开发数据库原最新 revision：`20261016100000_retire_role_data_scope`。
- `password_reset_challenges` 表、`users.dingtalk_mobile_normalized` 列原均不存在。
- 将 `schema_migrations.version` 与 `server/migrations/*.sql` 文件名逐一比较，唯一待应用文件为 `20261019100000_mobile_login_password_reset.sql`。
- 已审阅该 SQL：新增手机号规范化检索投影并回填，增加非唯一索引，新建验证码挑战表及其索引和 CHECK；不改写账号身份、既有密码及组织关系。

## 实际执行命令

在仓库根目录执行现有开发迁移入口两次：

```bash
pnpm run migrate:dev
pnpm run migrate:dev
```

执行由临时 Node 辅助脚本调用 `spawnSync('pnpm', ['run', 'migrate:dev'])`；脚本通过 `node --env-file-if-exists=.env.local` 加载现有配置。数据库只读检查通过 `psql -X -q -t -A -v ON_ERROR_STOP=1` 执行，凭据仅放入子进程 PG 环境变量，不拼入命令行或输出。

- 两次退出码均为 `0`。
- 两次权限同步均为新增 `0`、更新 `0`、移除 `0`、补齐角色依赖 `0`。
- 两次订单主数据种子补齐均为 `0`。
- 没有发生已应用迁移校验和重录。
- 原始迁移日志位于 `/tmp/roncin-password-reset-migrate.log`，权限已核验为 `0600`，不纳入 Git。

## 数据保留与结构验证

迁移前将各表行数和排序后行 JSON 的摘要保留在辅助进程内存；每次迁移后重新计算，仅输出相等布尔值，不输出摘要本身、密码 hash 或个人资料。排除本次新增的 `dingtalk_mobile_normalized` 与标准迁移入口负责维护的 `search_keywords`；`users` 的其余字段（含密码 hash、用户名与钉钉身份）全部参与对照。

| 对照表 | 第一次后不变 | 第二次后不变 |
| --- | --- | --- |
| users | true | true |
| organizations | true | true |
| memberships | true | true |
| roles | true | true |
| role_assignments | true | true |
| role_permissions | true | true |
| partner_roles | true | true |

迁移后只读检查通过：

- 最新 revision 为 `20261019100000_mobile_login_password_reset`，该 revision 恰有 `1` 条。
- 手机号规范化列和验证码挑战表均存在。
- 索引齐全：`user_dingtalk_mobile_normalized`、`password_reset_challenges_pkey`、`passwordresetchallenge_mobile_key_hash`、`passwordresetchallenge_token_hash`、`passwordresetchallenge_user_id`、`passwordresetchallenge_updated_at`。
- CHECK 齐全：`password_reset_challenges_status_check`、`password_reset_challenges_failed_attempts_check`。
- 第二次执行后没有新增目标 revision，账号和组织关系仍相同。

## 实际 HTTP 烟测

辅助进程选择符合国内格式的测试号码，先查询确认 `users.username` 与 `users.dingtalk_mobile_normalized` 均无命中，再仅调用一次：

```text
POST http://127.0.0.1:8000/api/v1/auth/password-reset/code
Content-Type: application/json
```

请求体和响应 token 仅留在进程内存，没有输出或写入本记录。实际结果：

| 检查 | 结果 |
| --- | --- |
| HTTP 状态 | 200 |
| success | true |
| message | 若该手机号对应已开通的钉钉账号，验证码将发送到本人钉钉。 |
| challengeToken 存在 | true |
| expiresAt / resendAvailableAt / serverTime 存在 | 全部 true |
| Cache-Control 包含 no-store | true |
| 测试号码对应身份命中 | 0 |
| 该次挑战数 | 1 |
| 挑战 user_id 为空 | true |
| 挑战 status 为 FAILED | true |
| 挑战 code_hash 为空 | true |

未知目标只形成一条不可消费挑战；结合用例仅在 `record.UserID != nil` 时调用发送器的现有分支，确认该请求未向真实员工发送消息。没有调用密码确认接口，没有修改任何密码。运行中的后端无需重启即可读取已补齐结构，原申请接口已恢复正常成功信封。

## 清理与验证边界

- 匿名摘要只留在已退出的辅助进程内存，没有摘要临时文件；`0600` 辅助脚本已删除。
- 保留标准申请产生的不可消费测试挑战与对应限流记录，未直接删改业务表进行清理。
- 本次是既有正式迁移的环境落地，没有产品实现变更；执行了真实开发库迁移、重复执行、结构与数据摘要核验和真实 HTTP 烟测，不运行无关联的全栈门禁。
- 未执行真实钉钉本人收件联调；HTTP 200 表示公开申请接口恢复，不代表已验证真实消息送达。
- 长期运行的 `pnpm dev` 在新迁移合入后需另行执行 `pnpm run migrate:dev`，或重启 `pnpm dev` 触发启动迁移；air 热更新不替代迁移。生产仍使用 `pnpm run migrate:server`。

## 独立检查

检查代理以只读事务独立核验迁移前后 revision 与目标结构，确认目标 revision 恰 1、数据库 checksum 与仓库 SQL 一致、主键/索引/CHECK 齐全，迁移日志为 0600。审阅执行记录和未知目标发送分支后通过，无未解决项；git diff --check 通过。
