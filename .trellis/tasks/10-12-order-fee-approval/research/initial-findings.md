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

## 审批承载决策（2026-10-12 已确认）

用户已确认首期采用 Roncin 内审批、钉钉发送待审和结果提醒。完整费用、利润和版本复核集中在本系统，现有补录通知机制可作为参考。钉钉 OA 费用审批不纳入首期。

## 自审规则核查

`server/internal/biz/order_fee_supplement_usecase.go:442` 的补录批准与 `:649` 的驳回均实时检查锁授权，未在这两个用例中排除申请人本人；历史任务 `.trellis/tasks/archive/2026-09/09-17-locked-order-fee-policy/implement.md:122` 也列有自审验证。这只能证明原补录业务的处理方式，不能作为新整票费用审核允许自审的产品授权。

2026-10-12 用户明确允许提交人在持有审核权限且满足其余资格时批准自己的申请。本次不增加提交人排除规则。

## 批准冻结与状态筛选（2026-10-12 用户进一步明确）

用户期望“完成录入 → 提交审核 → 有权限的成员批准 → 费用不可改动 → 财务按审核状态筛选建账”。批准后冻结是明确要求，不能仅允许编辑再使批准失效。

- `server/internal/data/order_fee.go:460` 的 Update 当前允许满足公司配置和草稿账单条件的已建账费用编辑；新审核冻结必须优先阻止该路径，不能被 BilledFeeEditEnabled 放行。
- `server/internal/data/order_fee.go:681` 的 Remove 现有约束包含补录来源及有效账单占用，尚无整票批准冻结约束；批量、补录撤销、拆票等写入须一并核查。
- `server/internal/data/finance_bill_write.go:210` 的取消账单独立处理账单关联，不能把取消账单隐式等同于撤销费用批准。
- 新增审核状态筛选须与现有有效账单关联状态分别表达；“已批准”不意味着“已建账”，批准本身也不自动建账。

2026-10-12 用户已确认纠错入口：由具备审核资格的成员显式撤销批准，修改后重新提交；存在有效账单时先取消相关账单，再允许撤销批准。仍须遵守现有账单取消约束、业务锁与财务锁，不能以撤销批准绕过其他限制。

## 负利润展示核查

- `web/src/pages/orders/fees.tsx:591` 当前按应收、应付汇总差额计算界面利润。
- `web/src/pages/orders/components/fees/OrderFeeHeader.tsx:170`、`:180` 当前利润率标签与金额颜色已区分正负，负利润使用红色；颜色展示不构成审批权限或批准记录。
- 本轮检查的权限清单 `server/internal/access/manifest.go` 未发现独立负利润批准权限。2026-10-12 用户明确只设一个费用审核权限，亏损订单不另设批准权限；角色名称不构成自动授权。
- 2026-10-12 用户确认负利润审批意见必填：在一个输入框说明亏损原因，批量审核逐票填写。服务端以统一折算口径的利润快照执行校验，不能依赖前端浮点计算或颜色。

本轮只创建任务、记录需求及研究证据；未修改产品代码、数据库、公共规范或其他任务文件。

## 订单列表审核入口（2026-10-12 新增需求）

- 用户提出列表末列增加「审核」，入口亮起时悬停查看费用和利润，服务于审核人员批量处理订单。
- `web/src/pages/orders/list.tsx:155` 使用公共 OrderListTemplate，`:164` 已通过 extraColumns 传入提成摘要列；`:173` 目前在批量回调中处理标签维护。
- `web/src/components/ui/order-list-template/OrderListTemplate.tsx:272` 的 extraColumns 插入在中间，不能直接用它满足末列要求；`:435` 为已有右侧固定操作列，实施时需提供最小末列扩展入口，业务审批组件仍由订单页面持有，公共 UI 不反向依赖领域模块。
- 同文件 `:646` 已有 rowSelection；工具栏 `web/src/components/ui/order-list-template/OrderListToolbar.tsx:70` 起已有批量菜单结构，可在确定产品范围后接入真实批量审核动作。
- 悬停内容建议包含审核状态、应收／应付笔数及本位币合计、利润、利润率、折算口径，提供完整费用明细入口；整票审核不能只展示本次选中费用。负利润提示、审批意见仍遵守后续确认规则。
- 2026-10-12 用户进一步明确：同时支持逐行快速审核与勾选多票、查看逐票摘要后一次批准。负利润意见逐票必填已确认，批量失败处理粒度待确认。
- 金额／利润摘要由服务端授权并计算，避免列表对无权用户暴露成本；加载完整费用明细可按实际查看请求，不按每一行无条件加载全量费用。

