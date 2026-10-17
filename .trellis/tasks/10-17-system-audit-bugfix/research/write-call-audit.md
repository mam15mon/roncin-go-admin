# 写调用审计与验证记录

## 范围与判定

对 `web/src/services/roncin/*.ts` 中生成客户端声明进行 AST 扫描，提取非 GET 方法及其响应 DTO，再匹配 `web/src` 生产代码的直接调用。当前共 **214 处调用、94 个文件**（POST 151、PUT 49、DELETE 14），包含 POST 预览，不代表 214 个独立写接口。生成客户端、传输错误消费契约和后端接口均未手改。

实体写入统一以 `isWriteResponse(response, true)` 确认非空响应、`success !== false` 和必要 `data.id`；无实体操作要求 `success === true`。非标准响应按实际 DTO 核验锁状态的 `orderId`、改配事件、拆票事件及原订单、运输执行 ID 与版本、单证切换的订单 ID。批量标签接口只有计数且 protojson 可将零计数编码成 `{}`，仅该类显式使用 count 模式；其他接口不以空对象冒充成功。

`undefined`、`success:false`、缺少必要实体 ID 或请求 reject 均不可进入成功提示、关闭、清草稿、导航或刷新。`ModalForm.onFinish` 返回 false；useMutation 的 onSuccess 同样先守卫。订单已落库后的后台同步仍按既有 best-effort 契约处理。

## 间接消费与确认生命周期

- `useMasterDataCrud`：创建、更新、启停保存原始响应并判定；false 透过模板回调传至确认层。货币、国家、计费单位、编号规则及其他使用者按实际 DTO 消费。
- `SubEntityDrawerTemplate`：创建/更新必须有实体 ID；移除必须有 success；账户、合同、结算规则、开票资料和订单子实体等回调返回原始响应，失败不触发刷新和关窗。
- `BusinessTagModal`、`confirmWithReason`、`makeVersionActions`：保留失败语义到调用方，失败保留已选标签和原因。
- 已安装 Ant Design 的 Popconfirm 对任何 resolve 都请求关闭，false 不具备 ModalForm 的留窗语义；直接 reject 又会形成未处理拒绝。因此新增最薄共享 `AsyncPopconfirm`，沿用官方 Promise loading，仅阻止失败那次关闭；成功、用户取消和点击外部保留官方行为。已迁移 19 个生产文件、23 处真实异步写确认。剩余原生 Popconfirm 是联系人本地删除、建账本地候选删除和订单尚未接后端批操作，均不属于异步写。
- 登录表单通过已有显式实体验证；钉钉回调、资源图片预上传、费用审核/提成发放等已具备正确结果边界的调用保留原实现。POST 预览按实际数据形态判定，不能机械要求 data.id。

## 直接调用清单

以下逐文件记录本次扫描到的全部调用，方法及返回类型来自生成客户端和 typings；同文件重复调用保留次数。行号为收敛时的定位辅助，后续格式调整可能移动。

