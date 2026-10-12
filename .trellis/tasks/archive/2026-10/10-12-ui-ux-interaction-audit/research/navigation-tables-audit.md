# Research: 导航与列表交互审查

- Query: 使用 UI-UX-Pro-Max 审查公共导航、页签、组织工作区、列表筛选与返回恢复、列设置和列宽调整，以及窄屏入口、反馈与键盘交互。
- Scope: internal；按本地指定技能的方法检查现有 Web 后台，不做产品改动。
- Date: 2026-10-12
- 技能：`/home/vince/.claude/plugins/marketplaces/nextlevelbuilder-ui-ux-pro-max-skill/.claude/skills/ui-ux-pro-max/SKILL.md`；已读任务 PRD、design、research/skill-method.md、workflow、领域术语和前端规范。
- 证据状态：本代理研究为源代码审查；下列“浏览器检查”交由主会话复核。NAV-01 已补主会话回传的 390px 抽屉观察，其余未经浏览器观察的项不能写作“已实测”。

## Findings

### NAV-01 / P2：窄屏缺失企业资源与设置的完整导航入口

- 代码证据：`web/src/global.less:1393` 在 ≤768px 隐藏 `.roncin-header-menus`，注释称抽屉已有完整菜单；`web/config/routes.ts:38`、`:45`、`:74`、`:102`、`:251`、`:259` 等把企业资源配置、客户/供应商/国外代理、主数据、系统管理设置为 `hideInMenu: true`；`web/src/router/adaptRoutes.tsx:128` 构建侧栏菜单时直接剔除所有 `hideInMenu` 项；`web/src/app/AppLayout.tsx:42` 将该菜单树交给 ProLayout。
- 浏览器回传证据：主会话在 390px 已确认 `.roncin-header-menus` 为 `display:none`，Drawer 只有工作台、订单管理、费用管理。此观察不能扩大成所有入口均不可达：公司工作台 `web/src/pages/workbench/WorkbenchSideCards.tsx:240` / `:243` 分别提供“客户管理”和“主数据”按钮，`workbench/index.tsx:142` 无条件渲染该快捷区；系统工作台 `workbench/index.tsx:176` / `:179` / `:182` 还提供公共基础资料、公司管理、人员分配链接。
- 精确缺失范围：供应商列表 `/partners/suppliers`、国外代理列表 `/partners/foreign-agents` 和企业资源配置 `/enterprise-resources/config` 的正常主导航入口只在 `HeaderMenus.tsx:124` / `:129` / `:137` 出现；工作台快捷区没有它们。系统管理需要区分组织场景：系统工作台仍能从公司管理/人员分配进入 `/admin` 再通过内部页签访问已授权功能，不能称系统工作台全无管理入口；公司工作台没有 `/admin` 快捷项，设置中心隐藏后，其常规入口缺失。
- 触发步骤：有对应权限的用户在 390px 公司工作区进入 `/welcome`，展开抽屉，尝试找到供应商、国外代理、企业资源配置及公司组织的设置中心；比较桌面顶部“企业资源”“设置中心”。客户/主数据另从工作台快捷区验证可达。
- 用户影响：供应商/国外代理/企业资源配置及公司工作区设置的导航可发现性和常规直达路径在窄屏缺失。客户/主数据仍可绕回工作台进入，但缺少桌面端统一导航入口；不应描述为完全不可访问。首次进入或组织切换后没有历史页签可兜底。直接 URL 仍可打开，不属于权限缺陷。
- 限定说明：海运订单表单 `SeaBasicInfoSection.tsx:486` / `:496` 有供应商/国外代理的快捷新建路径；已有详情 `partner-detail.tsx:120` / `:127` 的返回按钮可回相应列表。这些都依赖先进入订单表单或该类详情，不能替代首次寻找列表的主导航，也说明不能声称全站“任何入口都不存在”。
- 最薄建议：保留可点击的窄屏快捷菜单入口，或把同一权限过滤后的企业资源/设置菜单加入移动抽屉，复用既有菜单数据；无需重建手机导航系统。
- 浏览器检查：375、767、768、769px；`.roncin-header-menus` 计算样式；侧栏 Drawer `.ant-drawer [role="menuitem"]` 和实际显示文字。`/welcome`、新工作区、组织切换后分别验证。≤768px 的样式与 ProLayout <768px 断点还有 768px 边界差异。
- 状态：代码证据 + 主会话 390px Drawer 实证；已排除客户/主数据及系统工作台管理快捷项的误报，优先级按窄屏场景调整为 P2。

### LIST-01 / P2：订单收起筛选会清空表单回显，但实际查询继续带旧条件

