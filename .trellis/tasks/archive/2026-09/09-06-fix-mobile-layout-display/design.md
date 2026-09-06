# 修复移动端页面布局与顶栏显示异常 - 技术方案设计

## 1. 架构与改动边界

本次改动纯粹聚焦于前端响应式布局（CSS 媒体查询）与基础布局组件（`HeaderMenus`、`AvatarDropdown`、`PageHeaderShell`）的移动端适配，不涉及任何服务端 API、数据契约或业务逻辑。

### 改动文件与定位

1. **`web/src/global.less`**：
   - 在 `@media (max-width: 768px)` 响应式代码块中补充高特异性规则：
     - 重置 `.ant-pro-layout .ant-pro-layout-header` 及其变体：`left: 0 !important; width: 100% !important; padding: 0 8px !important;`；
     - 重置 `.roncin-header-title`：`min-width: 0; flex: 1; overflow: hidden;`；
     - 重置 `.roncin-header-title-text`：`white-space: nowrap !important; overflow: hidden !important; text-overflow: ellipsis !important; display: block;`；
     - 隐藏移动端冗余的顶栏菜单 `.roncin-header-menus`：`display: none !important;`；
     - 优化移动端头像与操作区 `.ant-pro-global-header-header-actions`：紧凑对齐，窄屏下 `.roncin-avatar-name` 优雅适配或在极窄屏（<= 480px）下收起文字仅保留头像。
2. **`web/src/components/layout/HeaderMenus.tsx`**：
   - 为根 `<Space>` 容器赋予标准样式类 `roncin-header-menus`，以便通过样式精准控制桌面展示与移动端隐藏。
3. **`web/src/components/RightContent/AvatarDropdown.tsx`**：
   - 为用户名称 `span` 添加 `roncin-avatar-name` 标识类，确保样式可选择且无行内样式硬锁。
4. **`web/src/components/ui/page-shell/PageHeaderShell.tsx`**：
   - 将行内 `height: 52` 调整为 `minHeight: 52`，允许在移动窄屏下多行换行时自适应高度，避免溢出截断。

## 2. 交互与视觉规范对齐

- **断点标准**：以 Ant Design 与业界通行的 `768px` 作为平板/手机移动端分界线。
- **导航抽屉（Drawer）协同**：移动端完全保留 ProLayout 原生抽屉能力。汉堡图标（☰）贴左，点击从左侧滑出完整纯白菜单抽屉，可无缝展开「订单管理」、「费用管理」、「企业资源」、「设置中心」各级菜单并进行路由切换。
- **TagsView 多页签**：置于 `top: 48px` 紧贴顶栏，保持 `height: 36px` 与单行横向平滑滚动条，保留移动端已打开单据的快速切换能力。
- **桌面端零破坏**：媒体查询严格限定在 `max-width: 768px` 之内，屏幕宽度大于 768px 时所有现有设计（180px 全高常驻侧栏、48px 顶栏、右侧快捷下拉）100% 保持不变。

## 3. 验证与回归方案

1. **真实视口截图与断言对比**：
   - 使用 Playwright 驱动 Chromium 模拟 iPhone 13 (390x844) 与 Pixel 7 (412x915) 视口；
   - 验证登录后 `/welcome`、`/orders/sea-export` 页面：
     - 断言 Header `left === 0` 且 `width === window.innerWidth`；
     - 断言标题文字高度 `<= 48px`（未发生单字折行溢出）；
     - 断言汉堡菜单点击后弹出 Drawer 正常展示；
     - 断言 TagsView 正常位于 `y = 48px`。
2. **桌面端回归断言**：
   - 验证桌面端 (1280x800) 下 Header 仍然 `left === 180px`（展开态）或 `left === 48px`（收起态），桌面端功能菜单完整呈现。
3. **前端质量门禁**：
   - `pnpm --dir web tsc --noEmit`
   - `pnpm --dir web biome:lint`
   - `pnpm --dir web test`
