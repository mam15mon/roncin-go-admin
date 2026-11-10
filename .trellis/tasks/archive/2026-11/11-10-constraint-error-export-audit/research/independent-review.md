# 独立复核结果

日期：2026-11-10。复核角色：独立 `trellis-check`；实现由另外两名代理完成，复核未派生其他代理。

## 范围与依据

- 阅读任务 PRD、设计、执行清单、两份专项研究，以及根目录/服务端协作规范、后端索引质量检查、错误、审计、数据库及分层规范。
- 覆盖工作树中本任务的全部生产代码差异、修改/新增测试及四份现行文档差异；不修改并行客商工作，不扩大到 API、Schema、依赖、权限或生成物。
- 沿 Service 权限和当前公司解析 → Biz 导出 → Data 查询及审计 → HTTP 错误转换与结构化运行日志核对。

## 发现并修复

首轮完整 `pnpm run check:server` 在三个相邻唯一冲突用例失败：

| 文件 | 问题 | 修复 |
| --- | --- | --- |
| `server/internal/data/order_abnormal_case_test.go` | 仍以 `errors.New` 包装 pq 文案模拟唯一冲突，缺少结构化驱动字段 | 改用 `pgconn.PgError`，携带 `23505` 和完整异常标记约束名 |
| `server/internal/data/order_container_test.go` | 同类旧文案夹具不再满足当前驱动契约 | 改用当前 pgx 的结构化唯一错误与完整箱号约束名 |
| `server/internal/data/order_shipping_document_test.go` | 同类旧文案夹具导致分单唯一冲突回归失败 | 改用当前 pgx 的结构化唯一错误与完整分单约束名 |

修复仅更新测试夹具及对应中文注释，没有为通过旧测试添加生产兼容逻辑。首轮失败日志保留在 `/tmp/roncin-constraint-first-server.log`。

## 核对结论

- 命名约束坚持 Ent 类型门禁、`errors.As`、当前 pgx SQLSTATE 和完整 `ConstraintName`；未知错误原样返回，没有文本回退或公共自动截断。两处超长名与账单幂等名有正式迁移和实际 PostgreSQL 回归。
- 导出身份或公司范围无效时不伪造审计归属；有效边界内的参数、Count、超限、批读和取消失败收口为单次失败审计，失败均返回 nil 数据。
- 取消在读取前、批次间及读完后检查；审计使用保留元数据的 `WithoutCancel` 加 5 秒独立超时。活动/关闭共享事务及只读快照使用同一事务标识，审计入口明确拒绝，不绕过 `Data.client`。
- 关键字不记录原文；非法状态与日期不进入审计。复合错误只输出安全摘要，首个可发现的 Kratos 状态移除原始 cause 和 metadata，同时保留业务与审计两侧错误链。
- 已检查项目实际 `internal/server/error.go`：其通过 `errors.FromError` 取得安全状态，未知 500 进一步统一内部错误提示；既有传输转换及 JSON 日志安全测试可覆盖本次局部错误组合。
- Service 的未知数字/显式零枚举依照现有转换进入 Biz 参数拒绝，真实 ProtoJSON 形态已有回归；无身份、仅查询权限、系统/未知工作台和越权公司仍在用例前拒绝。
- 文档保留核查时点及历史台账，新增现行修复说明；社区公约只追加适用边界，保留上游原文。

未发现尚未修复的本任务阻塞问题。

## 验证

- 定向相邻回归：`go -C server test -p 32 ./internal/data -run '^(TestMapEnt|TestOrderAbnormalCaseRepo_|TestOrderContainerRepo_|TestOrderShippingDocumentRepo_)' -count=1`，退出 0，data 包 0.050 秒。
- 最终完整服务端门禁：`pnpm run check:server`，退出 0；包含 Go 重复检测、分层检查、Proto lint、全服务端默认测试、`go vet` 及可达漏洞扫描。最终日志：`/tmp/roncin-constraint-final-server.log`，漏洞扫描未发现未处理的可达漏洞。
- `git diff --check` 通过；错误目录生成器只读校验通过，403 个错误码一致。
- 已检查全部 18 份本次 Go 文件的格式；只有 `settlement_commission.go` 原有未改动字段行存在 gofmt 对齐差异，经 `git show HEAD:<path> | gofmt -d` 确认基线相同，按禁止无关格式化要求保留，不属于本次引入的问题。

## 验证边界

- 默认服务端门禁未提供专用数据库环境，依赖该环境的历史真实库用例按原配置跳过，不能将退出 0 描述为完整 PostgreSQL 验收。
- 实现代理另在本次一次性数据库运行 `TestMapEntConstraintPostgres` 及全部 `TestMapEnt` 定向测试，报告 PASS、无 FAIL/SKIP；本次独立复核阅读并核对该回归实现，没有重复运行其完整迁移准备。
- 导出审计取消、元数据、共享事务拒绝及自身期限以 sqlmock/领域和传输回归验证；本次未额外运行审计真实数据库持久化或共享事务内独立提交验收，后者本就不受支持。
- 不对外部宿主、设备、第三方服务或正式部署声明已验收。
