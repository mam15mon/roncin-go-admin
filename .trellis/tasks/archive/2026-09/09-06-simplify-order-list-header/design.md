# 技术方案设计：精简订单列表头部与快捷切签

## 1. 架构定位与改动边界

```text
web/src/pages/orders/
  ├── list.tsx                 # 消费端：移除 statusTabs 传参与未使用导入
  └── list-constants.ts        # 保留常量元数据，供筛选器与后续模块使用
web/src/components/ui/order-list-template/
  ├── types.ts                 # 增加 showBreadcrumb?: boolean 属性类型
  ├── OrderListTemplate.tsx    # PageContainer 隐藏面包屑，statusTabs 默认设为可选/undefined
  └── OrderListTemplate.test.tsx # 针对无切签、有切签两种场景补充与更新单测
```

## 2. 核心改动点设计

### 2.1 移除面包屑 (`PageContainer`)

在 `OrderListTemplate.tsx` 中，`PageContainer` 默认会从 Umi / Ant Design Pro 的路由上下文中计算面包屑树。
通过以下配置彻底消除：

```tsx
<PageContainer
  breadcrumbRender={showBreadcrumb ? undefined : false}
  header={{
    title,
    subTitle,
    breadcrumb: showBreadcrumb ? undefined : undefined,
  }}
  style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
>
```

- 默认 `showBreadcrumb = false`；
- 无冗余上级链接干扰，顶部只保留纯白底色、清晰的页面标题与业务说明。

### 2.2 移除顶部状态快捷切签卡片

- 目前 `OrderListTemplate.tsx` 中声明了内部 `defaultStatusTabs` 并将其作为默认参数值；
- 改造为：
  - 移除内部冗余的 `defaultStatusTabs`；
  - `statusTabs?: OrderStatusTabItem[]` 默认为空；
  - 渲染条件保持 `{statusTabs && statusTabs.length > 0 && <Card ...>}`；
  - 在 `currentTab` 状态管理上，当未设置 `statusTabs` 时，默认保持为 `activeStatusTab = 'all'`，不给 `queryOrders` 附带默认 stage 过滤；
- 在 `list.tsx` 中：
  - 移除 `import { orderStatusTabs } from './list-constants';`；
  - 移除 `<OrderListTemplate statusTabs={orderStatusTabs} ... />` 中的 `statusTabs` 属性。

### 2.3 保持筛选与请求无副作用

- `OrderListTemplate` 内部发起请求时：
  ```tsx
  stage: currentTab !== 'all' ? currentTab : currentFilter.stage
  ```
  当未渲染 Tabs 时，`currentTab` 维持 `'all'`，请求中生效的 `stage` 完全由 `OrderListSearchFilter` 展开面板中的表单输入项提供。重置时 `currentFilter` 清空，列表查询全部，逻辑完全自洽。

## 3. 测试与验证策略

1. **单测更新**：
   - 更新 `OrderListTemplate.test.tsx`，断言在无 `statusTabs` 传入时，不渲染状态切签栏，标题和表格正常渲染；
   - 增加带有 `statusTabs` 传参的单独用例，确保组件通用性与向后兼容性；
2. **构建与类型验证**：
   - 运行 `pnpm --dir web tsc --noEmit` 保证 TypeScript 零编译错误；
   - 运行 `pnpm --dir web test` 保证相关用例全部通过；
3. **视觉实测**：
   - 通过本地运行的 `pnpm dev` 实例，在浏览器中抓取实际渲染截图，检查 `/orders/sea-export` 页面视觉效果。
