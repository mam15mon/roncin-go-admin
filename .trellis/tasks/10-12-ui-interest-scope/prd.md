# 利息规则界面与业务范围对齐

## 目标与来源

修复UI-UX审查 A03，消除未完成计息配置被误认为已生效的业务误导。来源为 `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/audit-report.md`，所有原始浏览器证据仍在原审查归档。

## 已确认事实

- A03/P1：`web/src/pages/partners/components/InterestRuleModal.tsx:75` 承诺自动计息，`SettlementSection.tsx:261` 提供编辑和启用摘要；`web/src/pages/partners/partner-detail.tsx:86`、`:916` 仅本地state+成功toast，`:556` 保存及详情回填无字段。
- 已有财务上线范围 `.trellis/tasks/archive/2026-09/09-09-finance-launch-readiness/prd.md:225` 将逾期计息列为非目标。此为既有范围事实；本次处理方式以用户明确选择为准。
- 仅评价该配置链路，没有证明全系统不存在其它计息实现。

## 已确认产品决定

2026-10-12，用户明确选择“本次移除未完成计息入口（推荐）”；本任务按此处理，完整计息延期。

## 需求

R1：用户不再把局部未持久配置误认为已落库并参与计息；界面承诺必须与实际业务闭环一致。
R2：移除未完成的编辑入口、启用摘要、自动计息说明及仅服务该UI的本地state/组件；保留已实现的其它结算能力。不得仅修改toast保留误导性的配置链路。

## 验收

- AC1/R1：不能再通过无后端链路的本地弹窗得到“已更新/已启用/自动计息”承诺。
- AC2/R2：客商其它结算条款、信用额度、账期正常保存回填；移除后全仓库无该未完成UI的悬空引用。
- AC3/R2：删除行为仅作用于未完成前端入口及其专属代码，不修改后端契约、Schema、生成客户端或真实业务数据。

## 非目标

完整逾期计息算法、Schema、计息账单生成或财务写入；此类能力延期，未来需要独立业务规划。

## 共同约束

保持纯白高密度、PageContainer 流式全宽、12px 内容边距、既有权限和用户/组织隔离。复用现有能力，遵循 ProTable request/actionRef、ModalForm.onFinish；不加兼容回退，不手改生成物，不引入全站 KeepAlive 或自动保存平台。禁止 TDD，实现后按风险补回归验证；不修改真实业务数据进行验收。
最终规划已获用户批准，按总任务约定顺序实施；当前进度以 task.json 和 implement.md 为准。
