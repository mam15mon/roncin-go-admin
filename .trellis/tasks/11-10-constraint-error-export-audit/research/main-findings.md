# 本轮代码核对摘要

> 本文保留规划研究时点的结论与限制；实施后的验证及边界见本任务 validation.md，不将研究阶段的“未实测”解释为最终结果。
- 任务创建已批准，当前仅规划；未写产品代码。
- data/ent_error.go:29 用strings.Contains约束名；Ent生成ent.go:489-511有Unwrap保留wrap，生成builder包裹sqlgraph错误也保留原错误。pgx/v5当前驱动，PgError.ConstraintName可用。
- finance_bill_write.go:81仍传idempotency子串，正式20260826160000迁移索引financebill_organization_id_idempotency_key。精确匹配须同步调用方。
- finance_commission.go:142-169只在成功读完写审计，Count、超限和批读失败直接返回。commissionExportAudit固定success并原样写keyword/状态/日期，需要有限有效摘要。
- Service settlement_commission.go:43-61先RequirePrincipal及organizationIDsForRequestedOrganization，再调用用例；保持拒绝边界，不扩大权限。
- data/finance_commission_ledger.go:113-119通过Data.client(ctx)写审计。测试TestCommissionRepoSaveExportAuditWritesOutsideTransaction只在无事务ctx要求直接Exec，并未证明共享事务隔离。
- transaction.go:Data.client继承txCtx；WithoutCancel会保留其value，不能当作解除事务。本次最薄边界为SaveExportAudit拒绝共享事务上下文、普通请求独立直接写；不新建锁/版本/事务框架。
- AuditLog.schema已支持success/failure；元数据由audit.go resolveAuditValues获取request/trace/IP。现有password_reset.go:106已使用WithoutCancel+5秒模式。
- Kratos errors.FromError通过errors.As找首个类型化Error。双失败错误链需保证对外safe错误先于底层错误，并用errors.Is保留原因；不得把SQL错误拼进Message。
- 原导出测试明确断言超限/Count失败不写审计，本次须一次性更新契约测试；成功仍必须审计后返回，失败不交付部分数据。

## 规划边界

见prd/design/implement，实施阶段先实现再补定向测试；公共错误映射影响多仓储故最终check:server。未知驱动/名不匹配不字符串回退；命名约束截断以真实SQL逐项核查，不公共静默截断。
