# 前端清理复核

## 确认删除项与证据

- `src/locales/` 64 个文件、8 个语言目录（原报告 9 语言不准确）：目录外无导入；入口 `main.tsx` 使用 antd 内置中文和 dayjs 中文包，`AppLayout` 明确 `menu.locale=false`；Vite 只启用 React/Tailwind，无 Umi locale 约定加载。保留实际中文依赖。
- `OrderMasterDocGroupCard.tsx`、`finance/settlement-placeholder.tsx`：无静态引用且 `config/routes.ts` 不登记；`adaptRoutes` 虽用 glob 收录所有页面 TSX，但只依据固定 routes 配置取得 loader，因此两个文件没有运行入口。保留真实 `ShippingDocumentDrawer` 与财务页面。
- `loading.tsx`：没有导入，入口 HTML 使用 `public/scripts/loading.js`；该文件不是 Vite 约定入口，保留实际加载脚本。
- `service-worker.js`、`manifest.json`：位于 src 非 public，HTML 没有 manifest link，源码无注册入口，Vite 没有 Workbox 插件。仅 doctor 配置豁免残留，同步去除豁免。
- `components/ui/status-tag/` 两文件：三个常量仅定义和 barrel 转发，没有消费；移除对应 UI barrel 行。
- `RightContent/style.ts`：默认导出的样式 hook 无导入，现有 AvatarDropdown 等入口不依赖。
- `order-plan-constants.ts`：MasterDocGroup、两个主单选项、两个主单格式化函数只有孤儿组件和自身映射使用；删除五个导出与连带两个私有映射，保留 HouseDocItem、SelectOption、分单选项和格式化函数。
- `utils/format.ts`：formatNumber 仅被 formatYuan 调用，formatYuan 无消费；删除二者及 numberFormatter，保留日期金额小数工具及现有测试。
- `global.less`：colorWeak 无 class 赋值，defaultSettings 中 false 配置不会输出类，未挂 SettingDrawer；crumb-current 在页面壳组件中没有输出；仅删两段选择器。

## 保留与限制

- 不处理 Sentry、忽略的 .umi-production、重复实现、其他过度导出。
- `web/CLAUDE.md` 的 Umi/i18n 文档已落后于现行 Vite 实现，报告主会话，本次不扩大重写。
- 依赖由主会话负责；生产构建与全量门禁由主会话统一执行。

## 验证结果

- `pnpm --dir web tsc` 通过。
- `pnpm --dir web exec vitest run src/utils/format.test.ts src/router/adaptRoutes.test.tsx src/pages/orders/order-plan-fields.test.tsx`：3 文件、10 用例全部通过。
- `pnpm --dir web exec biome check src/components/ui/index.ts src/pages/orders/order-plan-constants.ts src/utils/format.ts doctor.config.json`：4 文件通过，无修正。
- `git diff --check -- web` 通过。
- 代码部分删除 72 个文件，编辑 5 个文件，共移除 3039 行（不含主会话的依赖变更）。
