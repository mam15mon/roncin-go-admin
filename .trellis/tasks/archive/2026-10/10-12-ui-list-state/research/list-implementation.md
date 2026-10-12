# 列表状态统一实施记录

## 实施范围及边界

本组实现 A04/A05/A12，所有产品修改已停止，交主会话独立复核及提交。未修改后端、生成客户端、依赖、真实业务数据或任务状态；未引入 KeepAlive、持久存储、请求代理或结果缓存。前置保存保护及计息移除代码保持。

### 查询快照公共入口

新增 `web/src/components/ui/list-query-state/index.tsx`：

- `ListQuerySnapshot<T>` 保存 `filters: T`、`labels: Record<string,string>`、`current`、`pageSize`。
- `ListQueryStateProvider({enabled,children})` 用实例 ref 持有 Map；`OrganizationWorkspace` 仅在当前用户 ID 和当前组织 ID 均存在时启用。既有工作区 key 在用户/组织变化时重建 Provider，缺失身份或没有 Provider 时仅本地默认查询。
- `useListQueryState<T>({tabKey,listId,defaultPageSize,defaultFilters})` 返回 `initial`、已提交 `snapshotRef`、`commit(filters,labels?,resetPage=true)`、官方 `pagination` 初值及 onChange。事件同步更新 ref 和快照后调用官方 `reload(true)`；普通刷新继续使用官方分页。
- `useClearListQueries()` 返回按页签 key 清理的方法，删除快照并递增 generation。旧 hook 租约随后提交或分页事件无法重建已关闭快照；新挂载获取新租约。没有卸载写回或异步结果写回。
- `TagsView` 当前、其它、右侧、左侧、工作台关闭其它（全部业务页签）及叉号均只在既有 dirty 确认完成后清理目标范围；普通切换和取消确认保留。

### 筛选及自动刷新

- 共享订单列表显式接收 `queryStateIdentity: {tabKey,listId}`，业务页传订单 kind 并用 key 重建。外层搜索区域与内部更多筛选均保持挂载，仅用 hidden 隐藏；已提交查询与未提交编辑继续分离。
- 五类订单日期按既有定义从 YYYY-MM-DD 恢复 Dayjs，提交时保持字符串区间协议。恢复后重置显式清空旧初值，保留原有默认单号类型和全部状态。
- 订单私有 RemoteSelect 保存完整已选 value/label，候选更新不会把旧名称套到新 ID；默认空数组引用固定，避免 effect 重复更新。订单多选标签快照以 `tagIds:ID` 保存名称，合并到候选显示，不保存全量候选。
- 公共 SearchFilterTemplate 新增 initialValues、initialLabels、onCommit；恢复 Form、chips 和名称，重置显式清字段，chip 删除仅提交原已提交条件。公共 RemoteSearchSelect 新增 selectedOptions，并与实时候选按 ID 合并。
- 订单 toolbar 自动刷新受控，新 `useOrderAutoRefresh` 开启时每 60 秒调用 actionRef.reload；失焦、隐藏、关闭、卸载停止；忙碌时跳过，监听与 interval 卸载清理。没有第二条结果/loading 数据链。

### 消费者接入

- 客商三角色各有稳定 pathname/listId；keyword、enabled、isCasual、blacklistView 与默认 20 分页恢复。查询与导出读取已提交 ref，黑名单即时筛选同步提交后刷新。
- 订单列表恢复所有既有筛选、名称与默认 20 分页。用户/组织工作区与 kind 分开隔离。
- 费用/账单分别保存原始搜索值、目标所属公司、标签 ID 和名称，默认 40 分页；恢复时公司和标签同时初始化，切公司同步清旧标签/名称。既有费用关键词优先级及时间字段转换保留。
- 资金流水保留真实 direction/status 内置表单，不新增 keyword 控件。通过 FinanceLedgerTemplate 暴露官方 pagination、formRef、form、onSubmit、onReset；form.initialValues 包含恢复 current/pageSize，避免官方 onInit 回默认 1/40。reset 用 formRef 清字段且快照清条件，request 从已提交快照读取实际筛选。
- 财务模板仍完全使用官方 ProTable request/actionRef；仅透传原生生命周期及分页配置，不缓存结果。

