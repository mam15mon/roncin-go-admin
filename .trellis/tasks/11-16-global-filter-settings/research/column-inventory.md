# 研究：全站表头与表格展示设置盘点

- 研究问题： 表头的全部设置如何跨浏览器保存，公共入口覆盖到哪里，是否有独立配置和漏项。
- 研究范围：仓库内部，源码静态盘点及关键实现精读；未运行产品、测试或数据库。
- 日期： 2026-11-16

## 研究发现

### 结论与本期边界建议

本期不能只把 `ColumnSettingsPreference` 的 `order/hidden/freeze` 搬到服务端。已经存在的用户列宽操作、费用专有列配置、费用台账行配色也要纳入账号保存。行密度也是现成的用户展示设置（当前实例状态），建议同样纳入并在最终方案明示。全屏、刷新、排序查询、分页当前页、勾选及业务表单草稿是临时动作或业务状态，不能因“所有设置”自动扩展为跨设备同步。

现有大量表头通过公共 hook 自动覆盖；最近新增的一些业务嵌套表没有列设置入口。用户明确的是所有筛选区有设置，以及表头所有设置保存，不应擅自把本期扩大为所有无入口表格新增显隐/冻结功能。下文列出缺口，主会话应在方案中明确：至少保存这些表已有的调宽能力；新增表头入口按现行全站规范及本期范围统一判断。订单审核完整费用核对表有明确列偏好例外，保持固定全列，不将个人显隐应用到审核表。

### 展示维度与持久化实际

| 维度 | 真实代码依据 | 当前保存方式 | 本期建议 |
| --- | --- | --- | --- |
| 列显隐、顺序、左右逐列冻结 | `web/src/components/ui/column-settings/types.ts:16`；`useColumnSettings.tsx:267`、`:328` | 本浏览器 localStorage，按 tableKey+用户+组织 | 服务端正式来源，保留字段 key、必显和结构约束 |
| 恢复默认 | `useColumnSettings.tsx:340` | 写入业务默认值，非另一个可配置选项 | 服务端恢复默认，确保另一浏览器恢复业务默认 |
| 订单费用专有配置 | `web/src/pages/orders/components/fees/feeColumnPreference.ts:149`、`:260`、`:274`；`OrderFeeTableTabs.tsx:372`、`:400`、`:455` | 独立 localStorage，用户+组织，应收/应付共享，跨订单共享 | 同步服务端，保持共享语义，不能只改公共 hook |
| 拖拽列宽 | `web/src/components/ui/responsive-table/useResizableColumns.tsx:31`、`:57` | 当前组件 useState；刷新失效 | 稳定表格 key+字段身份保存；先给未有 tableKey 的响应式表补稳定标识 |
| 双击展开／还原宽度 | 同文件 `:64`，`ResizableHeaderCell.tsx` | 与 widths 同一实例状态，无持久化 | 与拖宽保存同一宽度；保留原始业务默认基线，不把服务端保存宽度误作双击初始宽度 |
| 费用台账行配色（更多设置） | `web/src/pages/finance/fees/index.tsx:198`、`:346`；`feeLedgerViewPreference.ts:29`、`:45` | 独立 localStorage，用户+当前组织，7类状态颜色 | 作为 finance-fees:ledger 的配套展示配置随账号保存；恢复默认一并验证 |
| 行密度 | `FinanceLedgerTemplate.tsx:130`、`:246`；多个页面 options.density:true | 财务模板 useState，其余由 ProTable 实例管理，未发现保存 | 建议本期纳入 table 展示偏好，用官方 size/onSizeChange 接口；不持久化单纯 CSS 媒体查询结果 |
| 全屏、刷新 | `FinanceLedgerTemplate.tsx:386` 等 options | 临时动作 | 不保存 |
| 响应式窄屏临时解冻 | `ResponsiveTable.tsx:90`、`:95` | 实时容器计算 | 不写回偏好，宽屏恢复原冻结 |

### 公共 hook、响应式层与原生 ProTable 状态的边界

