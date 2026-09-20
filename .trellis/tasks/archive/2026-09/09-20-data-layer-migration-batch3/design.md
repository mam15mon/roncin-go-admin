# 技术设计：数据层迁移第三批

设计基线沿用前两批：模式映射表、v5 红线、错误提示约定见
`state-management.md` 与归档任务 `09-20-data-layer-migration-batch2/design.md`。
本批无新基建。

## 批次 A：useMasterDataCrud → React Query

- 列表查询：`['master-data', '<entityName>', query]`，query 状态（page/
  pageSize）驱动；`placeholderData: keepPreviousData` 保持翻页不闪空。
- 统计查询：`['master-data', '<entityName>', 'stats']`（enabled true/false
  两次调用并行，保持原 Promise.all 形态或拆两条——以文案与失败语义等价为准，
  原 reloadStats 失败静默）。
- 写操作：`useMutation` ×3（create/update/toggle），onSuccess 内
  `invalidateQueries({ queryKey: ['master-data', entityName] })` 等价替代原
  `saveResponse + reload + reloadStats`；原局部 catch 文案进 mutation
  `meta.errorMessage` 或保持局部 catch（以现行为为准）。
- 返回 API 兼容：`data`（query.data ?? []）、`setData`（`queryClient.
  setQueryData` 薄封装，保留函数式更新签名）、`loading`（isFetching）、
  `reload`（refetch）、query/setQuery 原样。
- 消费方：Airlines/Airports/ShippingLines/Ports/Countries 五个面板 +
  `useMasterDataCrud.test.tsx`（用户修过无限重取语义，测试中的内联 mapItem
  场景等价保留——queryKey 模式下内联函数不再进依赖数组）。面板测试按需
  Provider 补齐。

## 批次 B：事件驱动型三处

1. `useCreditLimitIntervention.ts` → `useQuery({ queryKey: ['settlement',
   'credit-limit-policy'], queryFn, enabled, meta: { silent: true } })`；
   返回派生布尔（enabled=false 或读取失败或未激活 → false，判定逻辑
   `!== true` 注释原样保留）。grep 消费方补 Provider。
2. fees 标签筛选（`tagFilterRequestRef`）：改 `tagSearchKeyword` state +
   `useQuery({ queryKey: ['finance','fee-tag-options',{organizationId,
   keyword}], enabled: !!organizationId, meta: { silent: true } })`；
   「已选项保活合并」改渲染侧 useMemo（当前选中集合 ∪ 服务端结果）；
   原命令式 `loadTagFilterOptions(keyword, selectedIds)` 调用点改 setState
   驱动；tagFilterRequestRef 删除。
3. `ProFormSearchableSelect`：核实其 `request` 形态——若为 pro-components
   原生管理（预期如此），判定豁免不迁移，在 spec「事件驱动豁免清单」记录；
   若发现调用方存在手写 effect 链则迁移调用方。

## spec 更新

- `state-management.md` 增补「事件驱动型请求处置约定」：pro 原生 request
  豁免；命令式搜索一律防抖关键词进 queryKey；小布尔策略查询也走 useQuery
  （含 silent 约定）。

## 风险与回滚

- useMasterDataCrud 消费面广（5 面板 + 模板），测试覆盖充分
  （MasterDataTemplate.test + useMasterDataCrud.test + 治理测试），逐面板
  验证；独立提交可回滚。
