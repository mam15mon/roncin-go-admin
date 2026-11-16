# 全站服务端状态迁移静态盘点

日期：2026-11-16。依据：用户粘贴的状态管理规范及当前工作区源码。

**结论：尚未迁完。已确认 31 个查询消费文件仍有手工请求链，另有 2 个模块级缓存实现。** 按文件去重；公共组件影响的路由数量更多。

## 范围和判断口径

扫描 web/src 的 pages、features、components、hooks、app、请求与缓存基础设施，排除生成客户端、测试和 fixture 命中，并核对调用方。区分 Query、官方表格/表单 request、手工查询、显式命令和认证初始化；不能凭文件没有 import Query 就判违规。

以下为任务建立之前的原始静态盘点，保留作为逐文件范围依据；当时未运行浏览器、测试或构建。后续任务实施及验收状态以本任务文档和实际验证记录为准。

## 已确认的查询迁移缺口

| 领域 | 代码入口 | 残留实现 |
| --- | --- | --- |
| 公共 | [components/ui/searchable-select/RemoteSearchSelect.tsx:29](../../../../web/src/components/ui/searchable-select/RemoteSearchSelect.tsx:29) | options/loading + seqRef + 手工 load；挂载 Effect 与 onSearch 直接请求。SearchFilterTemplate 的远程字段复用这里，影响多类列表；这是自研请求链，不属于 ProForm 原生 request 豁免。 |
| 公共 | [pages/orders/components/PartnerQuickAddSelect.tsx:109](../../../../web/src/pages/orders/components/PartnerQuickAddSelect.tsx:109) | 客商候选由 useLatestAsync + options state 管理；应让关键词和依赖进入 Query key，渲染时合并已选及新建候选。 |
| 订单 | [pages/orders/list-resources.ts:48](../../../../web/src/pages/orders/list-resources.ts:48) | 主数据、港口、机场、客商使用 Effect + Promise.all + requestIdRef，分别保存为 state。 |
| 订单 | [pages/orders/use-fee-exchange-preview.ts:51](../../../../web/src/pages/orders/use-fee-exchange-preview.ts:51) | 费用表单汇率解析由事件直接请求，使用 exchangeRateRequestRef 和多份状态；费用页、补录弹窗共用。 |
| 订单 | [pages/orders/components/fees/OrderFeeTableTabs.tsx:680](../../../../web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:680) | 行内汇率查询使用 rateRequestSeqRef + then/catch 更新 rowPreviews。这里仅指出汇率查询缺口，表格官方 request 本身允许保留。 |
| 订单 | [pages/orders/components/detail/SameBatchOrdersSection.tsx:37](../../../../web/src/pages/orders/components/detail/SameBatchOrdersSection.tsx:37) | 同批订单使用 Effect + active + orders/loading/error。 |
| 订单 | [pages/orders/components/detail/UnlockRequestHistoryDrawer.tsx:102](../../../../web/src/pages/orders/components/detail/UnlockRequestHistoryDrawer.tsx:102) | 审批历史使用 load + state，另外 setInterval 每 5 秒轮询。 |
| 订单 | [pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx:118](../../../../web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx:118) | 历史预览、分页列表及详情三条查询均手工管理状态。 |
| 订单 | [pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx:108](../../../../web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx:108) | 拆票/改配动作能力查询使用 requestIdRef、aliveRef、changeActionsState。 |
| 订单 | [pages/orders/templates/components/sea/SeaDocumentSection.tsx:169](../../../../web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx:169) | 放货记录、无 documents prop 时的单证补读仍是手工请求序号链；还存在模式变更命令预览。 |
| 订单 | [pages/orders/templates/components/sea/SeaExternalConfirmationFields.tsx:40](../../../../web/src/pages/orders/templates/components/sea/SeaExternalConfirmationFields.tsx:40) | 确认附件候选使用 Effect + active + attachments/loading；拆票、改配、修订等多入口复用。 |
| 订单 | [pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:283](../../../../web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:283) | MBL/HBL 版本和事件历史使用手工分页、加载更多、historyGenerationRef；还存在修订/作废预览。 |
| 订单 | [pages/orders/release-pod-panel.tsx:160](../../../../web/src/pages/orders/release-pod-panel.tsx:160) | 打开抽屉时手工加载关联提单，以 activeOrderIdRef 判断采纳；记录表格的官方 request 正常。 |
| 财务 | [pages/finance/bills/components/BillEditModal.tsx:74](../../../../web/src/pages/finance/bills/components/BillEditModal.tsx:74) | 客商信用与未核销余额、币种、结算账户三个查询链均未迁移，使用 cancelled、requestSequenceRef 和本地状态。 |
| 财务 | [features/finance/bill-creation/BillGroupCard.tsx:65](../../../../web/src/features/finance/bill-creation/BillGroupCard.tsx:65) | 每个建账分组挂载时 Effect + cancelled 加载币种，保存到 state。 |
| 财务 | [pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx:47](../../../../web/src/pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx:47) | 申请详情使用 refreshToken + sequenceRef + Effect；父组件仍通过 token 递增驱动重读。 |
| 财务 | [pages/finance/commissions/my-supplement-source.tsx:23](../../../../web/src/pages/finance/commissions/my-supplement-source.tsx:23) | 调整来源页使用 requestSequenceRef + Effect 管理 source/loading/notFound。 |
| 财务 | [pages/finance/commissions/components/CommissionRulesDrawer.tsx:81](../../../../web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx:81) | 员工名单与筛选候选由命令式搜索直接写 options；不是 ProForm 原生 request。 |
| 财务 | [pages/finance/bills/index.tsx:282](../../../../web/src/pages/finance/bills/index.tsx:282) | 账单详情打开与刷新直接 GET 后 setDetail；取消前的影响范围读取是独立命令校验，不宜机械替换。 |
| 财务 | [pages/finance/commissions/index.tsx:166](../../../../web/src/pages/finance/commissions/index.tsx:166) | 提成详情打开、刷新仍直接请求，维护 detail/loading。 |
| 财务 | [pages/finance/invoices/index.tsx:167](../../../../web/src/pages/finance/invoices/index.tsx:167) | 发票详情点击时直接 GET 后 setDetail。 |
| 财务 | [pages/finance/fee-settings/components/CustomSettingsPanel.tsx:72](../../../../web/src/pages/finance/fee-settings/components/CustomSettingsPanel.tsx:72) | 信用与已建账费用编辑策略使用 Effect + load 函数管理状态，更新后手工回读。 |
| 客商 | [pages/partners/components/AuditLogSection.tsx:136](../../../../web/src/pages/partners/components/AuditLogSection.tsx:136) | 日志手工分页，使用 requestSequenceRef + logs/total/loading/error。 |
| 客商 | [pages/partners/components/ContractCardList.tsx:82](../../../../web/src/pages/partners/components/ContractCardList.tsx:82) | 合同卡片使用 Effect + fetchContracts + state，保存后手工重读；客商详情经 ContractsSection 使用。 |
| 客商 | [pages/partners/components/ShippingPresetSection.tsx:139](../../../../web/src/pages/partners/components/ShippingPresetSection.tsx:139) | 单证预设使用 Effect + fetchPresets + state，保存后手工重读；客商详情经 PresetsSection 使用。 |
| 企业资源 | [pages/enterprise-resources/index.tsx:132](../../../../web/src/pages/enterprise-resources/index.tsx:132) | 客商、负责人候选直接搜索后 setOptions；896 行 capability 也在 Effect 中直接请求。列表官方 request 正常。 |
| 企业资源 | [pages/enterprise-resources/useRegionOptions.ts:45](../../../../web/src/pages/enterprise-resources/useRegionOptions.ts:45) | 省市区三级 Effect 直接请求并 then(setOptions)，没有 Query key 隔离父级。 |
| 管理 | [pages/admin/organizations.tsx:72](../../../../web/src/pages/admin/organizations.tsx:72) | 组织树/拓扑用 loadData + Effect + state；loadData 依赖 selectedId，选择节点也触发重新取整棵树。 |
| 管理 | [pages/admin/components/NumberRulesPanel.tsx:77](../../../../web/src/pages/admin/components/NumberRulesPanel.tsx:77) | 编号规则使用 fetchRules + Effect + state，写后手工 fetchRules。 |
| 主数据 | [pages/master-data/components/CurrenciesPanel.tsx:31](../../../../web/src/pages/master-data/components/CurrenciesPanel.tsx:31) | 仍使用 fetchCurrencies + Effect + data/loading。应保留全量接口与本地分页，只迁移请求状态。 |
| 主数据 | [pages/master-data/components/CitiesPanel.tsx:42](../../../../web/src/pages/master-data/components/CitiesPanel.tsx:42) | 仍使用 fetchServerData + Effect + data/loading；全量接口与前端分页本身符合规范。 |

