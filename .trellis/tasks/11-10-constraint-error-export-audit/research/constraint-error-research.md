# Research: Ent 约束错误结构化映射

- Query: 审计 mapEntConstraint/mapEntConstraints 的错误链、调用方约束名称及精确匹配迁移影响。
- Scope: internal
- Date: 2026-11-10

## Findings

### 文件与现有模式

- `server/internal/data/ent_error.go:29`：`mapEntConstraint` 先经 `ent.IsConstraintError` 判断，再用 `strings.Contains(err.Error(), constraintName)` 匹配；`mapEntConstraints`（37 行）依次调用同一 helper。字符串包含会受错误文本影响，且允许不完整名称误命中。
- `server/internal/data/ent/ent.go:490`：生成的 `ConstraintError` 保存 `wrap error`；501 行 `Unwrap()` 返回该错误，底层 `pgconn.PgError` 可经标准库 `errors.As` 提取；506 行附近 `IsConstraintError` 自身也使用 `errors.As`。无需修改生成物。
- `server/go.mod:6`、`:15`：当前锁定 Ent v0.14.6、pgx/v5 v5.9.2。应采用 `github.com/jackc/pgx/v5/pgconn.PgError.ConstraintName`，保留 Ent 类型门禁并完整相等匹配；无结构化驱动错误、空配置、不匹配时保留原错误，不增加字符串回退。
- 已有扫描记录为约 64 个调用、36 个名称；实施前应重新扫描最终工作树，计数仅作影响规模参考。

### 必须同步的调用方名称

1. `server/internal/data/finance_bill_write.go:81` 使用子串 `idempotency`；正式 SQL `server/migrations/20260826160000_finance_bills.sql:88` 的唯一索引完整名是 `financebill_organization_id_idempotency_key`，必须同步替换。
2. `server/internal/data/finance_commission_application_submit.go:496` 使用 `financecommissionapplication_organization_id_employee_id_application_month`（74 字节）。正式 SQL 来源为 `server/migrations/20260920110000_commission_applications.sql:60`。按 PostgreSQL 标准 63 字节标识符边界，实际存储名称应为 `financecommissionapplication_organization_id_employee_id_applic`。
3. `server/internal/data/order_write_sea_master_bill.go:230`、`:418` 与 `server/internal/data/sea_order_change_shared.go:128` 使用 `seamasterbill_organization_id_shipping_line_id_normalized_master_no`（67 字节）。正式 SQL 来源为 `server/migrations/20260906120000_sea_shipping_line_identity.sql:45`。按同一边界，实际存储名称应为 `seamasterbill_organization_id_shipping_line_id_normalized_maste`。

两处超长名称必须在调用方显式使用实际数据库名，不能在公共 helper 对任意配置自动截断；Ent schema 与生成迁移元数据仍保留逻辑全名，不手改生成物。相关已有测试引用见 `server/internal/data/commission_application_schema_integration_test.go:183`、`server/internal/data/ent/migrate/commission_application_schema_test.go:57`；它们的逻辑 schema 名称断言与运行时 PgError 名称断言应区分。

### 相关规范

- `.trellis/spec/server/backend/error-handling.md`：驱动错误映射集中在 data 层；禁止字符串匹配驱动错误细节；事务回调错误原样外传。
- 根目录 `AGENTS.md`：唯一约束由数据库兜底，使用 `ent.IsConstraintError` 判断并映射领域错误；不得手改生成物；实现后做触发场景及邻近路径验证。
- 已读取 `.trellis/workflow.md`、领域术语表与架构地图，研究产出仅写当前任务 research 目录。

### 验证建议

实现后验证：nil、普通错误、Ent 无底层驱动结构、正确名称、错误名称、名称前缀/子串、空名称/空领域错误、外层包装错误、多映射顺序，以及错误文本含目标名但 `ConstraintName` 不同。采用正式 SQL 的数据库集成验证，覆盖 finance bill 幂等与两个超长名称的真实返回值。

## External References

本次未使用外网；版本来自本仓库 go.mod，驱动链证据来自现有生成代码。PostgreSQL 63 字节截断值由 ASCII 名称逐字节计算，仍须实施时用实际 catalog/真实违规错误核对。

## Caveats / Not Found

- 尚未实际查询 PostgreSQL catalog 验证两个截断名称；上述实际名按标准 63 字节边界推导，非数据库实测结果。
- 尚未穷尽验证全部调用名称与正式 SQL 的对应关系，约 64 调用/36 名称需实施复核。
- SQLSTATE 的允许范围及当前 pgx/Ent 对 UNIQUE、FK、CHECK、NOT NULL 等具体包装边界未完成实测，实施时需核对，避免普通 PgError 被当作约束映射。
- 未改代码、生成物、schema、规格或执行 Git 操作。