- `useColumnSettings.tsx:201` 返回 `{columns, entry}`，没有向响应式层暴露 tableKey/宽度/密度保存句柄。单独迁移 localStorage 并不能让 `ResponsiveTable` 知道保存位置，需要明确公共适配协议。
- `ResponsiveTable.tsx:48` 调用 `useResizableColumns(props.columns, props.components)`，当前无任何持久化 key；三种 ResponsiveTable/ResponsiveProTable/ResponsiveEditableProTable 共用。
- 原生 ProTable `columnsState` 仅出现在 `ResponsiveTable.tsx:151` 和 `:178`：默认受控空 map 防止旧 fixed 缓存覆盖新列，没发现页面自行传 columnsState/persistenceKey；不能恢复原生持久化作为第二真相源。
- `useColumnSettings.tsx:89` 以 key/dataIndex 建立稳定身份，结构列由 standard key、valueType 或显式 structuralKeys 判断，不按中文名称猜；`applyColumnSettings:141` 保留结构锚点；至少保留一普通列，lockVisible 不可隐藏。
- 现有 hook 的 deriveFields/applyColumnSettings 实现主要处理顶层数组；调宽层对 children 递归。若本期碰到分组头，必须核对实际结构，不能假定两层已经共享同一身份编码；复合 dataIndex 在 hook 内 join('-')、调宽 JSON.stringify 编码也存在差别，服务端映射需统一或显式转换，避免同字段两份宽度。
- `useResizableColumns.tsx:39` 测量同表叶子与辅助列快照，拖动时会顺带记录其他列实际宽度和 @selection/@expand；持久化时要区分用户选择与本次几何快照，避免把窄屏临时分配结果当成用户全局选择。
- 宽度范围 50–600px，序号/选择/展开/分组头无用户拖拽手柄，普通业务叶子及操作列可拖。用户调宽不应触发业务 request、清空选择或重挂编辑内容。
- `ColumnSettingsEntry` 的面板/高级弹窗会话失效与 fullscreen 挂载点行为已有规范，迁移异步服务端保存时继续防止旧面板回调写到切换后的身份。
- 在 `server/api/**/*.proto` 搜索 column/config/preference/setting 未发现服务端列配置契约。当前“财务增强版保持原契约”指前端高级内容自行存取，不代表已有服务端账户设置接口。

### 共享模板扩散范围

| 模板 | 保存入口锚点 | 扩散范围及实例 key |
| --- | --- | --- |
| MasterDataTemplate | `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:457` | Countries、Cities、Currencies、Airlines、Airports、Ports、ShippingLines、ShippingAgents；缺省 `master-data:${title}` |
| SettingTableTemplate | `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx:86` | 管理异常档案、财务费用项目/计费单位/税务服务/费用模板；缺省 `setting:${entityName}` |
| SubEntityDrawerTemplate | `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx:197` | 订单货物、集装箱抽屉；缺省 `orders:sub-entity:${entityName}` |
| OrderListTemplate | `web/src/components/ui/order-list-template/OrderListTemplate.tsx:699`；`pages/orders/list.tsx:243` | 四品类订单列表，`orders:${definition.kind}`，保持品类隔离 |
| FinanceLedgerTemplate | `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:133` | 六类台账 fees/bills/cashflows/invoices/verifications/nettings 与四类报表；有 columnSettingsKey 才把公共 columns/entry 接入，缺省占位 unset 不可作为真实保存区域 |

以下直接消费者清单来自非测试 .tsx 中 `useColumnSettings<...>(` 或 `useColumnSettings(` 调用的逐条静态索引。表格锚点是调用处，不把导入行计作实例；动态 key 行需沿调用方读取。公共 hook 的真实 DOM 展示由现有返回值与消费模板代码确认，未逐页面运行交互验收。

### 公共列设置调用点清单

