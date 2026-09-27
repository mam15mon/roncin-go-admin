# 技术设计：列设置浮层化改造

## 现状与波及面

- `web/src/components/ui/column-settings/`：`ColumnSettingsEntry`（齿轮按钮）、
  `ColumnSettingsModal`（640px 弹窗，草稿 + 保存）、`useColumnSettings`（显隐排序应用 +
  localStorage 持久化，store `roncin:column-settings:v1:<tableKey>:<userId>:<orgId>`）、
  `preference.ts`、`types.ts`。
- hook 返回 `{ columns, entry, modal }`，全站约 70 个调用点全部经此契约接入；
  5 个模板组件（FinanceLedgerTemplate、MasterDataTemplate、OrderListTemplate、
  SettingTableTemplate、SubEntityDrawerTemplate）内部渲染 `entry` 与 `modal`，
  其余页面/抽屉直接解构使用。**无任何调用方使用 `persist: false` 与 hook `advanced`**。
- 费用台账绕行链路：`fees/index.tsx` → `FinanceLedgerTemplate(onOpenColumnConfig)` →
  `TableColumnConfigModal`（双页签：列配置 + 高级设置 RowColorSettings），整份
  `FinanceLedgerViewConfig { columns, rowColors }` 经 `feeLedgerViewPreference.ts`
  （`roncin:fee-ledger-view:v1:*`）草稿式保存。

## 目标组件结构

```
column-settings/
  ColumnSettingsEntry.tsx   齿轮按钮 + 受控 Popover 容器（浮层开关状态自持）
  ColumnSettingsPanel.tsx   新增：浮层内容（搜索/筛选/分区列表/拖拽/恢复默认入口）
  useColumnSettings.tsx     即时生效 + 自动持久化；管理「更多设置」二级 Modal
  preference.ts / types.ts  不变（存储 key、隔离口径、ColumnSettingsValue 不动）
```

- `ColumnSettingsModal.tsx` 删除；列表 UI（分区渲染、搜索筛选、拖拽、上下移、
  必显/至少一列规则）整体平移进 `ColumnSettingsPanel`，仅去掉 Modal 外壳与页脚。
- Panel 从「内部草稿 state」改为**受控**：单一真相源是 hook 的 `value`；
  每次勾选/拖拽/上下移直接计算 next value 并回调 `onChange(next)`，
  关闭浮层不丢状态（本就无草稿）。

## 状态流与持久化

```
Panel 交互 ──onChange(next)──▶ hook.handleValueChange
                                 ├─ setValue(next)            // 表格立即重渲染（即时生效）
                                 ├─ saveColumnSettingsPreference(...)  // 自动持久化
                                 └─ 写入失败：message.error 提示，页面内生效保留
```

- 写入失败提示防刷屏：记录「失败待提示」标志，同一浮层会话内只提示一次，
  下次写入成功或关闭浮层后复位。
- 「恢复默认」走同一条 `onChange(defaultValue)` 链路，即时生效并持久化。
- 浮层 open 状态收在 Entry 内部；`disabled` 时入口禁用，浮层状态无副作用。

## API 变更（一次性同步，不留兼容层）

`useColumnSettings(options)`：

| 项 | 变更 |
|---|---|
| 返回值 | `{ columns, entry, modal }` → `{ columns, entry }` |
| `persist?: false` | 删除（无使用者；持久化恒开） |
| `advanced?: ReactNode` | 恢复启用：传入时浮层底部出现「更多设置」按钮，点击打开 hook 内置二级 Modal（`advancedTitle` 可配，默认「高级设置」），内容即 `advanced` |
| `onOpenAdvanced` | 不引入（由 `advanced` 注入即可覆盖费用台账场景） |
| `title` / `disabled` / `disabledReason` / `structuralKeys` / `lockVisibleKeys` / `defaultHiddenKeys` / `titleOverrides` / `scope` | 不变 |

二级 Modal 放在 hook 内部管理（open state + `advanced` 内容注入），
通用层不依赖任何领域组件，依赖方向符合 `components/ui` 边界。

### 调用点更新（约 70 文件，机械改动）

- 模板组件内部：删除 `settings.modal` / `columnSettings.modal` 渲染行；
  `FinanceLedgerTemplate` 增 `advancedSettings?: ReactNode` 透传给 hook。
- 页面/抽屉调用点：删除解构中的 `modal` 与对应 JSX 渲染行；
  `useColumnSettings` 的 `advanced` 参数名沿用（原 hook 参数存在但无使用者，语义升级为注入）。
- 验证兜底：`grep -rn "settings.modal\|columnSettings.modal\|\.modal}"` 无残留 + tsc 全绿。

## 费用台账收敛

- `fees/index.tsx` 改为标准接入：`FinanceLedgerTemplate(columnSettingsKey='finance-fees:ledger',
  advancedSettings=<RowColorSettings .../>)`，删除 `onOpenColumnConfig` 绕行。
- 列偏好由统一 hook 持久化到 `roncin:column-settings:v1:finance-fees:ledger:*`；
  `feeLedgerViewPreference.ts` 收窄为仅存行配色（存储 key 升级为
  `roncin:fee-ledger-row-colors:v1:*`，结构校验同步收窄），旧 v1 数据不迁移
  （上线前无生产契约，开发期偏好按默认重建，PRD 已声明）。
- `TableColumnConfigModal.tsx` 及其测试删除；`FinanceLedgerViewConfig` 类型收窄为
  `{ rowColors }`（`columns` 字段随类型一并移除，调用方一次性同步）。
- `RowColorSettings` 交互同步改即时：每次改色/重置直接回调页面 state + 即时写偏好，
  去掉「打开弹窗同步草稿 + 保存提交」的草稿逻辑（页面级小弹窗，遮挡影响可接受）。

## 浮层交互细节与风险

- `Popover trigger="click"` 受控 open（`onOpenChange` 仅响应外部关闭）；
  浮层宽约 320px，列表区 `maxHeight` 收敛为浮层内自适应（约 420px 上限，可滚动）。
- 拖拽沿用现有原生 HTML5 平移：mousedown 发生在浮层内不会触发外部关闭；
  drop 在浮层外不产生 click，不误关。风险点在实现后以定向测试 + 手动验证覆盖。
- 保留「搜索中禁用排序」防错与提示文案；上下移按钮保留（可访问性兜底路径）。

## 测试策略

- `ColumnSettingsModal.test.tsx` → 重写为 `ColumnSettingsPanel.test.tsx`
  （勾选即时回调、分区/搜索/筛选、必显与至少一列、恢复默认）。
- `useColumnSettings.test.tsx` 重写断言：onChange 即时 setState + 每次写入 localStorage、
  写入失败提示一次、恢复默认持久化、advanced 注入打开二级 Modal、
  调用点不再有 modal 返回值（编译期保证）。
- `TableColumnConfigModal.test.tsx` 删除；费用台账以
  `FinanceLedgerTemplate.test.tsx` 增补「advancedSettings 注入 → 更多设置入口」用例，
  行配色持久化逻辑如已有测试则随收窄同步改写。
- 长测试文件不超过 10 个重型用例（AGENTS 长尾约束），拆分按常规/重交互维度。

## 兼容与回滚

- 单任务内一次性完成组件、hook、全部调用点、测试与存储收窄，不设双轨；
  回滚 = revert 对应提交序列（见 implement.md 的提交切分）。
- 存储变更仅涉及浏览器 localStorage，无服务端契约与迁移。
