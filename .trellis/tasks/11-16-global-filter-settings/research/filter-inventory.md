# 研究：全站筛选入口与布局接入清单

- 研究问题：全站哪些界面实际存在筛选，如何统一显隐、排序、单项宽度、收起展示数量与恢复默认，同时改为账号跨浏览器保存？
- 研究范围：仓库内部；仅盘点现有源码与规范，不改产品代码、不运行实施或浏览器验收。
- 日期：2026-11-16（Asia/Shanghai）

## 检查方法与结论边界

在 `web/src/pages`、`web/src/features`、`web/src/components/ui` 的非测试 TS/TSX 中检索共享搜索组件、ProTable `search`/`options.search`、`Input.Search`、独立 Input/Select/Segmented/日期控件以及查询状态，随后阅读命中组件的筛选渲染、提交与请求路径。共享控件内部远程候选搜索、业务录入字段、结构导航不直接等同于可隐藏的筛选字段。

这是**源码入口与相关控制流核对**，不是每个文件全文审计，也未完成全部页面浏览器验收。下表按实际区域分组，多个业务页复用同一组件时同时记录隔离维度；实施时须逐个身份完成覆盖记录，不能仅按组件个数验收。

## 已找到的公共能力

| 文件与锚点 | 现有职责 | 本期缺口 |
| --- | --- | --- |
| `web/src/components/ui/filter-settings/FilterSettingsEntry.tsx:15` | 360px 勾选浮层、恢复默认，勾选即时生效 | 缺排序、单项宽度、折叠数量与异步保存状态 |
| `web/src/components/ui/filter-settings/useFilterSettings.ts:34` | 用户＋组织＋pageKey 的浏览器 localStorage 显隐偏好 | 需统一服务端读取/写入、加载完成与保存失败语义 |
| `web/src/components/ui/filter-settings/types.ts:2` | 字段标识、名称、默认可见及 required | 需补稳定排序、宽度与折叠数量展示模型，不能混入查询值 |
| `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx:160` | grid 依 `visibleNames` 过滤、保持原 `allItems` 顺序 | 不能把 visibleNames 数组顺序误当成已生效的排序；需真正按稳定配置排列 |
| `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx:251` | 移除字段清 form、committed、labels；仅有已提交值时按剩余条件刷新 | 保留未提交其他输入、范围字段整体清除；对新 bar/自定义适配要复用同样语义 |
| `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx:505` | bar 的关键字与 quickFilters 固定渲染 | visibleNames 当前不决定 bar 的输入显隐/顺序/宽度，不能只添加齿轮入口 |
| `web/src/components/ui/search-filter-template/types.ts:29` | 开发者声明 `span`，默认 24 栅格 | 可复用布局，不需要引入自由画布或拖拽坐标系统 |
| `web/src/components/ui/order-list-template/OrderListSearchFilter.tsx:209` | 独立 Form/Row/Col、远程候选、日期转换、复合单号、hidden 折叠 | 需同一字段模型的适配；原生复合控件不能拆散 |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:101`、`:628` | 独立关键字/启用状态/自定义 Select，客户端与服务端两模式 | 需要共享设置布局并分别清理本地状态或服务端 query；不能额外增加本来没呈现的筛选项 |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:268` | 可选官方 ProTable 搜索；大量页面传 customSearch | 搜索协议必须保持官方 request/actionRef，不建立外层 data/loading 引擎 |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx:165` | search 默认关闭；可配置启用 ProTable 搜索 | 已读现有消费方未开启搜索，不能为了统一而新增业务筛选 |

`SearchFilterTemplate` 提供 custom 模式，但在上述目录的非测试代码中没有找到 `layout="custom"` 消费。现有订单独立表单不是该模式，不能声称 custom 模板改动会自动覆盖订单。

## 区域清单：订单、客商、主数据

| ID | 区域／隔离身份 | 当前机制与源码锚点 | 建议接入点及验收风险 |
| --- | --- | --- | --- |
| F01 | 四类订单列表：海出、海进、空出、空进分别隔离 | `pages/orders/list.tsx:226`、`:243` → `components/ui/order-list-template/OrderListTemplate.tsx:722` → `OrderListSearchFilter.tsx:209` | 同一可配置筛选布局适配独立 Form；复合单号 numberType＋numberKeyword 作为一项，日期范围整体，空运不显示船公司；移除条件同步 queryState/currentFilterRef、回第一页、清选择并使旧导出失效；不能重挂载丢未提交草稿 |
| F02 | 共享箱抽屉的同航次 HOUSE 订单货物候选 | `pages/orders/components/drawers/SeaSharedContainerAllocationTable.tsx:168` Input.Search；`SeaSharedContainerDrawer.tsx:92`、`:148` 查询状态 | 单关键字同样有设置；只清 candidateKeyword/candidateSearching 并回第1页，保留已分配货量草稿；身份按业务视图而非具体箱/订单ID存展示偏好 |
| F03 | 客户、供应商、国外代理等客商列表及普通/黑名单视图 | `pages/partners/index.tsx:519` bar：关键字＋enabled；`:591` 切视图 | bar 需真正支持设置；按实际路由/角色与视图定义稳定区域键，黑名单与普通视图仍属导航，不能通过设置改客商角色或黑名单范围；提交、导出沿用 queryState |
| F04a | 货币主数据 | `pages/master-data/components/CurrenciesPanel.tsx:85` → MasterDataTemplate | 关键字＋启用状态；客户端清值同步搜索/过滤和页码，无服务端搜索放大 |
| F04b | 中国行政区划 | `pages/master-data/components/CitiesPanel.tsx:76`、`:105` | 关键字＋启用状态＋区划层级；客户端清 filterValues.level，分页与统计同口径 |
| F04c | 船代 | `pages/master-data/components/ShippingAgentsPanel.tsx:77` | 关键字＋启用状态；沿用已声明数据范围，不把系统共享/公司数据范围做成可移除字段 |
| F04d | 全球港口 | `pages/master-data/components/PortsPanel.tsx:100`、`:110` | 服务端模式实际显示关键字＋启用状态；移除清 query.keyword/enabled 并回第1页；不循环翻页加载全量 |
| F04e | 全球机场 | `pages/master-data/components/AirportsPanel.tsx:101`、`:111` | 同服务端模式；按机场视图独立偏好 |
| F04f | 全球航司 | `pages/master-data/components/AirlinesPanel.tsx:104`、`:114` | 同服务端模式；不能把声明但当前不渲染的国家筛选当作已有交互 |
| F04g | 全球船公司 | `pages/master-data/components/ShippingLinesPanel.tsx:115`、`:125` | 同服务端模式；按船公司视图独立偏好 |
| F04h | 国家地区 | `pages/master-data/components/CountriesPanel.tsx:99`、`:109` | 当前传 query/onQueryChange 走服务端模式；同上 |

重要细节：MasterDataTemplate 的自定义 `filterOptions.map` 包在 `!serverMode` 条件内（`MasterDataTemplate.tsx:663`）。Ports/Airports/Airlines/ShippingLines/Countries 虽各自声明 filterOptions，但当前服务端模式**不渲染它们**。本期应按真实现有字段接入，不顺便新增这些筛选功能或扩服务端契约。Cities 客户端模式的层级过滤是真实存在的。

## 区域清单：系统管理与企业资源

| ID | 区域／隔离身份 | 当前机制与源码锚点 | 建议接入点及验收风险 |
| --- | --- | --- | --- |
| F05 | 用户列表 | `pages/admin/users.tsx:120` bar 单关键字；`:203` 请求 params | 单关键字也必须入口可达；隐藏后清 searchParams.keyword、选择和页码，未提交编辑表单不受影响 |
| F06 | 角色列表 | `pages/admin/roles.tsx:299` bar 单关键字，`:353` 本地过滤响应 | 与角色配置模态的权限搜索分开区域键；隐藏时刷新官方 request，不改变权限配置草稿 |
| F07 | 组织架构树表列表的组织搜索 | `pages/admin/organizations.tsx:113`、`:187`、`:405` 独立 Input 本地树过滤 | 关键字单项；清值恢复树、保持原有展开选择逻辑；拓扑/树表切换及方向是结构视图，不混作可移除业务范围 |
| F08 | 权限字典列表 | `pages/admin/permissions.tsx:23`、`:96`、`:107` 官方 ProTable：keyword＋module，本地对完整字典过滤 | 搜索字段应与列设置解耦，防止隐藏表格列连带隐藏搜索项；模块选项来自完整数据，不能随过滤缩减 |
| F09 | 操作与安全审计 | `pages/admin/audit.tsx:38`、`:116`、`:225` ProTable dateRange | 只有 timeRange 真正可搜索；移除完整清起止 UTC 请求参数，不能把日志展示时间列也变成筛选项 |
| F10 | 后台任务 | `pages/admin/background-tasks.tsx:80`、`:162`、`:272`、`:382` ProTable：timeRange＋kind＋status | 当前/历史 Tabs 是结构范围；切 phase 原本清 status，新设置不得恢复不适用状态；筛选偏好可按页保持但范围不被隐藏 |
| F11 | 角色新增/编辑弹窗的权限矩阵搜索 | `pages/admin/components/roles/components/PermissionMatrixHeader.tsx:95`；`RoleFormModal.tsx:187`、`:415` | 这是真实已展示权限列表搜索，不是下拉内部搜索，计入；显隐/宽度只作用关键词区域，隐藏清 keyword 后返回模块过滤，保留 selectedPermissionKeys、依赖联动与权限草稿；ModuleNavPanel/业务线条是矩阵导航，不拆成任意可隐藏权限范围 |
| F12 | 用户编辑弹窗中的组织关系当前/已停用视图 | `pages/admin/components/users/UserFormModal.tsx:391`、`:408` Segmented 切实际成员关系列表 | 这是改变可见列表的本地视图控制，应在总清单登记；建议作为必要视图项保留不可隐藏，提供设置入口及宽度/恢复默认，不把某个用户ID作为展示偏好身份；是否合并到该列表的独立布局按最终方案说明 |
| F13 | 企业资源六类列表：地址、备注、图片、收货人、发货人、通知人 | `router/workspaceTabs.ts:9`；`pages/enterprise-resources/index.tsx:368`、`:669` 官方 ProTable 搜索 | 每个资源类型独立区域键；通用 keyword/linked/partnerId，地址另有 addressType/assigneeId；删除范围及候选需按真实字段映射清理，清 selectedKeys 和第1页；企业关联搜索下拉内部 keyword 本身不另建区域 |

## 区域清单：财务已有设置区与报表

| ID | 区域／隔离身份 | 当前机制与源码锚点 | 建议接入点及验收风险 |
| --- | --- | --- | --- |
| F14 | 费用台账 | `pages/finance/fees/index.tsx:104`、`:258`；`components/FeeLedgerSearchFilter.tsx:72` | 已有 settings＋grid；新增布局维度、服务端偏好读取；页签状态初始化必须等偏好就绪后 sanitize，避免异步默认值让隐藏条件继续过滤 |
| F15 | 账单列表 | `pages/finance/bills/index.tsx:100`、`:455`、`:473` | 已有 settings＋grid，同上；统计/勾选/导出保持官方采纳快照 |
| F16 | 发票列表 | `pages/finance/invoices/index.tsx:121`、`:496`、`:502` | 已有设置，保持登记/修订入口不受布局改变影响 |
| F17 | 资金流水 | `pages/finance/cashflows/index.tsx:119`、`:500`、`:506` | 已有设置；完整日期字段清除，汇总及导出不读编辑草稿 |
| F18 | 核销记录 | `pages/finance/verifications/index.tsx:70`、`:435`、`:441` | 已有设置；独立于工作台双表候选搜索 |
| F19 | 对冲记录 | `pages/finance/nettings/index.tsx:93`、`:493`、`:499` | 已有设置；不能把方向或币种范围错误当成隐藏后默认不限的业务写入参数 |
| F20 | 提成台账 | `pages/finance/commissions/index.tsx:102`、`:707`、`:773` | 已有 settings，custom 日期 item 仍是一个设置项，保持原日期规范化；台账/申请/冲减等分段视图不是字段显隐 |
| F21a | 利润报表 | `pages/finance/reports/profit.tsx:128` → `ReportShell.tsx:122`、`:292` | 已有设置，默认本月日期仅在 dateRange 可见时初始化；业务日期、候选及完整导出保持同口径 |
| F21b | 利润分析 | `pages/finance/reports/profit-analysis.tsx:1` → profit 页面 analysisMode | 与利润报表不同 pageKey，布局偏好隔离；月图与列表仍共用同一筛选快照 |
| F21c | 实际收付款报表 | `pages/finance/reports/cashflows.tsx:88` → ReportShell | 已有设置，实际交易日期与利润业务日期不能混淆 |
| F21d | 当前账龄表 | `pages/finance/reports/aging.tsx:108` → ReportShell | 已有设置，bucket/币种/对象可配；不新增历史截止日筛选 |
| F22 | 利润分析月图的本币展示控制 | `pages/finance/reports/ProfitChart.tsx:44` Select，本地选币种投影 monthlySummary | 属于图表视图控制，记录到覆盖清单，建议声明为必要项不可隐藏，单项宽度可设置；保存布局不保存具体 currency 查询/展示值，不能将其伪装为新的跨币折算 |

## 区域清单：财务次级面板、工作台及弹窗候选

| ID | 区域／隔离身份 | 当前机制与源码锚点 | 建议接入点及验收风险 |
| --- | --- | --- | --- |
| F23 | 提成月度申请管理 | `pages/finance/commissions/components/CommissionApplicationsPanel.tsx:272` grid：status/applicationMonth/employeeId | 尚无设置；月选择器 Dayjs→YYYY-MM，隐藏清对应 committedFiltersRef，只保留实际后端默认待审批语义，不意外提交未保存审批意见 |
| F24 | 提成待执行冲减 | `pages/finance/commissions/components/PendingDecreasePanel.tsx:258` grid：keyword/status | 尚无设置；已有后端默认状态含义需保持，不将“隐藏控件”当成悄悄恢复默认条件；刷新走 reload/actionRef |
| F25 | 提成方案抽屉 | `pages/finance/commissions/components/CommissionRulesDrawer.tsx:416` 独立 employee Select＋`:441` ProTable 搜索（name/personnelRole/calculationBasis/enabled） | 合并为同一区域可声明员工及官方字段，避免员工在设置之外；员工查询清值并刷新；注意现有 request `:470` 未透传 calculationBasis，属于已存在 UI/请求不一致，应核对并记录，不能擅自扩后端接口 |
| F26 | 核销工作台待核销资金候选 | `pages/finance/verifications/VerificationWorkbench.tsx:730`、`:739` options.search keyword | 单关键字也有入口，使用官方 request；原规范搜索/分页保留已选业务来源与金额分配，不套用普通财务台账筛选清选择规则 |
| F27 | 核销工作台待核销账单候选 | 同文件 `:784`、`:793` | 与资金表单独区域键；不因同步偏好重挂载整个工作台丢分摊草稿；两表相互独立页码 |
| F28 | 建账工作台费用候选 | `features/finance/bill-creation/BillCandidateSelectionStep.tsx:54`；`billWorkbenchFeeColumns.tsx:115` keyword＋direction 官方搜索 | 尚无设置；沿用当前公司、canBill 与来源范围；fixedSelection 带入场景本来没有候选筛选，不额外新增；不能因隐藏关键字清掉已选费用 |
| F29 | 发票登记弹窗的账单候选 | `features/finance/invoice-registration/InvoiceRegistrationModal.tsx:227`、`:459` keyword 官方搜索 | 尚无设置；`form={{ component: false }}` 防止嵌套表单，Enter 只能查询候选不能提交发票登记；隐藏/重排不丢 bills 及发票金额草稿；固定带入账单/修订模式不渲染候选区域 |
| F30 | 账单增费候选 | `pages/finance/bills/components/BillFeeMaintenance.tsx:274` Input.Search | 单关键字；清 keyword＋page，不改原已选候选费用或账单编辑草稿；同公司、同币种、同对象、同方向是不可移除业务范围 |
| F31 | 我的提成明细抽屉 | `pages/workbench/MyCommissionDrawer.tsx:196` 独立 status Select | 尚无设置；隐藏清 query.status＋page=1，保持本人范围、本币展示；列表与冲减下钻独立表头 |
| F32 | 可申请提成明细抽屉 | `pages/workbench/MyApplicationCandidatesDrawer.tsx:156` commissionMonth Select | 尚无设置；隐藏只清展示月过滤，不能更改“截至上月末全部合格项”申请范围；偏好不保存具体月份 |
| F33 | 我的申请历史抽屉 | `pages/workbench/MyApplicationHistoryDrawer.tsx:428` status Select | 尚无设置；主列表和明细返回保持原上下文；不影响被驳回重提操作或固化金额快照 |

## 表头业务列过滤核对

在上述目录非测试 TS/TSX 中检索 `filters:`、`onFilter:`、`filterDropdown:`、`filteredValue:`、`defaultFilteredValue:`、`filterIcon:`、`filterMode:`、`filterSearch:` 及相应 JSX 属性形式，**未发现实际 Ant Design 业务列过滤实现**。`filters` 命中为 list-query-state、领域查询对象与局部变量；不是列配置。ProTable 的 `valueType: select` 默认搜索也不等于列头过滤。

因此当前没有需要额外迁入筛选设置的表头过滤入口。列排序属于表格排序交互，应保留；不能误纳入字段显隐。实施后仍应以全量静态复核加浏览器代表页核对防止遗漏新增入口。

## 排除项与已核对无筛选区域

以下区域已读相应控件或明确 `search={false}`／数据来源，排除是基于职责，不是忽略用户“所有筛选”的要求：

| 类别 | 证据／范围 | 排除理由 |
| --- | --- | --- |
| 下拉候选内部搜索 | `components/ui/searchable-select/RemoteSearchSelect.tsx`、`features/finance/bill-creation/BillVerificationFields.tsx:157`、CommissionCreateModal 的核销/对冲来源 Select、用户组织/员工候选等 | 仅搜索一个输入控件的选项，不是独立业务结果列表；隐藏会影响业务录入，不应自动包装为筛选设置 |
| 业务录入 | `orders/components/split/SplitFeesSection.tsx:65`、`SplitAllocationSection.tsx:79` 的归属结果票 Select；费用输入、单证正文、汇率录入等 | 改的是业务草稿，不是筛选条件 |
| 页签/角色/业务类型结构导航 | `partners/index.tsx:591`、`finance/commissions/index.tsx:684`、后台任务 Tabs、企业资源类型页签、角色权限模块/业务线导航 | 决定视图/业务范围，不能个人设置移除或隐式改变；可配置区只包含对应实际筛选字段 |
| 列设置面板内字段搜索 | `components/ui/column-settings/ColumnSettingsPanel.tsx:61` | 设置面板自身的定位功能，不能递归添加“筛选设置的筛选设置” |
| 无业务筛选的已有表 | 已查：订单费用表 `OrderFeeTableTabs.tsx:1833`、订单异常/放单回单面板、附件/单证/拼单抽屉、空运主单成员/历史、进口事件；客商二级 Accounts/Contracts/Attachments/InvoiceProfiles/SettlementRules 的 search=false；工作台待办 TodoTable 与应收下钻无额外筛选 | 不应凭空新增筛选，只纳入表头偏好服务端保存部分 |
| 配置表与汇率列表 | `SettingTableTemplate.tsx:165` 默认 search=false，现有费用项目/计费单位/应税劳务/费用模板/异常类型未传 search；`ExchangeRatesPanel.tsx:267` search=false；编号规则无搜索控件 | 本期不增加业务查询字段，表头设置仍需跨浏览器保存 |

F12 成员关系当前/已停用与 F22 图表本币明确登记为视图控制，不与普通业务录入混淆；最终方案需说明其“必要项不可隐藏”的规则，不能口头声称所有控件均任意隐藏。

## 最小公共演进建议

1. 继续复用 `components/ui/filter-settings` 的公开入口与字段声明，补 `order`、每字段 `span`、`collapsedVisibleCount`；持久化统一走服务端偏好能力，页面只声明字段及稳定区域键。列表查询值、日期和候选 label 不进偏好载荷。
2. grid/bar/独立区域统一渲染能力。grid 以已解析顺序布局；bar 可映射为紧凑横排字段后共享栅格，保留原即时下拉提交行为。独立复合控件用 field.render 适配，避免重写订单远程候选与日期格式化逻辑。
3. 宽度建议使用既有 24 栅格模型：提供“默认、1/6、1/4、1/3、1/2、整行”等可读选项，默认保留开发者原 span，窄屏降为单列/双列；不保存像素坐标。日期范围和复合单号应有合理最小可用宽度。收起数量是可见字段中前 N 项；0 项是否允许需在最终设计声明，齿轮/查询/重置不可被一起隐藏。
4. 可选字段允许全部隐藏，设置入口仍必须可达；required 字段可重排/调宽但不可隐藏。排序、宽度及折叠数量只改变展示，不提交业务筛选、清勾选或请求列表。
5. 新增服务端异步读取后，先确定当前用户/工作区/区域的偏好，再初始化查询状态并发起业务查询。当前 `useListQueryState.initial` 和表单 `initialValues` 都只首挂载初始化，不能先挂默认并让远端偏好迟到后悄悄覆盖输入。加载失败显式错误和重试，不把读取失败当成“未设置”。
6. 移除字段的统一清理需映射到表单字段及真实请求键；日期范围清起止，复合单号清 type＋keyword。只能提交清理后的**原已提交条件**，不能趁设置变化提交其它未保存草稿。对核销/建账/发票候选按原业务规则保留已选来源，不能全局机械清选择。
7. 官方 ProTable 搜索字段应来自与列设置独立的完整搜索 schema；若迁移为 customSearch，则保持官方 request、actionRef、onLoad 和采纳快照机制，不引入平行 data/loading 状态。

## 分组执行与验证建议

- 子任务 A：服务端账号偏好（另有专项研究）先完成，明确身份、读取、保存、失败与切换隔离。
- 子任务 B：前端共享偏好提供层、持久化与表头接入；为筛选区提供统一读取/写入来源。
- 子任务 C：筛选设置与布局公共模型、grid/bar/独立渲染、条件清除与恢复语义。先用费用台账、用户单关键词和订单复合控件验证三种真实用法。
- 子任务 D：全站筛选接入。包括订单、客商、八主数据、系统管理、企业资源，财务七台账＋四报表升级，以及次级面板、工作台三个抽屉、核销双表和建账/发票/账单增费/共享箱候选补覆盖；特殊保留选择路径独立验证。后端业务查询契约保持原样。
- 最终逐项核对清单中的每个实际区域身份，再做两浏览器同账号恢复、两账号/两工作区隔离、保存失败/迟到响应、窄屏与键盘排序/设置操作验收。
- 高风险来自跨领域公共组件、服务端偏好契约和异步初始化，最终运行完整前后端门禁；定向测试先覆盖隐藏已提交条件、隐藏未提交值、折叠保值、排序调宽不查询、范围/复合字段、读取失败、切身份响应隔离和嵌套表单 Enter。

## 相关规范

- `.trellis/workflow.md`：研究只写任务 research，规划未批准前不实施。
- `.trellis/spec/web/frontend/capability-navigation.md:31`：现有筛选设置公共入口；其中“仅存本浏览器、不支持排序/同步”需在本任务实施完成时更新。
- `.trellis/spec/web/frontend/finance-ledger-snapshots.md`：官方 onLoad、完整汇总、跨页选择；现有筛选移除清值与本地偏好说明需同步。
- `.trellis/spec/web/frontend/state-management.md:350`、`:355`、`:361`：列表查询快照、initialValues 单次初始化、订单 hidden 折叠保护输入。
- `.trellis/spec/web/frontend/component-guidelines.md:47`：页签已确定范围不加重复筛选。
- 根 `AGENTS.md`：禁止 TDD、保留已有修改、正式生成流程、跨领域依赖边界、全站满宽与窄屏可用。

## 外部参考

无外部依赖或第三方 API 选型研究；以上仅基于仓库当前实现，不对 Ant Design/ProComponents 新版本能力作外部推断。

## 限制与未发现项

- 没有运行页面、请求、持久化或数据库验证；所有建议仍需最终设计和实施验收。
- 没有审计全站所有编辑字段；检索命中后只阅读与筛选/查询相关的源码区段。
- 尚未将 F01/F03/F13 等多视图分组展开成最终验收身份清单，实施计划应根据四订单 kind、实际客商路由和六企业资源类型展开。
- 服务端偏好并发写入策略、确切保存时机、版本/事务取舍由专项后端研究与最终方案确定，本文件不替用户决定。
