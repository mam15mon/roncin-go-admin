# Research: 列表筛选、自动刷新与页签查询上下文实施预研

- Query: A04/A05/A12 已批准修复如何以最薄公共机制落地，覆盖订单、客商、费用、账单及资金流水，不架空 ProTable。
- Scope: internal
- Date: 2026-10-12

## Findings

### 1. 范围与结论

本预研为后续 `.trellis/tasks/10-12-ui-list-state` 的交接资料，仅写当前保存保护任务的 research。已读取该兄弟任务 PRD/design/implement；不修改产品、任务状态或其它任务文档。保存保护实现者正在改客商详情及财务写路径，列表实现必须按已批准顺序在前置提交后启动。

推荐在 `OrganizationWorkspace` 内增加一个工作区实例级的查询快照 Provider，通过通用公开入口让页签层和列表消费。只保存已提交条件、显示名称和分页，不保存响应、loading、选中行、未提交表单或完整 React 页面；不使用全局 Map、匿名共享 namespace、sessionStorage 或 KeepAlive。

搜索展开/收起与跨页签恢复应分别处理：订单筛选 DOM/Form 保持挂载仅隐藏，保留未提交编辑；跨业务页签卸载再挂载只恢复已提交查询。

### 2. 相关规范和代码地图

- `.trellis/spec/web/frontend/state-management.md`：本地 UI 状态、服务端状态边界，已提交筛选同步引用后才 reload；订单草稿身份和页签关闭语义可参考，不复用表单草稿存储。
- `.trellis/spec/web/frontend/component-guidelines.md`、`hook-guidelines.md`、`quality-guidelines.md`：模板请求/actionRef、稳定动作引用和定向验证。
- `.trellis/spec/web/frontend/capability-navigation.md`：已有公共搜索模板 chips/label 快照，新增跨域真实共用入口后需更新能力导航。
- `.trellis/spec/domain/architecture-map.md`：订单、客商、财务领域边界；不改变生成请求客户端和业务接口。
- `web/src/app/AppLayout.tsx:90`：以 `getOrganizationWorkspaceKey(currentUser)` 为 key 挂载 `OrganizationWorkspace` 和 Outlet，身份切换有现成整体重建边界。
- `web/src/components/layout/OrganizationWorkspace.tsx:4`：现有 key 在缺少身份时含 anonymous/no-organization，不能把该字符串当允许保留快照的认证证明。
- `web/src/components/layout/TagsView.tsx:221`：直接关闭入口；`:270` 起右键当前/其他/右/左关闭，另有关闭全部。均在 dirty 确认成功回调里真正关闭。
- `web/src/components/layout/routeUtils.ts:107`：稳定 tabKey 解析；客商三角色、订单、费用列表与同组详情共享各自页签 key，不能用当前完整 pathname 代替所属 tabKey 清理。
- `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx:138`：committed、labelSnapshot 与 labelMapRef 已区分编辑和已提交；`:177` 提交、`:184` 重置、`:193` 单 chip 删除。
- `web/src/components/ui/order-list-template/OrderListTemplate.tsx:64`：已提交条件 ref；`:567` 条件卸载筛选；`:617` 默认20分页；`:630` 官方 request。
- `web/src/components/ui/order-list-template/OrderListToolbar.tsx:64`：autoRefresh 目前只有本地状态；`:219` 文案60秒、`:223` Switch。
- `web/src/components/ui/order-list-template/OrderListSearchFilter.tsx:172`：私有 Form；`:200` Dayjs/date 范围规范化；`:256` numberType/shareStatus/isLocked 默认值。
- `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:231`：固定默认40分页，当前没有向页面开放完整搜索/分页接点；`:373` 仍由 ProTable request 驱动。
- `web/src/components/ui/searchable-select/RemoteSearchSelect.tsx:15`：私有远程 options，最终覆盖 rest.options；要恢复候选名称不能只在父层传普通 options。

### 3. 公共机制建议（签名仅为实施参考）

可新增 `web/src/components/ui/list-query-state/index.ts` 的最小导出面，内部仅 generic 类型/React Context，不能导入 pages/features。布局用该入口，模板和页面也用同一入口。通用层不得读取某个财务/客商请求，更不能反向导入领域代码。

