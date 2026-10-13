# 待审补录与结案准入设计

## 业务不变量

业务结案不要求回款/付款结清；已结案仍允许受控补录或更正，不需反结案。新增检查只针对“执行结案时已存在 PENDING 补录”。因此 CLOSED 与 PENDING 可以合法共存，不能以数据库互斥约束或全局 CLOSED 拒绝实现本需求。

## 服务端

在 `biz.OrderClosureReadiness` 增加明确的待审补录事实字段。只读 `ClosureReadiness` 与 `TransitionClosure` 事务内复核共用 `hasPendingFeeSupplement`（已存在于 data/order_fee_review.go），显式传入正确 client；后者传 tx.Client()，避免读取跑出事务。

领域用例在目标为 CLOSED 时纳入该条件；仓储在现有 Order FOR UPDATE 和版本/来源状态复核之后重新查询，不能只信任领域预读。沿用 HTTP 409 与 ORDER_CLOSURE_BLOCKED，针对该原因给出明确中文提示“存在待审核补录，请先处理后再结案”；使用请求内错误实例，不修改共享错误对象。

反结案不增加此条件。受控补录 Create/Approve/Reject/Withdraw 不新增 CLOSED 限制。批准补录生成未建账费用时，结案仍由既有未建账条件阻断。

## 前端

现有 `Order.fee_review.has_pending_supplement` 已由 Proto 提供，优先复用现有服务端投影提示“存在待审核补录，请先处理后再结案”；不引入轮询、新 readiness API 或前端独立查全量补录。

入口位于 `web/src/pages/orders/detail.tsx` 的状态操作及 `OrderStatusSection`。利用最新订单投影控制结案按钮和提交前提示，反结案不受影响。补录操作成功后按既有查询机制刷新投影；无预读权限或数据尚未加载时不得假造“无待审”，最终由服务端权威门禁拒绝。

`confirmOrderClosure` 已验证 `response?.data` 才提示成功；保留这一行为，后端拒绝时不得提示结案成功或执行成功刷新。实现时追踪 `orderAllowedActions` 与 fee_review 富化顺序，避免只在一个入口修改能力。现有 allowed_actions 主要描述生命周期资格；不将这次修复扩大成全量 readiness API 重构。

预计无需 Proto/Schema 变更，若实现发现必须改契约，则先更新设计，再从源生成。

## 并发

复用现有结案及补录提交/批准的 Order 行锁，不增加锁字段或改变授权父行 → Order 的顺序。

| 实际提交先后 | 期望 |
| --- | --- |
| 补录先提交，结案后检查 | 看见 PENDING，结案拒绝，无生命周期/审计部分写入 |
| 结案先提交，补录后提交 | 补录正常进入待审，订单保持 CLOSED |
| 补录批准先提交，结案后检查 | 新费用未建账时拒绝结案 |
| 已结案后批准补录 | 生成费用及对应提成调整，保留 CLOSED |

只读预检可能早拒绝，允许刷新重试；不能用过期预检覆盖事务内读取。测试使用明确并发屏障，不靠 sleep 猜先后。

## 验证和风险

领域覆盖待审拒绝及清理后可结案，真实库覆盖上述顺序及 CLOSED 后完整补录审批、异常/未建账/权限/版本相邻路径。前端覆盖待审原因、刷新、后台状态变化导致的提交失败以及反结案入口。

不扫描重置已有 CLOSED + PENDING 数据，该组合按新规则可能合法。发布/回退仅涉及本子任务代码和测试；回退会恢复遗漏门禁，不改历史结案事件。