| 文件及调用行 | tableKey 证据（调用附近） |
| --- | --- |
| `web/src/pages/admin/audit.tsx:110` | `'admin:audit'` |
| `web/src/pages/admin/background-tasks.tsx:266` | `'admin:background-tasks'` |
| `web/src/pages/admin/components/NumberRulesPanel.tsx:351` | `'admin:number-rules'` |
| `web/src/pages/admin/components/org/OrgDetailCard.tsx:125` | `'admin:org-detail'` |
| `web/src/pages/admin/components/org/OrgInspectorPanel.tsx:101` | `'admin:org-inspector'` |
| `web/src/pages/admin/components/users/UserFormModal.tsx:205` | `'admin:user-memberships'` |
| `web/src/pages/admin/permissions.tsx:90` | `'admin:permissions'` |
| `web/src/pages/admin/roles.tsx:292` | `'admin:roles'` |
| `web/src/pages/admin/users.tsx:111` | `'admin:users'` |
| `web/src/pages/enterprise-resources/index.tsx:520` | ``enterprise-resources:list:${activeTab}`` |
| `web/src/pages/finance/bills/components/BillDetailDrawer.tsx:105` | `'finance:bill-lines'` |
| `web/src/pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx:137` | `'finance:commission-application-lines'` |
| `web/src/pages/finance/commissions/components/CommissionApplicationsPanel.tsx:265` | `'finance:commission-applications'` |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:267` | `'finance:commission-candidate-nettings'` |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:274` | `'finance:commission-candidate-lines'` |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx:195` | `'finance:commission-lines'` |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx:202` | `'finance:commission-adjustments'` |
| `web/src/pages/finance/commissions/components/CommissionRuleRosterModal.tsx:94` | `'finance:commission-rule-roster'` |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx:400` | `'finance:commission-rules'` |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx:251` | `'finance:commission-pending-decrease'` |
| `web/src/pages/finance/commissions/index.tsx:674` | `'finance:commission-list'` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx:235` | `'finance:exchange-import-preview'` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx:690` | `'finance:exchange-sync-preview'` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx:245` | `'finance:exchange-rates'` |
| `web/src/pages/finance/fees/detail.tsx:77` | `'finance:fee-detail-lines'` |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx:91` | `'finance:invoice-lines'` |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx:98` | `'finance:invoice-records'` |
| `web/src/pages/finance/nettings/index.tsx:468` | `'finance:netting-allocation-list'` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:576` | `'finance:verification-cashflow-candidates'` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:582` | `'finance:verification-bill-candidates'` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:586` | `'finance:verification-allocations'` |
| `web/src/pages/finance/verifications/index.tsx:410` | `'finance:verification-allocation-list'` |
| `web/src/pages/orders/abnormal-case-panel.tsx:244` | `'orders:abnormal-cases'` |
| `web/src/pages/orders/components/detail/SameBatchOrdersSection.tsx:118` | `'orders:same-batch'` |
| `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx:260` | `'orders:sub-entity:附件'` |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx:68` | `'orders:consolidation-summary'` |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx:74` | `'orders:consolidation-members'` |
| `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx:306` | `'orders:change-history'` |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx:457` | `'orders:reassignment-voyage-difference'` |
| `web/src/pages/orders/components/drawers/SeaSharedContainerAllocationTable.tsx:156` | `'orders:shared-container-allocation'` |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx:317` | `'orders:voyage-difference'` |
| `web/src/pages/orders/components/drawers/ShippingDocumentDrawer.tsx:275` | `'orders:shipping-documents'` |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx:106` | `'orders:split-containers'` |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx:553` | `动态参数／见调用方` |
| `web/src/pages/orders/components/split/SplitAttachmentsAndNotesSection.tsx:93` | `'orders:split-attachments'` |
| `web/src/pages/orders/components/split/SplitFeesSection.tsx:89` | `'orders:split-fees'` |
| `web/src/pages/orders/release-pod-panel.tsx:369` | `'orders:release-pod'` |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx:60` | `'orders:doc-field-differences'` |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx:82` | `'orders:doc-downstream-impacts'` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:124` | `'orders:doc-field-differences'` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:146` | `'orders:doc-downstream-impacts'` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:496` | `'orders:doc-versions'` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:528` | `'orders:doc-events'` |
| `web/src/pages/partners/components/PartnerExcelImportModal.tsx:343` | `'partners:excel-import-preview'` |
| `web/src/pages/partners/components/secondary/AccountsPanel.tsx:153` | `'partners:accounts'` |
| `web/src/pages/partners/components/secondary/AttachmentsPanel.tsx:94` | `'partners:attachments'` |
| `web/src/pages/partners/components/secondary/ContractsPanel.tsx:138` | `'partners:contracts'` |
| `web/src/pages/partners/components/secondary/InvoiceProfilesPanel.tsx:121` | `'partners:invoice-profiles'` |
| `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx:293` | `'partners:settlement-rules'` |
| `web/src/pages/partners/index.tsx:500` | ``partners:list:${currentView.roleType}${` |
| `web/src/pages/workbench/MyApplicationCandidatesDrawer.tsx:137` | `'workbench:application-candidates'` |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx:251` | `'workbench:application-history-lines'` |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx:324` | `'workbench:application-history'` |
| `web/src/pages/workbench/MyCommissionDrawer.tsx:172` | `'workbench:my-commissions'` |
| `web/src/pages/workbench/MyCommissionDrawer.tsx:303` | `'workbench:my-adjustments'` |
| `web/src/pages/workbench/MyReceivablesDrawer.tsx:120` | `'workbench:my-receivables'` |
| `web/src/pages/workbench/MyTodosDrawer.tsx:210` | `动态参数／见调用方` |
| `web/src/features/finance/bill-creation/BillCandidateSelectionStep.tsx:35` | `'finance:bill-candidate-fees'` |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx:101` | `'finance:bill-creation-group-results'` |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx:139` | `'finance:bill-creation-nettings'` |
| `web/src/features/finance/bill-creation/BillGroupCard.tsx:79` | `'finance:bill-group-fees'` |
| `web/src/features/finance/bill-creation/NettingPairsCard.tsx:88` | `'finance:bill-netting-pairs'` |
| `web/src/features/orders/fee-correction/FeeCorrectionSection.tsx:321` | `'orders:fee-corrections'` |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:133` | `columnSettingsKey ?? 'finance-ledger:unset'` |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:457` | `columnSettingsKey ?? `master-data:${title}`` |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx:699` | `columnSettingsKey` |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx:86` | `columnSettingsKey ?? `setting:${entityName}`` |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx:197` | `columnSettingsKey ?? `orders:sub-entity:${entityName}`` |

共索引 78 个公共 hook 调用点（含共享模板，每个模板可扩散多个业务实例，不等于用户界面表格总数）。

动态实例补充：`MyTodosDrawer.tsx:323` / `:333` 分别为 `workbench:my-unbilled-fees`、`workbench:my-open-abnormal-cases`；`SplitAllocationSection.tsx:258` / `:420` 分别为 `orders:split-allocations`、`orders:split-result`。同 `SeaCreateDocumentModeField` 与 `SeaDocumentHistoryActions` 共用 differences/impacts key 是现有业务视图共享，不能按组件实例随机拆 key。

### 未直接接入公共列设置的表格

下表是按 `<Table/ProTable/EditableProTable/Responsive*>` 实例提取、且文件不含 `useColumnSettings/ColumnSettingsEntry` 的静态候选，再精读主要新增财务表、海空新增表和审核例外。它不是“本期必须给所有表新增齿轮”的结论，也不包含一文件已用 hook 但某个嵌套表可能漏用的情况。所有导入别名 Table/ProTable 需按实际文件解释；本批多处 Table 是 ResponsiveTable 别名，已有拖宽能力但无稳定持久化区域。

| 文件及表格行 | 判断／后续 |
| --- | --- |
| `web/src/pages/admin/components/users/DingTalkDirectorySyncPanel.tsx:268` | 同步失败详情，无入口；实际为 ResponsiveTable 别名。 |
| `web/src/pages/finance/bills/components/BillFeeMaintenance.tsx:245, :283` | 账单当前费用与可搜索新增候选，均无列入口；候选还要接筛选设置。 |
| `web/src/pages/finance/bills/components/BillInvoiceExemptionSection.tsx:51` | 免票历史表，无入口。 |
| `web/src/pages/finance/bills/components/BillInvoiceRegistrationSection.tsx:54` | 账单内发票记录，options=false 无入口。 |
| `web/src/pages/finance/commissions/components/CommissionLineTable.tsx:87` | 提成费用构成展开快照，无入口。 |
| `web/src/pages/finance/commissions/components/CommissionPaymentModal.tsx:140` | 发放前调整清单，无入口。 |
| `web/src/pages/finance/reports/ProfitChart.tsx:131` | 图下月份数值表，无入口。 |
| `web/src/pages/finance/reports/aging.tsx:82` | 完整筛选汇总分桶表，主列表已由 ReportShell 接入。 |
| `web/src/pages/finance/reports/cashflows.tsx:56` | 完整筛选汇总币种表，主列表已接入。 |
| `web/src/pages/finance/reports/profit.tsx:91` | 完整筛选汇总金额表，主列表已接入。 |
| `web/src/pages/orders/air-master.tsx:482, :580` | 主单活动成员／历史 options=false，无入口。 |
| `web/src/pages/orders/components/detail/OrderLockFeeDetails.tsx:127` | 规范明确例外：完整核对费用，不读个人显隐列偏好。 |
| `web/src/pages/orders/import-handling/ImportHandlingPanel.tsx:91, :515` | 进口服务步骤表、变更历史表，无入口。 |
| `web/src/pages/orders/order-kinds/air-export/AirExportDetailFeatures.tsx:126` | 空运分单版本，无入口。 |
| `web/src/pages/orders/order-kinds/air-export/AirWeightPanel.tsx:285` | 重量确认历史，无入口。 |
| `web/src/features/finance/invoice-registration/InvoiceAllocationDetails.tsx:21` | 发票费用分配追溯，无入口。 |
| `web/src/features/finance/invoice-registration/InvoiceRegistrationModal.tsx:459` | 独立发票登记候选账单，官方搜索、options=false，无入口。 |
| `web/src/features/finance/invoice-registration/InvoiceRevisionDiff.tsx:141` | 修订对照三列表，无入口。 |
| `web/src/features/finance/invoice-registration/InvoiceRevisionHistory.tsx:144` | 修订历史内费用快照表，无入口。 |

### 分阶段实施与验证建议

1. 先统一服务端账号设置边界及前端异步状态源；公共 columns、订单费用专有 columns、列宽、密度与高级行配色各有明确区域 key，不能保留 localStorage 作为回退或双写。
2. 公共 hook 消费者优先全覆盖；共享模板把持久化上下文传给响应式层，页面不再各维护一套宽度。
3. 将 `feeColumnPreference` 的业务字段定义/权限裁剪保留，只替换存取；应收应付原有共享配置且各自编辑输入不丢。当前 `OrderFeeTableTabs.tsx` 有其他会话未提交修改，实施协调并保留。
4. 给已能调宽而未持有 tableKey 的表明确稳定区域标识；不使用订单 ID、数组位置、中文标题或临时 DOM 索引作为账号偏好身份。
5. 验证两浏览器上下文相同账号相同组织同表恢复显隐/顺序/冻结/宽度/密度/行配色；同账号切组织、不同账号隔离；权限裁剪、必显列、结构锚点、冻结空数组、隐藏重显、恢复默认及服务端保存失败。
6. 测量真实浏览器固定列、少列表铺满、390px 窄屏到宽屏、Pro 全屏和 Editable 输入保护；不可只用 col style 断言通过。配置加载与存储响应乱序不应重挂表格或覆盖用户较新的修改。

## 相关规范

- `.trellis/spec/web/frontend/column-settings.md`：公共表头、费用例外、冻结、列宽、身份与全屏规范。
- `.trellis/spec/web/frontend/component-guidelines.md:270`：全站列设置与审核表例外。
- `.trellis/spec/web/frontend/capability-navigation.md`：表格响应式和统一拖宽公开入口。
- 根 `AGENTS.md`：不兼容旧存储、正式迁移/生成物、不得回退他人改动、按风险验证与中文文档。

## 外部引用

未调用外部文档；结论全部来自仓库源码和本地规范，未依赖第三方 API 的记忆。

## 限制与未发现项

- 本报告未执行测试、真实浏览器或数据库验证，所有“现有行为”依据实现与规范，不冒充真实环境验收。
- “全部调用点”仅指上述词法范围的直接 hook 调用，模板重复实例和文件内未绑定设置的嵌套表需另行按实例验证；新增业务代码在本研究期间可能变化。
- 没发现页面级原生 columnsState/persistenceKey 消费者或服务端既有表头偏好 API；只能说明本次限定目录检索未发现，不能据此断言所有第三方库内部没有实例状态。
- 没找到用户可配置的列标题重命名、字体、对齐等能力，不能作为现有维度加入。汇率同步加成 markups、登录记住设置、草稿 sessionStorage 是其他业务／身份能力，不纳入表头设置。
- 行密度是否保存与无入口表新增齿轮的范围在最终方案写清；主会话可按用户“表头所有设置”推荐前者，但不要把临时动作一起同步。