建议数据形态：`ListQuerySnapshot<T> = { filters: T; labels: Record<string,string>; current: number; pageSize: number }`。页面特有的即时筛选（组织、标签、黑名单）可纳入具体 T，而非再造一个平行全局字段体系。字段值保留业务原始类型，boolean false、enum 0 不得通过 truthy 过滤丢失。

建议最小 API：

- `ListQueryStateProvider({ enabled, children })`：由 workspace 明确完整认证身份才 enabled，store 用实例 ref 初始化；身份缺失时 hook 只能使用本地默认值，读写共享快照均禁用。
- `useListQueryState<T>({ tabKey, listId, defaultPageSize })`：显式稳定身份；初始化读取 snapshot，给页面当前已提交值 ref/初值以及同步提交和分页记录动作。不要依赖每次 render 新建的 inline 回调去重新初始化。
- Provider 向页签层提供 `clearTabs(tabKeys)`，只清所属标签下所有列表项，不影响别的角色/页签。

Key = `tabKey + listId`（内部可嵌套 Map），Map 仅在已认证 workspace 实例内；客商 listId 含 roleType，订单 listId 含 definition.kind。不得默认 `orders:list` 让不同业务类型串用。父级已处理用户+当前组织重建，但 Provider 必须验证身份存在，不能凭匿名 key 启用。

不需要 store 订阅总线或新增平台：页面每次挂载读取一次，此后仍由自身 Form/ProTable 维护交互；事件同步更新已提交 ref 与 snapshot。若身份可能在同一组件实例内变化，优先在已存在页面/模板边界提供 `key` 重建（客商角色/订单 kind），不用复杂 watcher 在 render 中修改 store。

### 4. 关闭清理，避免卸载复活

所有清理放在 TagsView 实际关闭的确认成功回调，与 clearTabDrafts 同一位置。必须覆盖顶部叉号、右键当前、其他、右侧、左侧和全部；拒绝 dirty 确认不能清查询。普通切页签或从列表进入同页签详情不能清快照。

优先在提交、重置、即时筛选变化、分页变化事件记录，**不在组件卸载 cleanup 保存**。保存未提交输入不仅违反需求，还会在关闭删除后由旧实例 cleanup 重建快照。

为了防迟到事件/请求，Provider 可为每个 tabKey 保留 generation，hook 首次取得当前 generation 的租约；clearTabs 先删除快照并递增 generation；旧租约 write 与当前 generation 不匹配则拒绝。重新打开 hook 取得新代际。不要用 tombstone 永久禁用 tabKey，否则关闭重开无法正常记录新查询。store 不接受来自 request 完成的 filters 写回，也不接受跨身份旧实例写入；整工作区卸载则整 store 自然失效。

### 5. 每个消费者的真实接点

| 消费者 | 现状与请求接点 | 要记录与恢复 | 最薄修改位置 |
| --- | --- | --- | --- |
| 客商三个角色 | `pages/partners/index.tsx:478` searchParams；`:526` onSearch/:530 reset；`:608` request调用 partnerServiceListPartners | keyword/enabled/isCasual，页码/默认20；`:154` blacklistView 是立即生效查询，不是结果偏好；roleType是身份边界 | 同步提交 ref 替代单纯 setState+立即reload；SearchFilterTemplate恢复 Form/chips；blacklist切换也进入快照，保留现有reset语义；实际 ProTable.pagination记录 |
| 共享订单 | `OrderListTemplate.tsx:570` currentFilterRef同步；`:630` request调用queryOrders；`pages/orders/list.tsx:184`进入queryOrderList | `OrderListFilterParams`中的规范条件、页码/默认20；五类日期范围；remote labels | 搜索表单不卸载，给恢复初值/候选 label 入口；模板明确 tabKey/listId，业务调用方传 kind；筛选提交/重置记录；不能用列设置key隐式代替页签身份 |
| 费用明细 | `pages/finance/fees/index.tsx:119` filterParams；`:143`handleSearch；`:467` settlementServiceListFeeLedger | FeeLedgerFilterParams全部字段、expenseDateRange，默认40分页，顶部 organizationId/tagFilterIds及标签名 | FeeLedgerSearchFilter透传通用恢复值/labels/提交事件；财务模板开放pagination；顶部变化同步更新 ref+快照后reload，切所属公司清旧tagIDs/label；保持已有后端关键词映射 |
| 账单 | `pages/finance/bills/index.tsx:149`searchParams；`:441`提交；`:489`settlementServiceListBills | keyword/direction/status、billDateRange/dueDateRange、onlyUnsettled/onlyOverdue、settlementPartyId/currency，默认40；顶部organizationId/tagFilterIds | SearchFilterTemplate恢复；查询读已提交 ref；FinanceLedgerTemplate传pagination；保留公司切换时标签请求失效机制，恢复所选标签显示名 |
| 资金流水 | `pages/finance/cashflows/index.tsx:468`直接消费ProTable p；`:434`FinanceLedgerTemplate内置搜索 | 当前真实内置筛选 direction/status，加顶部organizationId与默认40分页 | 财务模板开放原生 form.initialValues/onSubmit/onReset及pagination透传，不在页面另建data/loading；ProTable搜索仍由官方引擎消费 |