## 修改文件

产品文件共 20 个（18 个既有文件及 2 个新增公共机制）：

- `components/layout/OrganizationWorkspace.tsx`、`TagsView.tsx`。
- `components/ui/list-query-state/index.tsx`（新增）。
- `components/ui/order-list-template/OrderListSearchFilter.tsx`、`OrderListTemplate.tsx`、`OrderListToolbar.tsx`、`types.ts`、`useOrderAutoRefresh.ts`（新增）。
- `components/ui/search-filter-template/SearchFilterTemplate.tsx`、`types.ts`。
- `components/ui/searchable-select/RemoteSearchSelect.tsx`、`types.ts`。
- `components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx`、`types.ts`。
- `pages/orders/list.tsx`、`pages/partners/index.tsx`。
- `pages/finance/fees/index.tsx`、`components/FeeLedgerSearchFilter.tsx`，`bills/index.tsx`、`cashflows/index.tsx`。

针对性回归新增 SearchFilterTemplate.restore、OrderListSearchFilter.restore、TagsView.query-state 和三财务消费者 query-state；既有 OrderListTemplate.test 适配已提交名称第二参数。公共 listQueryState/useOrderAutoRefresh 两 suite 由独立 checker 完善，未并行改写其测试。

## 验证结果

实现后按风险补测试，没有 TDD。定向验证共 **17 个 suite、88 个用例通过**：

- 核心 9 suite / 56 例通过：真实 TagsView 既有行为、新关闭清理矩阵、OrderListTemplate、Provider 路由卸载返回与角色/工作区/代际、自动刷新假计时器、公共搜索恢复重置、三财务请求参数/分页/标签/所属公司恢复。
- OrderListSearchFilter.restore 独立 2 例最后重跑通过：五日期及候选外标签恢复、收展和未提交值保留、重置不回旧初值；改选客户乙后候选缺失仍显示乙且提交 ID/名称一致（suite 4.96 秒，整体 7.63 秒）。早先失败为弹层残留造成重复文本定位，已限定实际输入的选中显示元素。
- 相邻 7 suite / 30 例通过：SearchFilterTemplate、RemoteSearchSelect、FinanceLedgerTemplate、客商列表、FeeLedgerSearchFilter、账单与资金流水保存保护（整体 18.13 秒）。
- 修改文件 Biome check --write 27 文件通过；最后三个修改文件再次 check 无改动。未格式化 checker 独占的两个 suite。
- 最终 `pnpm --dir web tsc` 通过。
- 最终 `pnpm run check:architecture` 18 检查通过，417 产品文件、1531 本地依赖，没有边界违规。
- 订单筛选修改文件 `antd lint --format json` 无问题（0 条）。
- `git diff --check` 通过。

主会话真实浏览器七项全部通过，详见 `research/browser-verification.json`：客户/费用/账单关键词往返，资金流水 direction 及第 2 页/20 条恢复，订单未提交收展与已提交刷新/跨页签恢复，客户关闭重开默认；自动刷新开启 65 秒实际 1 次 GET，关闭 62 秒 0 次 GET。均只读真实接口或仅 GET fixture，没有改真实业务数据。

## 交接与限制

无需重复上述耗时浏览器轮询。最后订单名称修复已由真实 Select 回归验证，如独立浏览器抽查，可恢复客户甲后改选客户乙、再让空关键词候选不含乙，确认当前输入和提交请求均为乙；订单标签初批不含已选标签时应显示已保存中文名称。

整页浏览器刷新清空内存快照，是批准设计边界；只保已提交条件，跨业务页签不恢复未提交编辑。完整前端门禁留给主会话最终批准阶段执行，本组未重复全量 check:web 或后端/build。
