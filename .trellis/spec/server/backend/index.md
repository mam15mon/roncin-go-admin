# Server 后端开发规范（Go / Kratos / Ent）

> 本层规范的唯一真相源是根目录 `AGENTS.md`；本目录是面向 AI 任务执行的浓缩版。
> 冲突时以 `AGENTS.md` 为准。

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | 分层职责与依赖方向 | ✅ |
| [Database Guidelines](./database-guidelines.md) | Ent、事务封装、并发锁、分页 | ✅ |
| [Sea Export Document Contract](./sea-export-document-contract.md) | 海运操作票、共享 MBL、运输执行与 HBL 跨层契约 | ✅ |
| [Order Lock and Document Version](./order-lock-and-document-version.md) | 海运出口订单业务锁、不可变版本与共享 MBL 固定锁序 | ✅ |
| [Error Handling](./error-handling.md) | 领域错误与驱动错误映射 | ✅ |
| [Quality Guidelines](./quality-guidelines.md) | 分层禁令、生成物、常用命令 | ✅ |
| [Logging Guidelines](./logging-guidelines.md) | 结构化日志与审计 | ✅ |

## Pre-Development Checklist

动手写代码前确认：

1. 接口变更是否已先修改 `.proto` 契约（禁止手改 `*.pb.go` 等生成物）。
2. 涉及的层是否清楚：DTO 转换在 `service`、业务规则在 `biz`、Ent 查询在 `data`。
3. 是否需要事务 / 并发防护；新实体是否需要 `version` 字段（判定不清先问用户）。
4. 列表接口分页是否复用 `internal/biz` 公共校验（`1 <= pageSize <= 200`）。
5. 新增权限码是否登记 `internal/access/manifest.go` 并重新生成前端权限键。
6. 订单锁、解锁或共享 MBL 写入是否遵循不可变版本与固定锁序规范。

## Quality Check

提交前确认：

- `go -C server vet ./...` 与 `go -C server test ./...` 通过（按风险选取最小集）。
- 契约变更已重新生成绑定代码与 OpenAPI / 前端客户端，生成物无手改痕迹。
- 迁移文件已随代码提交，未绕过 Ent 手写业务 SQL。

**语言**：文档与面向开发者的注释使用中文。