| 文件 | 调用（行号；方法；响应 DTO） |
| --- | --- |
| `web/src/components/OrganizationSwitcher.tsx` | `authServiceSwitchOrganization`（31；POST；`SwitchOrganizationResponse`） |
| `web/src/components/RightContent/AvatarDropdown.tsx` | `authServiceLogout`（30；POST；`LogoutResponse`） |
| `web/src/components/business-tag/BusinessTagModal.tsx` | `enterpriseResourceServiceCreateEnterpriseResource`（135；POST；`CreateEnterpriseResourceResponse`） |
| `web/src/features/finance/bill-creation/BillCreationWorkbench.tsx` | `settlementServicePreviewBillBatch`（67；POST；`PreviewBillBatchResponse`）<br>`settlementServiceCreateBillBatch`（488；POST；`CreateBillBatchResponse`）<br>`settlementServiceConfirmBillBatch`（552；POST；`ConfirmBillBatchResponse`） |
| `web/src/features/orders/fee-correction/FeeCorrectionSection.tsx` | `orderFeeServicePreviewOrderFeeCorrectionApproval`（182；POST；`PreviewOrderFeeCorrectionApprovalResponse`）<br>`orderFeeServiceWithdrawOrderFeeCorrection`（242；POST；`WithdrawOrderFeeCorrectionResponse`）<br>`orderFeeServiceRejectOrderFeeCorrection`（272；POST；`RejectOrderFeeCorrectionResponse`）<br>`orderFeeServiceApproveOrderFeeCorrection`（359；POST；`ApproveOrderFeeCorrectionResponse`） |
| `web/src/pages/admin/background-tasks.tsx` | `backgroundTaskServiceRequeueBackgroundTask`（247；POST；`RequeueBackgroundTaskResponse`） |
| `web/src/pages/admin/components/AbnormalCasesPanel.tsx` | `masterDataServiceCreateItem`（67；POST；`CreateItemResponse`）<br>`masterDataServiceUpdateItem`（78；PUT；`UpdateItemResponse`） |
| `web/src/pages/admin/components/NumberRulesPanel.tsx` | `masterDataServiceUpdateNumberRule`（130；PUT；`UpdateNumberRuleResponse`）<br>`masterDataServiceUpdateNumberRule`（160；PUT；`UpdateNumberRuleResponse`）<br>`masterDataServiceCreateNumberRule`（174；POST；`CreateNumberRuleResponse`） |
| `web/src/pages/admin/components/org/OrgCreateModal.tsx` | `adminServiceCreateOrganization`（55；POST；`CreateOrganizationResponse`） |
| `web/src/pages/admin/components/org/OrgEditModal.tsx` | `adminServiceUpdateOrganization`（52；PUT；`UpdateOrganizationResponse`） |
| `web/src/pages/admin/components/roles/RoleFormModal.tsx` | `adminServiceUpdateRole`（300；PUT；`UpdateRoleResponse`）<br>`adminServiceCreateRole`（312；POST；`CreateRoleResponse`） |
| `web/src/pages/admin/components/users/DingTalkAssignmentModal.tsx` | `adminServiceBatchAuthorizeDingTalkUsers`（91；POST；`BatchAuthorizeDingTalkUsersResponse`） |
| `web/src/pages/admin/components/users/DingTalkDirectorySyncPanel.tsx` | `adminServiceStartDingTalkDirectorySync`（85；POST；`StartDingTalkDirectorySyncResponse`） |
| `web/src/pages/admin/components/users/ResetPasswordModal.tsx` | `adminServiceResetUserPassword`（41；PUT；`ResetUserPasswordResponse`） |
| `web/src/pages/admin/components/users/UserFormModal.tsx` | `adminServiceDeleteUserMembership`（184；DELETE；`DeleteUserMembershipResponse`）<br>`adminServiceUpdateUser`（249；PUT；`UpdateUserResponse`）<br>`adminServiceCreateUser`（266；POST；`CreateUserResponse`） |
| `web/src/pages/admin/components/users/UserMembershipModal.tsx` | `adminServiceUpdateUserMembership`（77；PUT；`UpdateUserMembershipResponse`）<br>`adminServiceCreateUserMemberships`（97；POST；`CreateUserMembershipsResponse`） |
| `web/src/pages/admin/roles.tsx` | `adminServiceDeleteRole`（103；DELETE；`DeleteRoleResponse`） |
| `web/src/pages/admin/users.tsx` | `adminServiceTerminateUser`（99；POST；`TerminateUserResponse`） |
| `web/src/pages/enterprise-resources/components/ResourceEditorModal.tsx` | `enterpriseResourceServicePrepareEnterpriseResourceImageUpload`（314；POST；`PrepareEnterpriseResourceImageUploadResponse`） |
| `web/src/pages/enterprise-resources/components/TagGroupModal.tsx` | `enterpriseResourceServiceDeleteEnterpriseTagGroup`（82；DELETE；`DeleteEnterpriseTagGroupResponse`） |
| `web/src/pages/enterprise-resources/index.tsx` | `enterpriseResourceServiceUpdateEnterpriseResource`（299；PUT；`UpdateEnterpriseResourceResponse`）<br>`enterpriseResourceServiceCreateEnterpriseResource`（306；POST；`CreateEnterpriseResourceResponse`）<br>`enterpriseResourceServiceDeleteEnterpriseResource`（512；DELETE；`DeleteEnterpriseResourceResponse`）<br>`enterpriseResourceServiceBatchCreateAssociations`（555；POST；`BatchCreateAssociationsResponse`）<br>`enterpriseResourceServiceBatchDeleteAssociations`（556；POST；`BatchDeleteAssociationsResponse`）<br>`enterpriseResourceServiceBatchAssignAddressTypes`（574；POST；`BatchAssignAddressTypesResponse`）<br>`enterpriseResourceServiceBatchRemoveAddressTypes`（575；POST；`BatchRemoveAddressTypesResponse`）<br>`enterpriseResourceServiceBatchAssignAssignees`（595；POST；`BatchAssignAssigneesResponse`）<br>`enterpriseResourceServiceBatchRemoveAssignees`（596；POST；`BatchRemoveAssigneesResponse`）<br>`enterpriseResourceServiceUpdateEnterpriseTagGroup`（616；PUT；`UpdateEnterpriseTagGroupResponse`）<br>`enterpriseResourceServiceCreateEnterpriseTagGroup`（630；POST；`CreateEnterpriseTagGroupResponse`）<br>`enterpriseResourceServicePreviewEnterpriseResourceImport`（646；POST；`PreviewEnterpriseResourceImportResponse`）<br>`enterpriseResourceServiceCommitEnterpriseResourceImport`（660；POST；`CommitEnterpriseResourceImportResponse`） |
| `web/src/pages/finance/bills/index.tsx` | `settlementServiceUpdateBill`（282；PUT；`UpdateBillResponse`）<br>`settlementServiceConfirmBill`（317；POST；`ConfirmBillResponse`）<br>`settlementServiceCancelBill`（337；POST；`CancelBillResponse`）<br>`settlementServiceBatchAssignFinanceBillTags`（646；POST；`BatchAssignFinanceBillTagsResponse`）<br>`settlementServiceBatchRemoveFinanceBillTags`（653；POST；`BatchRemoveFinanceBillTagsResponse`） |
| `web/src/pages/finance/cashflows/index.tsx` | `settlementServiceConfirmCashflow`（184；POST；`ConfirmCashflowResponse`）<br>`settlementServiceCancelCashflow`（203；POST；`CancelCashflowResponse`）<br>`settlementServiceCreateCashflow`（559；POST；`CreateCashflowResponse`） |
| `web/src/pages/finance/commissions/components/CommissionAdjustmentModal.tsx` | `settlementServiceCreateCommissionAdjustment`（48；POST；`CreateCommissionAdjustmentResponse`） |
| `web/src/pages/finance/commissions/components/CommissionApplicationsPanel.tsx` | `settlementServiceApproveCommissionApplication`（120；POST；`ApproveCommissionApplicationResponse`）<br>`settlementServiceRejectCommissionApplication`（145；POST；`RejectCommissionApplicationResponse`） |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | `settlementServiceCreateCommission`（344；POST；`CreateCommissionResponse`）<br>`settlementServicePreviewCommission`（531；POST；`PreviewCommissionResponse`） |
| `web/src/pages/finance/commissions/components/CommissionPaymentModal.tsx` | `settlementServicePreviewCommissionPayment`（34；POST；`PreviewCommissionPaymentResponse`）<br>`settlementServiceMarkCommissionPaid`（45；POST；`MarkCommissionPaidResponse`） |
| `web/src/pages/finance/commissions/components/CommissionRuleRosterModal.tsx` | `settlementServiceAssignCommissionRuleEmployees`（191；POST；`AssignCommissionRuleEmployeesResponse`）<br>`settlementServiceRemoveCommissionRuleEmployees`（199；POST；`RemoveCommissionRuleEmployeesResponse`） |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx` | `settlementServiceUpdateCommissionRule`（196；PUT；`UpdateCommissionRuleResponse`）<br>`settlementServiceCreateCommissionRule`（218；POST；`CreateCommissionRuleResponse`）<br>`settlementServiceCopyCommissionRule`（248；POST；`CopyCommissionRuleResponse`） |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx` | `settlementServiceMarkCommissionAdjustmentPaid`（99；POST；`MarkCommissionAdjustmentPaidResponse`） |
| `web/src/pages/finance/commissions/index.tsx` | `settlementServiceConfirmCommissionAdjustment`（224；POST；`ConfirmCommissionAdjustmentResponse`）<br>`settlementServiceMarkCommissionAdjustmentPaid`（231；POST；`MarkCommissionAdjustmentPaidResponse`）<br>`settlementServiceCancelCommissionAdjustment`（271；POST；`CancelCommissionAdjustmentResponse`）<br>`settlementServiceConfirmCommission`（312；POST；`ConfirmCommissionResponse`）<br>`settlementServiceCancelCommission`（361；POST；`CancelCommissionResponse`） |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx` | `exchangeRateServicePreviewExchangeRateImport`（88；POST；`PreviewExchangeRateImportResponse`）<br>`exchangeRateServiceConfirmExchangeRateImport`（125；POST；`ConfirmExchangeRateImportResponse`） |
| `web/src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.tsx` | `exchangeRateServiceFetchExchangeRates`（253；POST；`FetchExchangeRatesResponse`）<br>`exchangeRateServiceSyncExchangeRates`（340；POST；`SyncExchangeRatesResponse`） |
| `web/src/pages/finance/exchange-rates/components/ExchangeRatesPanel.tsx` | `exchangeRateServiceDisableExchangeRateSetting`（202；POST；`DisableExchangeRateSettingResponse`）<br>`exchangeRateServiceUpdateExchangeRateSetting`（394；PUT；`UpdateExchangeRateSettingResponse`）<br>`exchangeRateServiceCreateExchangeRateSetting`（402；POST；`CreateExchangeRateSettingResponse`） |
| `web/src/pages/finance/fee-settings/components/BillingUnitsPanel.tsx` | `feeCatalogServiceCreateBillingUnit`（124；POST；`CreateBillingUnitResponse`）<br>`feeCatalogServiceUpdateBillingUnit`（141；PUT；`UpdateBillingUnitResponse`） |
| `web/src/pages/finance/fee-settings/components/CustomSettingsPanel.tsx` | `settlementServiceUpdateBilledFeeEditPolicy`（116；PUT；`UpdateBilledFeeEditPolicyResponse`）<br>`settlementServiceUpdateBilledFeeEditPolicy`（144；PUT；`UpdateBilledFeeEditPolicyResponse`）<br>`settlementServiceUpdateCreditLimitControlPolicy`（168；PUT；`UpdateCreditLimitControlPolicyResponse`） |
| `web/src/pages/finance/fee-settings/components/FeeItemsPanel.tsx` | `feeCatalogServiceCreateFeeSetting`（155；POST；`CreateFeeSettingResponse`）<br>`feeCatalogServiceUpdateFeeSetting`（171；PUT；`UpdateFeeSettingResponse`） |
| `web/src/pages/finance/fee-settings/components/FeeTemplatesPanel.tsx` | `feeCatalogServiceCreateFeeSettingTemplate`（83；POST；`CreateFeeSettingTemplateResponse`）<br>`feeCatalogServiceUpdateFeeSettingTemplate`（87；PUT；`UpdateFeeSettingTemplateResponse`） |
| `web/src/pages/finance/fee-settings/components/TaxableServicesPanel.tsx` | `feeCatalogServiceCreateTaxableService`（81；POST；`CreateTaxableServiceResponse`）<br>`feeCatalogServiceUpdateTaxableService`（90；PUT；`UpdateTaxableServiceResponse`） |
| `web/src/pages/finance/fees/index.tsx` | `settlementServiceBatchAssignFinanceFeeTags`（628；POST；`BatchAssignFinanceFeeTagsResponse`）<br>`settlementServiceBatchRemoveFinanceFeeTags`（636；POST；`BatchRemoveFinanceFeeTagsResponse`） |
| `web/src/pages/finance/invoices/index.tsx` | `settlementServiceCreateInvoice`（163；POST；`CreateInvoiceResponse`）<br>`settlementServiceIssueInvoice`（193；POST；`IssueInvoiceResponse`）<br>`settlementServiceCancelInvoice`（234；POST；`CancelInvoiceResponse`）<br>`settlementServiceRedFlushInvoice`（266；POST；`RedFlushInvoiceResponse`） |
| `web/src/pages/finance/nettings/index.tsx` | `settlementServiceConfirmNetting`（93；POST；`ConfirmNettingResponse`）<br>`settlementServiceCancelNetting`（111；POST；`CancelNettingResponse`）<br>`settlementServiceReverseNetting`（133；POST；`ReverseNettingResponse`） |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | `settlementServiceCreateVerification`（306；POST；`CreateVerificationResponse`） |
| `web/src/pages/finance/verifications/index.tsx` | `settlementServiceReverseVerification`（84；POST；`ReverseVerificationResponse`） |
| `web/src/pages/master-data/components/AirlinesPanel.tsx` | `masterDataServiceCreateAirline`（83；POST；`CreateAirlineResponse`）<br>`masterDataServiceUpdateAirline`（95；PUT；`UpdateAirlineResponse`） |
| `web/src/pages/master-data/components/AirportsPanel.tsx` | `masterDataServiceCreateAirport`（82；POST；`CreateAirportResponse`）<br>`masterDataServiceUpdateAirport`（93；PUT；`UpdateAirportResponse`） |
| `web/src/pages/master-data/components/CountriesPanel.tsx` | `masterDataServiceCreateItem`（61；POST；`CreateItemResponse`）<br>`masterDataServiceUpdateItem`（77；PUT；`UpdateItemResponse`） |
| `web/src/pages/master-data/components/CurrenciesPanel.tsx` | `masterDataServiceSetCurrencyEnabled`（65；PUT；`SetCurrencyEnabledResponse`） |
| `web/src/pages/master-data/components/PortsPanel.tsx` | `masterDataServiceCreatePort`（82；POST；`CreatePortResponse`）<br>`masterDataServiceUpdatePort`（91；PUT；`UpdatePortResponse`） |
| `web/src/pages/master-data/components/ShippingLinesPanel.tsx` | `masterDataServiceCreateShippingLine`（93；POST；`CreateShippingLineResponse`）<br>`masterDataServiceUpdateShippingLine`（105；PUT；`UpdateShippingLineResponse`） |
| `web/src/pages/orders/abnormal-case-panel.tsx` | `orderAbnormalCaseServiceResolveAbnormalCase`（188；POST；`ResolveAbnormalCaseResponse`）<br>`orderAbnormalCaseServiceRemoveAbnormalCase`（209；DELETE；`RemoveAbnormalCaseResponse`）<br>`orderAbnormalCaseServiceMarkAbnormalCase`（302；POST；`MarkAbnormalCaseResponse`） |
| `web/src/pages/orders/components/PartnerQuickAddSelect.tsx` | `partnerServiceCreatePartner`（356；POST；`CreatePartnerResponse`） |
| `web/src/pages/orders/components/detail/OrderLockControl.tsx` | `orderLockServiceLockOrder`（210；POST；`LockOrderResponse`）<br>`orderLockServiceRequestOrderUnlock`（257；POST；`RequestOrderUnlockResponse`） |
| `web/src/pages/orders/components/drawers/AttachmentDrawer.tsx` | `orderAttachmentServiceRemoveAttachmentReference`（105；DELETE；`RemoveAttachmentReferenceResponse`）<br>`orderAttachmentServiceRegisterAttachment`（146；POST；`RegisterAttachmentResponse`） |
| `web/src/pages/orders/components/drawers/CargoItemDrawer.tsx` | `orderCargoItemServiceAddCargoItem`（111；POST；`AddCargoItemResponse`）<br>`orderCargoItemServiceUpdateCargoItem`（128；PUT；`UpdateCargoItemResponse`）<br>`orderCargoItemServiceRemoveCargoItem`（156；DELETE；`RemoveCargoItemResponse`） |
| `web/src/pages/orders/components/drawers/ContainerDrawer.tsx` | `orderContainerServiceAddContainer`（136；POST；`AddContainerResponse`）<br>`orderContainerServiceUpdateContainer`（151；PUT；`UpdateContainerResponse`）<br>`orderContainerServiceRemoveContainer`（177；DELETE；`RemoveContainerResponse`） |
| `web/src/pages/orders/components/drawers/MilestoneDrawer.tsx` | `orderMilestoneServiceSetMilestone`（114；PUT；`SetMilestoneResponse`）<br>`orderMilestoneServiceSetMilestone`（132；PUT；`SetMilestoneResponse`） |
| `web/src/pages/orders/components/drawers/PersonnelDrawer.tsx` | `orderPersonnelServiceAssignPersonnel`（98；POST；`AssignPersonnelResponse`）<br>`orderPersonnelServiceRemovePersonnel`（108；DELETE；`RemovePersonnelResponse`） |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx` | `seaOrderChangeServicePreviewSeaOrderReassignment`（175；POST；`PreviewSeaOrderReassignmentResponse`）<br>`seaOrderChangeServiceExecuteSeaOrderReassignment`（410；POST；`ExecuteSeaOrderReassignmentResponse`） |
| `web/src/pages/orders/components/drawers/SeaSharedContainerDrawer.tsx` | `seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft`（348；PUT；`SaveSeaSharedContainerAllocationsDraftResponse`）<br>`seaSharedContainerServiceConfirmSeaSharedContainer`（383；POST；`ConfirmSeaSharedContainerResponse`）<br>`seaSharedContainerServiceWithdrawSeaSharedContainer`（414；POST；`WithdrawSeaSharedContainerResponse`）<br>`seaSharedContainerServiceDeleteSeaSharedContainer`（444；DELETE；`DeleteSeaSharedContainerResponse`）<br>`seaSharedContainerServiceCreateSeaSharedContainer`（475；POST；`CreateSeaSharedContainerResponse`） |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx` | `seaOrderChangeServicePreviewSeaTransportExecutionUpdate`（129；POST；`PreviewSeaTransportExecutionUpdateResponse`）<br>`seaOrderChangeServiceExecuteSeaTransportExecutionUpdate`（164；POST；`ExecuteSeaTransportExecutionUpdateResponse`） |
| `web/src/pages/orders/components/drawers/ShippingDocumentDrawer.tsx` | `orderShippingDocumentServiceTransitionShippingDocumentStatus`（198；POST；`TransitionShippingDocumentStatusResponse`）<br>`orderShippingDocumentServiceTransitionShippingDocumentStatus`（228；POST；`TransitionShippingDocumentStatusResponse`）<br>`orderShippingDocumentServiceRemoveShippingDocument`（258；DELETE；`RemoveShippingDocumentResponse`）<br>`orderShippingDocumentServiceUpdateShippingDocument`（372；PUT；`UpdateShippingDocumentResponse`）<br>`orderShippingDocumentServiceAddShippingDocument`（389；POST；`AddShippingDocumentResponse`） |
| `web/src/pages/orders/components/fees/FeeCorrectionCreate.tsx` | `orderFeeServiceCreateOrderFeeCorrection`（52；POST；`CreateOrderFeeCorrectionResponse`） |
| `web/src/pages/orders/components/fees/FeeSupplementSection.tsx` | `orderFeeServiceCreateOrderFeeSupplement`（251；POST；`CreateOrderFeeSupplementResponse`）<br>`orderFeeServiceApproveOrderFeeSupplement`（323；POST；`ApproveOrderFeeSupplementResponse`）<br>`orderFeeServiceRejectOrderFeeSupplement`（376；POST；`RejectOrderFeeSupplementResponse`）<br>`orderFeeServiceWithdrawOrderFeeSupplement`（431；POST；`WithdrawOrderFeeSupplementResponse`）<br>`orderFeeServiceWithdrawOrderFeeSupplement`（493；POST；`WithdrawOrderFeeSupplementResponse`）<br>`orderFeeServiceCancelApprovedOrderFeeSupplement`（578；POST；`CancelApprovedOrderFeeSupplementResponse`） |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | `orderFeeServiceAddFee`（869；POST；`AddFeeResponse`）<br>`orderFeeServiceUpdateFee`（880；PUT；`UpdateFeeResponse`）<br>`orderFeeServiceBatchAssignOrderFeeTags`（1698；POST；`BatchAssignOrderFeeTagsResponse`）<br>`orderFeeServiceBatchRemoveOrderFeeTags`（1706；POST；`BatchRemoveOrderFeeTagsResponse`）<br>`orderFeeServiceBulkUpdateOrderFees`（1748；POST；`BulkUpdateOrderFeesResponse`）<br>`orderFeeServiceBulkUpdateOrderFees`（1807；POST；`BulkUpdateOrderFeesResponse`）<br>`orderFeeServiceBulkRemoveOrderFees`（1869；POST；`BulkRemoveOrderFeesResponse`） |
| `web/src/pages/orders/components/fees/QuickAddFeeModal.tsx` | `feeCatalogServiceCreateFeeSetting`（65；POST；`CreateFeeSettingResponse`） |
| `web/src/pages/orders/components/fees/QuickAddPartnerModal.tsx` | `partnerServiceCreatePartner`（55；POST；`CreatePartnerResponse`） |
| `web/src/pages/orders/components/fees/fee-review/BatchFeeReviewModal.tsx` | `orderFeeServicePreviewOrderFeeReviews`（57；POST；`PreviewOrderFeeReviewsResponse`） |
| `web/src/pages/orders/detail.tsx` | `orderServiceUpdateOrder`（382；PUT；`UpdateOrderResponse`） |
| `web/src/pages/orders/fees.tsx` | `orderFeeServiceUpdateFee`（447；PUT；`UpdateFeeResponse`）<br>`orderFeeServiceAddFee`（459；POST；`AddFeeResponse`）<br>`orderFeeServiceRemoveFee`（499；DELETE；`RemoveFeeResponse`） |
| `web/src/pages/orders/list.tsx` | `orderTagServiceBatchAssignOrderTags`（362；POST；`BatchAssignOrderTagsResponse`）<br>`orderTagServiceBatchRemoveOrderTags`（370；POST；`BatchRemoveOrderTagsResponse`） |
| `web/src/pages/orders/new.tsx` | `orderServiceCreateOrder`（229；POST；`CreateOrderResponse`） |
| `web/src/pages/orders/order-detail-transitions.tsx` | `orderServiceTransitionOrderStatus`（74；POST；`TransitionOrderStatusResponse`）<br>`orderServiceTransitionOrderTermination`（160；POST；`TransitionOrderTerminationResponse`）<br>`orderServiceTransitionOrderClosure`（232；POST；`TransitionOrderClosureResponse`） |
| `web/src/pages/orders/order-fee-panel.tsx` | `orderFeeServiceRemoveFee`（263；DELETE；`RemoveFeeResponse`）<br>`orderFeeServiceUpdateFee`（319；PUT；`UpdateFeeResponse`）<br>`orderFeeServiceAddFee`（340；POST；`AddFeeResponse`）<br>`orderFeeServiceBatchAssignOrderFeeTags`（503；POST；`BatchAssignOrderFeeTagsResponse`）<br>`orderFeeServiceBatchRemoveOrderFeeTags`（510；POST；`BatchRemoveOrderFeeTagsResponse`） |
| `web/src/pages/orders/release-pod-panel.tsx` | `orderReleasePodServiceTransitionReleasePodStatus`（328；POST；`TransitionReleasePodStatusResponse`）<br>`orderReleasePodServiceRemoveReleasePod`（355；DELETE；`RemoveReleasePodResponse`）<br>`orderReleasePodServiceUpdateReleasePod`（485；PUT；`UpdateReleasePodResponse`）<br>`orderReleasePodServiceAddReleasePod`（492；POST；`AddReleasePodResponse`） |
| `web/src/pages/orders/split.tsx` | `seaOrderChangeServicePreviewSeaOrderSplit`（589；POST；`PreviewSeaOrderSplitResponse`）<br>`seaOrderChangeServiceExecuteSeaOrderSplit`（803；POST；`ExecuteSeaOrderSplitResponse`） |
| `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | `seaDocumentServicePreviewSeaDocumentAmendment`（398；POST；`PreviewSeaDocumentAmendmentResponse`）<br>`seaDocumentServicePreviewSeaDocumentVoid`（405；POST；`PreviewSeaDocumentVoidResponse`）<br>`seaDocumentServiceExecuteSeaDocumentAmendment`（427；POST；`ExecuteSeaDocumentAmendmentResponse`）<br>`seaDocumentServiceExecuteSeaDocumentVoid`（438；POST；`ExecuteSeaDocumentVoidResponse`） |
| `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx` | `seaDocumentServiceUpdateSeaHouseBill`（376；PUT；`UpdateSeaHouseBillResponse`）<br>`seaDocumentServiceUpdateSeaMasterBillContent`（402；PUT；`UpdateSeaMasterBillContentResponse`）<br>`seaDocumentServicePreviewChangeSeaDocumentMode`（449；POST；`PreviewChangeSeaDocumentModeResponse`）<br>`seaDocumentServiceExecuteChangeSeaDocumentMode`（482；POST；`ExecuteChangeSeaDocumentModeResponse`） |
| `web/src/pages/partners/components/ContractCardList.tsx` | `partnerServiceUpdatePartnerContract`（134；PUT；`UpdatePartnerContractResponse`）<br>`partnerServiceUpdatePartnerContract`（168；PUT；`UpdatePartnerContractResponse`）<br>`partnerServiceCreatePartnerContract`（187；POST；`CreatePartnerContractResponse`） |
| `web/src/pages/partners/components/PartnerExcelImportModal.tsx` | `partnerServiceImportPartners`（265；POST；`ImportPartnersResponse`） |
| `web/src/pages/partners/components/RoleSwitchModal.tsx` | `partnerServiceUpdatePartner`（83；PUT；`UpdatePartnerResponse`） |
| `web/src/pages/partners/components/ShippingPresetSection.tsx` | `partnerServiceUpdatePartnerShippingPreset`（209；PUT；`UpdatePartnerShippingPresetResponse`）<br>`partnerServiceUpdatePartnerShippingPreset`（267；PUT；`UpdatePartnerShippingPresetResponse`）<br>`partnerServiceCreatePartnerShippingPreset`（278；POST；`CreatePartnerShippingPresetResponse`） |
| `web/src/pages/partners/components/secondary/AccountsPanel.tsx` | `partnerServiceUpdatePartnerAccount`（233；PUT；`UpdatePartnerAccountResponse`）<br>`partnerServiceCreatePartnerAccount`（240；POST；`CreatePartnerAccountResponse`） |
| `web/src/pages/partners/components/secondary/AttachmentsPanel.tsx` | `partnerServiceRegisterPartnerAttachment`（161；POST；`RegisterPartnerAttachmentResponse`） |
| `web/src/pages/partners/components/secondary/ContractsPanel.tsx` | `partnerServiceUpdatePartnerContract`（226；PUT；`UpdatePartnerContractResponse`）<br>`partnerServiceCreatePartnerContract`（238；POST；`CreatePartnerContractResponse`） |
| `web/src/pages/partners/components/secondary/InvoiceProfilesPanel.tsx` | `partnerServiceUpdatePartnerInvoiceProfile`（198；PUT；`UpdatePartnerInvoiceProfileResponse`）<br>`partnerServiceCreatePartnerInvoiceProfile`（211；POST；`CreatePartnerInvoiceProfileResponse`） |
| `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx` | `partnerServiceUpdatePartnerSettlementRule`（400；PUT；`UpdatePartnerSettlementRuleResponse`）<br>`partnerServiceCreatePartnerSettlementRule`（417；POST；`CreatePartnerSettlementRuleResponse`） |
| `web/src/pages/partners/index.tsx` | `partnerServiceSetPartnerRoleBlacklist`（680；POST；`SetPartnerRoleBlacklistResponse`） |
| `web/src/pages/partners/partner-detail.tsx` | `partnerServiceUpdatePartner`（621；PUT；`UpdatePartnerResponse`）<br>`partnerServiceCreatePartner`（647；POST；`CreatePartnerResponse`） |
| `web/src/pages/user/login/components/organization-picker.tsx` | `authServiceSwitchOrganization`（64；POST；`SwitchOrganizationResponse`） |
| `web/src/pages/user/login/dingtalk-callback.tsx` | `authServiceDingTalkLogin`（46；POST；`DingTalkLoginResponse`） |
| `web/src/pages/user/login/index.tsx` | `authServiceLogin`（60；POST；`LoginResponse`） |
| `web/src/pages/workbench/MyApplicationCard.tsx` | `workbenchServiceSubmitMyCommissionApplication`（122；POST；`SubmitMyCommissionApplicationResponse`） |
| `web/src/pages/workbench/MyApplicationHistoryDrawer.tsx` | `workbenchServiceResubmitMyCommissionApplication`（174；POST；`ResubmitMyCommissionApplicationResponse`） |

