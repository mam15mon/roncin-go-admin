# 结案待审补录门禁实施记录

日期：2026-10-13。子代理仅实现与定向验证，主会话负责规范、提交及最终门禁。

## 实现

- `biz.OrderClosureReadiness` 增加 `HasPendingFeeSupplement`；目标 CLOSED 时拒绝待审补录，返回请求独立的 HTTP 409 / `ORDER_CLOSURE_BLOCKED`，中文提示“存在待审核补录，请先处理后再结案”。不修改共享错误对象。
- `ClosureReadiness` 及 `TransitionClosure` 的 Order 行锁后都调用现有 `hasPendingFeeSupplement`；事务复核显式传 `tx.Client()`。
- 详情页复用 `feeReview.hasPendingSupplement` 禁用结案按钮并提示原因；确认弹窗提交前重读 `orderRef`。已有服务端 allowedActions 继续表示生命周期/权限资格，不扩大成新的 readiness API。
- 受控补录所有成功流转通过既有 `refreshCapabilities` 新增 `order-detail` 查询失效，刷新详情投影；失败不触发成功刷新。
- 未改补录服务端入口的生命周期资格；结案后仍可提交、撤回、退回、批准，批准会生成正式费用和确认提成冲减，订单保持 CLOSED。
- 核对 `fees.tsx` 补录入口只依赖当前公司工作台、品类 `fee.create` 权限与费用批准，不用普通业务写门禁隐藏 CLOSED 订单补录入口。详情费用抽屉只有普通费用编辑，补录经现有费用页办理。

## 实现后定向验证

| 检查 | 结果 |
| --- | --- |
| `go -C server test -p 32 ./internal/biz -run 'TestOrderClosure' -count=1 -v` | PASS，既有主流程/异常/未建账及新增待审、处理后重试、反结案路径 |
| 专用 PostgreSQL 环境 `go -C server test -p 32 ./internal/data -run 'TestOrderClosurePendingSupplement\|TestOrderLifecycleGatePostgres\|TestFeeSupplementApprovalAdmissionPostgres' -count=1 -v` | PASS，16.705 秒，零 SKIP |
| `pnpm --dir web exec vitest run src/pages/orders/detail-status-transition.test.tsx src/pages/orders/order-detail-transitions.test.tsx src/pages/orders/components/fees/FeeSupplementSection.state.test.tsx` | 详情编排与补录状态两套 PASS；确认函数初次存在一个空拒绝值的断言错误，修正后单独重跑 7/7 PASS，合计 24 用例通过 |
| `pnpm --dir web exec vitest run src/pages/orders/order-detail-transitions.test.tsx` | 修正后 PASS，7 用例，2.23 秒 |
| `pnpm --dir web tsc` | PASS |
| 修改的 5 个前端文件 `biome check` | PASS |
| `git diff --check` | PASS |

真实库使用主会话创建的专用集成数据库，通过现有 `getIntegrationData` 按正式 SQL 迁移建立隔离 Schema，未读取或打印凭据，未重置开发库。未发送钉钉通知，仅验证事务内提醒任务入队。

## 真实库验收证据

- 有待审、没有未建账正式费用时，readiness 与锁内门禁均拒绝；拒绝后订单版本/结案状态/closedAt、生命周期事件及成功审计均不产生部分写入。
- 没有费用金额读取权限时仍有 pending 状态投影，Summary 不泄露；撤回/退回清理待审后，已业务锁定且没有收付款事实的订单可结案。
- CLOSED 后依次真实提交/撤回、提交/退回、提交/批准；普通请求者无批准权限；批准创建费用及 CONFIRMED 提成冲减，仍为 CLOSED，普通费用修改仍拒绝。
- CLOSED 后存在待审及未建账费用时仍可反结案；撤回待审后未建账继续阻断结案，正式账单关联创建后可结案。
- 使用通道及 Order 查询拦截器建立有序并发屏障，首操作写入并持真实锁，等次操作开始查询才提交。补录先提交→结案拒绝；结案先提交→补录允许且 CLOSED；批准先提交→未建账费用拒绝结案。无 sleep 猜测先后。
- 新的补录真实库夹具为 SI；既有生命周期回归覆盖 SE 业务锁、反结案、版本冲突等相邻路径。准入代码为所有品类共用实现，没有品类分支。

## 前端与验证边界

- 待审按钮原因、状态投影刷新恢复、确认打开后后台新增待审、已结案反结案、无费用投影由后端复核均有定向覆盖。
- 空响应和接口拒绝均不误报结案成功、不调用成功刷新；受控补录创建/撤回/批准/退回均刷新订单详情投影，保持 CLOSED 事实。
- 新增未超过 10 个重型详情编排用例；本次最慢前端套件低于 20 秒，测试输出没有新增 act/antd 警告。
- 无 Proto、Schema 或权限清单变化；最终 `check:fast` 由主会话统一执行。第二阶段通用费用更正不在本子任务实现。
