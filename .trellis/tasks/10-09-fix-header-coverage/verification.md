# 固定顶部栏修复验收

## 根因与结果
- 当前已安装 ProComponents 3.1.14-6 实际使用 `.ant-pro-layout-header` 外层和 `.ant-pro-global-header` 内层；历史 `layout-mix` 内层选择器已失效。
- 修复前真实工作台折叠侧栏后，侧栏宽 48px，但固定 header 仍从 208px 开始；原折叠选择器未匹配实际 DOM。
- 内层默认 margin-inline 16px，与全局外层 padding 14px 叠加；现将外层 padding 清零、内层 margin 清零并铺满，水平留白交由内层。
- `menu.collapsedWidth: 48` 将组件默认 64px 占位同步为 48px，页签和正文与顶部栏边界一致。
- header 响应式规则采用桌面 >=768px、移动 <768px，与实际 ProLayout 判定一致；外层滚动前后均不透明，z-index 保持 100。

## 浏览器验收
在已有 Vite 开发服务 `http://127.0.0.1:8001/welcome` 使用 Chromium/Playwright 验证真实 AppLayout、工作台和全局样式。以浏览器路由拦截提供非敏感用户及只读 API fixture，正文仅增加 min-height 1800px 以保证可滚动，不修改 header/页签样式、不写数据库。

11 个场景全部通过：1440px 展开初始/滚动、折叠滚动/初始/再次滚动、重新展开滚动；767px 移动滚动、768px/769px 折叠滚动；390px 移动初始/滚动。

| 场景 | header 左边界 | 内外宽度 | 页签顶部 | 结果 |
| --- | --- | --- | --- | --- |
| 1440px 展开 | 208px | 1232px | 48px | 通过 |
| 1440px 折叠 | 48px | 1392px | 48px | 通过 |
| 767px 移动 | 0px | 767px | 48px | 通过 |
| 768px 折叠 | 48px | 720px | 48px | 通过 |
| 769px 折叠 | 48px | 721px | 48px | 通过 |
| 390px 移动 | 0px | 390px | 48px | 通过 |

所有场景检查固定 header 顶部为 0、高度为 48、右边界等于视口右边界，内部与外部尺寸相等，背景为不透明白色，无水平溢出。header/页签区域的左右边缘及中间多点 hit-test 均命中顶部栏或页签，不命中正文；浏览器 pageerror 为 0。此验收验证布局，不覆盖真实用户权限或业务数据。

## 质量门禁
- `pnpm run check:web`：退出码 0；权限键、Proto 常量、架构边界、重复扫描、Biome lint、tsc、Vitest 全部通过。
- Vitest：184 个文件通过、1 个跳过；1089 个用例通过、12 个跳过，47.04 秒。
- 修改文件 `Biome check src/app/AppLayout.tsx`：通过。
- 最终 LESS 编译：通过；Biome 不支持 LESS，未用 CSS 字符串单测替代真实布局检查。
- `git diff --check`：通过。
- 门禁中有范围外的 localhost:3000 连接与 socket hang up 日志，以及未修改测试文件的 7 条 Biome info；未导致门禁失败。act/弃用/Warning 日志均为 0，未修改无关文件。
- 本次仅前端布局修复，未运行后端全量门禁或生产构建。

## 本地复核产物
- 浏览器脚本：`/tmp/roncin-header-validate.cjs`；结果：`/tmp/roncin-header-validation.json`、`/tmp/roncin-header-browser.log`。
- 前端门禁日志：`/tmp/trellis-header-check-web.log`。
- 折叠滚动与移动端截图：`/home/vince/.codex/visualizations/2026/10/09/01a11f7e-4dcb-7780-9d1f-6bb5495fab9e/header-collapsed.png`、`header-mobile.png`。