## 外层有 Query 也无法自动消除的模块缓存

| 入口 | 现状和影响 |
| --- | --- |
| [orderOptionsCache.ts:9](../../../../web/src/features/orders/options/orderOptionsCache.ts:9) | 四个模块级 Map 缓存字典、港口、机场、人员。正常组织切换及登录/退出已有手动 clear，不能误报为完全无隔离；但缓存生命周期仍独立于工作区 QueryClient，按组织/品类建键，需和消费方一起收敛。 |
| [currencies.ts:5](../../../../web/src/features/master-data/currencies/currencies.ts:5) | currenciesRequest 为模块级 Promise，没有用户/工作区作用域；外层 Query 失效仍可能取得原 Promise。货币维护额外靠 getCurrencies(true) 强刷，形成两套失效机制。 |

## 命令式预览和显式带入的补充入口

这些与声明式查询分开处理。有些属于显式业务命令，适合 useMutation 或当前客户端 fetchQuery；不能为消除 await/ref 而删掉表单编辑版本、幂等意图或会话保护。

| 入口 | 现状 |
| --- | --- |
| [CommissionCreateModal.tsx:573](../../../../web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:573) | 候选已用 Query；计算预览仍用 useLatestAsync + preview/loading/signature。 |
| [SeaTransportExecutionUpdateModal.tsx:179](../../../../web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx:179) | 地点和订单读取已用 Query；航程修改预览仍用 previewRequestId + previewing。 |
| [SeaBillContentFormFields.tsx:304](../../../../web/src/pages/orders/templates/components/sea/SeaBillContentFormFields.tsx:304) | 从客户、国外代理带入抬头仍直接 GET + importing；空运对应入口已用当前客户端 fetchQuery。 |
| [air-master.tsx:689](../../../../web/src/pages/orders/air-master.tsx:689) | 主单详情已用 Query；表单提交阶段的影响预览仍直接调用接口后 setImpact，loading 由 onFinish 接管，不属于手写请求 Effect。 |
| [AirTransportMaintenance.tsx:257](../../../../web/src/pages/orders/order-kinds/air-export/AirTransportMaintenance.tsx:257) | 详情已用 Query；onFinish 阶段的影响预览仍直接请求后 setImpact。 |
| [ExchangeRateSyncModal.tsx:251](../../../../web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx:251) | 点击抓取牌价用 fetching/error state 并把结果拆为 raw/draft；请求状态可由 mutation 管理，人工草稿仍保留。 |
| [ExchangeRateImportModal.tsx:91](../../../../web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx:91) | 导入预检直接请求后保存 token/batch；预检产生业务令牌，应按 mutation 语义处理，不自动后台 refetch。 |

