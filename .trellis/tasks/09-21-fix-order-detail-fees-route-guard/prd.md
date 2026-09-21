# PRD：收口订单详情与费用录入路由权限并加固取数 undefined 防护

## Goal

订单详情与费用录入两条路由按权限收口（费用路由兼容仅持 lock 权限的补录审批人入口），订单详情与费用工作台取数链对请求层 resolve undefined 加明确业务错误防护。

## 背景

上一任务（archive/2026-09/09-21-fix-order-create-entry-guard）收口了 `/orders/:kind/new`，PRD 明确遗留 `/orders/:kind/:id`（订单详情）与 `/orders/:kind/:id/fees`（费用录入）两条路由未挂 access 守卫。本次一并收口。

### 调查结论（2026-09-21 会话）

1. **详情页**：数据链 `orderServiceGetOrder`（要求订单 read）等 5 个并行请求；`use-order-detail-data.ts:83-85` 直接读 `orderRes.data` / `unwrapList(docsRes)` / `unwrapList(personnelRes)`，请求层错误被全局消费后 resolve undefined 时抛 `Cannot read properties of undefined (reading 'data')`，与上一任务修复的崩溃同类。
2. **费用页**：数据链 `orderServiceGetOrder` + `orderFeeServiceListFeeOptions`（fee.read）；`use-order-fee-options.ts:74-75` 同类裸读。
3. **费用页特殊入口**：工作台 `FinanceSummaryCard`「前往处理」按钮跳转费用页处理锁后补录审批。后端 `ApproveOrderFeeSupplement` 仅要求登录态 + 分公司身份 + 实时 lock grant（`order_fee_supplement_usecase.go:346`），lock grant 资格 = 持有 `business.order.<kind>.lock` 权限码（`order_lock_qualification.go:140`），**不要求 fee.read**。费用路由守卫若只看 fee.read 会误伤该审批流。
4. **详情页入口**：列表行点击（kind 列表页已有 canReadSEOrders 等守卫）、工作台订单提醒卡、变更历史抽屉；均为订单读取人群，粗粒度 `canReadAnyOrders` 不误伤。

## Requirements

1. **路由守卫**：
   - `web/config/routes.ts` 的 `/orders/:kind/:id`（订单详情）挂既有布尔键 `canReadAnyOrders`；按 kind 的精确读取仍由后端 `GetOrder`（order read）执行，形成与新建路由一致的双层结构。
   - `/orders/:kind/:id/fees`（费用录入）挂新布尔键 `canAccessAnyOrderFees`：`[1,2,3,4].some(t => canOrder(t, 'fee.read') || canOrder(t, 'lock'))`，同时覆盖费用录入人群与仅持 lock 权限的补录审批人（工作台「前往处理」入口）。
2. **取数防护（同上一任务的 ensureListResponse 模式）**：
   - `use-order-detail-data.ts`：queryFn 内对 `orderRes` 为 undefined 抛「订单详情加载失败，请稍后重试」；`docsRes` / `personnelRes` 经 `ensureListResponse` 转明确业务错误。
   - `use-order-fee-options.ts`：对 `orderRes` / `optionsRes` 任一 undefined 抛「加载费用信息失败，请稍后重试」。
   - 不改 `requestClient.ts` 全局契约；不静默返回空数据。
3. **入口按钮不加改**：详情页「费用录入」按钮已有 `fee.read` 门控（detail.tsx:544）；工作台「前往处理」按服务端裁剪的待审批列表展示，不做第二套前端权限判断。

## 非目标

- 不改后端任何权限规则、lock grant 逻辑与补录审批流程。
- 不为详情/费用页补页面级 per-kind 403 兜底（后端 403 + 错误态已覆盖，维持与列表入口一致的边界）。
- 不处理 finance 域路由（`/finance/fees/detail/:orderId` 已有 canReadFinanceFees 守卫）。

## Acceptance Criteria

- [ ] 无任何订单 read 权限的用户直接访问 `/orders/sea-export/<id>` 被路由守卫拦为 403，不发起 GetOrder 请求。
- [ ] 既无 fee.read 也无 lock 权限的用户直接访问 `/orders/sea-export/<id>/fees` 被拦为 403。
- [ ] 仅持 `business.order.se.lock` 的补录审批人可进入费用页（工作台「前往处理」不被误伤）。
- [ ] 详情/费用取数在请求层 resolve undefined 时展示明确中文错误（如「订单详情加载失败，请稍后重试」），不再出现 `Cannot read properties of undefined (reading 'data')`。
- [ ] 定向测试通过：access 新键用例、use-order-detail-data / use-order-fee-options undefined 防护用例、既有 orders 相关用例回归；改动文件 Biome 与 `pnpm --dir web tsc` 通过。

## 验证命令

```bash
pnpm --dir web exec vitest run src/access.test.ts \
  src/pages/orders/use-order-detail-data.test.ts src/pages/orders/use-order-fee-options.test.ts \
  src/pages/orders/fees.test.tsx src/pages/orders/detail-draft-lifecycle.test.tsx
pnpm --dir web tsc
```