- 代码证据：`web/src/components/ui/order-list-template/OrderListTemplate.tsx:64` 的 `currentFilterRef` 留存已提交条件；`:567` 条件挂载整个 `OrderListSearchFilter`，工具栏在 `:601` 切换 `filterVisible`；`:633` 列表请求继续消费 ref；`OrderListSearchFilter.tsx:172` 自建表单，`:256` 仅初始化单号类型和全选状态，没有传入父级已提交筛选值。
- 触发步骤：`/orders/sea-export` → “展开筛选” → 单号框填标记并查询 → “收起筛选” → “展开筛选” → 检查单号框、总条数及刷新 GET 的 `numberKeyword`。
- 用户影响：表单显示无条件，而数据仍是筛选后的子集；用户容易以为订单缺失。单纯收起筛选本应改变展示，不应丢失可见查询口径。尚未提交的筛选输入也会丢失。
- 最薄建议：收起时保持同一表单实例/组件挂载，仅隐藏展示；或由父级持有 Form 实例及完整原始值，展开时正确还原日期和选项。无需引入页面全局状态。可加已生效条件摘要，但保留现有表单已足以修复。
- 浏览器检查：`input[placeholder="输入单号"]`；“展开筛选”“收起筛选”“查询”“刷新”；比较查询前后只读列表请求。表单内部“展开更多/收起”依赖 Antd Form preserve，不应与整个面板卸载混淆。
- 状态：代码强证据，待浏览器确认。

### LIST-02 / P2：订单“自动刷新”开关承诺每 60 秒刷新，实际只改变外观

- 代码证据：`web/src/components/ui/order-list-template/OrderListToolbar.tsx:64` 仅声明 local state；`:219` 文案“开启后每 60 秒自动静默刷新表格数据”；`:223` 开关只调用 `setAutoRefresh`；整个模块没有 effect、计时器、轮询或把状态回传父层。`OrderListTemplate.tsx:595` 只提供手动 `onRefresh`。
- 触发步骤：`/orders/sea-export` → 开启“自动刷新” → 保持页面前台观察列表 GET 至少 65 秒；手动刷新应能正常请求。
- 用户影响：用户相信订单状态自动更新而实际停留旧数据，可能遗漏其他员工的最新处理结果。
- 最薄建议：若当前不交付自动刷新，移除该开关和承诺；若保留，则真实接入现有 `actionRef.reload()`，卸载清理计时器，并在后台/弹层编辑等需要保持稳定的场景暂停。先明确当前业务需要，不为此建设轮询平台。
- 浏览器检查：“自动刷新”相邻 `[role="switch"]`、`.ant-table-wrapper`，采集列表 GET 次数；该 switch 没有 aria-label，键盘无障碍名称可并入此项修复。
- 状态：代码确认；65 秒验证为可选补强，不需要修改任何业务记录。

### NAV-02 / P2：页签及关闭按钮主动移除焦点描边，键盘位置不可见

- 代码证据：`web/src/components/layout/TagsView.tsx:378` 页签 `role="tab"` 和 `tabIndex={0}`，`:386` 能通过 Enter/Space 激活；`:401` 关闭按钮有 aria-label；但 `web/src/global.less:653` 和 `:693` 明确 `outline: none`，整个页签样式未提供 `:focus-visible`。
- 触发步骤：打开至少两个业务页签，移开鼠标，用 Tab 连续进入未激活页签及关闭按钮，观察是否有独立于 hover/active 的可见焦点。
- 用户影响：不是键盘完全不能激活，而是用户无法判断焦点在哪一页和哪个关闭按钮，容易误导航/关闭。
- 最薄建议：页签和 close 添加一致的 `:focus-visible` 描边，必要时补左/右/Home/End 键及 roving tabindex，但后者可后续增强。不能将所有按钮缺 aria 一概归为本问题。
- 浏览器检查：`.roncin-chrome-tab[role="tab"]`、`.roncin-chrome-tab-close`，比较 focus 时 outline/box-shadow/background；若实际浏览器存在其他覆盖样式，应以观察为准。
- 状态：代码强证据，待浏览器焦点实证。

### TABLE-01 / P2 增强：统一列宽调整仅支持鼠标，键盘与触屏没有等价操作