资金流水 request 虽读取 `p.keyword`，当前列里没有 keyword 搜索字段，flowNo/settlementPartyName 均 search:false；本次不要为了“统一”新增关键字控件或扩大API范围。可针对当前实际 direction/status 写回归。

费用 keyword 目前从 orderNo/masterNo/houseNo/feeName等多个字段按优先级合并为一个后端 keyword（fees:479–490）；此行为是现存业务映射，查询恢复保存全部原始筛选字段，不能顺便重构为独立服务端过滤。

财务顶部“所属公司”是查询目标组织，不等同于 workspace 身份当前组织，必须作为查询条件恢复，不能拿来切 Provider namespace。tagIDs依赖该目标公司，切换公司必须清旧ID和labels；快照恢复时公司+tags同时初始化，避免用默认undefined先触发条件清理后丢掉标签。

### 6. 日期、远程候选与公共 SearchFilterTemplate

内存快照无需 JSON 序列化，可以保存对应 Form Dayjs 值；订单 request 协议中的日期范围仍为 YYYY-MM-DD 字符串，应由订单筛选入口只对 createdAtRange/etaRange/etdRange/lockedAtRange/statusTimeRange 还原 Dayjs。不对任意字符串值自动猜日期。

公共搜索模板目前所有模式 Form 未接 initialValues，committed 也固定为空。建议显式 props 提供 initialValues（已提交表单值）、initialLabels，以及提交/重置/chip删除时的已提交快照通知。初始 props 只在挂载时使用；不要每render setFieldsValue 覆盖未提交编辑。resetFields会回到 Form初值，恢复后“重置”必须显式清字段至真正默认查询，再清 chips/ref/store，不能又回到恢复值。

labelSnapshot是提交时抓取的名称镜像，labelMapRef是编辑候选捕获；初始化时两者同步到恢复 labels。删除chip同时清字段及labels，保留现有行为：按已提交 next触发请求，不意外把其它未提交字段带上。

RemoteSearchSelect 目前没有已选候选保活入口；仅恢复chip labels仍不足，因为输入框会显示裸ID。最小扩展为显式已选 `{value,label}` option并与远程候选渲染合并/按value去重，不改远程查询协议，也不把整个候选数据存入快照。当前RemoteSelect（订单私有实现）也需按已选ID注入label选项；当前缺失于首批空关键字候选的已选记录不应因远程结果覆盖而失去中文显示。

不建议临时改 labelInValue：所有现有请求/清洗约定是原始ID，labelInValue会扩大契约变化。标签多选可保存ID→label轻量映射，只保选中项；金额、组织对象、整份候选响应不得进快照。

### 7. 分页与官方请求协议

使用 ProTable 的 pagination.defaultCurrent/defaultPageSize恢复首次页码，每次 pagination.onChange同步记录实际current/pageSize；只需一个回调，避免 onChange/onShowSizeChange重复记录。请求本身继续仅从 ProTable params取分页，不能改成外部data/loading链。

FinanceLedgerTemplate当前固定40，应增加具体 ProTableProps['pagination'] 类型的入口，以默认40+showSizeChanger为基础合并；初值不要每render反复重设受控current以阻断官方分页行为。

提交/重置新条件时统一回第1页，保持已选pageSize；使用官方 `actionRef.reload(true)`（实施时确认安装类型）或官方setPageInfo/reload路径。普通手动/定时刷新使用reload()保留页码。snapshot页码应与实际官方分页回调一致，不能在API请求结束后推测页码。

