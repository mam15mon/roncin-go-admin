# PRD：修复总部用户新建订单入口未收口与请求层 TypeError

## Goal

订单新建入口（路由与列表按钮）按 create 权限收口，新建页候选请求按权限门控，并修复请求层错误被消费后 resolve undefined 导致 unwrapList 抛 TypeError 的崩溃链路。

## 背景

总部（headquarters）角色未授予订单创建权限，但总部用户在订单列表仍能看到并点击「新增订单」按钮，进入 `/orders/:kind/new` 后收到报错文案 `Cannot read properties of undefined (reading 'data')`。

### 根因（诊断结论，2026-09-21 会话）

1. **入口未按权限收口**：
   - `web/config/routes.ts:136` 的 `/orders/:kind/new` 路由没有挂 `access` 守卫（同级拆票路由挂了 `canSplitSEOrders`）。
   - `web/src/pages/orders/list.tsx:195` 将 `onCreateOrder` 无条件传给 `OrderListTemplate`；`OrderListToolbar.tsx:175` 只要有回调且非 `readonly` 就渲染「新增订单」按钮，未判断 `canOrder(businessType, 'create')`。
2. **无权限仍发请求 + 请求层崩溃链**：
   - `use-order-create-options.ts:42` 的查询 `enabled` 只看 `definition && organizationId`，不看 create 权限；页面 403 兜底渲染前查询已发出。
   - 其中 `orderServiceListPersonnelOptions` 后端要求订单 create 权限且当前组织在可写范围内（`order.proto:36`、`server/internal/server/auth.go:318`），总部用户被 403。
   - `web/src/utils/requestClient.ts:98-105`：错误被全局 errorHandler 消费后 `request()` 以 `undefined` 正常 resolve（复刻 umi-request 行为）。
   - `orderOptionsCache.ts` 的 `.then(unwrapList)` 在 `undefined` 上读 `response.data`（`utils/api.ts:13` 无空值保护）→ TypeError。
   - TypeError 文案经 `utils/queryClient.ts:33` 全局 onError / 新建页「主数据加载失败」兜底页直接展示给用户。

## Requirements

1. **新建订单入口按 create 权限收口**：
   - `web/src/access.ts` 暴露布尔键 `canCreateAnyOrders`（`[1,2,3,4].some(t => canOrder(t, 'create'))`，与 `canReadAnyOrders` 同型），供路由守卫消费。
   - `web/config/routes.ts` 的 `/orders/:kind/new` 挂 `access: 'canCreateAnyOrders'`；按 kind 的精确判断继续由页面内 `new.tsx` 的 `canOrder(kind, 'create')` 兜底（与现有拆票路由双层结构一致）。
   - `web/src/pages/orders/list.tsx` 仅在 `access.canOrder(definition.businessType, 'create')` 为真时传入 `onCreateOrder`，无权限不渲染按钮。不改 `OrderListToolbar` 模板。
2. **新建页候选查询按权限门控**：
   - `useOrderCreateOptions` 增加 create 权限入参（由 `new.tsx` 用 `access.canOrder(...)` 计算后传入），无权限时 `enabled` 为 false，不发起 personnel options 等请求，从源头消除无权限 403。
3. **消除 `.data` TypeError 崩溃链（防御纵深）**：
   - `request` 在错误被全局处理后 resolve `undefined` 是既有契约，不改 `requestClient.ts`（影响面大）。
   - 在 `web/src/features/orders/options/orderOptionsCache.ts` 的 4 个取数包装（`getMasterDataOptions` / `getCachedPorts` / `getCachedAirports` / `getOrderPersonnelOptions`）中，对 resolve 为 `undefined` 的响应抛出明确中文业务错误，不允许 `unwrapList` 在 undefined 上崩溃，也不允许静默返回空列表掩盖错误。
   - `web/src/pages/orders/common.ts` 的 `searchOrderLocations`（地点联想，同页面同类读 `.data` 的路径）同步加 undefined 防护，抛同一风格的明确业务错误。

## 非目标（明确不做）

- 不改 `requestClient.ts` 的「错误被处理后 resolve undefined」全局契约。
- 不给 `/orders/:kind/:id`（订单详情）、`/orders/:kind/:id/fees`（费用录入）路由补 access（另属同类风险，单独评估，不在本任务扩散范围）。
- 不改后端权限规则与种子数据；后端行为正确。
- 不为「新增订单」按钮做禁用置灰，无权限直接不渲染（与现有 `onCreateOrder` 缺省即隐藏的模板行为一致）。

## Acceptance Criteria

- [ ] 总部（无 `business.order.*.*.create` 权限）用户：订单列表不出现「新增海运出口订单」按钮；直接访问 `/orders/sea-export/new` 被路由守卫拦为 403；全程无「Cannot read properties of undefined (reading 'data')」报错，也无 personnel options 请求发出。
- [ ] 有 create 权限的分公司用户：按钮可见、可进入新建页、主数据与人员候选正常加载、可正常创建订单（回归不受影响）。
- [ ] `orderOptionsCache` 各取数函数在底层请求 resolve `undefined` 时 reject 出明确中文错误（而非 TypeError），且缓存条目被清除，下次调用可重试。
- [ ] 定向测试通过：`access` / `list` / `new` / `orderOptionsCache` / `use-order-create-options` / `common` 相关用例；改动文件 Biome 与 `pnpm --dir web tsc` 通过。

## 验证命令

```bash
pnpm --dir web exec vitest run src/access.test.ts src/access.workspace.test.ts \
  src/pages/orders/new.test.tsx src/pages/orders/list.test.tsx \
  src/features/orders/options/orderOptionsCache.test.ts \
  src/pages/orders/use-order-create-options.test.ts src/pages/orders/common.test.ts
pnpm --dir web tsc
```
