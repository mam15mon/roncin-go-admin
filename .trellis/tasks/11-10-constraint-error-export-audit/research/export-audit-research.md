# Research: 提成导出最薄失败审计

> 本文保留规划研究时点的结论与限制；实施后的验证及边界见本任务 validation.md，不将研究阶段的“未实测”解释为最终结果。
- Query: 核对 ExportScoped 导出失败审计的身份、范围、取消、共享事务与错误传播边界，并提出最小实现及验证。
- Scope: internal
- Date: 2026-11-10

## Findings

### 现有文件与调用链

- `server/internal/service/settlement_commission.go:43`：ExportCommissions 先 RequirePrincipal，再按 FinanceCommissionExport 经 organizationIDsForRequestedOrganization 解析范围，随后调用 ExportScoped；DTO 转换在用例成功后进行。
- `server/internal/service/organization_scope.go:32`：先权限解析，再校验可选 OrganizationId 的 UUID 与允许范围；范围外返回 ErrPermissionDenied。身份、权限和公司拒绝发生在用例前，不应在用例内伪造该次导出审计。
- `server/internal/biz/finance_commission.go:142`：统一 trim 筛选后校验组织、actor、状态与日期；再 CountScoped、10000 上限、每批200行读取，只有全部读取成功才 SaveExportAudit。
- `server/internal/biz/finance_commission.go:173`：成功审计保留排序后的 organization_ids，多组织不冒充单一组织；当前原样保存 keyword/status/date，失败参数加入审计后必须限制任意输入。
- `server/internal/data/finance_commission_ledger.go:113`：SaveExportAudit 仅 Data.client(ctx) 再 writeAudit，并未主动隔离事务或取消。
- `server/internal/data/transaction.go:100`：Data.client 看上下文事务值；活动 txCtx 复用事务，结束 txCtx 返回 errTransactionContextClosed。
- `server/internal/data/audit.go:13`：单次 AuditLog INSERT；resolveAuditValues 在事件字段为空时从 requestmeta 获取 request/trace/IP；可复制事件补元数据后使用独立背景上下文，无需新通用事务框架。
- `server/internal/biz/finance_commission_test.go:578`：既有测试覆盖上限、批次、成功审计；超限和 Count 失败明确断言不写审计，需要同步修改。
- `server/internal/data/finance_commission_test.go:373`：OutsideTransaction 测试仅用 Background 直写，没有证明共享 txCtx 下独立提交。
- `server/internal/data/finance_commission_test.go:293`：结束事务上下文测试也包含导出审计；若审计改为显式独立写入，应将这一项移入专用隔离测试，不能同时保留互斥断言。

### 建议实现

主会话已选定的最薄契约：导出不支持共享事务上下文，SaveExportAudit 对活动及结束 txCtx 均显式拒绝；普通调用使用 WithoutCancel + 5秒独立审计期限。下面同时保留被比较的隔离方案及取舍，实施以主设计 design.md 为准。

1. 保留身份和组织范围验证的前置拒绝；actor 为 Nil、组织列表空/Nil/重复时，不构造带虚假归属的审计。组织有效且 actor 有效后建立单个收尾审计流程，保证合法上下文中的参数失败、Count失败、超限、任一批读取失败及成功各写一次。
2. 失败返回 nil items；失败 row_count 定义为已实际返回给用户的行数，因此固定0；如需诊断只另记有限整数 read_row_count。count未知不伪造总数，已知 total 可用单独 matched_row_count。成功保持真实导出行数（含零行）。
3. 安全失败类别用固定阶段 validation/count/limit/batch/canceled/deadline 与稳定内部分类，禁止记录 err.Error、SQL、驱动细节或任意错误 Reason。状态只保存允许枚举；日期只保存验证后的 YYYY-MM-DD，非法日期仅记标记；keyword 可只记字符数/是否存在，或使用明确短上限的摘要，不能把巨大输入原样放进审计。
4. 取消必须成为已发生导出的失败：Count前、批次间、全部读取后且成功审计前检查 ctx.Err；仓储返回 context.Canceled/DeadlineExceeded 时优先标记相应类别。审计成功后请求立即取消不可绝对杜绝，但不能将早已取消仍被 stub 返回成功的路径记成功。
5. 若需要支持共享事务内审计，可选的专用隔离方案（本期不采用）：在 SaveExportAudit 内复制 AuditEvent，先从原 ctx 补 request/trace/IP，再以 context.WithTimeout(context.Background(), 5*time.Second) 建立专用审计上下文，经 Data.client(auditCtx) 获取普通客户端并单次写入。该异常边界只用于此导出审计，不改通用事务框架、不直接访问 d.db、不重试。文档应明确这是独立审计提交的专门契约，所有普通业务读取仍遵守结束 txCtx 拒绝规则。
6. context.WithoutCancel(ctx) 保留 Value，因此单独使用它加 timeout **不能**实现事务隔离。若主实现决定只支持无共享事务的调用，也可在 SaveExportAudit 显式拒绝活动/结束 txCtx，但这只约束调用前提，无法声称支持共享事务内失败审计；主设计已选择拒绝共享ctx并限制调用前提，必须将该边界写入规范与测试，不能声称支持共享事务内导出审计。
7. 只读导出无需新增事务、Schema、API、权限、队列或依赖。独立审计是单条 INSERT 的自动提交；不要为了它搭建事务隔离平台。

