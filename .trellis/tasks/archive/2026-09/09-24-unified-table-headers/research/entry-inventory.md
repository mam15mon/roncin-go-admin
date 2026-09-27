# 全站列设置入口盘点初稿

## 范围与统计口径

用户已确认所有表格都要有设置入口。下列为源码静态扫描所得的 108 个表格／模板调用点，包含公共模板定义和消费者，不能视为 108 张独立业务表格。实施时还须追踪动态封装、组件别名与透传配置；逐个关联实际页面与入口，不因当前缺少设置按钮而排除。

## 逐项补全要求

每个实际表格记录：业务入口、稳定表格标识、使用模板、现有入口类型、新增或迁移、标准／增强版、存储范围、必显／固定规则、分组／汇总约束、验证结果。普通表默认标准版，财务费用台账增强版；未逐项核验前不得勾选全站完成。

## 静态调用点

| 文件 | 行号 | 调用片段 |
| --- | --- | --- |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx` | 567 | `<ProTable<T>` |
| `web/src/pages/admin/components/NumberRulesPanel.tsx` | 500 | `<Table` |
| `web/src/pages/admin/components/AbnormalCasesPanel.tsx` | 47 | `<SettingTableTemplate<API.MasterDataItem, AbnormalCaseFormValues>` |
| `web/src/pages/admin/users.tsx` | 164 | `<ProTable<API.AdminUser>` |
| `web/src/pages/admin/permissions.tsx` | 57 | `<ProTable<API.AdminPermission>` |
| `web/src/pages/admin/background-tasks.tsx` | 263 | `<ProTable<API.BackgroundTask>` |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx` | 209 | `<ProTable<TItem>` |
| `web/src/pages/admin/audit.tsx` | 109 | `<ProTable<API.AdminAuditLog>` |
| `web/src/pages/admin/roles.tsx` | 323 | `<ProTable<API.AdminRole>` |
| `web/src/pages/admin/components/users/UserFormModal.tsx` | 401 | `<Table<API.AdminUserMembership>` |
| `web/src/pages/admin/dingtalk-invitations.tsx` | 280 | `<ProTable<API.DingTalkInvitation>` |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx` | 147 | `<ProTable<TRecord>` |
| `web/src/pages/admin/dingtalk-registrations.tsx` | 136 | `<ProTable<API.DingTalkRegistration>` |
| `web/src/pages/admin/components/org/OrgInspectorPanel.tsx` | 285 | `<Table<API.AdminOrganization>` |
| `web/src/pages/admin/components/org/OrgDetailCard.tsx` | 187 | `<Table<API.AdminOrganization>` |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx` | 436 | `<ProTable<T>` |
| `web/src/pages/master-data/components/CitiesPanel.tsx` | 76 | `<MasterDataTemplate<RegionItem>` |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx` | 85 | `<Table<API.FinanceBill>` |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx` | 131 | `<Table<API.FinanceNetting>` |
| `web/src/pages/partners/components/secondary/AccountsPanel.tsx` | 155 | `<ProTable<API.PartnerAccount>` |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx` | 595 | `<ProTable<OrderListItem>` |
| `web/src/pages/master-data/components/CountriesPanel.tsx` | 96 | `<MasterDataTemplate<CountryItem>` |
| `web/src/pages/master-data/components/PortsPanel.tsx` | 106 | `<MasterDataTemplate<PortItem>` |
| `web/src/pages/partners/components/secondary/AttachmentsPanel.tsx` | 96 | `<ProTable<API.PartnerAttachment>` |
| `web/src/features/finance/bill-creation/BillGroupCard.tsx` | 424 | `<ProTable<API.FeeLedgerItem>` |
| `web/src/features/finance/bill-creation/NettingPairsCard.tsx` | 34 | `<Table<API.BillBatchNettingPair>` |
| `web/src/pages/partners/components/secondary/InvoiceProfilesPanel.tsx` | 66 | `<ProTable<API.PartnerInvoiceProfile>` |
| `web/src/features/finance/bill-creation/BillCandidateSelectionStep.tsx` | 68 | `<ProTable<API.FeeLedgerItem>` |
| `web/src/pages/master-data/components/ShippingLinesPanel.tsx` | 125 | `<MasterDataTemplate<ShippingLineItem>` |
| `web/src/pages/partners/index.tsx` | 564 | `<ProTable<API.Partner>` |
| `web/src/pages/master-data/components/AirportsPanel.tsx` | 110 | `<MasterDataTemplate<AirportItem>` |
| `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx` | 281 | `<ProTable<SettlementRuleItem>` |
| `web/src/pages/master-data/components/CurrenciesPanel.tsx` | 84 | `<MasterDataTemplate<CurrencyItem>` |
| `web/src/pages/partners/components/secondary/ContractsPanel.tsx` | 137 | `<ProTable<API.PartnerContract>` |
| `web/src/pages/master-data/components/AirlinesPanel.tsx` | 113 | `<MasterDataTemplate<AirlineItem>` |
| `web/src/pages/workbench/MyApplicationCandidatesDrawer.tsx` | 165 | `<Table<Candidate>` |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx` | 379 | `<Table<ApplicationLine>` |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx` | 408 | `<Table<Application>` |
| `web/src/pages/workbench/MyCommissionDrawer.tsx` | 196 | `<Table<MyCommission>` |
| `web/src/pages/workbench/MyCommissionDrawer.tsx` | 214 | `<Table<MyAdjustment>` |
| `web/src/pages/finance/fees/detail.tsx` | 84 | `<Table<API.FeeLedgerItem>` |
| `web/src/pages/partners/components/PartnerExcelImportModal.tsx` | 461 | `<Table` |
| `web/src/pages/orders/list.tsx` | 155 | `<OrderListTemplate` |
| `web/src/pages/workbench/MyReceivablesDrawer.tsx` | 131 | `<Table<MyReceivable>` |
| `web/src/pages/orders/abnormal-case-panel.tsx` | 232 | `<ProTable<API.OrderAbnormalCase>` |
| `web/src/pages/finance/fees/index.tsx` | 286 | `<FinanceLedgerTemplate<API.FeeLedgerItem>` |
| `web/src/pages/orders/order-fee-panel.tsx` | 426 | `<ProTable<API.OrderFee>` |
| `web/src/pages/orders/release-pod-panel.tsx` | 395 | `<ProTable<API.OrderReleasePod>` |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx` | 63 | `<Table<API.SeaDocumentFieldDifference>` |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx` | 75 | `<Table<API.SeaDocumentDownstreamImpact>` |
| `web/src/pages/enterprise-resources/index.tsx` | 664 | `<ProTable<API.EnterpriseResource>` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 122 | `<Table<API.SeaDocumentFieldDifference>` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 134 | `<Table<API.SeaDocumentDownstreamImpact>` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 460 | `<Table<API.SeaDocumentVersion>` |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 536 | `<Table<API.SeaDocumentEvent>` |
| `web/src/pages/orders/components/fees/FeeSupplementSection.tsx` | 734 | `<ProTable<SupplementRequest>` |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx` | 120 | `<Table<API.SeaOrderSplitContainerItem>` |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx` | 239 | `<Table` |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx` | 480 | `<Table` |
| `web/src/pages/orders/components/drawers/SeaSharedContainerAllocationTable.tsx` | 174 | `<Table` |
| `web/src/pages/finance/bills/components/BillDetailDrawer.tsx` | 154 | `<Table<API.FinanceBillLine>` |
| `web/src/pages/orders/components/split/SplitAttachmentsAndNotesSection.tsx` | 101 | `<Table<API.SeaOrderSplitAttachmentItem>` |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | 1422 | `<EditableProTable<API.OrderFee>` |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | 1564 | `<EditableProTable<API.OrderFee>` |
| `web/src/pages/orders/components/split/SplitFeesSection.tsx` | 96 | `<Table<API.SeaOrderSplitDraftFeeItem>` |
| `web/src/pages/orders/components/drawers/MilestoneDrawer.tsx` | 90 | `<SubEntityDrawerTemplate<` |
| `web/src/pages/orders/components/drawers/CargoItemDrawer.tsx` | 91 | `<SubEntityDrawerTemplate<` |
| `web/src/pages/finance/bills/index.tsx` | 362 | `<FinanceLedgerTemplate<API.FinanceBill>` |
| `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx` | 120 | `<SubEntityDrawerTemplate<` |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx` | 278 | `<Table<API.VoyageDifferenceItem>` |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx` | 41 | `<ProTable<API.OrderConsolidationSummary>` |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx` | 68 | `<ProTable<API.OrderConsolidationMember>` |
| `web/src/pages/orders/components/drawers/ShippingDocumentDrawer.tsx` | 296 | `<ProTable<API.OrderShippingDocument>` |
| `web/src/pages/finance/commissions/index.tsx` | 727 | `<ProTable<API.FinanceCommission>` |
| `web/src/pages/orders/components/detail/SameBatchOrdersSection.tsx` | 82 | `<Table<API.SameBatchOrderSummary>` |
| `web/src/pages/finance/invoices/index.tsx` | 517 | `<FinanceLedgerTemplate<API.FinanceInvoice, InvoiceLedgerFilterParams>` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 521 | `<Table` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 538 | `<Table` |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 595 | `<Table` |
| `web/src/pages/orders/components/drawers/PersonnelDrawer.tsx` | 77 | `<SubEntityDrawerTemplate<` |
| `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx` | 105 | `<SettingTableTemplate<API.FeeSetting, FeeSettingFormValues>` |
| `web/src/pages/finance/nettings/index.tsx` | 388 | `<FinanceLedgerTemplate<API.FinanceNetting, NettingLedgerFilterParams>` |
| `web/src/pages/finance/nettings/index.tsx` | 513 | `<Table<API.FinanceNettingAllocation>` |
| `web/src/pages/orders/components/drawers/ContainerDrawer.tsx` | 114 | `<SubEntityDrawerTemplate<` |
| `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx` | 313 | `<Table<API.SeaOrderChangeEventSummary>` |
| `web/src/pages/finance/fee-settings/components/FeeTemplatesPanel.tsx` | 61 | `<SettingTableTemplate<TemplateRow, API.FeeSettingTemplateInput>` |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx` | 775 | `<Table<API.VoyageDifferenceItem>` |
| `web/src/pages/finance/fee-settings/components/BillingUnitsPanel.tsx` | 110 | `<SettingTableTemplate<API.BillingUnit, BillingUnitFormValues>` |
| `web/src/pages/finance/verifications/index.tsx` | 280 | `<FinanceLedgerTemplate<` |
| `web/src/pages/finance/verifications/index.tsx` | 422 | `<Table<API.FinanceVerificationAllocation>` |
| `web/src/pages/finance/fee-settings/components/TaxableServicesPanel.tsx` | 72 | `<SettingTableTemplate<API.TaxableService, TaxableServiceFormValues>` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx` | 259 | `<Table<API.ExchangeRateImportRow>` |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx` | 455 | `<ProTable<API.FinanceCommissionRule>` |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | 423 | `<Table<API.FinanceNetting>` |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | 625 | `<Table<API.FinanceCommissionLine>` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx` | 778 | `<Table<ExchangeRateSyncDraftRow>` |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx` | 141 | `<Table<API.FinanceInvoiceLine>` |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx` | 164 | `<Table` |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx` | 369 | `<Table<API.FinanceCommissionLine>` |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx` | 391 | `<Table<API.FinanceCommissionAdjustment>` |
| `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx` | 249 | `<ProTable<API.ExchangeRateSetting>` |
| `web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx` | 229 | `<ProTable<API.FinanceBill>` |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx` | 357 | `<ProTable<Adjustment>` |
| `web/src/pages/finance/cashflows/index.tsx` | 429 | `<FinanceLedgerTemplate<API.FinanceCashflow, CashflowLedgerFilterParams>` |
| `web/src/pages/finance/commissions/components/CommissionLineTable.tsx` | 93 | `<Table<API.CommissionFeeDetail>` |
| `web/src/pages/finance/commissions/components/CommissionRuleRosterModal.tsx` | 61 | `<Table<API.CommissionRuleAssignmentProjection>` |
| `web/src/pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx` | 222 | `<Table<Line>` |
| `web/src/pages/finance/commissions/components/CommissionApplicationsPanel.tsx` | 301 | `<ProTable<Application>` |

