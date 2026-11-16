# 公共查询与候选实现记录

## 公共协议与缓存组

已提交 `dee1312a`（主会话提交）。30 个精确路径保存于 `/tmp/foundation-files.txt`，内容覆盖：

- `RemoteSearchSelect`、公开候选类型和 `SearchFilterTemplate`：资源 key + 防抖关键词 Query；回调 ref 稳定、已选结果从当前资源查询缓存渲染保活。模板 `requestKey` 表达同挂载内的真实依赖变化。
- 订单 options 四套模块 Map 删除，薄取数签名保留；登录/退出/工作区切换/新建重试不再调用手工 clear；所有过时测试同步。
- 货币模块 Promise/forceRefresh 删除。新增 `CURRENCY_QUERY_PREFIX`、`currencyQueryOptions(enabledOnly=true)` 和 `currencyOptionsQueryOptions()`，纯 `getCurrencies/getCurrencyOptions` 不缓存。
- `CurrenciesPanel` 与 `CitiesPanel` 迁 Query；货币启停同时失效直接前缀和明确 `meta.currencyDependent` 聚合。
- `ensureListResponse` 保留原信封子类型；拒绝 undefined、success:false、非法 data、无成功标志且缺省 data，接受数组或 success:true 的省略 repeated。`useMasterDataCrud` 的列表与统计均先守卫再读字段。

验证：10 个文件、74 例通过（公共 API、两个 options feature、远程下拉、主数据 Hook、货币页、订单新建 Hook、搜索模板本体/恢复/筛选设置）。登录/退出/组织切换相关回归另批通过。Biome、diff 检查通过；该阶段无本组 TypeScript 错误。

## 订单列表与客商快建组

已提交 `4e5ab166`（主会话提交）。精确 7 个路径：

1. `web/src/pages/orders/list-resources.ts`
2. `web/src/pages/orders/list-resources.test.ts`
3. `web/src/pages/orders/components/PartnerQuickAddSelect.tsx`
4. `web/src/pages/orders/components/PartnerQuickAddSelect.fixtures.tsx`
5. `web/src/pages/orders/components/PartnerQuickAddSelect.disabled.test.tsx`
6. `web/src/pages/orders/components/PartnerQuickAddSelect.failure.test.tsx`
7. `web/src/pages/orders/components/PartnerQuickAddSelect.test.tsx`

主聚合 Query 替换本地远端状态；原生 ProForm 的公开搜索 Promise 使用当前客户端 fetchQuery，组织/运输方式/业务类型均参与身份和迟到返回检查，候选名称与港口补入对应 Query 缓存，不再 set 本地远端 state。未开放类型及缺组织保持全部入口零请求。

客商快建查询用防抖关键词 Query，渲染合并当前结果/已选/新建缓存；创建仍由 useAsyncGuard 保护。失败信封含 ID 及缺 ID 均不回填、不关闭，原输入保留。

验证命令：

```bash
pnpm --dir web exec vitest run src/pages/orders/list-resources.test.ts src/pages/orders/components/PartnerQuickAddSelect.test.tsx src/pages/orders/components/PartnerQuickAddSelect.create.test.tsx src/pages/orders/components/PartnerQuickAddSelect.isolation.test.tsx src/pages/orders/components/PartnerQuickAddSelect.disabled.test.tsx src/pages/orders/components/PartnerQuickAddSelect.failure.test.tsx src/pages/orders/components/PartnerQuickAddSelect.pinyin.test.tsx
```

结果：7 文件 35 例通过，无 stderr，最长单文件约 10 秒。TypeScript、Biome、diff 及架构门禁通过；架构当时扫描 589 个产品文件、2410 条本地依赖。

## 追加币种消费同步与真实 Query 专项

授权依据：主会话确认 `FeeItemsPanel` 与 `ExchangeRatesPanel` 的候选 state 属于删除旧货币缓存后的必要调用方同步。

精确 7 个路径：

1. `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx`
2. `web/src/pages/finance/fee-settings/components/FeeItemsPanel.queries.test.tsx`
3. `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx`
4. `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.currencies.test.tsx`
5. `web/src/pages/master-data/components/CurrenciesPanel.invalidation.test.tsx`
6. `web/src/components/ui/search-filter-template/SearchFilterTemplate.request-key.test.tsx`
7. `web/src/components/ui/searchable-select/RemoteSearchSelect.test.tsx`

费用设置的单位/应税服务/异常类型独立 Query；币种复用统一 Query，表格 request 仅取列表且校验信封。汇率设置删除 currencies state 和「仅初次为空时请求」分支，币种展示/表单候选消费共享 Query；列表官方 request 保留。

专项验证涵盖：表格翻页不重复取辅助候选，币种失效独立更新费用候选和汇率名称，币种启停同时刷新直接消费者和标记聚合且不刷新无关 Query；同一筛选模板 enabled false→true→false 能重取/隐藏当前候选；旧关键词缓存仅用于已选项名称保活。

```bash
pnpm --dir web exec vitest run src/pages/finance/fee-settings/components/FeeItemsPanel.queries.test.tsx src/pages/finance/exchange-rates/components/ExchangeRatesPanel.currencies.test.tsx src/components/ui/searchable-select/RemoteSearchSelect.test.tsx src/components/ui/search-filter-template/SearchFilterTemplate.request-key.test.tsx src/pages/master-data/components/CurrenciesPanel.invalidation.test.tsx
```

结果：5 文件 12 例通过，7 个修改文件的 Biome 和 diff 检查通过。最新 TypeScript 临时仅报其他代理正在修改的 `SeaOrderChangeHistoryDrawer` data 推导问题，已通知该组，不修改其文件；最终完整类型门禁由主会话统一执行。

## 交叉 review 待处理证据

静态发现：`CurrenciesPanel.handleToggleActive` 仍沿用原 `Promise<void>`，无效写信封时返回 undefined，catch 后同样返回 undefined；`MasterDataTemplate` 把非 false 当作成功，因此失败后 AsyncPopconfirm 会关闭。已报告主会话，建议返回 `false` 并补失败保持弹层验证。按主会话要求未擅自扩写产品文件。
