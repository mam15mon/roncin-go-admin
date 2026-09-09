# 组件规范（纯白高密度企业级视觉）

全站统一纯白高密度视觉，公共模板由 `@/components/ui` 导出，新页面优先复用：

## 页面类型 → 模板

| 页面类型 | 模板 | 要点 |
|----------|------|------|
| 整页分节表单 / 详情页 | `PageHeaderShell` + `SectionCard` + `StickyFooterBar` | 吸顶导航（返回列表、层级路径、编码 Tag、主操作组）；区块卡片带 `3px × 15px` 品牌蓝竖标（`#1677ff`）、白底细边框（`#f0f0f0`），支持 `collapsible`；底部吸底操作栏（左摘要、右取消+提交） |
| 订单多品类表单 | `OrderFormTemplate` | 配置式分节结构，不另造表单骨架 |
| 主数据 / 通用 CRUD | `MasterDataTemplate` | 顶部指标统计卡、关键字+下拉筛选、标准分页表格、快捷模态表单 |
| 表格列表页 | ProTable 高密度样式 | 搜索卡片与表格卡片细边框微圆角，操作列靠右 |

## 侧边栏

- 折叠收起宽度基准 48px，菜单项固定 36px 居中圆角卡片；折叠时彻底隐藏文本与
  展开箭头，图标正中居中。
- 含子级的菜单项折叠态 Hover 弹出纯白圆角子菜单浮层（`.ant-menu-submenu-popup`）。

## 权限相关组件

- 按钮级权限复用路由级 `access` 的同一判断结果（如
  `access.financeCommissionExport`），不在页面里硬编码第二套规则。

## 多页签导航（TagsView 页面复用）

- 菜单入口为稳定页签身份（`key`）；菜单内部的新建、详情、编辑、费用录入、拆票等子页面共用该菜单的稳定页签，不裂变生成新页签。
- 页签记录该菜单最后访问的完整地址（`path`，包含 `pathname`、`search`、`hash`）；切换离开再切回时无缝恢复最后停留的内部页面与筛选状态。
- 页签标题随当前子页面动态呈现（列表展示列表名、新建展示新建名、详情与费用在数据加载后展示真实单号），标题变更不影响页签稳定 key。
- 新增具有内部整页子路由的菜单入口时，统一在 `src/components/layout/routeUtils.ts` 的 `TAB_KEY_RULES` 与测试矩阵中集中登记。

### 复用页面的业务数据隔离

- TagsView 菜单内部从实体 A 导航到实体 B 时，React 可能复用同一个页面组件实例；所有与实体相关的页面状态（选择项、汇总、弹窗、编辑对象、工作台上下文）必须绑定当前业务 ID。渲染和提交前均校验状态所属 ID，不能只依赖 `useEffect` 在切换后清空。
- 异步请求同时记录“请求所属业务 ID”和单调递增请求序号；仅当业务 ID 仍等于当前路由且序号仍为最新时，才允许写入状态。该规则同时适用于成功、失败、卸载、延迟回填和手工刷新路径。
- 含内部预览、选择或提交状态的工作台，在业务 ID 变化时使用稳定的业务 ID 作为 React `key` 重新挂载；提交载荷中的实体 ID/子项 ID 必须来自同一个已校验上下文。
- ProTable 等自行保存数据源的组件除拦截迟到响应外，还应在业务 ID 变化时重新挂载或立即提供空数据，禁止 B 加载期间暂时显示 A 的旧行。
- 回归测试必须在**同一个组件实例**内把路由参数从 A 改为 B，并至少覆盖：切换后立即隐藏 A、B 先返回/A 后返回、A 迟到失败、B 失败、组件卸载和已打开提交工作台。通过重新挂载页面得到的测试不能证明页面复用安全。

```tsx
// 错误：迟到的 A 响应可以覆盖当前 B。
const result = await loadEntity(id);
setEntity(result);

// 正确：响应的业务 ID 和请求序号都必须仍然属于当前页面。
const requestedId = id;
const requestSequence = ++requestSequenceRef.current;
const result = await loadEntity(requestedId);
if (
  requestedId === activeIdRef.current &&
  requestSequence === requestSequenceRef.current
) {
  setEntity({ id: requestedId, value: result });
}
```

## 页头与面包屑规范 (PageHeaderShell / OrderPageHeader)

- **面包屑契约**：`breadcrumbs` 仅传**上级路径**，所有节点必须对应真实存在的实体页面（禁止加入无实体页面的折叠菜单或纯重定向路由，如“订单管理”）；分隔符全站统一使用 `/`；支持真实链接语义（优先基于 Umi `Link` 的 `href`），支持快捷点击与右键新标签打开；当前页面名称**仅由 `title` 渲染**，不在 `breadcrumbs` 末尾重复输出且不可点击。
- **层级命名统一**：
  - 列表页：不显示路由级面包屑及顶部业务状态快捷切签，仅通过页面标题和当前菜单表达所在位置；状态筛选统一放在筛选表单中；
  - 新建页：`海运出口`，标题为 `新建订单`，主要返回按钮为 `返回列表`；
  - 详情页：`海运出口`，标题为 `[订单号]`（未加载时为 `[ID]`），主要返回按钮为 `返回列表`；
  - 费用页：`海运出口 / [订单号]`，标题为 `费用录入`，主要返回按钮为 `返回订单详情`；
  - 拆票页：`海运出口 / [订单号]`，标题为 `拆票`，主要返回按钮为 `返回订单详情`。
