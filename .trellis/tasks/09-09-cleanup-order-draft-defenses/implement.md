# 清理订单草稿冗余防御代码实施计划

## 1. 实施前提

- 本任务属于前端组件级精简重构，不涉及后端契约变更与数据库迁移；
- 实施时保持对同分公司内草稿能力与多页签守卫的完整兼容；
- 改动完成后按最小定向测试与 Biome 检查进行验证。

## 2. 实施步骤

### Step 1：精简 `OrderFormTemplate.tsx`
目标文件：
- `web/src/components/ui/order-template/OrderFormTemplate.tsx`
- `web/src/components/ui/order-template/OrderFormTemplate.test.tsx`

实施内容：
1. 移除 `previousDraftKeyRef`；
2. 移除 `draftContextChanged` 比对与重置；
3. 移除 `<ProForm key={draftKey}>` 上的 `key={draftKey}`；
4. 调整测试用例，移除人工变更 `draftScope` 期望重置的非真实场景用例。

定向验证：
```bash
pnpm --dir web exec vitest run src/components/ui/order-template/OrderFormTemplate.test.tsx
pnpm --dir web exec biome lint src/components/ui/order-template/OrderFormTemplate.tsx
```
（`pnpm --dir web` 执行时工作目录已是 `web/`，路径统一写相对 `web/` 的 `src/...`；biome 按真实路径解析，带 `web/` 前缀会找不到文件。）

### Step 2：精简 `detail.tsx`
目标文件：
- `web/src/pages/orders/detail.tsx`
- `web/src/pages/orders/detail-change-actions.test.tsx`

实施内容：
1. 移除 `useEffect` 中对 `draftScope` 的无用依赖；
2. 为 `OrderFormTemplate` 补充 `key={orderId}`（记录级身份 key，行为不变、结构兜底，见 design 2.3）；
3. 收敛重复的手动清理逻辑，保留 `pendingExplicitFormRefreshRef` 时序语义（见 design 2.2 收敛项）。

定向验证：
```bash
pnpm --dir web exec vitest run src/pages/orders/detail-change-actions.test.tsx
pnpm --dir web exec biome lint src/pages/orders/detail.tsx
```

### Step 3：规范同步

目标文件：
- `.trellis/spec/web/frontend/state-management.md`

实施内容：
1. 第 3 节「身份命名空间变化时表单必须重新挂载」改写为工作区级机制：身份切换由 `OrganizationWorkspace` key 卸载重挂载兜底，组件不再自行防御；
2. 第 6 节「模板测试：在同一个组件实例切换 draftScope」改写为：身份切换后以新实例渲染新身份的 `initialValues` / 草稿，组件内不再要求 draftScope 切换用例；
3. 补充不变量条款：draftKey 在组件挂载期内恒定，记录级身份由页面 `key={orderId}` 保证。

### Step 4：全链路回归与门禁
运行受影响的前端集成测试：
```bash
pnpm --dir web exec vitest run src/components/ui/order-template/OrderFormTemplate.test.tsx src/pages/orders/detail-change-actions.test.tsx src/components/layout/TagsView.test.tsx
pnpm --dir web tsc
git diff --check
```

建议提交：
```text
refactor(web): 清理工作区建立后的订单草稿冗余防御代码
```