### 错误组合与原文泄漏

本地版本 `server/go.mod:94` 为 Kratos v3.0.0；`/home/vince/go/pkg/mod/github.com/go-kratos/kratos/v3@v3.0.0/errors/errors.go:128` 的 FromError 首先 errors.As 找 Kratos 错误，否则用 err.Error 构造 Unknown 错误。`errors.go:28` 的 Kratos Error.Error 会打印 cause；因此直接 errors.Join 或 WithCause(rawDatabaseErr) 都可能通过日志打印SQL/驱动原文。

`server/internal/platform/requestmeta/requestmeta.go` 的 Logging 直接 slog.Any("error", err)，不能只证明 HTTP Message 安全而忽略 Error() 字符串。

- 成功但审计失败：nil items，返回安全的稳定审计失败业务错误。
- 业务失败且审计成功：保留既有领域错误语义；未知底层故障应外层呈现安全内部错误。
- 双失败：必须同时可观察原失败与审计失败，且不把审计失败覆盖原领域错误。可用局部复合 error：Error() 仅返回固定中文摘要，Unwrap() []error 保留两者；将安全的 Kratos 错误置于首个可发现位置，确保 FromError 得到原领域语义或固定内部错误。errors.Is 对原始错误仍成立，审计失败有稳定标记。
- 不建议直接 errors.Join(原数据库错误, 审计数据库错误)，也不建议安全 Kratos.WithCause(raw)；两者 Error() 都会暴露原文。应测试 FromError 的 Message/Reason、返回错误 Error()、errors.Is 三个面向，明确安全包装的优先级。

### 验证边界

- 用例：成功0行/200行/201行/恰好10000行；10001超限不发批读但写失败审计。
- 用例：有效身份公司下未知状态/非法日期/逆序日期记 validation，Count完全不调用；空actor/空组织/重复组织不写审计。
- 用例：Count故障；首批失败；后续批失败（不返回前面已读数据）；每次仅一条失败审计，安全阶段和行数一致。
- 用例：开始前取消、Count取消、批次间取消、最后一批后取消、超时；auditCtx 没有继承取消并具约5秒截止时间，原请求元数据保留。
- 用例：成功+审计失败返回nil数据；业务失败+审计失败同时 errors.Is 可观察，HTTP领域语义不丢，Error()/Message/审计details不含注入的SQL秘密样本。
- Service：无principal、缺导出权限、非公司工作区/范围外指定公司，均不调用Count/批读/审计；现有成功DTO及公司范围测试保持。
- 仓储（已选契约）：普通ctx单条INSERT且没有新Begin；活动共享txCtx和结束txCtx均明确拒绝且不发SQL；普通请求取消后用例给出的审计ctx仍能写入；request/trace/IP保留；审计自己超时返回错误。原有结束txCtx断言可保留并补活动ctx测试。
- sqlmock 能验证选择连接及Begin/Rollback序列；真实PostgreSQL隔离/回滚证明若未执行须明确记录，不能把普通Background测试当证明。

### 相关规范与外部参考

已读 `.trellis/workflow.md`、`.trellis/spec/domain/glossary.md`、`architecture-map.md`，以及后端 `logging-guidelines.md`、`error-handling.md`、`database-guidelines.md` 的事务部分。日志规范已要求导出成功/失败审计，当前实现缺失败路径；数据库规范要求普通仓储经Data.client且结束txCtx不静默新建事务，专门隔离审计应在规范中明确适用边界。

无外部联网研究；错误传播行为依据仓库固定依赖的本地源码，版本为Kratos v3.0.0、Go 1.26.10。

## Caveats / Not Found

- 本研究未修改产品代码、未执行测试、未操作Git；仅形成实现建议。
- Service身份/权限拒绝属于上游门禁，本次最薄导出审计不额外建设认证拒绝审计。
- API未知枚举如何转换需实现时补核 financeCommissionStatusFromAPI；用例直接调用的非法状态测试不能替代真实传输形态。
- 导出当前Count与批读之间没有一致性快照，本任务不扩展为导出快照重构。