## 验证证据

- 初轮写调用相关增量测试发现旧成功夹具缺失 success/ID；修复真实契约夹具后，15 文件 80 测试通过。未减少业务断言或跳过失败。
- 真实 axios 请求层与实际组件：重置密码、角色表单、客商角色、账单取消、对冲取消，模拟 409 被全局消费后得到 undefined；验证无成功反馈/刷新、原因或输入保留，补齐缺 ID 和成功重试。账单/对冲使用真实 Ant Design 确认框。
- 特殊 DTO 与相邻路径：锁定、海运改配、共享集装箱、航次调整、拆票、费用更正、提成创建 7 文件 38 测试通过（`/tmp/r1-final-special.log` 中这 7 文件全部 PASS；同轮新确认组件的测试选择器问题随后单独修正并通过）。
- `AsyncPopconfirm.test.tsx` 真实组件验证等待 Promise 时 loading，false/reject 保持弹层且不产生未处理拒绝，重试成功才关闭。最终专项日志 `/tmp/r1-popconfirm-final.log` 为 PASS。
- 全仓库 `pnpm --dir web tsc` 通过；当前 133 个修改/新增 TypeScript 文件 Biome lint 与 `git diff --check` 通过；架构规则 18 测试通过，446 个产品文件扫描无边界违规。全量门禁、数据库与最终浏览器验收由主会话统一运行。

### 全量门禁后的测试替身修正

首轮 `check:fast` 的前端失败集中为 2 文件 4 测试：`roles.test.tsx` 和 `UserFormModal.test.tsx` 对公共 UI 模块的替身遗漏新增 `AsyncPopconfirm`，导致渲染异常；后者的 `membership-m-system` 缺失是该异常的后果。两文件补用实际 `AsyncPopconfirm` 导出，保留已有 Ant Design 浅层替身及角色/成员关系业务断言，未修改产品实现。

定向运行两文件 **14 个测试全部通过**，无 SKIP；日志 `/tmp/r1-final-mocks.log`。两个文件 Biome lint 和 `git diff --check` 通过。检查代理另行负责 7 条本地 catch 的失败传播与真实弹层回归，详见独立复核记录；不将其尚未完成的结果预记为通过。

## 保留边界

全局错误已经反馈的 undefined 分支不追加技术错误；服务端实际落库结果不能由刷新失败推翻。身份切换的旧响应保护由本任务 R2 工作区边界负责。审计覆盖前端现有调用及间接模板，没有增加新接口、兼容分支或后端业务能力。
