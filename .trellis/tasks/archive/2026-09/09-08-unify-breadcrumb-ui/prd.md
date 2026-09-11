# 统一全站面包屑UI规范与组件

## Goal

彻底收口并统一全站面包屑组件、视觉样式与导航交互规范，消除多套自研实现的割裂，剔除虚拟无实体重定向层级，建立纯白高密度企业级后台的一致体验。

## Confirmed Facts & Technical Constraints

1. **多套实现割裂**：
   - `PageHeaderShell` 自带一套手写内联面包屑（使用 `/`，`rgba(0,0,0,0.45)`）。
   - `DocumentDetailLayout` 自带一套手写面包屑（使用 `>`，`#64748b`，末级高亮 `#1677ff`），并在 `OrderDetailHeader` 中被硬传 `breadcrumbs={[]}` 掩盖。
   - Ant Design Pro 自带的原生 `.ant-breadcrumb` 在多数模板中被强关，缺少统一封装。
2. **虚拟目录层级问题**：
   - `OrderPageHeader` 中硬编码了 `{ label: '订单管理', href: '/orders' }`，但 `/orders` 是纯重定向路由（直跳 `/orders/sea-export`），在其他品类（如海运进口、空运）中点击会发生错误的跨业务跳转；且与全站其他模块（如客商直接展示“客户管理”）不一致。
3. **规范真相源**：
   - `PageHeaderShell` 是全站整页分节表单与详情页的统一页头规范容器。
   - 顶栏已有全局 `TagsView`（多页签）+ `HeaderTitle`，一级列表/配置页不需要面包屑，面包屑仅服务于二级及以下详情、编辑、新建与子流程页面的层级上下文导航。

## Requirements

1. **规范组件收口与去重**：
   - 废弃/移除 `DocumentDetailLayout` 中手写的重复面包屑实现，详情页统一由 `PageHeaderShell` 或公共组件承载面包屑。
   - 在 `PageHeaderShell` 中标准化面包屑渲染：统一使用斜杠 `/` 作为分隔符，统一 Design Token 语义色，统一使用 Umi `<Link>` 保持路由预加载。
2. **剔除无实体的虚拟层级**：
   - 面包屑各层级必须对应**真实存在的实体页面**，禁止将纯折叠/纯重定向的一级菜单（如“订单管理”）作为面包屑节点。
   - 修正订单模块的面包屑层级：
     - 新建页：`[海运出口] / 新建订单`
     - 详情页：`[海运出口] / ORD202608260001`
     - 费用页：`[海运出口] / [ORD202608260001] / 费用录入`
     - 拆票页：`[海运出口] / [ORD202608260001] / 拆票`
3. **对齐其他业务模块**：
   - 客商详情（`partners/partner-detail`）：`[客户管理] / 客户名称` 或 `[供应商管理] / 供应商名称`。
   - 财务费用详情（`finance/fees/detail`）：`[费用明细] / 订单号费用详情`。
4. **规范与文档更新**：
   - 同步修正 `.trellis/spec/web/frontend/component-guidelines.md` 中关于页头与面包屑规范的陈述（移除“订单管理 >”历史表述）。

## Out of Scope

- 列表页与主数据配置页新增面包屑（列表页保持无面包屑高密度设计）。
- 全局顶栏 `HeaderTitle` 与 `TagsView` 结构改动。

## Acceptance Criteria

- [x] `OrderPageHeader` 彻底移除无实体的「订单管理」虚拟节点，海运进口/空运等详情页点击上一级只准确返回对应的品类列表。
- [x] 全站详情/表单页面包屑视觉样式统一（分隔符统一为 `/`，字体大小 13px，颜色遵从设计变量，无硬编码非标 hex 颜色）。
- [x] `DocumentDetailLayout` 中的自研面包屑废除或与规范对齐，`OrderDetailHeader` 不再需要 hack 式传入 `breadcrumbs={[]}`。
- [x] 相关前端单测（`PageHeaderShell.test.tsx`、`OrderListTemplate.test.tsx`、订单相关测试）全量通过，无编译报错。
- [x] `.trellis/spec/web/frontend/component-guidelines.md` 文档与代码实现保持一致。

