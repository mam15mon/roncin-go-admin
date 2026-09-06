# 精简订单列表头部与快捷切签

## Goal

统一移除订单列表页面的冗余面包屑导航与顶部快捷状态切签卡片，使用户进入海运出口及后续订单列表页面时直接聚焦于订单数据与核心操作工具栏，符合全站纯白高密度、专注紧凑的企业级视觉体验。

## Background

- 用户明确反馈：不希望海运出口订单列表页面展示面包屑，也不喜欢表格上方包含 12 项状态标签（全部订单、草稿、已订舱、已配舱、拖车已安排、已截单、报关已安排、已放单、退关中、已退关、已完结、异常挂起）的快捷筛选切签栏。
- 当前顶部的面包屑由 Ant Design Pro 的 `PageContainer` 默认从路由配置推导生成，在顶级列表页中属于重复信息（已有左侧高亮菜单项与顶部 TagsView 页签）。
- 快捷切签卡片占用了较大纵向首屏空间，且用户在业务操作中主要依赖工具栏「展开筛选」抽屉进行多维度联合过滤。

## Requirements

- **FR-1: 移除订单列表面包屑**
  - 在 `OrderListTemplate` 中将 `PageContainer` 的 `breadcrumbRender` 设置为 `false`，并在 `header` 中显式指定 `breadcrumb: undefined`。
  - 属性接口上提供 `showBreadcrumb?: boolean`（默认 `false`），保持未来特殊场景的可配置性。
- **FR-2: 移除顶部状态快捷切签卡片**
  - 在 `OrderListTemplate` 中将 `statusTabs` 属性的默认值调整为 `undefined`（不再回退到内部预置的 `defaultStatusTabs`）；仅当外部显式传入非空数组时才渲染 `<Tabs>` 卡片。
  - 在 `web/src/pages/orders/list.tsx` 中移除向 `OrderListTemplate` 传递的 `statusTabs={orderStatusTabs}` 及未使用的引用。
- **FR-3: 保证数据查询与筛选逻辑不受影响**
  - 当无快捷切签时，列表默认加载全部订单；
  - 工具栏「展开筛选」抽屉中的业务流转状态、节点日期、单号类型等复合筛选功能保持 100% 完整可用，无任何行为变更。

## Non-Requirements

- 不修改订单业务接口契约、服务端查询逻辑或数据库模型。
- 不影响订单新建页（`new.tsx`）、详情页（`detail.tsx`）、费用录入页（`fees.tsx`）等下级子页面的合法面包屑导航（这些子页面通过 `OrderPageHeader` 规范展示）。
- 不破坏 `list-constants.ts` 中针对订单流程状态的元数据定义，保留相关常量供筛选器及测试复用。

## Acceptance Criteria

- [x] AC-1: 访问 `/orders/sea-export` 页面，顶部不渲染任何面包屑导航链接（如 `订单管理 / 海运出口`），仅展示清晰的主标题与副标题。
- [x] AC-2: 访问 `/orders/sea-export` 页面，表格上方不渲染包含 12 项状态切签的白色卡片栏，首屏直接展示操作工具栏与订单数据表格。
- [x] AC-3: 列表默认查询全部状态订单；展开筛选抽屉后，仍可正常按各种流转状态进行精确筛选与重置。
- [x] AC-4: `OrderListTemplate.test.tsx` 更新测试用例（覆盖默认不显示切签、以及传入切签时的兼容渲染），全部前端测试（`vitest`）与类型检查（`tsc`）均通过。