账单/费用/客商当前多处 setState后立即reload，存在读旧closure窗口；以已提交ref为同步请求真相，setState只用来渲染，事件固化快照→ref→reload。资金流水内置表单已有官方同步搜索处理，不另造一个镜像查询引擎。

### 8. 自动刷新最薄落点

Toolbar改受控 `autoRefresh: boolean`/`onAutoRefreshChange`，真实状态与计时器放OrderListTemplate；Switch补中文可访问名。手动按钮继续actionRef.reload。

effect仅在开启且document.visibilityState为visible时建立60秒timer；visibilitychange隐藏时清timer，恢复前台重建一个timer（不额外立即请求）；关掉、卸载也清timer与监听。若验收将窗口blur也视为失去前台，可同一effect补focus/blur，不要另起第二计时器。

ProTable支持onLoadingChange，可用ref同步官方请求状态（boolean或SpinProps时注意布尔转换），tick只有!busyRef且actionRef存在才reload()，不自建data/loading/query、不修改选中行，也不Form.submit把未提交筛选生效。如果request包装处计数busy，务必finally复位并处理重叠手动请求；优先消费官方loading通知。

轮询只保持当前已提交筛选/当前页；失败仍走官方request错误处理，不能清查询。autoRefresh开关与未提交筛选不是跨页签持久化要求，默认卸载再挂载关掉，避免无依据扩大快照字段。

### 9. 安装组件实证与验证建议

本地真实 `@ant-design/pro-components` 为3.1.14-6（package声明^3.1.14-2），antd6.6.1。Table源码位于 `web/node_modules/@ant-design/pro-components/es/table/`，不要参考仓库同时安装的旧独立 `@ant-design/pro-table@3.21.0` 当作当前执行引擎。

- `typing.d.ts:237`公开onLoadingChange，`:247`polling，`:268`form，`:297`onSubmit，`:299`onReset。
- `Table.js:238`内置搜索首次等待初始化；`:249`开始已有formSearchRef同步源，防reset/reload旧条件；`:330`官方分页变化调用pagination.onChange/onShowSizeChange。
- `components/Form/FormRender.js:162`onInit从initialValues格式化后自动提交，并使用值里的current/pageSize覆盖pageInfo；恢复内置搜索时form.initialValues需与分页初始值一致，避免初始查询又回1页。

实施后再补测试：

1. Provider完整身份隔离/缺失禁用，关闭→旧租约写失败→重开新租约正常，真实路由卸载返回读快照，不以同实例rerender冒充页签恢复。
2. TagsView关闭当前/其它/左右/全部的确认与取消，取消保留，关闭删除且旧cleanup/迟到事件不复活。
3. 客商三个角色独立；boolean false不丢；黑名单与已提交搜索一起恢复。
4. 订单已提交和未提交输入收起再展开都留存；跨页签只已提交；日期转回Dayjs；恢复候选不在首批时中文名称仍正确。
5. 资金流水真实方向/状态查询与默认40分页恢复，内置搜索首次初始化/重置不回旧条件；费用账单顶部组织+tags恢复与公司切换清理。
6. 假计时器验证60秒tick/请求忙跳过/关闭与隐藏停止/恢复只有一个timer/卸载清理；浏览器65秒观察真实GET，不修改业务数据。
7. 修改文件Biome、tsc、architecture与相关定向Vitest；公共多领域集成最后check:web。公共组件测试按功能拆分，避免单suite重型渲染超过20秒。

## 外部参考

未查询外网；本次是安装版本的内部实现预研，API以仓库实际pro-components类型和源码为依据。没有引入依赖或更改node_modules。

## Caveats / Not Found

- 仅原审查浏览器实证客户关键字丢失，其它首批消费者与分页是批准的统一行为，实施后必须逐入口回归，不能把本预研静态分析当成已通过验证。
- react-router同模块角色切换是否复用现有页面实例需要测试核对；明确key是成本最低的重建边界，不依赖未经验证的路由自动卸载假设。
- 自动刷新文案称“静默”；官方reload是否显示短loading由现有ProTable决定，不为隐藏loading架空官方引擎。若视觉要求额外定义，交主会话确认实现边界。
- 订单列表部分sorter声明目前未映射给实际API（list-query.ts只映射已存在筛选）；排序恢复不在本期PRD，不顺便补服务端排序契约。
- 所有结论是只读预研；现有保存保护并发修改可能移动财务写路径行号，后续实现以提交后的实际源码为准。
