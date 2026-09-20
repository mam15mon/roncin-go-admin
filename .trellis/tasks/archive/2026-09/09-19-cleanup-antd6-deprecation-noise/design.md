# 技术设计：antd6 弃用 API 迁移与 act 噪音治理

## 一、弃用 API 迁移映射（依据 `@ant-design/cli` 对 antd 6.6.0 的离线元数据）

| # | 旧用法（处数） | v6 新契约 | 说明 |
|---|----------------|-----------|------|
| 1 | Alert `message`（31） | `title` | 等价改名；`description` 不变 |
| 2 | Space `direction`（15） | `orientation` | 等价改名 |
| 3 | Drawer `width`（12） | `size` | `size` 自 6.2.0 支持 `number \| string`，`width={720}` 直接改名 `size={720}`；预设值 `default`(378)/`large`(736) |
| 4 | Drawer/Modal `destroyOnClose`（2+6） | `destroyOnHidden` | 等价改名（5.25.0+） |
| 5 | Modal `maskClosable`（4 文件，lint 未覆盖） | `mask={{ closable }}` | 运行时警告 52 次的源头 |
| 6 | Select `onDropdownVisibleChange`（2） | `onOpenChange` | 等价改名 |
| 7 | Select `filterOption`（13）/`onSearch`（12）/`optionFilterProp`（4） | 并入 `showSearch={{ ... }}` 对象 | `showSearch` 对象形态自 6.0.0；对象字段含 `filterOption`、`onSearch`、`optionFilterProp`（6.1.0 起支持 `string[]` OR 匹配）、`filterSort`、`autoClearSearchValue`、`searchValue`。传对象即开启搜索，原布尔 `showSearch` 吸收进对象；`filterOption={false}`（服务端过滤）语义保持 |
| 8 | Spin `tip`（1） | `description` | 仅 Spin 包裹子元素场景 |
| 9 | Timeline `items.children`（1，lint 未覆盖） | `items.content` | 定位：`OrderAuditTimelineSection` / `AuditLogSection` / `orders/detail.tsx` 三处嫌疑，运行时确认 |
| 10 | Input `addonAfter`（2） | `Space.Compact` 包裹 | `BasicInfoSection.tsx:191`、`InvitationQrModal.tsx:164`；结构改造：`<Space.Compact><Input/><后缀/></Space.Compact>`，保持紧凑拼接视觉 |
| 11 | `List` 组件（9 处 / 4 文件，整体弃用） | `Listy`（6.6.0+） | 映射：`dataSource`→`items`、`rowKey`→`rowKey`、`renderItem`→`itemRender(item, index)`、`List.Item` 行 JSX 原样移入 `itemRender`；`List.Item.Meta` 拆为普通 flex JSX；`header`/`footer`/`pagination`/`bordered` 若存在则用外层容器等价表达。涉及：`FinanceSummaryCard`、`WorkbenchSideCards`、`UnlockRequestHistoryDrawer`、`SeaOrderChangeHistoryDrawer` |

### 迁移通用规则

- 只做等价替换，不顺手重构业务逻辑、不改文案、不改布局参数。
- `showSearch` 合并时注意保持原过滤行为逐字段搬移，不新增过滤字段。
- 每个 touched 文件同步修正对应测试中对旧 DOM 结构的断言（若有）；断言意图不变。
- 迁移后单文件运行 `pnpm --dir web exec vitest run <对应测试>` 验证。

## 二、act 警告治理策略

### 成因分类（从 check:fast 日志抽样归纳）

1. **异步落点未被等待**：触发动作后组件仍有 Promise/防抖回调在 act 作用域外落地
   （如 `BillCreationWorkbench` 的预览竞态用例、`partner-detail` 的按需请求）。
2. **测试收尾后的迟到更新**：用例断言完成后，被触发的异步流才 resolve。
3. **antd 内部异步渲染**（CSSMotion/Tree/虚拟 List）：多由 1、2 引发，随等待修复自然消失。

### 修复原则

- 逐文件治理：先跑该文件测试收集警告定位，再补 `await waitFor(...)` / `await findByXxx(...)`
  或将触发动作移入 `act`，让测试真正等到异步流收敛。
- 每修复一个文件：`pnpm --dir web exec vitest run <file>` 确认 ①用例全过 ②stderr 无
  `not wrapped in act`。
- 禁止：全局关闭 `IS_REACT_ACT_ENVIRONMENT`、mock console、删除/放宽断言、
  无意义 `setTimeout` 帧等待（AGENTS.md 已禁）。
- 测试文件超过 10 个重型用例时遵守长尾拆分约束（本次以修复为主，不主动拆文件）。

## 三、配置项

- `web/vitest.config.ts:40` `cache: { dir: '/dev/shm/roncin-vitest-cache' }` → 顶层
  `cacheDir: '/dev/shm/roncin-vitest-cache'`，行为等价（Vitest 官方新写法）。

## 四、风险与回滚

| 风险 | 缓解 |
|------|------|
| `List → Listy` 视觉/交互回归 | 行内容 JSX 原样复用；受影响 4 个组件各有测试；工作台卡片人工抽查 |
| `addonAfter → Space.Compact` 拼接样式变化 | 仅 2 处；保持 Compact 紧凑语义；定向测试验证 |
| act 修复改动测试语义 | 只加等待不改断言；受影响用例断言保持原意图 |
| 大范围机械改名引入笔误 | 分批提交 + `pnpm --dir web tsc` + `biome:lint` + 全量 vitest 兜底 |

回滚：两阶段各自独立 commit，可单独 revert。

## 五、兼容性

- 上线前阶段，无历史契约负担；一次性迁移，不留新旧双写。
- 不触碰 `.proto` / OpenAPI 生成物 / 权限码，无生成流程介入。