- 代码证据：`web/src/components/ui/responsive-table/ResizableHeaderCell.tsx:138` 创建无 tabindex 的 div；`:144` 手柄宽 7px；`:156` 只接 `onMouseDown`，`:158` 双击展开/还原；`:127` 监听 mousemove/mouseup，没有 Pointer/Touch 或键盘调整事件。`:119` 支持拖动中 Escape 取消，这一点已做好。
- 触发步骤：任意列表 → 只用 Tab 找调整列宽入口；或触屏设备拖动表头分隔线。鼠标能拖动和双击，键盘无操作入口。
- 用户影响：影响偏好设置便利性，对主要桌面鼠标用户不会阻断业务录入；不能按 P1 业务阻断描述，也不要求把所有高密度按钮变成 44px。
- 最薄建议：在列设置中增加可键盘操作的宽度/还原控件，或给手柄补 separator 语义、tabindex、左右调整与可见 focus；如确有触屏需要，再使用 Pointer Events 和 capture。优先提供等价简单操作，而不是建立通用手势系统。
- 浏览器检查：`[data-column-resize-handle]`、列设置 `[aria-label="列设置"]`；与原表头排序点击互不干扰；新实现需保留 Escape/blur/unmount 的拖动清理。
- 状态：代码确认。刷新恢复初始列宽属于本次已认可行为，不列为缺陷。

### LIST-03 / P2：多页签仅保存路径，跨页面返回不恢复列表条件/页码

- 代码证据：`web/src/app/AppLayout.tsx:90` 的 OrganizationWorkspace 内仅渲染 Outlet，没有保留路由组件；`TagsView.tsx:108` 和 `:216` 只保存/跳转完整 path；客户列表 `web/src/pages/partners/index.tsx:478` 的搜索参数初始 `{}`、`:607` 分页初始 20，未读取 URL 搜索状态。订单 `OrderListTemplate.tsx:64` 本地 ref 同样如此。
- 触发步骤：客户页提交关键字/状态、跳到第 2 页 → 进入费用页 → 点击既有“客户”页签 → 或打开客户详情“返回列表” → 检查条件与页码是否保留。
- 用户影响：视觉上保留了页签，操作上却重新开始查询；在客户、订单、账单反复核对时重复输入成本高。与未保存表单草稿不是同一个机制。
- 最薄建议：先将代表列表已提交关键词/筛选/页码同步 URL，并使列表初始化消费它；或按现有用户/组织/业务页签 scope 保存轻量快照。不保存结果数组、不引入全站 KeepAlive；关闭页签与组织切换清理规则需同步。
- 浏览器检查：客户 `input[placeholder*="搜索单位编码"]`、分页 `.ant-pagination`、业务页签 `[role="tab"]`；切到真正不同的路由以触发卸载，而不是同一个列表内部角色切换。
- 状态：代码强证据，待浏览器确认。

## 待复核候选，不直接写入确认问题

1. **顶部下拉和列表“更多”是否纯键盘无法展开**：`HeaderDropdown/index.tsx:44` 原样传递 Antd Dropdown；`HeaderMenus.tsx:173`、`:195` 和 `OrganizationSwitcher.tsx:133` 没有显式 trigger/click/key handler。不能因没有自己的 onKeyDown 就否认框架能力；需聚焦按钮后按 Enter/Space/ArrowDown 测试。如果确实失败，再列 P2，薄修复使用可由键盘激活的 click trigger 并保留原 hover。
2. **重新查询未归第 1 页**：`partners/index.tsx:526`、`OrderListTemplate.tsx:571`、`finance/fees/index.tsx:143` 均只 reload，局部筛选未接 ProTable 搜索引擎；本地 `@ant-design/pro-components/es/table/useFetchData.js:403` reload 不主动改分页。但框架可能在 total 变化后自动纠正并补请求。需要数据多于一页，从第 2 页筛出少量数据并观察请求链/最终呈现再判断，不能只读这几行断言假空结果。
3. **列表首次失败与空数据显示混同**：这些公共模板没有自定义 onRequestError/error 行内展示，但统一 requestErrorConfig 已提供 notification，ProTable 有自己的失败处理。可模拟 GET 500 与 GET 200 空数组对照，待通知消失后看是否能辨别失败并重试。当前仅可列“待检查”，不能称无任何错误反馈。
4. **页签右键菜单的键盘入口/溢出导航**：`TagsView.tsx:376` 仅 contextMenu；横向容器 `global.less:634` auto 滚动但隐藏滚动条。页签主体/关闭均可 Tab，浏览器可能自动滚动焦点进入可视区；需实测 Shift+F10、ContextMenu key、较多页签主动激活后是否自动进入视野，不能仅因无自定义 scrollIntoView 判断页面无法导航。

## 已做好的交互

