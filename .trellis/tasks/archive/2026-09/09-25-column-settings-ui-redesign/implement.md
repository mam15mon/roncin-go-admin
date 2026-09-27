# 列设置浮层现代化重构执行计划

## 执行检查清单

### Step 1: 视觉与结构重构 `ColumnSettingsPanel.tsx`
- [ ] 顶部控制栏升级：
  - 增加全选/半选/反选 Checkbox 与展示计数 `(已显数/总数)`；
  - 右侧置入纯文本链接型或轻量按钮「恢复默认」，消除底部冗余；
  - 移除原 `Radio.Group`（`全部/已显示/已隐藏`）。
- [ ] 搜索栏升级：
  - 将搜索输入框调整为全宽（`width: 100%`），尺寸统一为 `size="small"`，增加 `allowClear`；
  - 输入框与列表区增加合适的外边距。
- [ ] 列表条目项升级：
  - 左侧添加 `HolderOutlined` 拖拽把手图标（带 `grab` 光标）；
  - 移除每一行原本的 `borderBottom: 1px solid #f0f0f0`，改为行微圆角；
  - 增加 hover 悬浮背景变色（`#fafafa`）与右侧操作按钮淡入淡出（常态 `opacity: 0`，hover `opacity: 1`）；
  - 将「必显」标签由深蓝实体 Tag 调整为浅灰轻量微标；
  - 保持原有拖拽与上下移核心排序算法和区域边界隔离不变。
- [ ] 底栏收尾：
  - 移除原底部的「恢复默认」按钮；
  - 底部左侧保留简洁计数，右侧保留「更多设置」二级入口。

### Step 2: 单测与边界校验 `ColumnSettingsPanel.test.tsx`
- [ ] 更新与补充测试用例：
  - 验证全选/反选 Checkbox 的 `checked`、`indeterminate` 状态与点击回调；
  - 验证必显列在全选/反选时的防误触；
  - 验证搜索过滤、拖拽、上下移边界及「恢复默认」正常运转；
  - 确保原有各项用例平滑通过。

### Step 3: 全局门禁与视觉回归验证
- [ ] 运行定向单测：`pnpm --dir web exec vitest run src/components/ui/column-settings/ColumnSettingsPanel.test.tsx`；
- [ ] 运行快速门禁：`pnpm run check:fast`；
- [ ] 检查代码格式与类型安全。

## 执行记录（2026-09-25）

- 三步全部完成，单提交交付。样式经新增 `ColumnSettingsPanel.less` 落地
  （沿用 SearchFilterTemplate 的组件级 less 先例），行悬浮背景/操作淡入/
  把手变色/投放目标高亮均在 less 中实现。
- 全选语义：仅作用于非必显列；全选状态下点击清空时，无必显列则保留排序
  首列兜底；全部为必显列时全选入口禁用。搜索状态下全选按全量列生效。
- 上下移按钮保持常驻 DOM（仅透明度切换），aria-label 不变，既有用例零改动通过。
- 移除「全部/已显示/已隐藏」筛选 Radio（PRD 要求），底栏恢复默认上提顶栏。
- 验证：column-settings 定向 21 例全绿；`test:changed` 661 例通过；
  tsc、biome 干净；`check:fast` 全栈门禁通过。
- 人工视觉走查待用户在 dev 页面确认（本会话无浏览器后端）。