## 同时发现的直接相关规范缺口

1. **账单基本信息编辑缺主动变动检查。** [BillEditModal.tsx:41](../../../../web/src/pages/finance/bills/components/BillEditModal.tsx:41) 与父列表没有接 useBillChangeNotice 或前台 15 秒/回焦检查。费用维护、无需发票入口已接，但不会自动覆盖该编辑弹窗。后端 expectedVersion 最终校验仍存在。
2. **失败响应守卫未全局落实。** [currencies.ts:10](../../../../web/src/features/master-data/currencies/currencies.ts:10) 的 then(unwrapList)、[useRegionOptions.ts:39](../../../../web/src/pages/enterprise-resources/useRegionOptions.ts:39)、[useMasterDataCrud.ts:101](../../../../web/src/components/ui/master-data-template/useMasterDataCrud.ts:101) 仍可能把请求层消费后的 undefined 交给 unwrapList/unwrapPage。最后一处虽已迁 Query，错误仍可能变成读取 data 的 TypeError。
3. **组织选择触发额外整树请求。** [organizations.tsx:104](../../../../web/src/pages/admin/organizations.tsx:104) 中 loadData 依赖 selectedId，Effect 又依赖 loadData；选择节点会重新请求列表。

## 已核实正常的部分和豁免

- 未发现生产代码继续使用 Umi useRequest；retry 仅在客户端工厂统一配置，未发现业务调用点覆盖。
- 公开 queryClient 单例仅由 main.tsx 引入。认证工作区按用户/工作台 key 重建 Provider，layout cleanup 清 client，并激活/释放请求作用域；未发现业务直接引入公开单例。
- 六类财务台账经 FinanceLedgerTemplate 的 onAcceptedSnapshot 更新指标；页面有 metricStats state 不等于违规镜像请求链。
- 拆票预览、改配匹配/预览、共享箱及候选、空运重量预览、工作台抽屉、订单详情基础查询及多数变动探测已用 Query。
- useMasterDataCrud 内部已用 Query，国家、港口、机场等消费者不因自身无 Query import 而判未迁；公共 Hook 的响应守卫问题另计。
- 官方 ProTable/ResponsiveProTable/SettingTableTemplate request 与 ProForm 原生 request 保留；SearchFilterTemplate 的自研 RemoteSearchSelect 则是真实缺口。
- 当前客户端 fetchQuery/prefetchQuery 属于 React Query，覆盖登录预热、建账批量预览、空运单正文带入。防抖后参数进入 queryKey 也符合规则。
- 认证身份初始化 AppProvider 在工作区 Provider 之前运行，应独立评估；不能直接套用工作区 Query 生命周期，否则可能打乱登录/身份建立顺序。
- 写入身份代际、dirty revision、保存基线、幂等键、本地 UI 状态应保留，不能按 ref/state 名字批量删除。

## 建议顺序

1. 公共入口：RemoteSearchSelect、PartnerQuickAddSelect、订单列表资源与两套模块缓存。
2. 业务输入相关：两处费用汇率、账单编辑及变动检查、提成/发票详情、企业资源级联。
3. 辅助查询：单证/审批/变更历史、客商区块、组织、编号规则、货币与行政区划。
4. 预览命令按实际语义迁移 mutation/fetchQuery，保留表单草稿、版本与幂等生命周期。

每组实现后验证 undefined/失败、乱序、换实体与 A→B→A、关闭重开、写后失效和输入保护，不以批量替换 Effect 或增加 Query import 作为完成标准。
