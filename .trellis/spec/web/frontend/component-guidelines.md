# 组件规范（纯白高密度企业级视觉）

全站统一纯白高密度视觉，公共模板由 `@/components/ui` 导出，新页面优先复用；
动手前先查[复用能力导航](./capability-navigation.md)，已有领域能力（建账工作台、
审计展示、状态映射等）直接消费 `features/` 公开入口，不复制实现。

## 页面类型 → 模板

| 页面类型 | 模板 | 要点 |
|----------|------|------|
| 整页分节表单 / 详情页 | `PageHeaderShell` + `SectionCard` + `StickyFooterBar` | 吸顶导航（返回列表、层级路径、编码 Tag、主操作组）；区块卡片带 `3px × 15px` 品牌蓝竖标（`#1677ff`）、白底细边框（`#f0f0f0`），支持 `collapsible`；底部吸底操作栏（左摘要、右取消+提交） |
| 订单多品类表单 | `OrderFormTemplate` | 配置式分节结构，不另造表单骨架 |
| 主数据 / 通用 CRUD | `MasterDataTemplate` | 顶部指标统计卡、关键字+下拉筛选、标准分页表格、快捷模态表单 |
| 表格列表页 | ProTable 高密度样式 | 搜索卡片与表格卡片细边框微圆角，操作列靠右 |

## 页面容器与宽度规范 (Container & Width Standards)

- 全站页面统一以 `<PageContainer>` 作为顶级骨架，全局布局遵循 `web/config/defaultSettings.ts` 中的 `contentWidth: 'Fluid'` 流式全屏自适应，底色统一使用 `#f5f7fa`。
- **严禁在任何页面、工作台或模板中私自硬编码 `maxWidth: 1440` 或自定义水平居中外层容器**。
- 吸顶页头（`PageHeaderShell`）、业务分节卡片（`SectionCard`）、数据表格与吸底操作栏（`StickyFooterBar`）在任何屏幕分辨率下必须 100% 满屏平铺与贴边对齐，仅保留全局统一的 12px 内容区内边距。

## Ant Design Pro / ProComponents 状态与请求核心范式

- **表格数据流一律走官方 `request` 协议**：严禁在外层自建 `data/loading/query` 状态去架空 ProTable；服务端分页列表与搜索一律优先通过 ProTable 的 `request={(params) => Promise<{ data, success, total }>}` 消费接口，由组件内建引擎自动调度分页与防竞态。
- **刷新与重置一律走官方 `actionRef`**：新增、编辑、删除或启停操作成功后，统一通过 `actionRef.current?.reload()` 触发列表刷新，严禁层层透传手写的 `reload/fetchList` 触发式回调。
- **模态表单生命周期一律走 `ModalForm.onFinish`**：异步提交必须返回 Promise，由 ProComponents 自动接管提交中 loading 态与成功关闭，严禁在外部手工维护 `confirmLoading` 镜像状态。
- **自定义 Hook 依赖防护**：在封装涉及异步请求的 Hook 时，纯动作型回调（如 api 函数、数据转换 map 函数）必须使用 `useRef` 保障引用稳定性，严禁将未 memoize 的内联函数作为 `useCallback` 依赖引发渲染死循环（`Maximum update depth exceeded`）。

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
- 页签标题随当前子页面动态呈现：列表与新建直接使用路由标题；订单详情、费用录入与拆票在
  `routeUtils` 中统一使用中性占位标题（`订单详情` / `订单费用录入` / `订单拆票`），由页面在
  数据加载成功后经 `roncin:update-tab-title` 事件回填带单号的真实标题；标题变更不影响页签稳定 key。
- 通用 layout 不维护订单类型字典：未注册订单类型（如 `/orders/sea-import`）的页签标题回落
  中性「订单管理」，不得显示未开放业务名。
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

- **面包屑契约**：`breadcrumbs` 仅传**上级路径**，所有节点必须对应真实存在的实体页面（禁止加入无实体页面的折叠菜单或纯重定向路由，如“订单管理”）；分隔符全站统一使用 `/`；支持真实链接语义（优先基于 react-router `Link` 的 `href`），支持快捷点击与右键新标签打开；当前页面名称**仅由 `title` 渲染**，不在 `breadcrumbs` 末尾重复输出且不可点击。
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

## 往来单位选择器与快捷建档（散客契约）

后端契约与完整口径见 `../../server/backend/partner-casual-contract.md`，此处只约束前端交互：

- **快捷建档角色由上下文决定，单选**：订单侧（`PartnerQuickAddSelect`）按触发字段静默写入单一角色；费用侧（`QuickAddPartnerModal`）客商类型为单选，经 `defaultRole` prop 由调用方按当前费用方向预选（应收→客户、应付→供应商、方向未选默认客户）。多角色勾选属主档维护，快捷弹窗不得提供。
- **快捷新增弹窗默认勾选「单次合作」**，可取消；提交透传 `isCasual`。
- **选择器元数据经契约字段渲染，不污染 label**：候选项携带 `isCasual` 时经 `optionRender` 动态渲染 `<Tag>散客</Tag>`；`label` 只拼 `legalName (code)`，禁止拼入散客等业务标注文本（防止单证 / 合同字符串污染，选中值不得出现多余前缀）。
- **散客交互的软硬边界**：向散客供应商出款时对方账户动态标星必填（服务端刚性）；散客应收账单改大账期只出黄色预警、不阻断提交（刻意弹性，禁止加前端硬拦截）。

## antd6 API 契约（禁用弃用用法，2026-09 清理后口径）

源码侧弃用用法已全部清零（`cd web && antd lint ./src --only deprecated` 保持
0 issues），新代码禁止再次引入以下旧写法：

- Alert `message` → `title`；Space `direction` → `orientation`；Spin `tip` → `description`。
- Drawer `width` → `size`（支持 `number | string`）；Drawer/Modal `destroyOnClose` →
  `destroyOnHidden`（ProForm `modalProps` 透传同样适用）；Modal `maskClosable` →
  `mask={{ closable }}`。
- Select 顶层 `filterOption` / `onSearch` / `optionFilterProp` 必须并入
  `showSearch={{ ... }}` 对象（传对象即开启搜索，布尔 `showSearch` 一并吸收）；
  `onDropdownVisibleChange` → `onOpenChange`；`filterOption={false}` 表服务端
  过滤，语义保持。
- Timeline items 元素 `children` → `content`。
- Input/InputNumber `addonAfter` → `Space.Compact`（+ `Space.Addon`）；ProForm
  字段改用字段级 `addonAfter`（pro 自渲染，不透传 InputNumber）。
- `List` 已整体弃用（下个大版本移除）：一律用 `Listy`（`items` + `rowKey` +
  `itemRender`，行内容 JSX 原样迁移；`rowKey` 必须显式给出，缺省会触发
  React key 警告）。
- antd `message`/`modal` 静态导入改 `App.useApp()`，避免「can not consume
  context」警告。
- 依赖 antd 内部实现细节（如 `.ant-form-item-has-error` 类名）的工具逻辑，
  必须收拢选择器常量并配「升级哨兵测试」：用真实 antd 渲染断言该细节仍存在
  （范例：`formErrorAntdContract.test.tsx`），antd 升级移除时测试显式失败，
  不允许静默失效。
