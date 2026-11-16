# 公共与基础组最终审查记录

审查时间：2026-11-16。职责范围为公共候选协议、订单候选取数与缓存消费、货币/行政区划 Query 的失败及取消行为。所有修改均位于本任务隔离工作区；没有写源工作区、提交或回退其他代理修改。

## 已修复发现

1. `CurrenciesPanel` 的启停处理原为 `Promise<void>`，失败信封、异常和本位币禁止停用都返回 undefined，公共模板会当作成功关闭确认。失败明确返回 false；只有真实成功才接受关闭和后续失效。真实 AsyncPopconfirm 回归覆盖 undefined、缺实体标识、失败信封携带实体及抛错，验证保留原操作并可重试。
2. `CurrenciesPanel` 和 `CitiesPanel` 首次读取失败原本仍会把 `query.data ?? []` 交给正常页面，显示合法空统计。首次失败改为就地 Alert 与重试；后台刷新失败保留旧数据显示持续提示。Query 使用 silent，读取已使用 skipErrorHandler，避免重复反馈；合法成功且省略 repeated 的空列表仍正常显示。
3. `RemoteSearchSelect` 与客商快建的 Query 取消原本未传给真实请求，旧关键词可能继续由请求默认错误处理器提示。公开请求签名传递 signal 与 skipErrorHandler，真实失败统一由中文 Query meta 反馈。`SearchFilterTemplate` 同组透传 signal；订单列表主聚合、显式搜索、客商快建及 new/detail 包装层同步传到薄取数函数和生成客户端。
4. 订单 options 四个薄函数、客商/船公司搜索增加独立 RequestOptions 参数；客商/船公司列表在 unwrapList 前验证信封，缺失响应产生中文领域错误而非 data TypeError。货币新增显式 fetchCurrencyOptions(RequestOptions) 入口，零参数 getCurrencyOptions 继续供原生 ProForm request 使用，避免把框架参数误当传输选项。Query 缓存仍仅属于工作区 QueryClient。
5. `CitiesPanel.mapRegion` 仍把非 optional `AdministrativeRegion.enabled` 的缺省值用 `?? true` 映射为启用；已核对 `masterdata.proto` 中该字段为非 optional bool，改为 Boolean(item.enabled)，补真实缺 key、显式 false、显式 true 三种 wire 回归。此修复限当前已迁移页面，未扩大到全仓映射改写。

## 精确可提交路径

以下为本专项审查实际修改的 30 个代码/测试路径，另包含本审查记录。已实现但早于本轮审查的其他路径见 foundation-implementation.md，不据此清单整体暂存其他代理文件。

```text
web/src/components/ui/search-filter-template/SearchFilterTemplate.test.tsx
web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx
web/src/components/ui/search-filter-template/types.ts
web/src/components/ui/searchable-select/RemoteSearchSelect.test.tsx
web/src/components/ui/searchable-select/RemoteSearchSelect.tsx
web/src/components/ui/searchable-select/types.ts
web/src/features/master-data/currencies/currencies.test.ts
web/src/features/master-data/currencies/currencies.ts
web/src/features/master-data/currencies/index.ts
web/src/features/master-data/shipping-lines/shippingLineOptions.test.ts
web/src/features/master-data/shipping-lines/shippingLineOptions.ts
web/src/features/orders/options/orderOptionsCache.test.ts
web/src/features/orders/options/orderOptionsCache.ts
web/src/features/partners/partnerOptions.test.ts
web/src/features/partners/partnerOptions.ts
web/src/pages/master-data/components/CitiesPanel.tsx
web/src/pages/master-data/components/CitiesPanel.query.test.tsx
web/src/pages/master-data/components/CurrenciesPanel.tsx
web/src/pages/master-data/components/CurrenciesPanel.failure.test.tsx
web/src/pages/master-data/components/MasterDataQuery.failure.test.tsx
web/src/pages/orders/common.test.ts
web/src/pages/orders/common.ts
web/src/pages/orders/detail.tsx
web/src/pages/orders/list-resources.test.ts
web/src/pages/orders/list-resources.ts
web/src/pages/orders/new.tsx
web/src/pages/orders/components/PartnerQuickAddSelect.tsx
web/src/pages/orders/components/PartnerQuickAddSelect.pinyin.test.tsx
web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx
web/src/pages/orders/templates/types.ts
```

## 未修复发现与交接

本专项没有遗留未修复发现。`FeeLedgerSearchFilter`、账单列表和 `ReportShell` 的远程筛选实际消费者由主会话同步 signal；本代理未修改这些路径。state-management/hook-guidelines/capability-navigation 的规范同步及完整前端门禁由主会话统一整合，不在这里宣称全任务完成。

## 验证

- 本轮此前公共协议、取数助手和列表/搜索回归：12 文件、58 例通过；原日志 `/tmp/review-core-tests-third.log`。
- 最后主数据回归：4 文件、10 例通过；日志 `/tmp/review-core-tests-final.log`。命令：

```bash
pnpm --dir web exec vitest run src/pages/master-data/components/CitiesPanel.query.test.tsx src/pages/master-data/components/MasterDataQuery.failure.test.tsx src/pages/master-data/components/CurrenciesPanel.failure.test.tsx src/pages/master-data/components/CurrenciesPanel.invalidation.test.tsx
```

- 旧订单消费方回归：4 文件、54 例通过；日志 `/tmp/review-core-consumer-tests.log`。命令：

```bash
pnpm --dir web exec vitest run src/pages/orders/new.test.tsx src/pages/orders/detail-personnel.test.tsx src/pages/orders/use-order-detail-data.test.ts src/pages/orders/use-order-fee-options.test.ts
```

- 上述 30 个修改文件的 `pnpm --dir web exec biome check` 通过，未关闭规则；日志 `/tmp/review-core-biome-final.log`。
- `pnpm --dir web tsc` 通过；日志 `/tmp/review-core-tsc-final.log`。
- `pnpm run check:architecture` 通过：18 个脚本测试，589 个产品文件、2421 条本地依赖；日志 `/tmp/review-core-architecture-final.log`。
- `git diff --check` 通过。
- 本代理没有执行浏览器验收、后端全量或构建；本专项无后端或构建配置修改。全任务完整 `check:web` 由主会话执行。
