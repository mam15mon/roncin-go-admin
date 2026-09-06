# 修复移动端页面布局与顶栏显示异常 - 实施计划

## 实施步骤

1. **结构标识与基础组件适配**：
   - 修改 `web/src/components/layout/HeaderMenus.tsx`，添加 `className="roncin-header-menus"`。
   - 修改 `web/src/components/RightContent/AvatarDropdown.tsx`，为用户名称 `span` 添加 `className="roncin-avatar-name"`。
   - 修改 `web/src/components/ui/page-shell/PageHeaderShell.tsx`，将行内 `height: 52` 调整为 `minHeight: 52`。

2. **全局响应式样式适配**：
   - 修改 `web/src/global.less` 中的 `@media (max-width: 768px)` 区块：
     - 覆写 `.ant-pro-layout-header` 及其变体，使其在移动端具有 `left: 0 !important; width: 100% !important; padding: 0 8px !important;`。
     - 设置 `.roncin-header-title` 与 `.roncin-header-title-text` 单行省略截断与弹性缩放规则。
     - 设置 `.roncin-header-menus` 在移动端 `display: none !important;`。
     - 设置极窄屏（`@media (max-width: 480px)`）下 `.roncin-avatar-name` 隐藏仅保留头像圆形徽标，确保在 360~390px 窄屏下空间充裕。

3. **测试与质量门禁验证**：
   - 编写或补充针对顶栏响应式类名与 PageHeaderShell 弹性高度的单元/UI测试。
   - 运行自动化视口截图与布局断言脚本，验证移动端与桌面端的双端布局表现。
   - 执行质量门禁：
     ```bash
     pnpm --dir web tsc --noEmit
     pnpm --dir web biome:lint
     pnpm --dir web test
     git diff --check
     ```
