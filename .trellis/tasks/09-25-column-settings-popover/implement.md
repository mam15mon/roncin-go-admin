# 执行计划：列设置浮层化改造

> 按提交切分推进；每个提交点都是可验证的稳定态。提交前跑该步标注的验证命令，
> 并 `git diff --check`。全程禁止手改生成物；本任务不涉及服务端与契约生成。

## 步骤清单

### Step 1：column-settings 组件浮层化（核心改造）

- [ ] 新建 `ColumnSettingsPanel.tsx`：从 `ColumnSettingsModal` 平移列表 UI
      （搜索、全部/已显示/已隐藏筛选、三分区、拖拽、上下移、必显/至少一列、
      搜索禁排序提示、恢复默认入口），去掉 Modal 外壳与页脚；
      由内部草稿 state 改为受控 `value` + `onChange(next)`。
- [ ] 改写 `ColumnSettingsEntry.tsx`：受控 Popover 锚定齿轮按钮；
      底部「更多设置」按钮（`onOpenAdvanced` 注入时出现）；
      保留 disabled / disabledReason。
- [ ] 删除 `ColumnSettingsModal.tsx`。
- [ ] `useColumnSettings.tsx`：
      - `handleValueChange` = setValue + 同步持久化（失败提示一次并置标志，
        成功或关闭浮层复位）；
      - 返回值收敛为 `{ columns, entry }`；删除 `persist` 选项；
      - `advanced?: ReactNode` 注入 + 内置二级 Modal（`advancedTitle`，默认「高级设置」）。
- [ ] 更新 `index.ts` 导出面（Panel 如需单测则导出；移除 Modal 导出）。
- [ ] 重写 `ColumnSettingsPanel.test.tsx`（替代 Modal 测试）；
      重写 `useColumnSettings.test.tsx`（即时生效 + 自动持久化 + 失败提示一次 +
      advanced 二级弹窗）。

验证：`pnpm --dir web exec vitest run src/components/ui/column-settings` +
`pnpm --dir web biome:lint`（改动文件）+ `pnpm --dir web tsc`
（此时调用点未同步，tsc 预期报 modal 残留——以报错清单作为 Step 2 的作业清单，
Step 1 本身以定向测试与 biome 通过为门槛）。

**提交 1**：`refactor(web): 列设置改为齿轮锚定浮层并即时生效`（含组件、hook、新测试；
调用点残留属已知状态，提交说明中注明由下一步收口）。

### Step 2：全站调用点一次性同步

- [ ] 以 Step 1 的 tsc 报错清单为作业清单，逐点删除 `modal` 解构与渲染行；
      模板组件（FinanceLedgerTemplate、MasterDataTemplate、OrderListTemplate、
      SettingTableTemplate、SubEntityDrawerTemplate）同步内部渲染行。
- [ ] `FinanceLedgerTemplate` 增 `advancedSettings?: ReactNode` 透传 hook `advanced`。
- [ ] `grep -rn "settings\.modal\|columnSettings\.modal\|\.modal}" web/src` 无残留。

验证：`pnpm --dir web tsc` 全绿 + `pnpm --dir web test:changed` +
`pnpm run check:architecture`。

**提交 2**：`refactor(web): 列设置调用方契约收敛为 entry 单入口`。

### Step 3：费用台账收敛

- [ ] `fees/index.tsx` 改标准接入（`columnSettingsKey` + `advancedSettings`），
      删除 `onOpenColumnConfig` 绕行。
- [ ] `FinanceLedgerTemplate` 删除 `onOpenColumnConfig` 自定义入口分支。
- [ ] `TableColumnConfigModal.tsx` 与其测试删除；
      `feeLedgerViewPreference.ts` 收窄为仅行配色（key 升级
      `roncin:fee-ledger-row-colors:v1:*`），`FinanceLedgerViewConfig` 类型收窄，
      行配色改即时生效写入。
- [ ] `FinanceLedgerTemplate.test.tsx` 增补 advancedSettings 用例；
      费用台账相关测试随收窄改写。

验证：`pnpm --dir web exec vitest run src/components/ui/finance-ledger-template src/pages/finance/fees` +
`pnpm --dir web tsc`。

**提交 3**：`refactor(web): 费用台账列设置并入统一浮层，行配色独立持久化`。

### Step 4：收尾验收

- [ ] `grep` 终检：`ColumnSettingsModal`、`persist: false`（column-settings 范围）、
      `fee-ledger-view:v1`、`onOpenColumnConfig`、`TableColumnConfigModal` 无残留。
- [ ] 手动冒烟（dev server）：台账页浮层勾选/拖拽/恢复默认即时生效；
      刷新偏好保留；「更多设置」行配色即时生效；「至少一列」与必显规则正常。
- [ ] `pnpm run check:fast` 全量门禁通过。
- [ ] 视觉走查：浮层在 1366/1920 宽度下不遮关键操作区、滚动正常。

**提交 4（如冒烟有修补）**：`fix(web): 列设置浮层冒烟问题修补`。

## 回滚点

- 任一步骤失败且回退：`git revert` 对应提交；Step 2/3 依赖 Step 1 的组件形态，
  不跨提交部分回滚。
- 存储变更仅浏览器本地，无服务端数据回滚需求。

## 审查门

- Step 1 完成后：组件 API 形态确认（对照 design.md 的 API 表）。
- Step 3 完成后：费用台账接线与偏好存储收窄确认。
- Step 4：check:fast 通过后才进入 Phase 3（spec 更新 + 收尾）。

## 执行记录（2026-09-25）

- Steps 1–3 已全部完成，**合并为单提交**
  `refactor(web): 列设置改为齿轮锚定浮层并即时生效`：各步在同一工作批次内
  交错落地，中间切分会产生不可编译提交，故放弃四段切分，整体一组验证。
- 关键实现修正：Entry 外层结构在禁用切换时保持稳定（Popover > span >
  Tooltip > Button），避免禁用态切换时触发按钮 DOM 重建导致引用失效。
- 测试适配：`OrderFeeTableTabs.feeColumns/billTracking` 改浮层断言
  （`.ant-popover` 内勾选、即时生效、无保存按钮）；
  `VerificationWorkbench`/`UserFormModal` 的 antd 部分 mock 补 `Popover: () => null`。
- 验证结果：定向测试（column-settings 18 例 + 台账/费用相关）全绿；
  `pnpm test:changed` 102 文件 658 例全绿；`pnpm run check:fast` 全栈门禁通过
  （前端 183 文件 1073 例 + 后端）；tsc、biome、`git diff --check` 干净；
  残留 grep 终检为零（仅 `.umi-production` 旧构建缓存命中，未入 git）。
- 偏差：`FinanceLedgerTemplate.test.tsx` 未新增 advancedSettings 专项用例
  （透传为类型化 prop，「更多设置」二级弹窗行为已由 column-settings 用例覆盖）；
  浏览器人工冒烟未执行（本会话无浏览器后端），由用户在 dev 页面上确认。
- spec 同步：`capability-navigation.md` 列设置条目更新为新契约与交互形态约束。
