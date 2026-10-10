# 当前表格调用点清单

基于现有产品TSX AST静态扫描，测试文件排除；共享模板消费者另列，不与内部原生表重复计数。缺scroll仅为审查候选，小表可能合理；有spread需要继续追踪。

原生调用 82，共享模板消费者 24；原生无显式scroll 65。

| 路径 | 行 | 类型 | 横向滚动配置 | 展开属性 |
| --- | ---: | --- | --- | --- |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx` | 460 | ProTable | scroll={{ x: scrollX }} |  |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx` | 576 | ProTable | scroll={{ x: 'max-content' }} |  |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx` | 605 | ProTable | scroll={{ x: 2600 }} |  |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx` | 157 | ProTable | scroll={scroll} |  |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx` | 221 | ProTable | 未显式配置 |  |
| `web/src/features/finance/bill-creation/BillCandidateSelectionStep.tsx` | 74 | ProTable | 未显式配置 |  |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx` | 167 | Table | 未显式配置 |  |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx` | 195 | Table | 未显式配置 |  |
| `web/src/features/finance/bill-creation/BillGroupCard.tsx` | 439 | ProTable | scroll={{ x: 880 }} |  |
| `web/src/features/finance/bill-creation/NettingPairsCard.tsx` | 117 | Table | 未显式配置 |  |
| `web/src/pages/admin/audit.tsx` | 115 | ProTable | 未显式配置 |  |
| `web/src/pages/admin/background-tasks.tsx` | 272 | ProTable | 未显式配置 |  |
| `web/src/pages/admin/components/AbnormalCasesPanel.tsx` | 47 | SettingTableTemplate | 未显式配置 |  |
| `web/src/pages/admin/components/NumberRulesPanel.tsx` | 519 | Table | 未显式配置 |  |
| `web/src/pages/admin/components/org/OrgDetailCard.tsx` | 282 | Table | 未显式配置 |  |
| `web/src/pages/admin/components/org/OrgInspectorPanel.tsx` | 336 | Table | 未显式配置 |  |
| `web/src/pages/admin/components/users/DingTalkDirectorySyncPanel.tsx` | 265 | Table | 未显式配置 |  |
| `web/src/pages/admin/components/users/UserFormModal.tsx` | 442 | Table | 未显式配置 |  |
| `web/src/pages/admin/permissions.tsx` | 63 | ProTable | 未显式配置 |  |
| `web/src/pages/admin/roles.tsx` | 329 | ProTable | 未显式配置 |  |
| `web/src/pages/admin/users.tsx` | 177 | ProTable | 未显式配置 |  |
| `web/src/pages/enterprise-resources/index.tsx` | 675 | ProTable | scroll={{ x: 1100 }} |  |
| `web/src/pages/finance/bills/components/BillDetailDrawer.tsx` | 228 | Table | 未显式配置 |  |
| `web/src/pages/finance/bills/index.tsx` | 369 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/cashflows/index.tsx` | 434 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx` | 231 | Table | scroll={{ x: 1000 }} |  |
| `web/src/pages/finance/commissions/components/CommissionApplicationsPanel.tsx` | 307 | ProTable | 未显式配置 |  |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | 447 | Table | scroll={{ y: 260 }} |  |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | 652 | Table | scroll={{ x: 1080 }} |  |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx` | 386 | Table | scroll={{ x: 1080 }} |  |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx` | 411 | Table | scroll={{ x: 1040 }} |  |
| `web/src/pages/finance/commissions/components/CommissionLineTable.tsx` | 93 | Table | 未显式配置 |  |
| `web/src/pages/finance/commissions/components/CommissionRuleRosterModal.tsx` | 112 | Table | 未显式配置 |  |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx` | 463 | ProTable | 未显式配置 |  |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx` | 363 | ProTable | 未显式配置 |  |
| `web/src/pages/finance/commissions/index.tsx` | 734 | ProTable | scroll={{ x: 1900 }} |  |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx` | 359 | Table | 未显式配置 |  |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx` | 791 | Table | scroll={{ y: 400, x: 920 }} |  |
| `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx` | 258 | ProTable | 未显式配置 |  |
| `web/src/pages/finance/fee-settings/components/BillingUnitsPanel.tsx` | 110 | SettingTableTemplate | 未显式配置 |  |
| `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx` | 105 | SettingTableTemplate | scroll={{ x: 1460 }} |  |
| `web/src/pages/finance/fee-settings/components/FeeTemplatesPanel.tsx` | 61 | SettingTableTemplate | scroll={{ x: 1100 }} |  |
| `web/src/pages/finance/fee-settings/components/TaxableServicesPanel.tsx` | 72 | SettingTableTemplate | 未显式配置 |  |
| `web/src/pages/finance/fees/detail.tsx` | 126 | Table | 未显式配置 |  |
| `web/src/pages/finance/fees/index.tsx` | 304 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx` | 256 | ProTable | 未显式配置 |  |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx` | 215 | Table | 未显式配置 |  |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx` | 227 | Table | 未显式配置 |  |
| `web/src/pages/finance/invoices/index.tsx` | 522 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/nettings/index.tsx` | 448 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/nettings/index.tsx` | 584 | Table | 未显式配置 |  |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 553 | Table | 未显式配置 |  |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 579 | Table | 未显式配置 |  |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 645 | Table | 未显式配置 |  |
| `web/src/pages/finance/verifications/index.tsx` | 354 | FinanceLedgerTemplate | 未显式配置 |  |
| `web/src/pages/finance/verifications/index.tsx` | 506 | Table | 未显式配置 |  |
| `web/src/pages/master-data/components/AirlinesPanel.tsx` | 113 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/AirportsPanel.tsx` | 110 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/CitiesPanel.tsx` | 76 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/CountriesPanel.tsx` | 96 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/CurrenciesPanel.tsx` | 84 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/PortsPanel.tsx` | 106 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/master-data/components/ShippingLinesPanel.tsx` | 125 | MasterDataTemplate | 未显式配置 |  |
| `web/src/pages/orders/abnormal-case-panel.tsx` | 238 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/components/detail/SameBatchOrdersSection.tsx` | 139 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx` | 120 | SubEntityDrawerTemplate | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/CargoItemDrawer.tsx` | 91 | SubEntityDrawerTemplate | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx` | 94 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx` | 110 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/ContainerDrawer.tsx` | 114 | SubEntityDrawerTemplate | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/MilestoneDrawer.tsx` | 90 | SubEntityDrawerTemplate | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/PersonnelDrawer.tsx` | 77 | SubEntityDrawerTemplate | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx` | 331 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx` | 794 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/SeaSharedContainerAllocationTable.tsx` | 196 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx` | 302 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/drawers/ShippingDocumentDrawer.tsx` | 304 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/components/fees/FeeSupplementSection.tsx` | 753 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | 1398 | EditableProTable | scroll={{ x: 'max-content' }} |  |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | 1538 | EditableProTable | scroll={{ x: 'max-content' }} |  |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx` | 139 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx` | 566 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/split/SplitAttachmentsAndNotesSection.tsx` | 119 | Table | 未显式配置 |  |
| `web/src/pages/orders/components/split/SplitFeesSection.tsx` | 112 | Table | 未显式配置 |  |
| `web/src/pages/orders/list.tsx` | 155 | OrderListTemplate | 未显式配置 |  |
| `web/src/pages/orders/order-fee-panel.tsx` | 427 | ProTable | scroll={{ x: 1500 }} |  |
| `web/src/pages/orders/release-pod-panel.tsx` | 401 | ProTable | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx` | 104 | Table | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx` | 122 | Table | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 164 | Table | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 182 | Table | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 565 | Table | 未显式配置 |  |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 629 | Table | 未显式配置 |  |
| `web/src/pages/partners/components/PartnerExcelImportModal.tsx` | 475 | Table | 未显式配置 |  |
| `web/src/pages/partners/components/secondary/AccountsPanel.tsx` | 162 | ProTable | 未显式配置 |  |
| `web/src/pages/partners/components/secondary/AttachmentsPanel.tsx` | 102 | ProTable | 未显式配置 |  |
| `web/src/pages/partners/components/secondary/ContractsPanel.tsx` | 143 | ProTable | 未显式配置 |  |
| `web/src/pages/partners/components/secondary/InvoiceProfilesPanel.tsx` | 128 | ProTable | 未显式配置 |  |
| `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx` | 287 | ProTable | 未显式配置 |  |
| `web/src/pages/partners/index.tsx` | 573 | ProTable | scroll={{ x: 1480 }} |  |
| `web/src/pages/workbench/MyApplicationCandidatesDrawer.tsx` | 182 | Table | 未显式配置 |  |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx` | 404 | Table | 未显式配置 |  |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx` | 442 | Table | 未显式配置 |  |
| `web/src/pages/workbench/MyCommissionDrawer.tsx` | 218 | Table | 未显式配置 |  |
| `web/src/pages/workbench/MyCommissionDrawer.tsx` | 312 | Table | 未显式配置 |  |
| `web/src/pages/workbench/MyReceivablesDrawer.tsx` | 148 | Table | 未显式配置 |  |