## 订单级审核状态筛选（2026-10-12 已确认）

- 用户明确要求订单持有费用审核状态，方便审核人员筛选。审核以整票为单位，订单页与费用台账应消费同一审批结果。
- `web/src/pages/orders/list-query.ts:23` 将筛选映射到订单服务查询；`web/src/components/ui/order-list-template/OrderListSearchFilter.tsx:266` 负责提交筛选，`:492` 等已有状态筛选控件可参考。新增审核筛选须贯穿请求契约、服务端查询与总数，不采用当前页本地过滤。
- `web/src/components/ui/order-list-template/OrderListTemplate.tsx:646` 提供现有多选，批量审批应与审核状态筛选及完成后的列表刷新联动；多选不构成审批授权。

## 迟到成本与批准冻结的范围冲突

- 历史已交付需求 `.trellis/tasks/archive/2026-09/09-17-locked-order-fee-policy/prd.md:5` 明确支持提成已确认／发放后发现漏录成本，原费用冻结，专用审批新增正数应付并产生提成冲减建议。
- `server/internal/biz/order_fee_supplement_usecase.go:224` 附近的申请流程要求业务锁或财务锁事实；`:476` 附近的审批继续校验锁授权并生成新增费用。这不是普通修改已有费用。
- `server/internal/data/order_fee.go:280` 的普通写入检查业务锁及提成财务锁。因此简单要求所有新增费用都先取消原账单、撤销整票批准，会改变已有迟到成本业务，并可能使已核销／已产生提成的订单无法补录。
- 待确认建议：区分“改原费用”和“补新增成本”。前者继续执行用户已确认的取消账单、撤销批准、修改重审；后者保留原费用与账单，受控补录后停止新的建账，重新审阅整票收入／成本／利润，获批后允许新增费用建账。不能只审新增成本一行，也不能复用旧批准放行它。
- 此建议涉及产品例外，尚未批准；补录资格与现有冲减建议流程的整合设计待该决策明确后完成。

## 用户追问：当前是否能补成本、怎样影响提成

本节核实当前代码行为，不代表用户已同意在新整票审核流程中放行补录。

- 当前存在锁后应付补录申请与审批；审批用例 `server/internal/biz/order_fee_supplement_usecase.go:442` 在同一事务中创建新费用并调用提成影响计算、冲减建议创建。
- `server/internal/data/order_fee_supplement_approval.go:92` 的 LockCommissionImpactContext 读取目标订单 CONFIRMED／PAID 提成行；原提成金额和历史发放事实不在补录审批中重写。
- `server/internal/biz/order_fee_supplement.go:499` 的 CommissionSupplementCostSensitivity 明确已实现毛利计提受成本影响，已实现收入计提不受仅新增应付成本的影响；未知计算版本拒绝推测。
- 同文件 `:539` 计算本笔新增成本加入前后的提成差额，使用原提成快照及此前有效补录成本；`:553` 起按订单行与父单可用余额封顶，不将超额转扣其他订单或员工。
- `server/internal/data/order_fee_supplement_approval.go:233` 创建 DECREASE＋DRAFT 调整，属于待处理建议，不等同自动扣款；`web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx:86` 提供确认冲减，`:140` 起提供必填原因的忽略建议。实际少发后人工标记已扣回，系统不执行工资扣款。
- `server/internal/data/finance_commission_write.go:97` 的提成确认重新计算并比较来源指纹，费用变动导致旧草稿不能直接确认；同一确认流程还检查未建账费用。
- 用于说明的简化例子：同币种、收入全部实现、按毛利 10% 计提且可冲减余额足够，收入 10,000、原成本 8,000，原提成 200；补成本 500 后理论提成 150，生成 50 的冲减建议。实际金额受原计提口径、已实现收入比例、历史快照和可冲减余额约束。
- 用户担忧提成计算变化，不能将其追问视作授权取消补录或同意保留新例外；保留为待确认产品决策。