- **业务菜单名统一**：菜单名统一使用“海运出口”（或注册定义的 `navigationTitle`，由页面显式传入 `OrderPageHeader`），禁止在面包屑中混用“海运出口订单”或“海运出口订单列表”，页头组件不得再查任何类型字典。
- **单一边界与异常兜底**：
  - 详情与费用页面的加载中与 404/未找到档案状态必须保留公共页头与返回路径；
  - 无效业务类型（如未知 kind）明确展示 404 错误状态，禁止静默回退为海运出口；
  - 订单编号与操作按钮在各工作台间保持一致定位，避免在同一页面内提供多个重复的返回入口。

## 订单类型三类真相边界（order-kinds 注册表）

订单类型相关代码必须区分三类所有权，不得互相越权：

| 真相 | 唯一所有者 | 边界 |
|------|-----------|------|
| 订单类型元数据与表单/详情适配入口 | `pages/orders/order-kinds/` 注册表（`ORDER_KIND_REGISTRY` + `getOrderKindDefinition`） | kind、业务枚举、贸易方向、运输方式、标题、Sections、默认值、创建/更新转换、详情扩展组件 |
| 订单操作能力（谁能做什么） | 后端权限 Manifest + `access.canOrder` + `order.allowedActions` | 前端不得在注册表或组件内维护第二套 capability 集合；类型扩展组件的存在只表达 UI 实现位置 |
| 表单生命周期（草稿、dirty、恢复、resetTo） | `OrderFormTemplate` | 页面与类型扩展不得读写草稿键或 dirty；外部重置只经 `actionsRef.resetTo` |

注册契约：

- 注册项以 `satisfies Record<OrderKind, OrderKindDefinition>` 穷尽约束；只注册已真实交付的类型（当前仅 `sea-export`），不提前注册 SI/AE/AI/LAND/RAIL 占位。
- `getOrderKindDefinition(kindOrPath?)` 接受直接 kind 或 `/orders/<kind>` 路径；空值、未知、未注册类型一律返回 `undefined`，页面据此展示 404，禁止默认 Sea/Air 兜底，未知类型不得进入数据 Hook 或发起请求。API 只返回业务枚举时使用 `getOrderKindDefinitionByBusinessType` 反查已注册定义（如财务费用详情的订单链接），不得在各页维护数字枚举到路由的第二套映射。
- 页面（列表/新建/详情/费用）与列表查询、资源 Hook 只消费注册定义；`common.ts` 只保留跨类型共享的选项、搜索与主数据工具，不再维护类型字典。
- 运输方式（`OrderTransportMode`）相关的地点主数据、站点装载与候选项选择必须穷尽分发（如 `searchOrderLocations`、`fetchOrderMasterData`、`resolveOrderLocationOptions`）；land/rail 显式抛「尚未开放」，不得静默落入机场/空运分支。
- 通用 layout（`routeUtils`）不得维护订单 kind 字典：订单动态路由页签用中性占位标题，真实标题由订单页加载后经 `roncin:update-tab-title` 回填。
- 公共 `OrderListTemplate` 不反向依赖页面注册表：删除其本地 kind 联合与 kindMap，业务类型列只消费页面查询已映射好的 `businessType` 文案。

有状态类型详情扩展（`OrderKindDefinition.DetailFeatures`）：

- 扩展是正常 React 组件（如 `SeaExportDetailFeatures`），合法持有 Hook、请求序号与本地状态；禁止从注册字典动态调用 Hook，也禁止把全部 setter 塞进巨型 context。
- 页面以 `key={orderFormIdentity}` 重挂载扩展；扩展内请求必须同时校验请求序号、目标身份与实例存活，旧实例的同步续体不得对旧订单再发起请求。
- 扩展经 render-prop 贡献 `headerActions`（插在「异常情况」与「更多操作」之间）、`moreMenuItems`（置于通用菜单项之前）、`appendSections`（置于审计时间线之前）、`overlays` 与 `refreshTypeState`。
- `OrderDetailFeaturesContext` 只提供业务输入与命令（订单身份、写入口校验、`refreshOrderAndLock`、绑定业务类型的 `canOrder`、搜索函数与候选项）；不暴露草稿键、dirty setter、显式刷新令牌或模板 actions ref。
- 扩展的普通刷新走 `refreshOrderAndLock`（不清草稿、不 `resetTo`）；显式刷新令牌完全由通用详情页持有（见 state-management.md）。
- 通用详情组件（`detail.tsx`、`OrderDetailHeader`）不得直接 import Sea 覆盖层或品类服务；类型专属按钮经 `businessActions` 插槽注入，通用 Header 不认识具体业务动作。
