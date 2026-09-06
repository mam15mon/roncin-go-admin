# 修复移动端页面布局与顶栏显示异常

## Goal

修复手机等窄屏移动设备访问货代后台系统时，全局顶栏错位留白、标题被压缩竖排溢出遮挡内容、操作按钮挤出屏幕等布局显示异常，提供正常可用的移动端查看与基础操作体验。

## Background & Confirmed Facts

通过真实移动端视口（iPhone 13 / 390x844、Pixel 7 等）排查与 DOM 分析，确认了导致手机端严重显示异常的核心缺陷：

1. **顶栏绝对偏移写死（关键根因）**：
   - 文件：[web/src/global.less:358-365](file:///home/vince/roncin-go-admin/web/src/global.less#L358-L365)
   - 现状：样式对 `.ant-pro-layout-header` 强制设置了 `left: 180px !important; width: calc(100% - 180px) !important;`，这是为桌面端 180px 常驻侧边栏准备的。
   - 移动端表现：在手机端（屏幕宽度 <= 768px），ProLayout 自动折叠侧边栏并改由抽屉（Drawer）承载。由于页面不再存在左侧 180px 侧边栏，顶栏依然从 `x = 180px` 处开始渲染，导致左半屏 180px 完全空白，顶栏整体被暴力压缩至右侧仅剩约 195~210px 宽度。
2. **标题文字单字竖排溢出**：
   - 文件：[web/src/global.less:398-409](file:///home/vince/roncin-go-admin/web/src/global.less#L398-L409)
   - 现状：`.roncin-header-title-text` 未设置 `white-space: nowrap` 与 `min-width: 0`。
   - 移动端表现：在顶栏被压缩后，8 个字的标题（如“海运出口订单列表”）被挤为单字竖排折行，纵向高度膨胀至 384px，直接向下贯穿覆盖了 TagsView 多页签栏和页面主体内容。
3. **顶栏右侧菜单过多挤占宽度**：
   - 文件：[web/src/components/layout/HeaderMenus.tsx:213](file:///home/vince/roncin-go-admin/web/src/components/layout/HeaderMenus.tsx#L213) 与 [web/src/app.tsx:115](file:///home/vince/roncin-go-admin/web/src/app.tsx#L115)
   - 现状：桌面端顶栏放置了「企业资源」、「设置中心」快捷下拉。
   - 移动端表现：在宽度受限的手机顶部，上述快捷按钮挤占了宝贵的横向空间，导致用户头像、当前组织甚至汉堡菜单被挤出视口或换行重叠。而所有菜单项在左侧抽屉式导航中已有完整层级。
4. **详情/表单吸顶页头高度写死**：
   - 文件：[web/src/components/ui/page-shell/PageHeaderShell.tsx:30](file:///home/vince/roncin-go-admin/web/src/components/ui/page-shell/PageHeaderShell.tsx#L30)
   - 现状：`PageHeaderShell` 设置了固定 `height: 52`，在窄屏换行时内容溢出边框。

## Requirements

1. **REQ-1（顶栏宽度与定位响应式适配）**：
   - 在媒体查询 `@media (max-width: 768px)` 下，覆盖顶栏定位规则，强制 `.ant-pro-layout-header` 及其变体恢复为 `left: 0 !important; width: 100% !important;`，消除左侧 180px 空白。
2. **REQ-2（顶栏标题防折行与自适应省略）**：
   - 顶栏标题容器 `.roncin-header-title` 具备弹性收缩能力（`min-width: 0; flex: 1;`），标题文本 `.roncin-header-title-text` 强制单行 `white-space: nowrap; overflow: hidden; text-overflow: ellipsis;`，杜绝任何竖排溢出。
3. **REQ-3（移动端顶栏空间收敛与布局保护）**：
   - 在移动端窄屏（<= 768px）下，隐藏桌面端顶栏的快捷菜单组（`HeaderMenus`），统一引导通过汉堡抽屉菜单（Drawer）导航，将顶栏空间留给「汉堡按钮 + 页面标题 + 用户头像/切换」。
   - 头像区域（`AvatarDropdown`）在窄屏下支持紧凑模式（保留头像与点击下拉，长文本优雅自适应），防止撑破顶栏。
4. **REQ-4（TagsView 多页签在移动端的自适应）**：
   - 移动端保留 TagsView 多页签栏，位于 48px 顶栏下方（`top: 48px`），单行横向平滑滚动，与顶栏和主体内容无重叠遮挡。
5. **REQ-5（通用页头与表单外壳移动端微调）**：
   - `PageHeaderShell` 将固定高度改为 `min-height: 52px; height: auto`，内边距在窄屏下微调为紧凑边距，确保在手机端包含返回、标题、标签与操作时自适应折行不溢出。

## Acceptance Criteria

- [x] 在常见移动端视口（如 iPhone 13 390x844、SE 375x667、Pixel 7 412x915）下：
  - [x] 顶栏左侧无 180px 空白，左对齐汉堡菜单图标（☰），右对齐用户头像/操作区，横向占满 100% 视口宽度；
  - [x] 页面标题单行居中或靠左显示，超长时显示省略号，无任何纵向竖排溢出或遮挡下方组件现象；
  - [x] 点击左上角汉堡按钮能顺畅唤起侧边栏抽屉，可正常浏览多级菜单并跳转页面；
  - [x] 多页签栏（TagsView）贴合在顶栏正下方，支持单行左右滑动切换。
- [x] 桌面端（> 768px）现有布局与交互完全保持原有视觉效果，无任何样式回退或错位。
- [x] 工作台（`/welcome`）、订单列表（`/orders/sea-export`）及订单详情/创建页在移动视口下均能正常阅读和操作。
- [x] 通过全量前端类型检查（`tsc --noEmit`）、代码风格（`biome:lint`）与单元/UI测试。

## Out of Scope

- 不借本次任务对订单业务复杂表单进行完全移动端原生化重构（如专门设计移动端步骤表单/卡片流），保持企业级后台在响应式下的正常可用即可。
- 不修改任何后端接口、数据模型或权限逻辑。
