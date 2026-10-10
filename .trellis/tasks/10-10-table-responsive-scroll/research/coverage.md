# 全站表格接入覆盖记录

静态基线82处直接表格及24处模板消费者全部处置。独立AST复核：48普通表、27页面/feature Pro表、5模板内部Pro表、2编辑表均接入共享组件；只有共享适配器内部保留3个原始官方组件入口。

| 基线路径 | 原类型 | 处置 |
| --- | --- | --- |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:460` | ProTable | 接入 ResponsiveProTable |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:576` | ProTable | 接入 ResponsiveProTable |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx:605` | ProTable | 接入 ResponsiveProTable |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx:157` | ProTable | 接入 ResponsiveProTable |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx:221` | ProTable | 接入 ResponsiveProTable |
| `web/src/features/finance/bill-creation/BillCandidateSelectionStep.tsx:74` | ProTable | 接入 ResponsiveProTable |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx:167` | Table | 接入 ResponsiveTable |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx:195` | Table | 接入 ResponsiveTable |
| `web/src/features/finance/bill-creation/BillGroupCard.tsx:439` | ProTable | 接入 ResponsiveProTable |
| `web/src/features/finance/bill-creation/NettingPairsCard.tsx:117` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/audit.tsx:115` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/admin/background-tasks.tsx:272` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/admin/components/AbnormalCasesPanel.tsx:47` | SettingTableTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/admin/components/NumberRulesPanel.tsx:519` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/components/org/OrgDetailCard.tsx:282` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/components/org/OrgInspectorPanel.tsx:336` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/components/users/DingTalkDirectorySyncPanel.tsx:265` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/components/users/UserFormModal.tsx:442` | Table | 接入 ResponsiveTable |
| `web/src/pages/admin/permissions.tsx:63` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/admin/roles.tsx:329` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/admin/users.tsx:177` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/enterprise-resources/index.tsx:675` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/bills/components/BillDetailDrawer.tsx:228` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/bills/index.tsx:369` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/cashflows/index.tsx:434` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx:231` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionApplicationsPanel.tsx:307` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:447` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:652` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx:386` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx:411` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionLineTable.tsx:93` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionRuleRosterModal.tsx:112` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx:463` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx:363` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/commissions/index.tsx:734` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx:359` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx:791` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx:258` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/fee-settings/components/BillingUnitsPanel.tsx:110` | SettingTableTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx:105` | SettingTableTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/fee-settings/components/FeeTemplatesPanel.tsx:61` | SettingTableTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/fee-settings/components/TaxableServicesPanel.tsx:72` | SettingTableTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/fees/detail.tsx:126` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/fees/index.tsx:304` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx:256` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx:215` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/invoices/components/InvoiceDetailDrawer.tsx:227` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/invoices/index.tsx:522` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/nettings/index.tsx:448` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/nettings/index.tsx:584` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:553` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:579` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx:645` | Table | 接入 ResponsiveTable |
| `web/src/pages/finance/verifications/index.tsx:354` | FinanceLedgerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/finance/verifications/index.tsx:506` | Table | 接入 ResponsiveTable |
| `web/src/pages/master-data/components/AirlinesPanel.tsx:113` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/AirportsPanel.tsx:110` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/CitiesPanel.tsx:76` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/CountriesPanel.tsx:96` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/CurrenciesPanel.tsx:84` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/PortsPanel.tsx:106` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/master-data/components/ShippingLinesPanel.tsx:125` | MasterDataTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/abnormal-case-panel.tsx:238` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/components/detail/SameBatchOrdersSection.tsx:139` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx:120` | SubEntityDrawerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/components/drawers/CargoItemDrawer.tsx:91` | SubEntityDrawerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx:94` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/components/drawers/ConsolidationDrawer.tsx:110` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/components/drawers/ContainerDrawer.tsx:114` | SubEntityDrawerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/components/drawers/MilestoneDrawer.tsx:90` | SubEntityDrawerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/components/drawers/PersonnelDrawer.tsx:77` | SubEntityDrawerTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx:331` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx:794` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/drawers/SeaSharedContainerAllocationTable.tsx:196` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx:302` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/drawers/ShippingDocumentDrawer.tsx:304` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/components/fees/FeeSupplementSection.tsx:753` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:1398` | EditableProTable | 接入 ResponsiveEditableProTable |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:1538` | EditableProTable | 接入 ResponsiveEditableProTable |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx:139` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/split/SplitAllocationSection.tsx:566` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/split/SplitAttachmentsAndNotesSection.tsx:119` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/components/split/SplitFeesSection.tsx:112` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/list.tsx:155` | OrderListTemplate | 共享模板能力继承（24消费者） |
| `web/src/pages/orders/order-fee-panel.tsx:427` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/release-pod-panel.tsx:401` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx:104` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/templates/components/sea/SeaCreateDocumentModeField.tsx:122` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:164` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:182` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:565` | Table | 接入 ResponsiveTable |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx:629` | Table | 接入 ResponsiveTable |
| `web/src/pages/partners/components/PartnerExcelImportModal.tsx:475` | Table | 接入 ResponsiveTable |
| `web/src/pages/partners/components/secondary/AccountsPanel.tsx:162` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/partners/components/secondary/AttachmentsPanel.tsx:102` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/partners/components/secondary/ContractsPanel.tsx:143` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/partners/components/secondary/InvoiceProfilesPanel.tsx:128` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx:287` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/partners/index.tsx:573` | ProTable | 接入 ResponsiveProTable |
| `web/src/pages/workbench/MyApplicationCandidatesDrawer.tsx:182` | Table | 接入 ResponsiveTable |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx:404` | Table | 接入 ResponsiveTable |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx:442` | Table | 接入 ResponsiveTable |
| `web/src/pages/workbench/MyCommissionDrawer.tsx:218` | Table | 接入 ResponsiveTable |
| `web/src/pages/workbench/MyCommissionDrawer.tsx:312` | Table | 接入 ResponsiveTable |
| `web/src/pages/workbench/MyReceivablesDrawer.tsx:148` | Table | 接入 ResponsiveTable |

浏览器实测覆盖9类主页面，每页390/962/1440/1920宽度，用户表额外3组侧栏折叠，总计39组。各组无整页横向溢出，溢出表可滚到最后列；详见browser-page-verification.json。

公共能力另用真实组件夹具验证窄/宽容器、固定列恢复、列隐藏、小表、y260、嵌套、编辑输入、弹窗和抽屉；这些代表样本不等于每个业务弹层均手工打开验收。业务request/actionRef/selection/edit回归与静态全覆盖共同保障未逐一手工打开的入口。