- 列排序除了拖拽，还提供带字段名 aria-label 的上下按钮：`ColumnSettingsPanel.tsx:223`、`:231`；`ColumnSettingsPanel.less:22` 的 focus-within 使键盘聚焦时箭头显现。不会把原生拖拽方式误判为排序只有鼠标。
- 列设置统一中文入口标签、冻结 Switch 标签，至少保留一列/必显列，显示当前冻结目标，搜索时禁排序并解释原因：`ColumnSettingsPanel.tsx:163`、`:257`、`:305`。即时生效、无需保存且用户/组织/表格偏好隔离；存储失败会明确提示本次仅当前页面生效：`useColumnSettings.tsx:304`。
- 表格布局按实际容器测量横滚、自动释放窄容器过宽冻结列、避让顶部页头，并处理弹窗自身滚动区域：`useResponsiveTableLayout.ts:14`、`:38`、`:109`。没有把页面整体横滚作为宽表兜底。
- 统一列宽组件只在 mouseup 提交、鼠标移动采用 requestAnimationFrame，Escape/失焦/卸载清理拖动：`ResizableHeaderCell.tsx:93`、`:103`、`:119`，有参考线/像素反馈。
- 页签可 Enter/Space 激活，关闭按钮含具体标题 aria-label；批量关闭和组织切换均调用 dirty guard：`TagsView.tsx:233`、`:290`、`OrganizationSwitcher.tsx:66`。
- 组织切换展示目标组织 loading、禁止重复选择，失败不换工作区；以用户+组织 key 重挂工作区、清空相应候选缓存：`OrganizationSwitcher.tsx:29`、`:39`、`:159`，`OrganizationWorkspace.tsx:4`。本审查不为了偏好保留而取消组织隔离。
- 统一搜索模板对已提交条件提供 chips，远程下拉保留展示名称，删除 chip 会按完整剩余条件重查：`SearchFilterTemplate.tsx:137`、`:192`、`:240`。订单专属筛选没有消费这一能力，所以 LIST-01 是局部行为差异。
- 统一请求层对 401 返回原页面、403 使用业务中文提示、超时说明先确认结果再重试，普通失败有追踪信息；写入期间去重：`requestErrorConfig.ts:28`、`:170`、`:187`。路由异常页面提供明确重新加载入口：`router/RouteErrorPage.tsx:4`。

## Files found

| 文件 | 职责 |
| --- | --- |
| `web/config/routes.ts` | 路由与权限入口、菜单隐藏真相 |
| `web/src/router/adaptRoutes.tsx` | 统一路由适配与侧栏菜单过滤 |
| `web/src/app/AppLayout.tsx` | ProLayout 顶栏/侧栏/组织工作区壳 |
| `web/src/components/layout/TagsView.tsx` | 页签路径、关闭和右键操作 |
| `web/src/components/layout/OrganizationWorkspace.tsx` | 按用户组织身份隔离工作区 |
| `web/src/components/OrganizationSwitcher.tsx` | 组织切换反馈与 dirty guard |
| `web/src/components/layout/HeaderMenus.tsx` | 企业资源和设置中心快捷入口 |
| `web/src/components/HeaderDropdown/index.tsx` | 官方 Dropdown 透传与样式 |
| `web/src/components/ui/order-list-template/*` | 订单筛选、工具栏、官方 request 列表 |
| `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx` | 通用已提交条件摘要与搜索 |
| `web/src/components/ui/column-settings/*` | 显隐排序、冻结、隔离持久化 |
| `web/src/components/ui/responsive-table/*` | 容器横滚、吸顶和统一调宽 |
| `web/src/pages/partners/index.tsx` | 客商代表列表搜索与分页 |
| `web/src/pages/finance/fees/index.tsx` | 财务代表列表筛选 |
| `web/src/requestErrorConfig.ts` | 统一网络错误反馈与写入防重 |
| `web/src/global.less` | 顶栏/页签/窄屏导航关键样式 |

## External references / Related specs

- 本次未调用外部互联网资料；使用用户指定本地 UI-UX-Pro-Max 技能及现有本地框架实现。未把手机 App 的触控 44pt、safe-area、全站大字和双主题要求直接应用桌面后台。
- `.trellis/spec/web/frontend/column-settings.md`：偏好隔离、即时生效与冻结规范。
- `.trellis/spec/web/frontend/component-guidelines.md` 和根 `AGENTS.md`：高密度纯白、全宽、公共模板、官方 request/actionRef。
- `.trellis/spec/web/frontend/state-management.md`：服务端数据与页面状态边界。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：客户、订单、费用与权限工作区职责。

## Caveats / Not Found

- 未控制浏览器、未截图、未读取用户凭据、未提交业务写入。本文只输出候选、源代码证据与主会话可以安全验证的步骤。
- P2 为建议优先级；NAV-01 已补 390px Drawer 实证及工作台绕行入口限制；浏览器发现推翻静态推断时以实证为准，并更新最终审查报告。
- 表单/写入 Modal 未审查，由另一研究代理负责，避免重复工作。
- 一些数据依赖路径（多页搜索、未知权限角色）需要适当只读数据才能验证；不为审查创建订单、客户、账单或真实自动刷新修改记录。
- 未进行全站每一条路由的可达性证明；公共模板证据可覆盖实际消费者，但不能把一次样本实测扩大为所有页面均实测。
