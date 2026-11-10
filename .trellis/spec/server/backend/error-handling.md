# 错误处理规范

## 分层职责

- **领域错误定义在 `internal/biz`**（如 `biz.ErrOrderNotFound`、
  `biz.ErrOrderStatusConflict`），用 Kratos `errors` 构造，携带 HTTP 语义。
- **`internal/data` 负责驱动错误映射**：Ent 错误 → 领域错误，统一经
  `mapEntError` 之类的映射函数，不让 `ent.NotFoundError` 等驱动类型穿透到
  `biz`/`service`。
- `service` 层不做错误翻译，直接把用例返回的业务错误交给传输层。

## 常见映射

| 场景 | 处理 |
|------|------|
| 目标行不存在 | `mapEntError(err, biz.ErrXxxNotFound, nil)` |
| 唯一索引冲突 | `ent.IsConstraintError` 判断 → 对应「已存在」业务错误 |
| 版本不匹配 / 状态机拒绝 | `errors.Conflict` 业务错误（HTTP 409，中文提示「已被更新，请刷新后重试」） |
| 事务回调失败 | 原样外传触发 Rollback，禁止吞掉后继续提交 |
| 事务结束后使用 `txCtx` | 返回业务错误，不开新事务 |

## 命名约束映射

- 命名约束统一经 `mapEntConstraint` / `mapEntConstraints`：先通过
  `ent.IsConstraintError`，再用标准库 `errors.As` 取得当前 pgx/v5 的
  `pgconn.PgError`，核对 SQLSTATE 和完整 `ConstraintName`。
- 只映射当前 Ent 包装的唯一、外键、CHECK 约束错误；普通错误、未知驱动、
  空名称、名称不匹配或其他 SQLSTATE 原样返回。不能从 `Error()` 文本猜名称，
  不能添加字符串回退或多驱动兼容分支。
- 调用方填写正式迁移在 PostgreSQL 中的真实标识符，不使用 `idempotency`
  等子串。超长 ASCII 标识符受 PostgreSQL 63 字节边界影响时，逐项核实并显式
  填写实际名称；公共 helper 不自动截断配置，也不手改 Ent 生成元数据。
- 回归须经过真实 Ent builder 的错误包装链，覆盖消息包含目标名但结构化名称
  不符、名称前缀/子串、非约束 SQLSTATE、多映射与外层包装；真实库验证使用
  正式迁移和专用数据库，入口为 `TestMapEntConstraintPostgres`。

## 提成导出失败与审计失败

- 业务失败不返回已读的部分数据；成功读取后审计失败也不交付数据。
- 双失败保留原业务失败和审计失败的错误链，已知领域失败保留对外 HTTP 语义；
  未知故障对外使用固定安全错误。复合错误的 `Error()` 本身也须安全，不能仅
  检查 HTTP Message，因为结构化运行日志会打印错误文本。
- 禁止直接 `errors.Join` 原始驱动错误，或用会打印底层 cause 的 `WithCause`
  拼接安全错误。验证 `errors.Is`、Kratos `FromError` 及日志字符串三种行为；
  不为本地组合改写全局错误编码器，回归入口为
  `TestCommissionExportAuditErrorChainAndTransportSafety`。

## 禁止

- 禁止静默纠错、自动回退、旧数据兼容分支掩盖真实错误；发现兼容性风险记录在
  变更说明中。
- 禁止用字符串匹配解析驱动错误细节；映射逻辑集中在 `internal/data`。
