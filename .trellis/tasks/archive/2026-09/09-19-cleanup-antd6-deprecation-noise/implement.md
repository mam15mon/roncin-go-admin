# 执行计划：antd6 弃用 API 迁移与 act 噪音治理

> 基线：main@6ed1eeba 干净，check:fast 全绿（WEB 51.7s / SERVER 79.4s）。
> 迁移映射与修复原则见 design.md；精确位置清单见 `/tmp/antd-deprecated.json`
>（`antd lint ./src --only deprecated --format json` 可随时重扫）。

## 阶段一：弃用 API 迁移（feat 无、纯 refactor，commit 前缀 `refactor(web)`）

- [x] 1. `web/vitest.config.ts`：`cache.dir` → 顶层 `cacheDir`（保持 `/dev/shm` 内存盘路径）。
- [x] 2. 批次 A（机械改名，约 24 文件）：Alert `message`→`title`（31）、Space
      `direction`→`orientation`（15）、Drawer `width`→`size`（12）、Drawer/Modal
      `destroyOnClose`→`destroyOnHidden`（8）、Modal `maskClosable`→`mask.closable`（4 文件）、
      Select `onDropdownVisibleChange`→`onOpenChange`（2）、Spin `tip`→`description`（1）、
      Timeline `items.children`→`items.content`（1）。
- [x] 3. 批次 B（Select 搜索收敛，约 12 文件）：顶层 `filterOption`(13)/`onSearch`(12)/
      `optionFilterProp`(4) 并入 `showSearch={{ ... }}`，逐字段等价搬移。
- [x] 4. 批次 C（List→Listy，4 文件 9 处）：`FinanceSummaryCard`、`WorkbenchSideCards`、
      `UnlockRequestHistoryDrawer`、`SeaOrderChangeHistoryDrawer`；行 JSX 原样复用，
      同步修正对应测试断言。
- [x] 5. 批次 D（addonAfter→Space.Compact，2 处）：`BasicInfoSection:191`、
      `InvitationQrModal:164`。另有清单外收尾：11 处 `modalProps.destroyOnClose`、
      2 处 ProForm `fieldProps.addonAfter`（改字段级 `addonAfter`）。
- [x] 6. 阶段验证（review gate A）：
      - `cd web && antd lint ./src --only deprecated --format json` → 0 issues
      - `pnpm --dir web exec vitest run <受影响测试>` 逐批通过
      - `pnpm --dir web tsc && pnpm --dir web biome:lint`
      - 全量 `pnpm --dir web test` 后 grep 输出：`deprecated` 计数为 0（act 允许留待阶段二）
      - 提交：`refactor(web): 迁移 antd6 弃用 API 至新契约`（实际 b119bfe2）

## 阶段二：act 警告治理（commit 前缀 `test(web)`）

- [x] 7. 从阶段一后的全量测试输出提取 act 警告按文件分布（实测 21 文件 304 条）。
- [x] 8. 按域分批修复（财务 118→0 / 订单 80→0 / 伙伴与公共组件 109→0）：
      每文件「跑定向测试收集警告 → 补等待/act 包裹 → 复跑确认双零」。
      无 console 静音、无 act 环境开关、无断言删除、无组件行为修改。
- [x] 9. 阶段验证（review gate B）：
      - 全量 `pnpm --dir web test`：`not wrapped in act` 计数为 0，`deprecated` 计数为 0
      - 用例数不低于基线（853 passed / 12 skipped 持平）
      - 提交：`test(web): 修复测试异步等待并清零 act 警告`（实际 966c8c88）

## 追加范围：残余非弃用类警告清理（用户要求"噪音都清理"）

- [x] 10a. 静态 `message` ×2（ContainerDrawer/CargoItemDrawer 改 `App.useApp()`）。
- [x] 10b. Descriptions `span` 行宽不匹配 ×2（BillDetailDrawer 备注 span 3→2，视觉不变）。
- [x] 10c. Form `initialValues` 路径覆盖 ×2（SeaCreateDocumentModeField 移除字段级
      `initialValue`，默认 HOUSE 改挂载 effect 写入，三条分支终态不变）。
- [x] 10d. FormContext 缺失 ×1 + ProTable 缺 key ×1（均为测试替身缺陷，改真实
      Form 包裹与按列 key 渲染，断言零改动）。
      提交：`fix(web): 消除测试残余的静态 message 等五类警告`（实际 6d48b5e9）

## 终验与收尾

- [x] 10. `pnpm run check:fast` 全栈通过，WEB 段输出无弃用警告与 act 噪音。
- [x] 11. `git diff --check`；确认无秘密/调试输出；归档任务并记录 journal。
- [ ] 遗留（范围外，建议后续任务）：`useMasterDataCrud.test.tsx` 服务端分页用例存在
      存量「Maximum update depth exceeded」警告（本次改动前已有，计数 1~6 波动），
      疑似共享 Hook effect 依赖不稳定，需单独排查。

## 验证命令速查

```bash
# 精确弃用扫描
cd web && antd lint ./src --only deprecated --format json
# 定向测试（路径相对 web/）
pnpm --dir web exec vitest run src/pages/xxx/yyy.test.tsx
# 类型与 lint
pnpm --dir web tsc && pnpm --dir web biome:lint
# 终验
pnpm run check:fast
```
