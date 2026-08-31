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

## 禁止

- 禁止静默纠错、自动回退、旧数据兼容分支掩盖真实错误；发现兼容性风险记录在
  变更说明中。
- 禁止用字符串匹配解析驱动错误细节；映射逻辑集中在 `internal/data`。