## 迁移结果（2026-09-24 实施完成）

全部 108 个静态调用点已按「表格标识」收敛到统一列设置；共享模板内部接入后，
其消费者自动获得入口，无需逐页修改。存储一律为浏览器本地偏好
（`roncin:column-settings:v1:<表格标识>:<用户>:<组织>`），财务费用台账保留服务端偏好。

| 表格标识 | 覆盖调用点 | 版本 | 入口形态 |
| --- | --- | --- | --- |
| `finance:fees-ledger`（服务端偏好） | fees/index → FinanceLedgerTemplate(onOpenColumnConfig) | 增强版 | 模板工具栏齿轮 → 统一弹窗 + 高级页签 |
| `finance:bills` / `finance:invoices` / `finance:verifications` / `finance:cashflows` / `finance:nettings` | 5 个台账页 | 标准版 | FinanceLedgerTemplate 内置入口 |
| `master-data:<标题>` | 7 个主数据面板（Cities/Countries/Ports/ShippingLines/Airports/Currencies/Airlines） | 标准版 | MasterDataTemplate 内置入口 |
| `orders:list` | orders/list → OrderListTemplate | 标准版 | ProTable 工具栏 |
| `setting:<实体名>` | 4 个参数设置面板 + AbnormalCasesPanel | 标准版 | SettingTableTemplate 内置入口 |
| `orders:sub-entity:<实体名>` | 5 个订单子实体抽屉（里程碑/货物/附件/人员/箱） | 标准版 | SubEntityDrawerTemplate 内置入口 |
| `orders:fee-tabs`（既有键 `roncin:order-fee-columns:v1:*` 不变） | OrderFeeTableTabs 应收/应付两表 | 标准版 | 统一弹窗（保留必显/权限/编辑保护） |
| `admin:users` / `admin:roles` / `admin:permissions` / `admin:audit` / `admin:background-tasks` / `admin:dingtalk-invitations` / `admin:dingtalk-registrations` | admin 7 个 ProTable | 标准版 | ProTable 工具栏 |
| `admin:number-rules` / `admin:user-memberships` / `admin:org-inspector` / `admin:org-detail` | admin 4 个普通 Table | 标准版 | 表格上方右侧 |
| `partners:list` / `partners:accounts` / `partners:attachments` / `partners:contracts` / `partners:invoice-profiles` / `partners:settlement-rules` / `partners:excel-import-preview` | 客商主列表 + 5 二级面板 + 导入预览 | 标准版 | 工具栏 / 表格上方右侧 |
| `enterprise-resources:list` | 企业资源列表 | 标准版 | ProTable 工具栏 |
| `finance:commission-list` / `finance:commission-applications` / `finance:commission-pending-decrease` / `finance:commission-rules` | 提成域 ProTable | 标准版 | ProTable 工具栏 |
| `finance:exchange-rates` / `finance:exchange-import-preview` / `finance:exchange-sync-preview` | 汇率面板与导入/同步预览 | 标准版 | 工具栏 / 表格上方右侧 |
| `finance:verification-*`（3 表） / `finance:verification-allocation-list` / `finance:netting-allocation-list` | 核销工作台与分配明细 | 标准版 | 表格上方右侧 |
| `finance:bill-lines` / `finance:invoice-lines` / `finance:invoice-records` / `finance:invoice-candidate-bills` / `finance:fee-detail-lines` | 账单/发票/费用明细弹层 | 标准版 | 工具栏 / 表格上方右侧 |
| `finance:commission-candidate-*` / `finance:commission-lines` / `finance:commission-adjustments` / `finance:commission-rule-roster` / `finance:commission-application-lines` | 提成弹层明细 | 标准版 | 表格上方右侧 |
| `orders:abnormal-cases` / `orders:release-pod` / `orders:fee-panel` / `orders:shipping-documents` / `orders:consolidation-summary` / `orders:consolidation-members` / `orders:fee-supplements` | 订单面板与抽屉 ProTable | 标准版 | ProTable 工具栏 |
| `orders:same-batch` / `orders:change-history` / `orders:split-*`（5 表） / `orders:shared-container-allocation` / `orders:voyage-difference` / `orders:reassignment-voyage-difference` / `orders:doc-*`（4 表） | 订单普通 Table 与嵌套表 | 标准版 | 表格上方右侧 |
| `workbench:application-candidates` / `workbench:application-history(-lines)` / `workbench:my-commissions` / `workbench:my-adjustments` / `workbench:my-receivables` | 工作台抽屉 | 标准版 | 表格上方右侧 |
| `finance:bill-group-fees` / `finance:bill-candidate-fees` / `finance:bill-netting-pairs` / `finance:bill-creation-results` / `finance:bill-creation-nettings` | 建账工作台 | 标准版 | 工具栏 / 表格上方右侧 |

说明：
- 操作列（无 key/dataIndex 或显式结构列）不进设置列表且保持在原位置；
  固定列仅在原固定区域内排序，不跨区。
- 旧 `FeeColumnSettingsModal` 已删除；旧 `FieldConfigCard` 已删除；
  `TableColumnConfigModal` 重写为统一弹窗增强版适配层（外部 Props 不变）。
- 未接入例外：登录/回调页无业务表格；`WorkbenchSideCards` 无表格（Listy 卡片）。
