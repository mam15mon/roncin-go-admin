# ADR 0003: Ent Schema 同源，正式 SQL 迁移是生产唯一真相

- 状态：已采纳
- 日期：2026-08-21 版本化迁移；2026-09-18 补并发禁忌与 CHECK 同源规范
- 主题：契约与生成物（数据库）

## 背景

Ent 的运行期 `client.Schema.Create` 自动建表在早期开发很方便，但带来两类
已实际发生的问题：

1. `migrate.Tables` 包级表对象在 `Schema.Create`/Diff 时被框架就地回写，
   存在隐式顺序依赖与全局指针污染，集成测试并发建表会互相破坏。
2. 开发库若靠自动建表漂移生长，会缺少正式迁移里的 `CHECK` 约束；只改
   `field.Enum` 不扩值集时本地全绿，冷启动库（集成/生产按迁移链初始化）
   写入新枚举值直接违反 CHECK。仓库已连续两次踩中
   （`EXCHANGE_RATE_WEEKLY_REMINDER`、`FEE_SUPPLEMENT_APPROVAL_PENDING`），
   修复范本 `server/migrations/20260918093000_notification_delivery_template_check.sql`。

## 决策

- PostgreSQL Schema 的声明真相源是
  [../../../server/internal/data/ent/schema/](../../../server/internal/data/ent/schema/)：
  迁移中的 `CHECK`、外键 `ON DELETE` 等数据库强制条件必须在 Ent Schema
  同步声明（`entsql.Checks` / `entsql.OnDelete`），同名同表达式。
- 生产与集成验证以有序的正式 SQL 迁移
  （[../../../server/migrations/](../../../server/migrations/)）为唯一
  执行真相源，按 revision + 校验和严格校验；不得依赖 Ent 运行期自动建表。
- 集成测试禁止并发执行 `Schema.Create`，涉及 Schema 初始化的测试禁止
  `t.Parallel()`。
- 迁移校验和按环境双轨：`pnpm dev` / `migrate:dev` 允许把已应用迁移的
  校验和重录为当前文件（开发期迭代迁移属正常）；`pnpm run migrate:server`
  保持严格校验，禁止对生产库自动修复。
- 改 `field.Enum` 前先 `grep` 迁移链中该列是否已有 CHECK；命中则新迁移
  `DROP/ADD CONSTRAINT`、Ent 同名 `entsql.Checks`、元数据测试、冷启动
  实测四件事缺一不可。

## 理由

- Ent 自动建表面向「开发库从零长出来」，与「生产库按审计链演进」是两种
  世界观；混用导致开发库掩盖生产问题，这正是两次 CHECK 事故的根因。
- Ent 注解保留 Schema 真相源完整性，防止后续差异生成把正式约束识别为
  漂移而误删。

## 后果

- 每次生产结构变更必须生成、审阅并提交迁移文件；业务代码禁止散落手写
  SQL（确有必要时集中封装在 `internal/data` 并说明原因）。
- 空库迁移测试与生成元数据测试是 CHECK 类变更的强制验收。
- 完整规范见
  [../server/backend/database-guidelines.md](../server/backend/database-guidelines.md)。
