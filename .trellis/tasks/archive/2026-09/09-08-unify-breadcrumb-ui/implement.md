# 统一全站面包屑UI规范与组件 - 实施计划 (implement.md)

## Phase 1: 契约与规范确认
- [x] 完成全站面包屑现状排查并分析根因。
- [x] 确认剔除虚拟目录节点（如 `/orders`）的方案。
- [x] 更新 `prd.md` 与 `design.md`。

## Phase 2: 组件重构与虚拟节点清理
- [x] 步骤 2.1: 修改 `web/src/pages/orders/components/OrderPageHeader.tsx`，移除无实体的「订单管理」节点，只保留当前品类导航项。
- [x] 步骤 2.2: 优化 `web/src/components/ui/document-detail-layout/DocumentDetailLayout.tsx`，移除内部手写面包屑渲染，避免与 `PageHeaderShell` 产生双层冲突。
- [x] 步骤 2.3: 清理 `web/src/pages/orders/components/detail/OrderDetailHeader.tsx` 中的 `breadcrumbs={[]}` 兼容写法。
- [x] 步骤 2.4: 规范各业务页面（`fees/detail.tsx`、`partner-detail.tsx`）中的面包屑链接配置为标准 `href`。
- [x] 步骤 2.5: 更新 `.trellis/spec/web/frontend/component-guidelines.md` 规范文档。

## Phase 3: 针对性验证与质量把控
- [x] 步骤 3.1: 运行定向单元测试：
  - `pnpm --dir web exec vitest run src/components/ui/page-shell/PageHeaderShell.test.tsx`
  - `pnpm --dir web exec vitest run src/pages/orders/components/OrderPageHeader.test.tsx`
  - `pnpm --dir web exec vitest run src/components/ui/order-list-template/OrderListTemplate.test.tsx`
- [x] 步骤 3.2: 运行前端 Biome 与代码格式检查及 TypeScript 检查（`tsc --noEmit`）。
- [x] 步骤 3.3: 检查 `git diff`，确保无残留或多余改动。
