# 实施计划：精简订单列表头部与快捷切签

## 阶段与执行步骤

### 阶段 1：组件能力升级（OrderListTemplate）

- [x] **步骤 1.1**：修改 `web/src/components/ui/order-list-template/types.ts`
  - 在 `OrderListTemplateProps` 中新增 `showBreadcrumb?: boolean`，标注默认值为 `false`。
- [x] **步骤 1.2**：修改 `web/src/components/ui/order-list-template/OrderListTemplate.tsx`
  - 移除内部冗余的 `defaultStatusTabs` 声明；
  - 模板默认参数中 `statusTabs` 不设默认值（默认为 `undefined`），新增 `showBreadcrumb = false`；
  - `PageContainer` 显式设置 `breadcrumbRender={showBreadcrumb ? undefined : false}` 与 `header={{ title, subTitle, breadcrumb: showBreadcrumb ? undefined : undefined }}`。

### 阶段 2：页面配置瘦身（orders/list.tsx）

- [x] **步骤 2.1**：修改 `web/src/pages/orders/list.tsx`
  - 移除 `import { orderStatusTabs } from './list-constants';`；
  - 移除 `<OrderListTemplate statusTabs={orderStatusTabs} ... />` 中的 `statusTabs` 传递。

### 阶段 3：测试验证与覆盖

- [x] **步骤 3.1**：更新 `web/src/components/ui/order-list-template/OrderListTemplate.test.tsx`
  - 调整第一个测试用例，验证在默认无 `statusTabs` 时，页面不渲染状态切签卡片（`全部订单` 等文本不存在）；
  - 保留并强化第二个测试用例，传入自定义 `statusTabs` 验证向后兼容切签切换功能；
  - 运行 `vitest` 确保 4 个测试全部通过。

### 阶段 4：质量门禁与视觉验收

- [x] **步骤 4.1**：类型与格式检查
  - 运行 `pnpm --dir web tsc --noEmit`；
  - 运行 `pnpm --dir web biome:lint`。
- [x] **步骤 4.2**：端到端截图验收
  - 通过已有运行中的 `pnpm dev` 实例，在无头浏览器中访问 `/orders/sea-export`，截取页面首屏外观，确认面包屑与状态切签栏已彻底消除。
