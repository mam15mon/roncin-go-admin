# 统一审核锁初步调研

## 已核实的现状

| 能力 | 代码证据 | 对本任务的影响 |
| --- | --- | --- |
| 普通费用编辑 | `server/internal/data/order_fee.go:280` | 当前同时检查整票审批冻结、业务锁／生命周期、提成财务锁；必须统一判定，不能只改 UI。 |
| 建账审批前置 | `server/internal/data/order_fee_review.go:147`、`:514` | 当前建账仍校验整票批准及待审变更；需与未锁订单可分供应商先建账的新规则同步。 |
| 账单取消 | `server/internal/data/finance_bill_write.go:226` | 支持取消草稿及已确认账单；存在有效发票、核销、对冲或草稿对冲占用时拒绝。成功停用账单行并释放费用占用；当前按整张账单取消，合并账单会涉及其他费用。 |
| 发票回退 | `server/internal/data/finance_invoice.go:439`、`:467` | 已有取消与红冲入口；不能用费用状态变更冒充票据回退。 |
| 反核销 | `server/internal/data/finance_verification.go:347` | 保留核销历史、停用分配并联动提成重算；反核销不等于真实资金退款。 |
| 对冲回退 | `server/internal/data/finance_netting.go:460`、`:490` | 草稿取消与已确认反对冲分别处理，不能漏掉账单取消依赖。 |
| 锁后费用更正 | `server/internal/data/order_fee_correction_apply.go:69` | 当前支持保留原费、生成冲回及替代费用，已有账单及资金事实不覆盖；用户已选择 A，新流程不允许对仍已建账的原费用走该路径。 |
| 补审资格 | `server/internal/data/order_fee_supplement_lock.go:201`、`server/internal/data/order_fee_correction.go:56` | 当前按实时费用审核权限校验，并不限定原锁定人；用户已明确同公司有该订单审核权限的人均可补审，改用统一订单品类审核资格，不限定原锁定人。 |
| 自动锁 | `server/internal/data/order_lock_apply.go:37` | 自动锁定没有人工 lockedBy；不能直接当作人工审核通过。用户已确认取消自动锁，统一人工审核并锁定；移除各触发入口，同时保留真实财务事实及其审计。 |
| 财务进度 | `web/src/features/finance/fee-progress/index.ts:6`、`server/internal/data/settlement.go:94` | 用户列出的七种状态均已有；当前按整张关联账单开票及核销／对冲进度投影，不是费用行独立核销状态，避免新增重复状态机。 |

## 历史契约与本轮规则的差异

`.trellis/tasks/archive/2026-10/10-13-fee-commission-correction/prd.md` 已记录已建账费用采用冲回／替代、实际资金由财务另行执行的旧决策。本轮明确了未锁定费用必须先回退账务才能直接修改，用户随后选择 A，锁定后的原费用也须先回退原账单再补审。本轮决定替代该场景的旧冲回／替代路径。相关历史已经在当前会话和归档 PRD 内，无需再检索会话日志。

## 已确认：先回退账务，再补审

用户选择 A。已锁定订单修改／调减／作废已建账原费用：先由有权限人员按依赖办理票据、核销、对冲及账单回退；恢复未建账后才提交补审。批准时再次检查费用版本、原账单占用和订单审核依据。订单保持锁定，普通编辑仍被禁止；新增加的费用不要求回退其他费用账单。

当前账单取消和发票回退以整单为单位，合并单据可能需要整单回退、再建立有效单据；需展示影响范围，不能静默只改费用标签。反核销不等于真实退款，已发提成与实际资金历史保留。新流程不再用补审直接生成已建账原费用的冲回／替代账务来绕过回退。

## 后续设计关注

已确认取消自动锁。补审人员已确认采用同公司同品类审核资格，不限定原审核人。解锁／结案、合并单据回退及待审期间无关费用建账方案见 design.md，等待整体审阅。自动锁退役需同步清理 biz/order_auto_lock.go 及建账、费用删除、核销和对冲等触发方；人工审核保留单证历史保护。Schema、权限和正式迁移设计在产品边界明确后完成；当前只提交规划资料，不执行迁移、不修改产品代码。

## 本轮设计补充证据

- `server/internal/data/ent/schema/order_fee_correction_request.go`：现有 ADD／REPLACE／VOID、申请版本、前后快照和同票一份待审唯一索引可复用；批准依据当前为独立费用审核事件，须改为统一人工审核依据。
- `server/internal/data/order_lock_unlock.go`：现有锁定周期、解锁申请与直接解锁授权路径可复用；增加待审变更互斥以防跨代次批准。
- `server/internal/biz/order_transition.go:156`、`server/internal/data/order_write_lifecycle.go:187`：现有结案检查待审补录，延续统一变更判定；结案后受控变更已在规范中明确允许。
- `server/internal/data/ent/schema/order.go:77`：锁定人／时间／代次已有，同时存在整票费用状态与独立审批指针；统一运行态不能保留双状态自动同步。

本任务没有剩余必须单独提问的业务分歧；边界处理已作为完整设计提案写入 PRD，仍须用户审阅该最终版本后才实施。迁移异常数据处置是实际升级前的检查点，不能预先假定存在或自动修正。
