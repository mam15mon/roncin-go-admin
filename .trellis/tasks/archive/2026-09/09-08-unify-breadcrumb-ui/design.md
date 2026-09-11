# 统一全站面包屑UI规范与组件 - 技术方案 (design.md)

## 1. 架构与设计原则

遵循 `AGENTS.md` 的纯白高密度企业级视觉规范及 `@/components/ui` 收口原则：

1. **真实可达性原则**：
   面包屑中的每一级可点击节点必须对应**真实存在的页面**，禁止放置无实体内容的折叠菜单/纯重定向路由（如 `/orders`）。
2. **单一真相源原则**：
   - 全站整页分节表单与详情页的页头和面包屑统一由 `PageHeaderShell` 呈现。
   - `DocumentDetailLayout` 不再承担自研面包屑职责，移除其手写的 `breadcrumbs` 及 `extraBreadcrumb` 区域，专注于单据操作栏与内容排版。
3. **末级高亮与标题语义原则**：
   - 维持当前紧凑吸顶设计：面包屑作为前置链路（`crumbs`），当前页面标识作为主标题（`title`），平铺于页头左侧同一行。
   - 分隔符全站统一使用 `/`，字符颜色为 `rgba(0, 0, 0, 0.3)`。
   - 链接统一优先使用 Umi `<Link>`（支持 hover 预加载与标准的 a 标签无障碍），支持快捷点击与右键在新标签页打开。

## 2. 变更影响矩阵

| 文件 | 变更内容 |
| :--- | :--- |
| `web/src/pages/orders/components/OrderPageHeader.tsx` | 移除 `{ label: '订单管理', href: '/orders' }`，仅保留 `[navTitle, listPath]` 及详情单号节点 |
| `web/src/components/ui/document-detail-layout/DocumentDetailLayout.tsx` | 移除冗余的内部 breadcrumb 渲染逻辑（或标记废弃并清空默认渲染），仅保留 actions 和 children |
| `web/src/pages/orders/components/detail/OrderDetailHeader.tsx` | 清理传给 `DocumentDetailLayout` 的 `breadcrumbs={[]}` hack 代码 |
| `web/src/components/ui/page-shell/PageHeaderShell.tsx` | 保持斜杠 `/` 分隔与统一 token 规范，确保与 `<Link>` 及按钮点击的一致性 |
| `web/src/pages/finance/fees/detail.tsx` | 将 `onClick: () => history.push('/finance/fees')` 规范为 `href: '/finance/fees'` 语义链接 |
| `web/src/pages/partners/partner-detail.tsx` | 将 `onClick: () => history.push(listUrl)` 规范为 `href: listUrl` 语义链接 |
| `.trellis/spec/web/frontend/component-guidelines.md` | 同步更新页头与面包屑规范说明，删除“订单管理 >”历史写法 |

## 3. 测试与验证策略

- 运行 `PageHeaderShell.test.tsx`、`OrderPageHeader` 相关测试。
- 运行定向 vitest 测试：`pnpm --dir web exec vitest run src/components/ui/page-shell/PageHeaderShell.test.tsx`。
- 运行 Biome 检查：`pnpm --dir web biome:lint`。
- 检查 `git diff`，确保无不必要更改。
