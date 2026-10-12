# 首轮代码核查（2026-10-12）

## 已核实的现状

| 结论 | 证据 |
| --- | --- |
| 单张建账已有共享事务，用例内加载费用、校验币种和汇率后调用仓储；当前链路未检查整票审核 | `server/internal/biz/finance_bill.go:166` |
| 批量建账已有幂等重放与事务内预览令牌复核；预览有效不等于人工审核通过 | `server/internal/biz/finance_bill_batch.go:512` |
| 单张与批量仓储均锁定所选费用并校验有效账单占用 | `server/internal/data/finance_bill_write.go:23`、`server/internal/data/finance_bill_batch.go:99` |
| 建账也会递增费用行 version，因此原样聚合所有费用行 version 作为审核版本可能误使后续分批建账失效 | `server/internal/data/finance_bill_write.go:90`、`server/internal/data/finance_bill_batch.go:175` |
| 费用已有汇率、本位币及本位币金额快照，无通用审核状态 | `server/internal/data/ent/schema/order_fee.go:21` |
| 现有费用写入涵盖新增、修改、删除、批量修改和批量删除；另有补录生成、补录撤销及拆票转移，后续须完整枚举 | `server/internal/data/order_fee.go:380`、`:460`、`:681`、`:802`、`:877`；领域导航 `.trellis/spec/domain/glossary.md` |
| 建账权限与费用增删改查权限均已存在，尚无独立整票费用审核权限 | `server/internal/access/manifest.go:45`、`:175`、`:222`、`:309` |
| 原有锁授权资格谓词覆盖有效角色、成员关系和组织范围，可参考判定结构；不得继承其锁权限或管理员例外作为新审核规则 | `server/internal/data/order_lock_qualification.go:98` |
| 费用补录在 Roncin 内批准／驳回，并对候选人入队钉钉提醒 | `server/internal/biz/order_fee_supplement_usecase.go:442`、`:649`；`server/internal/data/order_fee_supplement_notifications.go:18` |
| 现有 OA 派发持久化逻辑依赖解锁申请，不能直接视为通用费用审批能力 | `server/internal/data/dingtalk_approval_repo.go:78`、`:83`、`:157` |

## 设计阶段必须解决的事项

- 整票费用内容的审核版本与费用行并发版本如何区分；先确认分批建账的产品口径，再确定字段或指纹方案。
- 不能仅锁定本次选择建账的费用：另一行成本变化、新增、删除或转移都可能改变整票审核依据；需对全部相关写入设计一致的事务与锁序。
- 费用发生时汇率与账单日期汇率分属不同环节，需明确审批利润口径，以及哪类汇率变化使批准失效。
- 幂等重放已有成功账单与发起新建账应明确区分，避免把返回已有结果误当成重新授权建账。
- 现有补录审批是否还需整票重新审核，须与本次“所有费用建账都经过有效整票批准”要求对齐。

## 首个产品决策建议（尚未批准）

建议首期采用 Roncin 内审批、钉钉只发送待审和结果提醒。完整费用、利润和版本复核集中在本系统，现有补录通知机制可作为参考。若选择钉钉 OA，则须额外设计审批实例、费用快照关联、回调后的资格与版本复核，以及重复回调处理，不能直接套用解锁业务。

本轮只创建任务、记录需求及研究证据；未修改产品代码、数据库、公共规范或其他任务文件。
