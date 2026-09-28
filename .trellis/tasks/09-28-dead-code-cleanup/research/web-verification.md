# 前端无用代码清理验证

## 删／保留判定

- 删除 `DocumentDetailLayout` 及公共出口：全仓引用核查仅命中组件自身与 `components/ui/index.ts` 的重导出，没有页面、路由、动态导入或测试消费者；现行页面骨架仍由 `PageHeaderShell` 等组件提供。
- 删除 `orderFeeColumns.tsx` 及 `fees.test.tsx` 中的 mock：唯一外部引用是该测试的模块 mock；真实订单费用页使用 `order-fee-panel.tsx` → `order-fee-panel-columns.tsx` → `feeBaseColumns.tsx`。
- `split.tsx` 只删除诊断出的未使用导入和由此失效的 `TextArea` 别名，保留组件、请求、JSX、拆票纯函数重导出与指纹逻辑。
- 删除 `web/tailwind.config.js`：现行 Vite 显式加载 `@tailwindcss/vite`；`web/tailwind.css` 使用 Tailwind v4 的 `@import`、`@source`，仓内 CSS 没有 `@config` 引用。移除配置前后生产构建生成的三份 CSS 文件名称和 SHA-256 完全相同，逐字节 `cmp` 也相同。Tailwind 插件和依赖保留。
- 保留实际使用的 `order-fee-panel-columns.tsx`、`feeBaseColumns.tsx`，以及全仓其他无用导入候选；本批不处理 189 处 React 导入基线。

## 验证

| 命令／检查 | 结果 |
| --- | --- |
| `pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx src/pages/orders/split.test.tsx src/pages/orders/components/fees/feeBaseColumns.test.tsx` | 3 文件、22 测试 PASS；stderr 有指向 `localhost:3000` 的 `ECONNREFUSED`，不影响本批用例结果。 |
| `pnpm --dir web exec biome check src/components/ui/index.ts src/pages/orders/fees.test.tsx src/pages/orders/split.tsx` | PASS，0 警告。 |
| `pnpm --dir web tsc` | PASS。 |
| `pnpm run check:architecture` | 18 项扫描测试 PASS，413 个产品文件无边界违规。 |
| `pnpm --dir web build` | 配置删除前与删除后均 PASS；最终代码的构建结果见 `/tmp/roncin-dead-code-web-build-final.log`。 |
| `git diff --check` | PASS。 |

未使用声明诊断仍按原命令 `pnpm --dir web exec tsc --noEmit --noUnusedLocals --noUnusedParameters` 执行，预期因仓库既有声明退出 1：错误从 212 降为 189，未使用 `React` 从 189 降为 187，`split.tsx` 从 22 条降为 0。诊断输出在 `/tmp/roncin-dead-code-unused-after-web.log`；正式 `tsc` 通过。

构建基线 CSS 副本在 `/tmp/roncin-dead-code-web-baseline/`；删除配置后的三个 CSS SHA-256 分别为 `3de844f66ac22142d7f256035dd896ac6d65005f77d9ec51b059848ba7f46875`、`a33397dfc4f5988eb4a3329e8fc00a3250872d9f7873b8d2b9f0dd48e7925dc6`、`359c96f207faa791da921305c4b257e6d99b52e53bc377f0f43bef2dc1aab911`，与基线相同。
