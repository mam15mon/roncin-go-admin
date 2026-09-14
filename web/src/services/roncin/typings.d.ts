declare namespace API {
  type AddCargoItemRequest = {
    orderId: string;
    cargoName: string;
    packageCount: number;
    grossWeightKg: number;
    volumeCbm: number;
    netWeightKg?: number;
    note?: string;
  };

  type AddCargoItemResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderCargoItem;
    traceId?: string;
  };

  type AddContainerRequest = {
    orderId: string;
    containerNo: string;
    containerSpecId: string;
    sealNo?: string;
    grossWeightKg: number;
    volumeCbm: number;
    note?: string;
    packageCount: number;
  };

  type AddContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer;
    traceId?: string;
  };

  type AddFeeRequest = {
    orderId: string;
    direction: number;
    settlementPartyId: string;
    quantity: string;
    unitPrice: string;
    currency: string;
    expenseDate: string;
    note?: string;
    exchangeRateOverride?: string;
    feeSettingId: string;
    billingUnitId: string;
    idempotencyKey: string;
    taxInclusive?: boolean;
  };

  type AddFeeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderFee;
    traceId?: string;
  };

  type AddReleasePodRequest = {
    orderId: string;
    shippingDocumentId?: string;
    releaseNo?: string;
    podNo?: string;
    note?: string;
    seaDocumentType?: number;
    seaDocumentId?: string;
  };

  type AddReleasePodResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod;
    traceId?: string;
  };

  type AddShippingDocumentRequest = {
    orderId: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
  };

  type AddShippingDocumentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument;
    traceId?: string;
  };

  type AdminAuditLog = {
    id?: string;
    organizationId?: string;
    userId?: string;
    action?: string;
    resourceType?: string;
    resourceId?: string;
    result?: string;
    requestId?: string;
    traceId?: string;
    ipAddress?: string;
    details?: Record<string, any>;
    createdAt?: string;
    actorDisplayName?: string;
    targetDisplayName?: string;
  };

  type AdministrativeRegion = {
    id?: string;
    code?: string;
    name?: string;
    level?: number;
    parentCode?: string;
    regionType?: string;
    source?: string;
    sourceVersion?: string;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type AdminOrganization = {
    id?: string;
    code?: string;
    name?: string;
    parentId?: string;
    enabled?: boolean;
    kind?: number;
    baseCurrency?: string;
  };

  type AdminPermission = {
    key?: string;
    name?: string;
    group?: string;
    description?: string;
    /** 勾选该权限时必须同时具备的基础权限码（如编辑类权限依赖对应查看权限）。 */
    requires?: string[];
  };

  type AdminRole = {
    id?: string;
    organizationId?: string;
    code?: string;
    name?: string;
    dataScope?: number;
    enabled?: boolean;
    permissionKeys?: string[];
    createdAt?: string;
    updatedAt?: string;
  };

  type AdminServiceApproveDingTalkRegistrationParams = {
    /** 注册用户的 ID（待审批队列中的 user_id）。 */
    id: string;
  };

  type AdminServiceAuthorizeDingTalkUserParams = {
    id: string;
  };

  type AdminServiceAuthorizeWeComUserParams = {
    id: string;
  };

  type AdminServiceCreateUserMembershipParams = {
    userId: string;
  };

  type AdminServiceDeleteUserMembershipParams = {
    userId: string;
    id: string;
  };

  type AdminServiceGetDingTalkInvitationParams = {
    id: string;
  };

  type AdminServiceListAuditLogsParams = {
    page?: number;
    pageSize?: number;
    action?: string;
    userId?: string;
    startTime?: string;
    endTime?: string;
    resourceType?: string;
    resourceId?: string;
  };

  type AdminServiceListDingTalkInvitationsParams = {
    page?: number;
    pageSize?: number;
    /** 按组织过滤；必须是调用者可写范围内的组织。 */
    organizationId?: string;
    status?: number;
  };

  type AdminServiceListDingTalkRegistrationsParams = {
    page?: number;
    pageSize?: number;
  };

  type AdminServiceListOrganizationRolesParams = {
    organizationId: string;
  };

  type AdminServiceListUserMembershipsParams = {
    userId: string;
  };

  type AdminServiceListUsersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type AdminServiceRejectDingTalkRegistrationParams = {
    id: string;
  };

  type AdminServiceResetUserPasswordParams = {
    id: string;
  };

  type AdminServiceRevokeDingTalkInvitationParams = {
    id: string;
  };

  type AdminServiceTerminateUserParams = {
    id: string;
  };

  type AdminServiceTransferDingTalkRegistrationParams = {
    userId: string;
  };

  type AdminServiceUpdateOrganizationParams = {
    id: string;
  };

  type AdminServiceUpdateRoleParams = {
    id: string;
  };

  type AdminServiceUpdateUserMembershipParams = {
    userId: string;
    id: string;
  };

  type AdminServiceUpdateUserParams = {
    id: string;
  };

  type AdminUser = {
    id?: string;
    username?: string;
    displayName?: string;
    email?: string;
    enabled?: boolean;
    roleIds?: string[];
    roleCodes?: string[];
    createdAt?: string;
    updatedAt?: string;
    wecomUserid?: string;
    wecomName?: string;
    dingtalkUnionid?: string;
    dingtalkName?: string;
    avatarUrl?: string;
    hasPassword?: boolean;
    dingtalkUserid?: string;
    status?: number;
    currentMembershipEnabled?: boolean;
    organizations?: AdminUserOrganizationSummary[];
  };

  type AdminUserMembership = {
    id?: string;
    userId?: string;
    organizationId?: string;
    organizationCode?: string;
    organizationName?: string;
    organizationKind?: number;
    primary?: boolean;
    enabled?: boolean;
    roleIds?: string[];
    roleCodes?: string[];
    roleNames?: string[];
    createdAt?: string;
    updatedAt?: string;
  };

  type AdminUserOrganizationSummary = {
    organizationId?: string;
    organizationName?: string;
    primary?: boolean;
  };

  type Airline = {
    id?: string;
    organizationId?: string;
    iataCode?: string;
    icaoCode?: string;
    awbPrefix?: string;
    nameZh?: string;
    nameEn?: string;
    countryCode?: string;
    cargoOnly?: boolean;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
    sourceVersion?: string;
    sourceHash?: string;
  };

  type Airport = {
    id?: string;
    organizationId?: string;
    iataCode?: string;
    icaoCode?: string;
    nameZh?: string;
    nameEn?: string;
    cityNameZh?: string;
    cityNameEn?: string;
    countryCode?: string;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
    sourceVersion?: string;
    sourceHash?: string;
  };

  type ApproveDingTalkRegistrationRequest = {
    /** 注册用户的 ID（待审批队列中的 user_id）。 */
    id: string;
    /** 同意时授予的初始角色，必须属于注册的目标组织。 */
    roleIds: string[];
  };

  type ApproveDingTalkRegistrationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkRegistration;
    traceId?: string;
  };

  type AssignPersonnelRequest = {
    orderId: string;
    userId: string;
    role: number;
    organizationId: string;
  };

  type AssignPersonnelResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderPersonnel;
    traceId?: string;
  };

  type AuthorizeDingTalkUserRequest = {
    id: string;
    organizationId: string;
    displayName: string;
    email?: string;
    roleIds: string[];
  };

  type AuthorizeDingTalkUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser;
    traceId?: string;
  };

  type AuthorizeWeComUserRequest = {
    id: string;
    organizationId: string;
    displayName: string;
    email?: string;
    roleIds: string[];
  };

  type AuthorizeWeComUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser;
    traceId?: string;
  };

  type AuthServiceGetDingTalkInvitationInfoParams = {
    /** 专属邀请 128-bit Token */
    token?: string;
  };

  type BackgroundTask = {
    id?: string;
    kind?: number;
    idempotencyKey?: string;
    status?: number;
    attempts?: number;
    maxAttempts?: number;
    nextRunAt?: string;
    lastError?: string;
    createdAt?: string;
    updatedAt?: string;
    recipientDisplayName?: string;
    recipientUserId?: string;
  };

  type BackgroundTaskServiceGetBackgroundTaskParams = {
    id: string;
  };

  type BackgroundTaskServiceListBackgroundTasksParams = {
    page?: number;
    pageSize?: number;
    status?: number;
    kind?: number;
    startTime?: string;
    endTime?: string;
    phase?: number;
  };

  type BackgroundTaskServiceRequeueBackgroundTaskParams = {
    id: string;
  };

  type BatchAssignAddressTypesRequest = {
    resourceIds: string[];
    addressTypes: number[];
  };

  type BatchAssignAddressTypesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchAssignAssigneesRequest = {
    resourceIds: string[];
    assigneeIds: string[];
  };

  type BatchAssignAssigneesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchAssignFinanceBillTagsRequest = {
    billIds: string[];
    tagIds: string[];
  };

  type BatchAssignFinanceBillTagsResponse = {
    assignedCount?: number;
    traceId?: string;
  };

  type BatchAssignFinanceFeeTagsRequest = {
    feeIds: string[];
    tagIds: string[];
    organizationId: string;
  };

  type BatchAssignFinanceFeeTagsResponse = {
    assignedCount?: number;
    traceId?: string;
  };

  type BatchAssignOrderFeeTagsRequest = {
    orderId: string;
    feeIds: string[];
    tagIds: string[];
  };

  type BatchAssignOrderFeeTagsResponse = {
    assignedCount?: number;
    traceId?: string;
  };

  type BatchAssignOrderTagsRequest = {
    businessType: number;
    orderIds: string[];
    tagIds: string[];
  };

  type BatchAssignOrderTagsResponse = {
    assignedCount?: number;
    traceId?: string;
  };

  type BatchCreateAssociationsRequest = {
    resourceIds: string[];
    partnerIds: string[];
  };

  type BatchCreateAssociationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchDeleteAssociationsRequest = {
    resourceIds: string[];
    partnerIds: string[];
  };

  type BatchDeleteAssociationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchRemoveAddressTypesRequest = {
    resourceIds: string[];
    addressTypes: number[];
  };

  type BatchRemoveAddressTypesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchRemoveAssigneesRequest = {
    resourceIds: string[];
    assigneeIds: string[];
  };

  type BatchRemoveAssigneesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    affectedCount?: number;
    traceId?: string;
  };

  type BatchRemoveFinanceBillTagsRequest = {
    billIds: string[];
    tagIds: string[];
  };

  type BatchRemoveFinanceBillTagsResponse = {
    removedCount?: number;
    traceId?: string;
  };

  type BatchRemoveFinanceFeeTagsRequest = {
    feeIds: string[];
    tagIds: string[];
    organizationId: string;
  };

  type BatchRemoveFinanceFeeTagsResponse = {
    removedCount?: number;
    traceId?: string;
  };

  type BatchRemoveOrderFeeTagsRequest = {
    orderId: string;
    feeIds: string[];
    tagIds: string[];
  };

  type BatchRemoveOrderFeeTagsResponse = {
    removedCount?: number;
    traceId?: string;
  };

  type BatchRemoveOrderTagsRequest = {
    businessType: number;
    orderIds: string[];
    tagIds: string[];
  };

  type BatchRemoveOrderTagsResponse = {
    removedCount?: number;
    traceId?: string;
  };

  type BillBatchNettingPair = {
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    receivableGrossAmount?: string;
    payableGrossAmount?: string;
    offsetAmount?: string;
    netReceivableAmount?: string;
    netPayableAmount?: string;
  };

  type BillBatchPreviewGroup = {
    groupKey?: string;
    direction?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    baseCurrency?: string;
    orderId?: string;
    orderNo?: string;
    taxRate?: string;
    fees?: FeeLedgerItem[];
    totalAmount?: string;
    netAmount?: string;
    taxAmount?: string;
    baseCurrencyAmount?: string;
    isTemporaryBillDate?: boolean;
    configurationComplete?: boolean;
    estimatedInvoiceCurrency?: string;
    estimatedInvoiceRate?: string;
    estimatedInvoiceAmount?: string;
    isCasual?: boolean;
    defaultPaymentTermsDays?: number;
    /** 信用额度比对信息（折本位币口径）：额度来自客户角色激活结算规则，余额来自已确认应收账单未核销总额。 */
    creditLimitAmount?: string;
    creditCurrency?: string;
    currentUnsettledAmount?: string;
    isCreditExceeded?: boolean;
  };

  type BillBatchPreviewGroupConfigInput = {
    groupKey: string;
    billDate?: string;
    settlementAccountId?: string;
    estimatedInvoiceCurrency?: string;
    estimatedInvoiceRate?: string;
  };

  type BilledFeeEditPolicy = {
    organizationId?: string;
    enabled?: boolean;
    editableFields?: number[];
    /** 未保存过策略时为 0；首次保存需携带 expected_version=0。 */
    version?: string;
    updatedAt?: string;
    updatedBy?: string;
  };

  type BillExpectedVersion = {
    billId: string;
    expectedVersion: string;
  };

  type BillGroupingPolicy = {
    splitByOrder?: boolean;
    splitByTaxRate?: boolean;
    mode?: number;
  };

  type BillingUnit = {
    id?: string;
    organizationId?: string;
    code?: string;
    name?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
    isContainerUnit?: boolean;
  };

  type BusinessTagSummary = {
    id?: string;
    name?: string;
    groupId?: string;
    groupName?: string;
    groupColor?: string;
    enabled?: boolean;
  };

  type BusinessTagSummary = {
    id?: string;
    name?: string;
    groupId?: string;
    groupName?: string;
    groupColor?: string;
    enabled?: boolean;
  };

  type CancelBillRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill;
    traceId?: string;
  };

  type CancelCashflowRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelCashflowResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCashflow;
    traceId?: string;
  };

  type CancelCommissionAdjustmentRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelCommissionAdjustmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionAdjustment;
    traceId?: string;
  };

  type CancelCommissionRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission;
    traceId?: string;
  };

  type CancelInvoiceRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelInvoiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice;
    traceId?: string;
  };

  type CancelNettingRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type CancelNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting;
    traceId?: string;
  };

  type CheckOrderReferenceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReferenceCheck;
    traceId?: string;
  };

  type CommissionCalculation = {
    verificationId?: string;
    verificationNo?: string;
    employeeId?: string;
    employeeName?: string;
    ruleId?: string;
    ruleName?: string;
    personnelRole?: string;
    calculationBasis?: string;
    ruleVersion?: string;
    calculationVersion?: string;
    baseCurrency?: string;
    realizedRevenue?: string;
    allocatedCost?: string;
    realizedProfit?: string;
    ratePercent?: string;
    commissionAmount?: string;
    lines?: FinanceCommissionLine[];
    customerCount?: number;
    orderCount?: number;
    feeCount?: number;
    commissionBaseAmount?: string;
    cnyExchangeRate?: string;
    cnyExchangeRateSource?: string;
    cnyExchangeRateDate?: string;
    cnyExchangeRateSettingId?: string;
    cnyCommissionAmount?: string;
    nettingId?: string;
    nettingNo?: string;
  };

  type CommissionCandidateSummary = {
    employeeId?: string;
    employeeName?: string;
    personnelRole?: string;
    customerCount?: number;
    orderCount?: number;
    feeCount?: number;
    baseCurrency?: string;
    realizedRevenue?: string;
    allocatedCost?: string;
    realizedProfit?: string;
    commissionBaseAmount?: string;
    ratePercent?: string;
    commissionAmount?: string;
    id?: string;
    displayName?: string;
  };

  type CommissionEmployeeOption = {
    id?: string;
    displayName?: string;
  };

  type CommissionExportItem = {
    commissionNo?: string;
    status?: number;
    verificationNo?: string;
    commissionDate?: string;
    employeeName?: string;
    personnelRole?: string;
    ruleName?: string;
    calculationBasis?: string;
    ratePercent?: string;
    baseCurrency?: string;
    createdAt?: string;
    commissionAmount?: string;
    cnyCommissionAmount?: string;
    adjustmentAmount?: string;
    cnyAdjustmentAmount?: string;
    effectiveCommissionAmount?: string;
    cnyEffectiveCommissionAmount?: string;
    organizationId?: string;
    organizationName?: string;
    nettingNo?: string;
  };

  type CommissionFeeDetail = {
    feeId?: string;
    direction?: string;
    feeCode?: string;
    feeName?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    totalAmount?: string;
    exchangeRate?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    expenseDate?: string;
    status?: number;
  };

  type CommissionRuleInput = {
    name: string;
    personnelRole: string;
    calculationBasis: string;
    ratePercent: string;
    effectiveFrom?: string;
    effectiveTo?: string;
    enabled?: boolean;
    note?: string;
  };

  type CommitEnterpriseResourceImportRequest = {
    resourceType: number;
    rows: EnterpriseResourceInput[];
    overwriteConflicts?: boolean;
  };

  type CommitEnterpriseResourceImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    rows?: EnterpriseResourceImportRow[];
    validCount?: number;
    invalidCount?: number;
    createdCount?: number;
    traceId?: string;
    conflictCount?: number;
    updatedCount?: number;
  };

  type ConfirmBillBatchRequest = {
    id: string;
    bills: BillExpectedVersion[];
  };

  type ConfirmBillBatchResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBillBatch;
    traceId?: string;
  };

  type ConfirmBillRequest = {
    id: string;
    expectedVersion: string;
  };

  type ConfirmBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill;
    traceId?: string;
  };

  type ConfirmCashflowRequest = {
    id: string;
    expectedVersion: string;
  };

  type ConfirmCashflowResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCashflow;
    traceId?: string;
  };

  type ConfirmCommissionAdjustmentRequest = {
    id: string;
    expectedVersion: string;
  };

  type ConfirmCommissionAdjustmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionAdjustment;
    traceId?: string;
  };

  type ConfirmCommissionRequest = {
    id: string;
    expectedVersion: string;
  };

  type ConfirmCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission;
    traceId?: string;
  };

  type ConfirmExchangeRateImportRequest = {
    previewToken: string;
    idempotencyKey: string;
  };

  type ConfirmExchangeRateImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateImportBatch;
    traceId?: string;
  };

  type ConfirmFeeRequest = {
    orderId: string;
    id: string;
    expectedVersion: string;
  };

  type ConfirmFeeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderFee;
    traceId?: string;
  };

  type ConfirmNettingRequest = {
    id: string;
    expectedVersion: string;
  };

  type ConfirmNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting;
    traceId?: string;
  };

  type ConfirmSeaSharedContainerRequest = {
    id: string;
    expectedVersion: string;
    /** 携带本次确认的分配输入，服务端在同一事务内保存并严格守恒确认，避免两步请求部分成功 */
    allocations?: SeaSharedContainerAllocationInput[];
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId: string;
  };

  type ConfirmSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };

  type CreateAirlineRequest = {
    iataCode: string;
    icaoCode?: string;
    awbPrefix?: string;
    nameZh?: string;
    nameEn: string;
    countryCode: string;
    cargoOnly?: boolean;
    source?: string;
    sortOrder?: number;
  };

  type CreateAirlineResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airline;
    traceId?: string;
  };

  type CreateAirportRequest = {
    iataCode: string;
    icaoCode?: string;
    nameZh: string;
    nameEn: string;
    cityNameZh: string;
    cityNameEn?: string;
    countryCode: string;
    sortOrder?: number;
  };

  type CreateAirportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airport;
    traceId?: string;
  };

  type CreateBillBatchGroupInput = {
    groupKey: string;
    statementTitle: string;
    billDate: string;
    dueDate?: string;
    paymentTermsDays?: number;
    note?: string;
    settlementAccountId: string;
    estimatedInvoiceCurrency?: string;
    estimatedInvoiceRate?: string;
  };

  type CreateBillBatchRequest = {
    feeIds: string[];
    groupingPolicy: BillGroupingPolicy;
    groups: CreateBillBatchGroupInput[];
    previewToken: string;
    idempotencyKey: string;
    organizationId: string;
  };

  type CreateBillBatchResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBillBatch;
    traceId?: string;
  };

  type CreateBillingUnitRequest = {
    code: string;
    name: string;
    sortOrder?: number;
    isContainerUnit?: boolean;
  };

  type CreateBillingUnitResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillingUnit;
    traceId?: string;
  };

  type CreateBillRequest = {
    feeIds: string[];
    billDate: string;
    dueDate?: string;
    note?: string;
    idempotencyKey: string;
    statementTitle?: string;
    paymentTermsDays?: number;
    settlementAccountId: string;
  };

  type CreateBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill;
    traceId?: string;
  };

  type CreateCashflowRequest = {
    direction: string;
    settlementPartyId: string;
    currency: string;
    amount: string;
    exchangeRate?: string;
    baseCurrency?: string;
    transactionDate: string;
    ourAccount: string;
    counterpartyAccount?: string;
    paymentMethod: string;
    bankReferenceNo?: string;
    note?: string;
    idempotencyKey: string;
    organizationId: string;
  };

  type CreateCashflowResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCashflow;
    traceId?: string;
  };

  type CreateCommissionAdjustmentRequest = {
    commissionId: string;
    orderId: string;
    direction: string;
    amount: string;
    reason: string;
    note?: string;
    idempotencyKey: string;
  };

  type CreateCommissionAdjustmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionAdjustment;
    traceId?: string;
  };

  type CreateCommissionRequest = {
    verificationId?: string;
    employeeId: string;
    note?: string;
    idempotencyKey: string;
    ruleId: string;
    nettingId?: string;
  };

  type CreateCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission;
    traceId?: string;
  };

  type CreateCommissionRuleRequest = {
    rule: CommissionRuleInput;
    organizationId: string;
  };

  type CreateCommissionRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionRule;
    traceId?: string;
  };

  type CreateDingTalkInvitationRequest = {
    /** 目标组织；必须是调用者可写范围内的组织。 */
    organizationId: string;
    /** 邀请类型：TARGETED（定向单人免审码）或 GENERIC（通用入职审批码）。缺省为 TARGETED。 */
    kind?: number;
    /** 国内手机号；TARGETED 必填，GENERIC 留空。 */
    mobile?: string;
    /** 激活后授予的初始角色（属于目标组织；通用码可选，留空则审批时指定）。 */
    roleId?: string;
    /** 备注姓名，仅供管理员识别，账号身份以钉钉返回为准。 */
    displayName?: string;
    /** 有效期（小时）；缺省 168（7 天），允许 1-720。 */
    expiresInHours?: number;
  };

  type CreateDingTalkInvitationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkInvitation;
    invitationUrl?: string;
    traceId?: string;
  };

  type CreateEnterpriseResourceRequest = {
    resource: EnterpriseResourceInput;
  };

  type CreateEnterpriseResourceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResource;
    traceId?: string;
  };

  type CreateEnterpriseTagGroupRequest = {
    group: EnterpriseTagGroupInput;
  };

  type CreateEnterpriseTagGroupResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseTagGroup;
    traceId?: string;
  };

  type CreateExchangeRateSettingRequest = {
    fromCurrency: string;
    toCurrency: string;
    /** 示例：2026-08-27T09:30:00+08:00。 */
    effectiveFrom: string;
    effectiveTo?: string;
    rate: string;
  };

  type CreateExchangeRateSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateSetting;
    traceId?: string;
  };

  type CreateFeeSettingRequest = {
    feeCode: string;
    nameZh: string;
    nameEn?: string;
    aliasName?: string;
    serviceTypeId?: string;
    defaultCurrency: string;
    billingUnitId: string;
    abnormalCaseId?: string;
    taxRate: string;
    taxableServiceId: string;
    sortOrder?: number;
  };

  type CreateFeeSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeSetting;
    traceId?: string;
  };

  type CreateInvoiceRequest = {
    billIds: string[];
    invoiceType: string;
    note?: string;
    idempotencyKey: string;
    invoiceProfileId: string;
  };

  type CreateInvoiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice;
    traceId?: string;
  };

  type CreateItemRequest = {
    kind: number;
    code: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
    attributes?: MasterDataAttributes;
  };

  type CreateItemResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem;
    traceId?: string;
  };

  type CreateNettingRequest = {
    organizationId: string;
    bills: NettingBillExpectedVersion[];
    note?: string;
    idempotencyKey: string;
  };

  type CreateNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting;
    traceId?: string;
  };

  type CreateNumberRuleRequest = {
    documentType: number;
    prefix?: string;
    dateFormat: number;
    sequenceLength: number;
    resetPolicy: number;
  };

  type CreateNumberRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: NumberRule;
    traceId?: string;
  };

  type CreateOrderRequest = {
    customerId: string;
    businessType: number;
    tradeDirection: number;
    tradeTerm?: number;
    paymentTerm: number;
    shippingLineId?: string;
    bookingAgentId?: string;
    shipmentType?: number;
    containerOwnership?: number;
    shipmentMode?: number;
    serviceTypeIds?: string[];
    cargoCategoryIds?: string[];
    originLocationId?: string;
    destinationLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    vesselVoyage?: string;
    etd?: string;
    eta?: string;
    siCutoff?: string;
    docCutoff?: string;
    customsCutoff?: string;
    vgmCutoff?: string;
    goodsDescription?: string;
    totalPackages?: number;
    totalPackageUnit?: string;
    specialRequirements?: string;
    orderDate?: string;
    notes?: string;
    customerReferenceNo?: string;
    foreignAgentId?: string;
    contractNo?: string;
    cargoValue?: string;
    cargoCurrency?: string;
    internalReferenceNo?: string;
    shippingAgentId?: string;
    insurancePremium?: string;
    insuranceCurrency?: string;
    unNumber?: string;
    hazardClass?: string;
    factoryName?: string;
    cargoReadyAt?: string;
    receivedAt?: string;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    personnelAssignments?: OrderPersonnelAssignmentInput[];
    shippingDocuments?: OrderShippingDocumentInput[];
    containerRequests?: OrderContainerRequestInput[];
    declarationCutoffAt?: string;
    totalGrossWeightKg?: number;
    totalVolumeCbm?: number;
    shipperShortName?: string;
    consigneeShortName?: string;
    seaMasterBill?: SeaMasterBillInput;
    seaDocument?: SeaOrderDocumentInput;
    bookingNo?: string;
    /** idempotency_key 创建幂等键：可选，传入即启用幂等；同键同意图重放返回原单。 */
    idempotencyKey?: string;
  };

  type CreateOrderResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type CreateOrganizationRequest = {
    code: string;
    name: string;
    parentId: string;
    kind: number;
    baseCurrency?: string;
  };

  type CreateOrganizationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization;
    traceId?: string;
  };

  type CreatePartnerAccountRequest = {
    partnerId: string;
    account: PartnerAccountInput;
  };

  type CreatePartnerAccountResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAccount;
    traceId?: string;
  };

  type CreatePartnerContractInput = {
    contractNo: string;
    name: string;
    status: number;
    startDate: string;
    endDate: string;
    paymentTerms?: string;
    disputeResolution?: string;
    otherNotes?: string;
  };

  type CreatePartnerContractRequest = {
    partnerId: string;
    contract: CreatePartnerContractInput;
  };

  type CreatePartnerContractResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerContract;
    traceId?: string;
  };

  type CreatePartnerInvoiceProfileRequest = {
    partnerId: string;
    invoiceTitle: string;
    taxpayerIdentificationNo: string;
    registeredAddress?: string;
    registeredPhone?: string;
    bankName?: string;
    bankAccount?: string;
    defaultInvoiceType: string;
    isDefault?: boolean;
  };

  type CreatePartnerInvoiceProfileResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerInvoiceProfile;
    traceId?: string;
  };

  type CreatePartnerRequest = {
    /** 客商代码；选填搜索辅助字段，留空表示未设置，不做自动生成。 */
    code?: string;
    legalName: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    roles?: PartnerRoleInput[];
    contacts?: PartnerContactInput[];
    aliases?: PartnerAliasInput[];
    profile?: PartnerProfile;
    assignments?: PartnerAssignmentInput[];
    isCasual?: boolean;
  };

  type CreatePartnerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
    traceId?: string;
  };

  type CreatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    rule: PartnerSettlementRuleInput;
  };

  type CreatePartnerSettlementRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerSettlementRule;
    traceId?: string;
  };

  type CreatePartnerShippingPresetRequest = {
    partnerId: string;
    preset: PartnerShippingPresetInput;
  };

  type CreatePartnerShippingPresetResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerShippingPreset;
    traceId?: string;
  };

  type CreatePortRequest = {
    unLocode: string;
    nameZh: string;
    nameEn: string;
    countryCode: string;
    transportModes?: string[];
    sortOrder?: number;
  };

  type CreatePortResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Port;
    traceId?: string;
  };

  type CreateRoleRequest = {
    /** 角色编码可选；留空时由服务端自动生成机器标识，组织内唯一性由数据库唯一索引兜底。 */
    code?: string;
    name: string;
    dataScope: number;
    permissionKeys?: string[];
  };

  type CreateRoleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole;
    traceId?: string;
  };

  type CreateSeaSharedContainerRequest = {
    input: SeaSharedContainerInput;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文 */
    orderId: string;
  };

  type CreateSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };

  type CreateShippingLineRequest = {
    scacCode: string;
    nameZh: string;
    nameEn: string;
    countryCode: string;
    trackingUrl?: string;
    alliance?: string;
    containerPrefixes?: string[];
    source?: string;
    sortOrder?: number;
  };

  type CreateShippingLineResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ShippingLine;
    traceId?: string;
  };

  type CreateTaxableServiceRequest = {
    name: string;
    shortName?: string;
    goodsCode?: string;
    defaultTaxRate: string;
  };

  type CreateTaxableServiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: TaxableService;
    traceId?: string;
  };

  type CreateUserMembershipRequest = {
    userId: string;
    organizationId: string;
    roleIds?: string[];
    primary?: boolean;
  };

  type CreateUserMembershipResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUserMembership;
    traceId?: string;
  };

  type CreateUserRequest = {
    username: string;
    displayName: string;
    password: string;
    email?: string;
    roleIds?: string[];
  };

  type CreateUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser;
    traceId?: string;
  };

  type CreateVerificationRequest = {
    allocations: VerificationAllocationInput[];
    verificationDate: string;
    note?: string;
    idempotencyKey: string;
  };

  type CreateVerificationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceVerification;
    traceId?: string;
  };

  type CreditLimitControlPolicy = {
    organizationId?: string;
    allowSelectionWhenCreditExceeded?: boolean;
    /** 未保存过策略时为 0；首次保存需携带 expected_version=0。 */
    version?: string;
    updatedAt?: string;
    updatedBy?: string;
  };

  type Currency = {
    id?: string;
    code?: string;
    name?: string;
    symbol?: string;
    minorUnit?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type CurrentUser = {
    id?: string;
    username?: string;
    displayName?: string;
    email?: string;
    currentOrganization?: Organization;
    organizations?: Organization[];
    permissions?: string[];
    roleScopes?: RoleScope[];
    avatarUrl?: string;
  };

  type DeleteEnterpriseResourceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type DeleteEnterpriseTagGroupResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type DeleteSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type DeleteUserMembershipResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type DingTalkInvitation = {
    id?: string;
    organizationId?: string;
    organizationName?: string;
    roleId?: string;
    roleName?: string;
    mobileMasked?: string;
    displayName?: string;
    status?: number;
    consumedName?: string;
    createdAt?: string;
    expiresAt?: string;
    inviterName?: string;
    kind?: number;
    token?: string;
  };

  type DingTalkInvitationPublicInfo = {
    organizationName?: string;
    inviterName?: string;
    expiresAt?: string;
  };

  type DingTalkLoginConfig = {
    enabled?: boolean;
    authorizeUrl?: string;
  };

  type DingTalkLoginRequest = {
    authCode: string;
    state: string;
  };

  type DingTalkLoginResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkLoginResult;
    traceId?: string;
  };

  type DingTalkLoginResult = {
    status?: number;
    currentUser?: CurrentUser;
    displayName?: string;
    /** REGISTRATION_REQUIRED 时可选的目标公司列表（启用中的公司组织），供注册确认页选择。 */
    registrationOrganizations?: OrganizationChoice[];
  };

  type DingTalkRegistration = {
    userId?: string;
    displayName?: string;
    avatarUrl?: string;
    requestedOrganizationId?: string;
    requestedOrganizationName?: string;
    registeredAt?: string;
  };

  type DingTalkRegistrationConfirmation = {
    displayName?: string;
    status?: string;
  };

  type DisableExchangeRateSettingRequest = {
    id: string;
  };

  type DisableExchangeRateSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type DownloadExchangeRateImportTemplateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    fileName?: string;
    contentType?: string;
    content?: string;
    templateVersion?: number;
    traceId?: string;
  };

  type EnterpriseResource = {
    id?: string;
    resourceType?: number;
    shortName?: string;
    enabled?: boolean;
    sortOrder?: number;
    partnerIds?: string[];
    addressTypes?: number[];
    assigneeIds?: string[];
    address?: EnterpriseResourceAddress;
    remark?: EnterpriseResourceRemark;
    party?: EnterpriseResourceParty;
    image?: EnterpriseResourceImage;
    tag?: EnterpriseResourceTag;
    createdBy?: string;
    updatedBy?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type EnterpriseResourceAddress = {
    contactName?: string;
    contactPhone?: string;
    countryCode?: string;
    provinceCode?: string;
    cityCode?: string;
    districtCode?: string;
    addressDetail?: string;
    remark?: string;
  };

  type EnterpriseResourceAssigneeOption = {
    id?: string;
    username?: string;
    displayName?: string;
  };

  type EnterpriseResourceImage = {
    fileName?: string;
    mimeType?: string;
    fileSize?: string;
    objectKey?: string;
    checksum?: string;
    width?: number;
    height?: number;
  };

  type EnterpriseResourceImportConflict = {
    existingResourceId?: string;
    existingShortName?: string;
    matchedFields?: string[];
  };

  type EnterpriseResourceImportRow = {
    rowNumber?: number;
    resource?: EnterpriseResourceInput;
    errors?: string[];
    conflicts?: EnterpriseResourceImportConflict[];
  };

  type EnterpriseResourceInput = {
    resourceType: number;
    shortName: string;
    enabled?: boolean;
    sortOrder?: number;
    partnerAssociations?: PartnerAssociations;
    addressTypes?: number[];
    assigneeIds?: string[];
    address?: EnterpriseResourceAddress;
    remark?: EnterpriseResourceRemark;
    party?: EnterpriseResourceParty;
    image?: EnterpriseResourceImage;
    tag?: EnterpriseResourceTag;
  };

  type EnterpriseResourcePartnerOption = {
    id?: string;
    code?: string;
    name?: string;
  };

  type EnterpriseResourceParty = {
    companyName?: string;
    businessCode?: string;
    address?: string;
    countryCode?: string;
    contactName?: string;
    contactPhone?: string;
    email?: string;
    taxIdentifier?: string;
    aeoCode?: string;
    customDisplay?: boolean;
    displayContent?: string;
    remark?: string;
  };

  type EnterpriseResourceRegionOption = {
    code?: string;
    name?: string;
    level?: number;
    parentCode?: string;
  };

  type EnterpriseResourceRemark = {
    remarkType?: number;
    content?: string;
  };

  type EnterpriseResourceServiceDeleteEnterpriseResourceParams = {
    id: string;
  };

  type EnterpriseResourceServiceDeleteEnterpriseTagGroupParams = {
    id: string;
  };

  type EnterpriseResourceServiceGetEnterpriseResourceImageAccessParams = {
    id: string;
  };

  type EnterpriseResourceServiceGetEnterpriseResourceParams = {
    id: string;
  };

  type EnterpriseResourceServiceListEnterpriseResourceRegionOptionsParams = {
    level?: number;
    parentCode?: string;
    page?: number;
    pageSize?: number;
  };

  type EnterpriseResourceServiceListEnterpriseResourcesParams = {
    resourceType?: number;
    partnerId?: string;
    linked?: boolean;
    enabled?: boolean;
    keyword?: string;
    page?: number;
    pageSize?: number;
    addressType?: number;
    assigneeId?: string;
    sortBy?: string;
    sortOrder?: string;
  };

  type EnterpriseResourceServiceSearchEnterpriseResourceAssigneeOptionsParams =
    {
      keyword?: string;
      page?: number;
      pageSize?: number;
    };

  type EnterpriseResourceServiceSearchEnterpriseResourcePartnerOptionsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type EnterpriseResourceServiceUpdateEnterpriseResourceParams = {
    id: string;
  };

  type EnterpriseResourceServiceUpdateEnterpriseTagGroupParams = {
    id: string;
  };

  type EnterpriseResourceTag = {
    groupId?: string;
  };

  type EnterpriseTagGroup = {
    id?: string;
    name?: string;
    color?: string;
    sortOrder?: number;
    createdAt?: string;
    updatedAt?: string;
  };

  type EnterpriseTagGroupInput = {
    name: string;
    color?: string;
    sortOrder?: number;
  };

  type ExchangeRateImportBatch = {
    id?: string;
    fileName?: string;
    fileChecksum?: string;
    templateVersion?: number;
    status?: string;
    totalCount?: number;
    validCount?: number;
    invalidCount?: number;
    importedCount?: number;
    canConfirm?: boolean;
    rows?: ExchangeRateImportRow[];
    expiresAt?: string;
    importedAt?: string;
    createdAt?: string;
  };

  type ExchangeRateImportRow = {
    rowNumber?: number;
    fromCurrency?: string;
    toCurrency?: string;
    rate?: string;
    effectiveFrom?: string;
    effectiveTo?: string;
    status?: string;
    errors?: string[];
  };

  type ExchangeRateServiceDisableExchangeRateSettingParams = {
    id: string;
  };

  type ExchangeRateServiceGetExchangeRateImportParams = {
    id: string;
  };

  type ExchangeRateServiceUpdateExchangeRateSettingParams = {
    id: string;
  };

  type ExchangeRateSetting = {
    id?: string;
    organizationId?: string;
    fromCurrency?: string;
    toCurrency?: string;
    /** effective_from 为带时区且精确到秒的 RFC 3339 时间，区间左边界包含该时刻。 */
    effectiveFrom?: string;
    /** effective_to 为带时区且精确到秒的 RFC 3339 时间，区间右边界不包含该时刻；空表示长期有效。 */
    effectiveTo?: string;
    isActive?: boolean;
    createdAt?: string;
    updatedAt?: string;
    /** rate 为原币折本位币的单一基准汇率。 */
    rate?: string;
  };

  type ExecuteChangeSeaDocumentModeRequest = {
    orderId: string;
    expectedOrderVersion: string;
    expectedLinkVersion: string;
    expectedHouseBillVersion?: string;
    expectedCurrentVersionId?: string;
    targetMode: number;
    newHouseBill?: SeaHouseBillInput;
    reason: string;
    confirmation: SeaExternalConfirmationInput;
    idempotencyKey: string;
  };

  type ExecuteChangeSeaDocumentModeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderDocuments;
    traceId?: string;
  };

  type ExecuteSeaDocumentAmendmentRequest = {
    orderId: string;
    documentType: number;
    documentId: string;
    expectedOrderVersion: string;
    expectedDocumentVersion: string;
    expectedCurrentVersionId: string;
    reason: string;
    idempotencyKey: string;
    input: SeaDocumentAmendmentInput;
    confirmation: SeaExternalConfirmationInput;
  };

  type ExecuteSeaDocumentAmendmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentVersion;
    traceId?: string;
  };

  type ExecuteSeaDocumentVoidRequest = {
    orderId: string;
    documentType: number;
    documentId: string;
    expectedOrderVersion: string;
    expectedDocumentVersion: string;
    expectedCurrentVersionId: string;
    reason: string;
    idempotencyKey: string;
    confirmation: SeaExternalConfirmationInput;
  };

  type ExecuteSeaDocumentVoidResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentEvent;
    traceId?: string;
  };

  type ExecuteSeaOrderReassignmentData = {
    reassignmentEventId?: string;
    createdAt?: string;
    orderId?: string;
    orderNo?: string;
    targetMasterBillId?: string;
    targetMasterNo?: string;
  };

  type ExecuteSeaOrderReassignmentRequest = {
    orderId: string;
    idempotencyKey: string;
    requestFingerprint: string;
    target: SeaOrderReassignmentTargetInput;
    reason: string;
    responsibilityType: string;
    responsiblePartnerId?: string;
    expectedOrderVersion: string;
    expectedLinkVersion: string;
    expectedCandidateMblVersion?: string;
    expectedCandidateTeVersion?: string;
    confirmation: SeaExternalConfirmationInput;
  };

  type ExecuteSeaOrderReassignmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExecuteSeaOrderReassignmentData;
    traceId?: string;
  };

  type ExecuteSeaOrderSplitData = {
    splitEventId?: string;
    createdAt?: string;
    originalOrder?: SeaOrderSplitOrderReference;
    createdOrders?: SeaOrderSplitCreatedOrder[];
    reassignmentEventIds?: string[];
  };

  type ExecuteSeaOrderSplitRequest = {
    orderId: string;
    idempotencyKey: string;
    requestFingerprint: string;
    note?: string;
    targets: SeaOrderSplitTargetInput[];
    results: SeaOrderSplitResultInput[];
    expectedVersions: SeaOrderSplitExpectedVersions;
    /** 任一结果目标不是当前母单（即产生内嵌改配）时必填 */
    confirmation?: SeaExternalConfirmationInput;
  };

  type ExecuteSeaOrderSplitResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExecuteSeaOrderSplitData;
    traceId?: string;
  };

  type ExecuteSeaTransportExecutionUpdateRequest = {
    orderId: string;
    expectedTransportExecutionVersion: string;
    input: SeaTransportExecutionUpdateInput;
    reason: string;
    confirmation: SeaExternalConfirmationInput;
    idempotencyKey: string;
  };

  type ExecuteSeaTransportExecutionUpdateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    transportExecution?: SeaTransportExecution;
    versionId?: string;
    traceId?: string;
  };

  type ExportCommissionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CommissionExportItem[];
    traceId?: string;
  };

  type ExportPartnersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerExportItem[];
    traceId?: string;
  };

  type FeeCatalogServiceSearchBillingUnitsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type FeeCatalogServiceSearchFeeSettingsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type FeeCatalogServiceSearchTaxableServicesParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type FeeCatalogServiceUpdateBillingUnitParams = {
    id: string;
  };

  type FeeCatalogServiceUpdateFeeSettingParams = {
    id: string;
  };

  type FeeCatalogServiceUpdateTaxableServiceParams = {
    id: string;
  };

  type FeeLedgerBaseCurrencyAmount = {
    baseCurrency?: string;
    receivableBaseAmount?: string;
    payableBaseAmount?: string;
    profitBaseAmount?: string;
  };

  type FeeLedgerColumnPreference = {
    fieldKey: string;
    visible?: boolean;
  };

  type FeeLedgerItem = {
    id?: string;
    orderId?: string;
    orderNo?: string;
    businessType?: string;
    direction?: string;
    status?: number;
    feeCode?: string;
    feeName?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    billingUnit?: string;
    quantity?: string;
    unitPrice?: string;
    totalAmount?: string;
    netAmount?: string;
    taxAmount?: string;
    currency?: string;
    exchangeRate?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    expenseDate?: string;
    note?: string;
    version?: string;
    createdAt?: string;
    updatedAt?: string;
    taxRate?: string;
    customerId?: string;
    customerName?: string;
    financialProgress?: number;
    billNo?: string;
    financeLocked?: boolean;
    tags?: BusinessTagSummary[];
    organizationId?: string;
    organizationName?: string;
  };

  type FeeLedgerOrderDetail = {
    orderId?: string;
    orderNo?: string;
    businessType?: string;
    customerName?: string;
    organizationId?: string;
    organizationName?: string;
    fees?: FeeLedgerItem[];
    amountsByBaseCurrency?: FeeLedgerBaseCurrencyAmount[];
  };

  type FeeLedgerPreference = {
    columns?: FeeLedgerColumnPreference[];
    pageSize?: number;
    sortField?: string;
    sortDirection?: string;
    rowColors?: FeeLedgerRowColors;
    version?: string;
    customized?: boolean;
    updatedAt?: string;
  };

  type FeeLedgerRowColors = {
    unbilled: string;
    unverifiedUninvoiced: string;
    invoicedUnverified: string;
    verifiedUninvoiced: string;
    completed: string;
    invoicedPartiallyVerified: string;
    partiallyVerifiedUninvoiced: string;
  };

  type FeeLedgerSummary = {
    activeCount?: string;
    amountsByBaseCurrency?: FeeLedgerBaseCurrencyAmount[];
  };

  type FeeSetting = {
    id?: string;
    organizationId?: string;
    feeCode?: string;
    nameZh?: string;
    nameEn?: string;
    aliasName?: string;
    serviceTypeId?: string;
    serviceTypeName?: string;
    defaultCurrency?: string;
    billingUnitId?: string;
    billingUnitName?: string;
    abnormalCaseId?: string;
    abnormalCaseName?: string;
    taxRate?: string;
    taxableServiceId?: string;
    taxableServiceName?: string;
    enabled?: boolean;
    sortOrder?: number;
    createdAt?: string;
    updatedAt?: string;
  };

  type FinanceBaseCurrencyAmount = {
    baseCurrency?: string;
    receivableBaseAmount?: string;
    payableBaseAmount?: string;
    unverifiedBaseAmount?: string;
    overdueReceivableBaseAmount?: string;
  };

  type FinanceBill = {
    id?: string;
    billNo?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    baseCurrency?: string;
    totalAmount?: string;
    netAmount?: string;
    taxAmount?: string;
    baseCurrencyAmount?: string;
    feeCount?: number;
    billDate?: string;
    dueDate?: string;
    note?: string;
    version?: string;
    confirmedAt?: string;
    confirmedBy?: string;
    cancelledAt?: string;
    cancelledBy?: string;
    cancellationReason?: string;
    lines?: FinanceBillLine[];
    createdAt?: string;
    updatedAt?: string;
    verifiedAmount?: string;
    unverifiedAmount?: string;
    batchId?: string;
    batchNo?: string;
    statementTitle?: string;
    paymentTermsDays?: number;
    exchangeRate?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    tags?: BusinessTagSummary[];
    organizationId?: string;
    organizationName?: string;
    settlementAccountId?: string;
    settlementAccountName?: string;
    settlementAccountHolder?: string;
    settlementBankName?: string;
    settlementBankAccount?: string;
    settlementAccountCurrency?: string;
    settlementSwiftCode?: string;
    estimatedInvoiceCurrency?: string;
    estimatedInvoiceRate?: string;
    estimatedInvoiceAmount?: string;
    /** netted_amount 是有效对冲分摊合计；unverified_amount 已扣除该抵销额。 */
    nettedAmount?: string;
    overdueDays?: number;
  };

  type FinanceBillBatch = {
    id?: string;
    batchNo?: string;
    splitByOrder?: boolean;
    splitByTaxRate?: boolean;
    feeCount?: number;
    billCount?: number;
    totalBaseAmount?: string;
    baseCurrency?: string;
    bills?: FinanceBill[];
    createdAt?: string;
    mode?: number;
    /** nettings 仅在对冲建账模式下返回：本批次原子生成的对冲结算单（初始为草稿）。 */
    nettings?: FinanceNetting[];
  };

  type FinanceBillLine = {
    id?: string;
    orderFeeId?: string;
    orderId?: string;
    orderNo?: string;
    businessType?: string;
    feeCode?: string;
    feeName?: string;
    totalAmount?: string;
    netAmount?: string;
    taxAmount?: string;
    currency?: string;
    exchangeRate?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    active?: boolean;
    taxRate?: string;
    quantity?: string;
    unitPrice?: string;
  };

  type FinanceBillSummary = {
    amountsByBaseCurrency?: FinanceBaseCurrencyAmount[];
  };

  type FinanceCashflow = {
    id?: string;
    flowNo?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    amount?: string;
    exchangeRate?: string;
    baseCurrency?: string;
    baseAmount?: string;
    transactionDate?: string;
    ourAccount?: string;
    counterpartyAccount?: string;
    paymentMethod?: string;
    bankReferenceNo?: string;
    note?: string;
    version?: string;
    confirmedAt?: string;
    cancelledAt?: string;
    cancellationReason?: string;
    createdAt?: string;
    updatedAt?: string;
    verifiedAmount?: string;
    unverifiedAmount?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    organizationId?: string;
    organizationName?: string;
  };

  type FinanceCashflowSummary = {
    amountsByBaseCurrency?: FinanceBaseCurrencyAmount[];
  };

  type FinanceCommission = {
    id?: string;
    commissionNo?: string;
    verificationId?: string;
    verificationNo?: string;
    employeeId?: string;
    employeeName?: string;
    status?: number;
    baseCurrency?: string;
    realizedRevenue?: string;
    allocatedCost?: string;
    realizedProfit?: string;
    ratePercent?: string;
    commissionAmount?: string;
    note?: string;
    version?: string;
    confirmedAt?: string;
    paidAt?: string;
    cancelledAt?: string;
    cancellationReason?: string;
    createdAt?: string;
    updatedAt?: string;
    ruleId?: string;
    ruleName?: string;
    personnelRole?: string;
    calculationBasis?: string;
    ruleVersion?: string;
    calculationVersion?: string;
    lines?: FinanceCommissionLine[];
    adjustments?: FinanceCommissionAdjustment[];
    adjustmentAmount?: string;
    effectiveCommissionAmount?: string;
    customerCount?: number;
    orderCount?: number;
    feeCount?: number;
    commissionBaseAmount?: string;
    commissionDate?: string;
    cnyExchangeRate?: string;
    cnyExchangeRateSource?: string;
    cnyExchangeRateDate?: string;
    cnyExchangeRateSettingId?: string;
    cnyCommissionAmount?: string;
    cnyAdjustmentAmount?: string;
    cnyEffectiveCommissionAmount?: string;
    organizationId?: string;
    organizationName?: string;
    confirmedBy?: string;
    paidBy?: string;
    cancelledBy?: string;
    nettingId?: string;
    nettingNo?: string;
  };

  type FinanceCommissionAdjustment = {
    id?: string;
    adjustmentNo?: string;
    commissionId?: string;
    commissionNo?: string;
    orderId?: string;
    orderNo?: string;
    employeeId?: string;
    employeeName?: string;
    direction?: string;
    status?: number;
    baseCurrency?: string;
    amount?: string;
    reason?: string;
    note?: string;
    version?: string;
    confirmedAt?: string;
    paidAt?: string;
    cancelledAt?: string;
    cancellationReason?: string;
    createdAt?: string;
    updatedAt?: string;
    sourceType?: string;
    sourceVerificationId?: string;
    organizationId?: string;
    organizationName?: string;
    confirmedBy?: string;
    paidBy?: string;
    cancelledBy?: string;
  };

  type FinanceCommissionLine = {
    id?: string;
    orderId?: string;
    orderNo?: string;
    employeeId?: string;
    employeeName?: string;
    personnelRole?: string;
    calculationBasis?: string;
    baseCurrency?: string;
    realizedRevenue?: string;
    allocatedCost?: string;
    realizedProfit?: string;
    ratePercent?: string;
    commissionAmount?: string;
    personnelOrganizationId?: string;
    personnelAssignedAt?: string;
    orderDate?: string;
    customerId?: string;
    customerCode?: string;
    customerName?: string;
    commissionBaseAmount?: string;
    customerAssignmentId?: string;
    customerAssignmentOrganizationId?: string;
    customerAssignedAt?: string;
    feeCount?: number;
    fees?: CommissionFeeDetail[];
  };

  type FinanceCommissionRule = {
    id?: string;
    name?: string;
    personnelRole?: string;
    calculationBasis?: string;
    ratePercent?: string;
    effectiveFrom?: string;
    effectiveTo?: string;
    enabled?: boolean;
    note?: string;
    version?: string;
    createdAt?: string;
    updatedAt?: string;
    organizationId?: string;
    organizationName?: string;
  };

  type FinanceInvoice = {
    id?: string;
    recordNo?: string;
    direction?: string;
    status?: number;
    invoiceType?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    totalAmount?: string;
    taxAmount?: string;
    billCount?: number;
    taxInvoiceNo?: string;
    invoiceDate?: string;
    note?: string;
    version?: string;
    issuedAt?: string;
    cancelledAt?: string;
    cancellationReason?: string;
    billLinks?: FinanceInvoiceBill[];
    createdAt?: string;
    updatedAt?: string;
    redInvoiceNo?: string;
    redInvoiceDate?: string;
    redFlushedAt?: string;
    redFlushReason?: string;
    netAmount?: string;
    invoiceProfileId?: string;
    invoiceTitle?: string;
    taxpayerIdentificationNo?: string;
    registeredAddress?: string;
    registeredPhone?: string;
    bankName?: string;
    bankAccount?: string;
    lines?: FinanceInvoiceLine[];
    baseCurrency?: string;
    exchangeRate?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    baseCurrencyAmount?: string;
    organizationId?: string;
    organizationName?: string;
  };

  type FinanceInvoiceBill = {
    id?: string;
    billId?: string;
    billNo?: string;
    amount?: string;
    taxAmount?: string;
    active?: boolean;
  };

  type FinanceInvoiceLine = {
    id?: string;
    lineNo?: number;
    itemCode?: string;
    itemName?: string;
    taxRate?: string;
    netAmount?: string;
    taxAmount?: string;
    totalAmount?: string;
    currency?: string;
    sourceLineCount?: number;
  };

  type FinanceInvoiceProfileOption = {
    id?: string;
    invoiceTitle?: string;
    taxpayerIdentificationNo?: string;
    defaultInvoiceType?: string;
    isDefault?: boolean;
  };

  type FinanceInvoiceProfilesForBill = {
    organizationId?: string;
    settlementPartyId?: string;
    data?: FinanceInvoiceProfileOption[];
  };

  type FinanceInvoiceSummary = {
    amountsByBaseCurrency?: FinanceBaseCurrencyAmount[];
    issuedCount?: string;
  };

  type FinanceNetting = {
    id?: string;
    nettingNo?: string;
    status?: number;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    amount?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    note?: string;
    version?: string;
    allocations?: FinanceNettingAllocation[];
    createdAt?: string;
    updatedAt?: string;
    confirmedAt?: string;
    cancelledAt?: string;
    cancellationReason?: string;
    reversedAt?: string;
    reversalReason?: string;
    organizationId?: string;
    organizationName?: string;
    batchId?: string;
    batchNo?: string;
    payableBaseAmount?: string;
    exchangeGainLoss?: string;
  };

  type FinanceNettingAllocation = {
    id?: string;
    billId?: string;
    billNo?: string;
    direction?: string;
    amount?: string;
    baseCurrencyAmount?: string;
    active?: boolean;
  };

  type FinanceNettingBaseCurrencyAmount = {
    baseCurrency?: string;
    nettingBaseAmount?: string;
  };

  type FinanceNettingBillBalance = {
    billId?: string;
    billNo?: string;
    billDate?: string;
    totalAmount?: string;
    verifiedAmount?: string;
    nettedAmount?: string;
    availableAmount?: string;
    version?: string;
  };

  type FinanceNettingPreview = {
    organizationId?: string;
    organizationName?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    receivableBills?: FinanceNettingBillBalance[];
    payableBills?: FinanceNettingBillBalance[];
    receivableAvailableAmount?: string;
    payableAvailableAmount?: string;
    offsetAmount?: string;
    netReceivableAmount?: string;
    netPayableAmount?: string;
  };

  type FinanceNettingSummary = {
    amountsByBaseCurrency?: FinanceNettingBaseCurrencyAmount[];
    confirmedCount?: string;
  };

  type FinanceOrganizationOption = {
    id?: string;
    code?: string;
    name?: string;
    baseCurrency?: string;
  };

  type FinanceSettlementAccountOption = {
    id?: string;
    name?: string;
    accountHolder?: string;
    bankName?: string;
    accountNo?: string;
    currency?: string;
    swiftCode?: string;
    isDefault?: boolean;
  };

  type FinanceSettlementPartyOption = {
    id?: string;
    code?: string;
    name?: string;
    isCasual?: boolean;
    creditExceeded?: boolean;
  };

  type FinanceVerification = {
    id?: string;
    verificationNo?: string;
    status?: number;
    direction?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    amount?: string;
    verificationDate?: string;
    note?: string;
    version?: string;
    reversedAt?: string;
    reversalReason?: string;
    allocations?: FinanceVerificationAllocation[];
    createdAt?: string;
    baseCurrency?: string;
    baseAmount?: string;
    billBaseAmount?: string;
    cashflowBaseAmount?: string;
    exchangeGainLoss?: string;
    organizationId?: string;
    organizationName?: string;
  };

  type FinanceVerificationAllocation = {
    id?: string;
    cashflowId?: string;
    billId?: string;
    cashflowNo?: string;
    billNo?: string;
    amount?: string;
    active?: boolean;
    billBaseAmount?: string;
    cashflowBaseAmount?: string;
    exchangeGainLoss?: string;
  };

  type FinanceVerificationSummary = {
    amountsByBaseCurrency?: FinanceBaseCurrencyAmount[];
  };

  type GetBackgroundTaskResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BackgroundTask;
    traceId?: string;
  };

  type GetBilledFeeEditPolicyResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BilledFeeEditPolicy;
    traceId?: string;
    /** can_update 表示当前主体是否可在当前组织更新本策略，由 bill.update 权限及其组织范围计算。 */
    canUpdate?: boolean;
  };

  type GetBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill;
    traceId?: string;
  };

  type GetCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission;
    traceId?: string;
  };

  type GetCreditLimitControlPolicyResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CreditLimitControlPolicy;
    traceId?: string;
    /** can_update 表示当前主体是否可在当前组织更新本策略，由 bill.update 权限及其组织范围计算。 */
    canUpdate?: boolean;
  };

  type GetDingTalkInvitationInfoResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkInvitationPublicInfo;
    traceId?: string;
  };

  type GetDingTalkInvitationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkInvitation;
    invitationUrl?: string;
    traceId?: string;
  };

  type GetDingTalkLoginConfigResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkLoginConfig;
    traceId?: string;
  };

  type GetEnterpriseResourceCapabilitiesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    imageEnabled?: boolean;
    imageMaxFileSize?: string;
    imageUsedStorageBytes?: string;
    imageStorageQuotaBytes?: string;
    traceId?: string;
  };

  type GetEnterpriseResourceImageAccessResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    url?: string;
    expiresAt?: string;
    traceId?: string;
  };

  type GetEnterpriseResourceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResource;
    traceId?: string;
  };

  type GetExchangeRateImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateImportBatch;
    traceId?: string;
  };

  type GetFeeLedgerOrderDetailResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerOrderDetail;
    traceId?: string;
  };

  type GetFeeLedgerPreferenceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerPreference;
    traceId?: string;
  };

  type GetInvoiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice;
    traceId?: string;
  };

  type GetNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting;
    traceId?: string;
  };

  type GetOrderLockStateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderLockStateData;
    traceId?: string;
  };

  type GetOrderResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type GetOrderUnlockRequestResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderUnlockRequestData;
    traceId?: string;
  };

  type GetPartnerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
    traceId?: string;
  };

  type GetSeaDocumentVersionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentVersion;
    traceId?: string;
  };

  type GetSeaOrderChangeActionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderChangeActionsData;
    traceId?: string;
  };

  type GetSeaOrderChangeEventResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderChangeEventDetailData;
    traceId?: string;
  };

  type GetSeaOrderDocumentsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderDocuments;
    traceId?: string;
  };

  type GetSeaOrderSplitContextResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderSplitContextData;
    traceId?: string;
  };

  type GetSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };

  type GetWeComLoginConfigResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: WeComLoginConfig;
    traceId?: string;
  };

  type ImportItemsRequest = {
    kind: number;
    source: string;
    mode: number;
    items: MasterDataImportItemInput[];
  };

  type ImportItemsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    createdCount?: number;
    updatedCount?: number;
    traceId?: string;
  };

  type ImportPartnersRequest = {
    source: string;
    mode: number;
    items: PartnerImportItemInput[];
  };

  type ImportPartnersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    createdCount?: number;
    updatedCount?: number;
    traceId?: string;
  };

  type IssueInvoiceRequest = {
    id: string;
    expectedVersion: string;
    taxInvoiceNo: string;
    invoiceDate: string;
  };

  type IssueInvoiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice;
    traceId?: string;
  };

  type ListAbnormalCasesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAbnormalCase[];
    traceId?: string;
  };

  type ListAdministrativeRegionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdministrativeRegion[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListAirlinesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airline[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListAirportsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airport[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListAttachmentsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAttachment[];
    traceId?: string;
  };

  type ListAuditLogsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminAuditLog[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListBackgroundTasksResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BackgroundTask[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListBillCreationCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerItem[];
    total?: string;
    traceId?: string;
  };

  type ListBillingUnitsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillingUnit[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListBillSettlementAccountCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceSettlementAccountOption[];
    traceId?: string;
  };

  type ListBillSettlementAccountUpdateCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceSettlementAccountOption[];
    traceId?: string;
  };

  type ListBillsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill[];
    total?: string;
    traceId?: string;
    summary?: FinanceBillSummary;
  };

  type ListCargoItemsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderCargoItem[];
    traceId?: string;
  };

  type ListCashflowsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCashflow[];
    total?: string;
    traceId?: string;
    summary?: FinanceCashflowSummary;
  };

  type ListCommissionCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CommissionCandidateSummary[];
    total?: string;
    traceId?: string;
    page?: number;
    pageSize?: number;
  };

  type ListCommissionEmployeesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CommissionEmployeeOption[];
    traceId?: string;
    total?: string;
    page?: number;
    pageSize?: number;
  };

  type ListCommissionNettingCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting[];
    total?: string;
    traceId?: string;
  };

  type ListCommissionRuleCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionRule[];
    total?: string;
    traceId?: string;
  };

  type ListCommissionRulesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionRule[];
    total?: string;
    traceId?: string;
  };

  type ListCommissionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission[];
    total?: string;
    traceId?: string;
  };

  type ListCommissionVerificationCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceVerification[];
    total?: string;
    traceId?: string;
  };

  type ListContainersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer[];
    traceId?: string;
  };

  type ListCurrenciesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Currency[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListDingTalkInvitationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkInvitation[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListDingTalkRegistrationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkRegistration[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListEnterpriseResourceRegionOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResourceRegionOption[];
    total?: string;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListEnterpriseResourcesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResource[];
    total?: string;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListEnterpriseTagGroupsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseTagGroup[];
    traceId?: string;
  };

  type ListExchangeRateSettingsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateSetting[];
    traceId?: string;
    baseCurrency?: string;
  };

  type ListFeeLedgerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerItem[];
    total?: string;
    summary?: FeeLedgerSummary;
    traceId?: string;
  };

  type ListFeeOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    settlementParties?: OrderFeeSettlementPartyOption[];
    currencies?: OrderFeeCurrencyOption[];
    traceId?: string;
    baseCurrency?: string;
    feeSettings?: OrderFeeSettingOption[];
    billingUnits?: OrderFeeBillingUnitOption[];
    financeLocked?: boolean;
    financeLockReason?: string;
    financeLockCommissionNos?: string[];
    customerId?: string;
    customerName?: string;
  };

  type ListFeeSettingsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeSetting[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListFeesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderFee[];
    traceId?: string;
  };

  type ListFinanceBillTagAssignmentOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListFinanceBillTagOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListFinanceFeeTagAssignmentOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListFinanceFeeTagOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListFinanceOrganizationOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceOrganizationOption[];
    traceId?: string;
  };

  type ListFinanceSettlementPartyOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceSettlementPartyOption[];
    total?: string;
    traceId?: string;
  };

  type ListInvoiceCreationBillsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill[];
    total?: string;
    traceId?: string;
  };

  type ListInvoiceProfilesForBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoiceProfilesForBill;
    traceId?: string;
  };

  type ListInvoicesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice[];
    total?: string;
    traceId?: string;
    summary?: FinanceInvoiceSummary;
  };

  type ListItemsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListMilestonesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderMilestone[];
    traceId?: string;
  };

  type ListNettingsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting[];
    total?: string;
    traceId?: string;
    summary?: FinanceNettingSummary;
  };

  type ListNumberRulesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: NumberRule[];
    traceId?: string;
  };

  type ListOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    traceId?: string;
  };

  type ListOrderConsolidationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderConsolidationSummary[];
    traceId?: string;
  };

  type ListOrderFeeTagOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListOrdersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListOrderTagOptionsResponse = {
    tags?: BusinessTagSummary[];
    total?: string;
    traceId?: string;
  };

  type ListOrderUnlockRequestsData = {
    items?: OrderUnlockRequestData[];
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListOrderUnlockRequestsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ListOrderUnlockRequestsData;
    traceId?: string;
  };

  type ListOrganizationRolesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole[];
    traceId?: string;
  };

  type ListOrganizationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization[];
    traceId?: string;
  };

  type ListPartnerAccountsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAccount[];
    traceId?: string;
  };

  type ListPartnerAssignmentOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAssignmentOption[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListPartnerAttachmentsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAttachment[];
    traceId?: string;
  };

  type ListPartnerAuditLogsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAuditLog[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListPartnerContractsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerContract[];
    traceId?: string;
  };

  type ListPartnerInvoiceProfilesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerInvoiceProfile[];
    traceId?: string;
  };

  type ListPartnerSettlementRulesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerSettlementRule[];
    traceId?: string;
  };

  type ListPartnerShippingPresetsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerShippingPreset[];
    traceId?: string;
  };

  type ListPartnersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListPermissionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminPermission[];
    traceId?: string;
  };

  type ListPersonnelOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderPersonnelOption[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListPersonnelResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderPersonnel[];
    traceId?: string;
  };

  type ListPortsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Port[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListReleasePodsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod[];
    traceId?: string;
  };

  type ListRolesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole[];
    traceId?: string;
  };

  type ListSameBatchOrdersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SameBatchOrderSummary[];
    traceId?: string;
  };

  type ListSeaDocumentEventsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentEvent[];
    total?: number;
    traceId?: string;
  };

  type ListSeaHouseBillVersionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentVersion[];
    total?: number;
    traceId?: string;
  };

  type ListSeaMasterBillVersionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentVersion[];
    total?: number;
    traceId?: string;
  };

  type ListSeaOrderChangeEventsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderChangeEventSummary[];
    total?: number;
    traceId?: string;
  };

  type ListSeaSharedContainerCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainerCandidateOrder[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListSeaSharedContainersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListShippingDocumentsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument[];
    traceId?: string;
  };

  type ListShippingLinesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ShippingLine[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListTaxableServicesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: TaxableService[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type ListTransferOrganizationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization[];
    traceId?: string;
  };

  type ListUserMembershipsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUserMembership[];
    traceId?: string;
  };

  type ListUsersResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ListVerificationCreationCandidatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: VerificationCreationCandidates;
    traceId?: string;
  };

  type ListVerificationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceVerification[];
    total?: string;
    traceId?: string;
    summary?: FinanceVerificationSummary;
  };

  type LockOrderRequest = {
    orderId: string;
    expectedOrderVersion: string;
    idempotencyKey: string;
  };

  type LockOrderResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderLockResultData;
    traceId?: string;
  };

  type LoginRequest = {
    username: string;
    password: string;
    /** 可选：显式指定本次登录进入的成员资格组织；缺省使用默认组织。 */
    organizationId?: string;
  };

  type LoginResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
    /** 本人启用中成员资格组织候选列表（默认组织置首并标记）。 */
    organizationChoices?: OrganizationChoice[];
  };

  type LogoutRequest = {};

  type LogoutResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type MarkAbnormalCaseRequest = {
    orderId: string;
    abnormalCaseId: string;
  };

  type MarkAbnormalCaseResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAbnormalCase;
    traceId?: string;
  };

  type MarkCommissionAdjustmentPaidRequest = {
    id: string;
    expectedVersion: string;
  };

  type MarkCommissionAdjustmentPaidResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionAdjustment;
    traceId?: string;
  };

  type MarkCommissionPaidRequest = {
    id: string;
    expectedVersion: string;
  };

  type MarkCommissionPaidResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommission;
    traceId?: string;
  };

  type MasterDataAttributes = {
    continent?: string;
    currencyCode?: string;
    regionLevel?: number;
  };

  type MasterDataImportItemInput = {
    code: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    teuFactor?: string;
    sortOrder?: number;
    enabled?: boolean;
    attributes?: MasterDataAttributes;
  };

  type MasterDataItem = {
    id?: string;
    organizationId?: string;
    kind?: number;
    code?: string;
    name?: string;
    nameEn?: string;
    parentCode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
    attributes?: MasterDataAttributes;
  };

  type MasterDataServiceListAdministrativeRegionsParams = {
    level?: number;
    parentCode?: string;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type MasterDataServiceListAirlinesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    enabled?: boolean;
  };

  type MasterDataServiceListAirportsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    enabled?: boolean;
  };

  type MasterDataServiceListItemsParams = {
    page?: number;
    pageSize?: number;
    kind?: number;
    keyword?: string;
    enabled?: boolean;
  };

  type MasterDataServiceListPortsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    enabled?: boolean;
  };

  type MasterDataServiceListShippingLinesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    enabled?: boolean;
  };

  type MasterDataServiceSearchCurrenciesParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type MasterDataServiceUpdateAirlineParams = {
    id: string;
  };

  type MasterDataServiceUpdateAirportParams = {
    id: string;
  };

  type MasterDataServiceUpdateItemParams = {
    id: string;
  };

  type MasterDataServiceUpdateNumberRuleParams = {
    id: string;
  };

  type MasterDataServiceUpdatePortParams = {
    id: string;
  };

  type MasterDataServiceUpdateShippingLineParams = {
    id: string;
  };

  type MatchSeaMasterBillCandidateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    matched?: boolean;
    candidate?: SeaMasterBillCandidate;
    conflicts?: SeaVoyageConflict[];
    traceId?: string;
  };

  type MeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
  };

  type NettingBillExpectedVersion = {
    billId: string;
    expectedVersion: string;
  };

  type NumberRule = {
    id?: string;
    organizationId?: string;
    documentType?: number;
    prefix?: string;
    dateFormat?: number;
    sequenceLength?: number;
    resetPolicy?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type Order = {
    id?: string;
    organizationId?: string;
    orderNo?: string;
    customerId?: string;
    shippingLineId?: string;
    bookingAgentId?: string;
    businessType?: number;
    tradeDirection?: number;
    tradeTerm?: number;
    paymentTerm?: number;
    shipmentType?: number;
    containerOwnership?: number;
    shipmentMode?: number;
    flowStatus?: number;
    serviceTypeIds?: string[];
    cargoCategoryIds?: string[];
    originLocationId?: string;
    destinationLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    vesselVoyage?: string;
    etd?: string;
    eta?: string;
    siCutoff?: string;
    docCutoff?: string;
    customsCutoff?: string;
    vgmCutoff?: string;
    goodsDescription?: string;
    totalPackages?: number;
    totalPackageUnit?: string;
    specialRequirements?: string;
    orderDate?: string;
    notes?: string;
    createdAt?: string;
    updatedAt?: string;
    customerReferenceNo?: string;
    foreignAgentId?: string;
    contractNo?: string;
    cargoValue?: string;
    cargoCurrency?: string;
    internalReferenceNo?: string;
    shippingAgentId?: string;
    insurancePremium?: string;
    insuranceCurrency?: string;
    unNumber?: string;
    hazardClass?: string;
    factoryName?: string;
    cargoReadyAt?: string;
    receivedAt?: string;
    organizationName?: string;
    canModify?: boolean;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    shippingDocuments?: OrderShippingDocument[];
    containerRequests?: OrderContainerRequest[];
    declarationCutoffAt?: string;
    totalGrossWeightKg?: number;
    totalVolumeCbm?: number;
    terminationStatus?: number;
    terminationType?: number;
    terminationReason?: string;
    terminatedAt?: string;
    terminatedBy?: string;
    closureStatus?: number;
    closureReason?: string;
    closedAt?: string;
    closedBy?: string;
    version?: string;
    hasActiveException?: boolean;
    activeExceptionCount?: number;
    allowedActions?: number[];
    shipperShortName?: string;
    consigneeShortName?: string;
    lockedAt?: string;
    isShared?: boolean;
    tags?: BusinessTagSummary[];
    allowedTargetFlowStatuses?: number[];
    seaMasterBill?: SeaMasterBillSummary;
    seaDocumentStructure?: number;
    seaDocumentLinkVersion?: string;
    seaDocumentSummary?: SeaOrderDocumentSummary;
    bookingNo?: string;
  };

  type OrderAbnormalCase = {
    id?: string;
    orderId?: string;
    abnormalCaseId?: string;
    status?: number;
    markedAt?: string;
    markedBy?: string;
    resolvedAt?: string;
    resolvedBy?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type OrderAbnormalCaseServiceListAbnormalCasesParams = {
    orderId: string;
  };

  type OrderAbnormalCaseServiceMarkAbnormalCaseParams = {
    orderId: string;
  };

  type OrderAbnormalCaseServiceRemoveAbnormalCaseParams = {
    orderId: string;
    id: string;
  };

  type OrderAbnormalCaseServiceResolveAbnormalCaseParams = {
    orderId: string;
    id: string;
  };

  type OrderAttachment = {
    id?: string;
    orderId?: string;
    docType?: string;
    idempotencyKey?: string;
    fileName?: string;
    mimeType?: string;
    fileSize?: string;
    objectKey?: string;
    checksum?: string;
    uploadedBy?: string;
    createdAt?: string;
    updatedAt?: string;
    assetId?: string;
  };

  type OrderAttachmentServiceListAttachmentsParams = {
    orderId: string;
  };

  type OrderAttachmentServiceRegisterAttachmentParams = {
    orderId: string;
  };

  type OrderAttachmentServiceRemoveAttachmentReferenceParams = {
    orderId: string;
    id: string;
  };

  type OrderCargoItem = {
    id?: string;
    orderId?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: number;
    volumeCbm?: number;
    netWeightKg?: number;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
    version?: string;
  };

  type OrderCargoItemServiceAddCargoItemParams = {
    orderId: string;
  };

  type OrderCargoItemServiceListCargoItemsParams = {
    orderId: string;
  };

  type OrderCargoItemServiceRemoveCargoItemParams = {
    orderId: string;
    id: string;
    expectedVersion?: string;
  };

  type OrderCargoItemServiceUpdateCargoItemParams = {
    orderId: string;
    id: string;
  };

  type OrderCargoMeasurement = {
    packages?: number;
    grossWeightKg?: number;
    volumeCbm?: number;
  };

  type OrderConsolidationMember = {
    orderId?: string;
    orderNo?: string;
    customerReferenceNo?: string;
    houseNos?: string[];
    entrusted?: OrderCargoMeasurement;
    actual?: OrderCargoMeasurement;
  };

  type OrderConsolidationSummary = {
    consolidationId?: string;
    masterNo?: string;
    memberCount?: number;
    entrusted?: OrderCargoMeasurement;
    actual?: OrderCargoMeasurement;
    members?: OrderConsolidationMember[];
  };

  type OrderContainer = {
    id?: string;
    orderId?: string;
    containerNo?: string;
    containerSpecId?: string;
    sealNo?: string;
    grossWeightKg?: number;
    volumeCbm?: number;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
    packageCount?: number;
    version?: string;
  };

  type OrderContainerRequest = {
    id?: string;
    orderId?: string;
    containerSpecId?: string;
    quantity?: number;
    createdAt?: string;
    updatedAt?: string;
  };

  type OrderContainerRequestInput = {
    id?: string;
    containerSpecId: string;
    quantity: number;
  };

  type OrderContainerServiceAddContainerParams = {
    orderId: string;
  };

  type OrderContainerServiceListContainersParams = {
    orderId: string;
  };

  type OrderContainerServiceRemoveContainerParams = {
    orderId: string;
    id: string;
    expectedVersion?: string;
  };

  type OrderContainerServiceUpdateContainerParams = {
    orderId: string;
    id: string;
  };

  type OrderFee = {
    id?: string;
    orderId?: string;
    direction?: number;
    feeCode?: string;
    feeName?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    billingUnit?: string;
    quantity?: string;
    unitPrice?: string;
    totalAmount?: string;
    currency?: string;
    exchangeRate?: string;
    expenseDate?: string;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    feeSettingId?: string;
    billingUnitId?: string;
    feeNameEn?: string;
    taxRate?: string;
    taxableServiceName?: string;
    status?: number;
    taxInclusive?: boolean;
    netAmount?: string;
    taxAmount?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    version?: string;
    cancelledAt?: string;
    cancelledBy?: string;
    cancellationReason?: string;
    tags?: BusinessTagSummary[];
  };

  type OrderFeeBillingUnitOption = {
    id?: string;
    code?: string;
    name?: string;
  };

  type OrderFeeCurrencyOption = {
    code?: string;
    name?: string;
    minorUnit?: number;
  };

  type OrderFeeServiceAddFeeParams = {
    orderId: string;
  };

  type OrderFeeServiceBatchAssignOrderFeeTagsParams = {
    orderId: string;
  };

  type OrderFeeServiceBatchRemoveOrderFeeTagsParams = {
    orderId: string;
  };

  type OrderFeeServiceConfirmFeeParams = {
    orderId: string;
    id: string;
  };

  type OrderFeeServiceListFeeOptionsParams = {
    orderId: string;
  };

  type OrderFeeServiceListFeesParams = {
    orderId: string;
  };

  type OrderFeeServiceListOrderFeeTagOptionsParams = {
    orderId: string;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type OrderFeeServiceRemoveFeeParams = {
    orderId: string;
    id: string;
    expectedVersion?: string;
    reason?: string;
  };

  type OrderFeeServiceReopenFeeParams = {
    orderId: string;
    id: string;
  };

  type OrderFeeServiceResolveFeeExchangeRateParams = {
    orderId: string;
    direction?: number;
    currency?: string;
    expenseDate?: string;
  };

  type OrderFeeServiceUpdateFeeParams = {
    orderId: string;
    id: string;
  };

  type OrderFeeSettingOption = {
    id?: string;
    feeCode?: string;
    nameZh?: string;
    nameEn?: string;
    aliasName?: string;
    defaultCurrency?: string;
    defaultBillingUnitId?: string;
    defaultBillingUnitName?: string;
    taxRate?: string;
    taxableServiceName?: string;
  };

  type OrderFeeSettlementPartyOption = {
    id?: string;
    code?: string;
    name?: string;
  };

  type OrderLockHouseBillSnapshotData = {
    id?: string;
    lockRecordId?: string;
    houseBillId?: string;
    houseBillVersionId?: string;
    houseNoSnapshot?: string;
    createdAt?: string;
  };

  type OrderLockRecordData = {
    id?: string;
    orderId?: string;
    orderNo?: string;
    generation?: string;
    lockedBy?: string;
    lockedByName?: string;
    lockedAt?: string;
    orderVersionAtLock?: string;
    masterBillId?: string;
    masterBillVersionId?: string;
    unlockedBy?: string;
    unlockedByName?: string;
    unlockedAt?: string;
    orderVersionAtUnlock?: string;
    unlockRequestId?: string;
    unlockReason?: string;
    unlockMode?: string;
    houseBillSnapshots?: OrderLockHouseBillSnapshotData[];
    businessType?: number;
  };

  type OrderLockResultData = {
    state?: OrderLockStateData;
    lockRecord?: OrderLockRecordData;
  };

  type OrderLockServiceGetOrderLockStateParams = {
    orderId: string;
  };

  type OrderLockServiceGetOrderUnlockRequestParams = {
    orderId: string;
    requestId: string;
  };

  type OrderLockServiceListOrderUnlockRequestsParams = {
    orderId: string;
    page?: number;
    pageSize?: number;
  };

  type OrderLockServiceLockOrderParams = {
    orderId: string;
  };

  type OrderLockServiceRequestOrderUnlockParams = {
    orderId: string;
  };

  type OrderLockStateData = {
    orderId?: string;
    orderNo?: string;
    isLocked?: boolean;
    lockGeneration?: string;
    lockedAt?: string;
    lockedBy?: string;
    lockedByName?: string;
    orderVersion?: string;
    canLock?: boolean;
    canRoleDirectUnlock?: boolean;
    canAdminEmergencyUnlock?: boolean;
    canRequestUnlock?: boolean;
    lockBlockedReasons?: string[];
    unlockBlockedReasons?: string[];
    activeUnlockRequest?: OrderUnlockRequestData;
    currentLockRecord?: OrderLockRecordData;
    businessType?: number;
  };

  type OrderMilestone = {
    id?: string;
    orderId?: string;
    type?: string;
    templateNodeCode?: string;
    templateNodeLabel?: string;
    occurredAt?: string;
    note?: string;
    updatedBy?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type OrderMilestoneServiceListMilestonesParams = {
    orderId: string;
  };

  type OrderMilestoneServiceSetMilestoneParams = {
    orderId: string;
    type: string;
  };

  type OrderPersonnel = {
    id?: string;
    orderId?: string;
    userId?: string;
    role?: number;
    assignedAt?: string;
    createdAt?: string;
    updatedAt?: string;
    organizationId?: string;
  };

  type OrderPersonnelAssignmentInput = {
    userId: string;
    organizationId: string;
    role: number;
  };

  type OrderPersonnelOption = {
    userId?: string;
    displayName?: string;
    organizationId?: string;
    organizationName?: string;
  };

  type OrderPersonnelServiceAssignPersonnelParams = {
    orderId: string;
  };

  type OrderPersonnelServiceListPersonnelParams = {
    orderId: string;
  };

  type OrderPersonnelServiceRemovePersonnelParams = {
    orderId: string;
    id: string;
  };

  type OrderReferenceCheck = {
    duplicate?: boolean;
    orderId?: string;
    orderNo?: string;
  };

  type OrderReleasePod = {
    id?: string;
    orderId?: string;
    shippingDocumentId?: string;
    releaseNo?: string;
    podNo?: string;
    status?: number;
    signedAt?: string;
    signedBy?: string;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
    allowedTargetStatuses?: number[];
    seaDocumentType?: number;
    seaDocumentId?: string;
  };

  type OrderReleasePodServiceAddReleasePodParams = {
    orderId: string;
  };

  type OrderReleasePodServiceListReleasePodsParams = {
    orderId: string;
  };

  type OrderReleasePodServiceRemoveReleasePodParams = {
    orderId: string;
    id: string;
  };

  type OrderReleasePodServiceTransitionReleasePodStatusParams = {
    orderId: string;
    id: string;
  };

  type OrderReleasePodServiceUpdateReleasePodParams = {
    orderId: string;
    id: string;
  };

  type OrderServiceCheckOrderReferenceParams = {
    referenceType?: number;
    referenceNo?: string;
    customerId?: string;
    excludeOrderId?: string;
  };

  type OrderServiceGetOrderParams = {
    id: string;
  };

  type OrderServiceListOrderConsolidationsParams = {
    id: string;
  };

  type OrderServiceListOrdersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    flowStatus?: number;
    businessType?: number;
    customerId?: string;
    terminationStatus?: number;
    closureStatus?: number;
    hasActiveException?: boolean;
    numberType?: number;
    numberKeyword?: string;
    createdAtFrom?: string;
    createdAtTo?: string;
    etdFrom?: string;
    etdTo?: string;
    etaFrom?: string;
    etaTo?: string;
    statusTimeFrom?: string;
    statusTimeTo?: string;
    lockedAtFrom?: string;
    lockedAtTo?: string;
    originLocationId?: string;
    destinationLocationId?: string;
    shippingLineId?: string;
    consigneeShortName?: string;
    shipperShortName?: string;
    operatorId?: string;
    operatorOrganizationId?: string;
    salesId?: string;
    salesOrganizationId?: string;
    customerServiceId?: string;
    customerServiceOrganizationId?: string;
    creatorId?: string;
    creatorOrganizationId?: string;
    tagIds?: string[];
    isLocked?: boolean;
    isShared?: boolean;
  };

  type OrderServiceListPersonnelOptionsParams = {
    businessType?: number;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type OrderServiceListSameBatchOrdersParams = {
    id: string;
  };

  type OrderServiceMatchSeaMasterBillCandidateParams = {
    shippingLineId?: string;
    masterNo?: string;
    originLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
  };

  type OrderServiceTransitionOrderClosureParams = {
    id: string;
  };

  type OrderServiceTransitionOrderStatusParams = {
    id: string;
  };

  type OrderServiceTransitionOrderTerminationParams = {
    id: string;
  };

  type OrderServiceUpdateOrderParams = {
    id: string;
  };

  type OrderShippingDocument = {
    id?: string;
    orderId?: string;
    houseNo?: string;
    releaseType?: string;
    status?: number;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type OrderShippingDocumentInput = {
    id?: string;
    houseNo: string;
    /** release_type 是分单（HBL）签放方式。 */
    releaseType?: string;
    note?: string;
  };

  type OrderShippingDocumentServiceAddShippingDocumentParams = {
    orderId: string;
  };

  type OrderShippingDocumentServiceListShippingDocumentsParams = {
    orderId: string;
  };

  type OrderShippingDocumentServiceRemoveShippingDocumentParams = {
    orderId: string;
    id: string;
  };

  type OrderShippingDocumentServiceTransitionShippingDocumentStatusParams = {
    orderId: string;
    id: string;
  };

  type OrderShippingDocumentServiceUpdateShippingDocumentParams = {
    orderId: string;
    id: string;
  };

  type OrderTagServiceListOrderTagOptionsParams = {
    businessType?: number;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type OrderUnlockApproverCandidateData = {
    id?: string;
    requestId?: string;
    userId?: string;
    membershipId?: string;
    roleId?: string;
    displayNameSnapshot?: string;
    dingtalkUseridSnapshot?: string;
  };

  type OrderUnlockRequestData = {
    id?: string;
    orderId?: string;
    orderNo?: string;
    lockRecordId?: string;
    lockGeneration?: string;
    requestedBy?: string;
    requestedByName?: string;
    requestedAt?: string;
    reason?: string;
    expectedOrderVersion?: string;
    idempotencyKey?: string;
    route?: string;
    status?: string;
    dingtalkProcessInstanceId?: string;
    dingtalkProcessCode?: string;
    decidedBy?: string;
    decidedByName?: string;
    decidedAt?: string;
    decisionSource?: string;
    failureCode?: string;
    failureMessage?: string;
    supersededByRequestId?: string;
    unlockedAt?: string;
    resultOrderVersion?: string;
    approverCandidates?: OrderUnlockApproverCandidateData[];
    businessType?: number;
  };

  type OrderUnlockResultData = {
    state?: OrderLockStateData;
    request?: OrderUnlockRequestData;
  };

  type Organization = {
    id?: string;
    code?: string;
    name?: string;
    baseCurrency?: string;
  };

  type OrganizationChoice = {
    organizationId: string;
    organizationName: string;
    organizationCode: string;
    isDefault?: boolean;
  };

  type Partner = {
    id?: string;
    organizationId?: string;
    code?: string;
    legalName?: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    enabled?: boolean;
    roles?: PartnerRole[];
    contacts?: PartnerContact[];
    aliases?: PartnerAlias[];
    createdAt?: string;
    updatedAt?: string;
    profile?: PartnerProfile;
    assignments?: PartnerAssignment[];
    isCasual?: boolean;
  };

  type PartnerAccount = {
    id?: string;
    partnerId?: string;
    name?: string;
    accountHolder?: string;
    currency?: string;
    bankName?: string;
    accountNo?: string;
    swiftCode?: string;
    usage?: number;
    isDefaultReceivable?: boolean;
    isDefaultPayable?: boolean;
    enabled?: boolean;
    remark?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerAccountInput = {
    name: string;
    accountHolder: string;
    currency: string;
    bankName: string;
    accountNo: string;
    swiftCode?: string;
    usage: number;
    isDefaultReceivable?: boolean;
    isDefaultPayable?: boolean;
    enabled: boolean;
    remark?: string;
  };

  type PartnerAlias = {
    id?: string;
    aliasName?: string;
    sortOrder?: number;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerAliasInput = {
    aliasName?: string;
    sortOrder?: number;
  };

  type PartnerAssignment = {
    id?: string;
    role?: number;
    userId?: string;
    organizationId?: string;
    createdAt?: string;
    updatedAt?: string;
    sortOrder?: number;
  };

  type PartnerAssignmentInput = {
    role: number;
    userId: string;
    organizationId: string;
  };

  type PartnerAssignmentOption = {
    userId?: string;
    displayName?: string;
    organizationId?: string;
    organizationName?: string;
    membershipEnabled?: boolean;
  };

  type PartnerAssociations = {
    partnerIds?: string[];
  };

  type PartnerAttachment = {
    id?: string;
    partnerId?: string;
    idempotencyKey?: string;
    fileName?: string;
    mimeType?: string;
    fileSize?: string;
    objectKey?: string;
    checksum?: string;
    uploadedBy?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerAuditLog = {
    id?: string;
    userId?: string;
    userDisplayName?: string;
    action?: string;
    result?: string;
    traceId?: string;
    details?: Record<string, any>;
    createdAt?: string;
  };

  type PartnerContact = {
    id?: string;
    name?: string;
    phone?: string;
    email?: string;
    note?: string;
    isPrimary?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerContactInput = {
    name?: string;
    phone?: string;
    email?: string;
    note?: string;
    isPrimary?: boolean;
  };

  type PartnerContract = {
    id?: string;
    partnerId?: string;
    contractNo?: string;
    name?: string;
    status?: number;
    startDate?: string;
    endDate?: string;
    paymentTerms?: string;
    disputeResolution?: string;
    otherNotes?: string;
    createdAt?: string;
    updatedAt?: string;
    allowedStatuses?: number[];
  };

  type PartnerExportItem = {
    code?: string;
    legalName?: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    enabled?: boolean;
    roles?: number[];
  };

  type PartnerImportItemInput = {
    code: string;
    legalName: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    roles?: PartnerRoleInput[];
    contacts?: PartnerContactInput[];
    aliases?: PartnerAliasInput[];
    profile?: PartnerProfile;
    assignments?: PartnerAssignmentInput[];
  };

  type PartnerInvoiceProfile = {
    id?: string;
    partnerId?: string;
    invoiceTitle?: string;
    taxpayerIdentificationNo?: string;
    registeredAddress?: string;
    registeredPhone?: string;
    bankName?: string;
    bankAccount?: string;
    defaultInvoiceType?: string;
    version?: string;
    isDefault?: boolean;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerProfile = {
    nameEn?: string;
    addressEn?: string;
    countryCode?: string;
    provinceCode?: string;
    cityCode?: string;
    districtCode?: string;
    addressDetail?: string;
    nature?: string;
    developmentMethod?: string;
    customerTypes?: number[];
    businessTypes?: number[];
    remark?: string;
  };

  type PartnerRole = {
    type?: number;
    enabled?: boolean;
    blacklisted?: boolean;
    blacklistReason?: string;
    blacklistedAt?: string;
    blacklistedBy?: string;
    settlementRule?: PartnerSettlementRule;
  };

  type PartnerRoleInput = {
    type?: number;
    enabled?: boolean;
    settlementRule?: PartnerSettlementRuleInput;
  };

  type PartnerServiceCreatePartnerAccountParams = {
    partnerId: string;
  };

  type PartnerServiceCreatePartnerContractParams = {
    partnerId: string;
  };

  type PartnerServiceCreatePartnerInvoiceProfileParams = {
    partnerId: string;
  };

  type PartnerServiceCreatePartnerSettlementRuleParams = {
    partnerId: string;
    roleType: number;
  };

  type PartnerServiceCreatePartnerShippingPresetParams = {
    partnerId: string;
  };

  type PartnerServiceExportPartnersParams = {
    keyword?: string;
    role?: number;
    enabled?: boolean;
  };

  type PartnerServiceGetPartnerParams = {
    id: string;
  };

  type PartnerServiceListPartnerAccountsParams = {
    partnerId: string;
    enabled?: boolean;
    usage?: number;
    currency?: string;
  };

  type PartnerServiceListPartnerAttachmentsParams = {
    partnerId: string;
  };

  type PartnerServiceListPartnerAuditLogsParams = {
    partnerId: string;
    page?: number;
    pageSize?: number;
  };

  type PartnerServiceListPartnerContractsParams = {
    partnerId: string;
    status?: number;
  };

  type PartnerServiceListPartnerInvoiceProfilesParams = {
    partnerId: string;
  };

  type PartnerServiceListPartnerSettlementRulesParams = {
    partnerId: string;
    roleType: number;
  };

  type PartnerServiceListPartnerShippingPresetsParams = {
    partnerId: string;
    presetType?: number;
    enabled?: boolean;
  };

  type PartnerServiceListPartnersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    role?: number;
    enabled?: boolean;
    isCasual?: boolean;
  };

  type PartnerServiceRegisterPartnerAttachmentParams = {
    partnerId: string;
  };

  type PartnerServiceSearchPartnerAssignmentOptionsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type PartnerServiceSetSupplierBlacklistParams = {
    id: string;
  };

  type PartnerServiceUpdatePartnerAccountParams = {
    partnerId: string;
    id: string;
  };

  type PartnerServiceUpdatePartnerContractParams = {
    partnerId: string;
    id: string;
  };

  type PartnerServiceUpdatePartnerInvoiceProfileParams = {
    partnerId: string;
    id: string;
  };

  type PartnerServiceUpdatePartnerParams = {
    id: string;
  };

  type PartnerServiceUpdatePartnerSettlementRuleParams = {
    partnerId: string;
    roleType: number;
    id: string;
  };

  type PartnerServiceUpdatePartnerShippingPresetParams = {
    partnerId: string;
    id: string;
  };

  type PartnerSettlementRule = {
    id?: string;
    partnerRoleId?: string;
    statementMode?: number;
    settlementMethod?: number;
    settlementDay?: number;
    settlementCycleDays?: number;
    settlementBase?: number;
    settlementCurrency?: string;
    isActive?: boolean;
    createdAt?: string;
    updatedAt?: string;
    creditLimitMinor?: string;
    creditCurrency?: string;
    paymentTermsDays?: number;
  };

  type PartnerSettlementRuleInput = {
    statementMode: number;
    settlementMethod: number;
    settlementDay?: number;
    settlementCycleDays?: number;
    settlementBase?: number;
    settlementCurrency: string;
    isActive?: boolean;
    creditLimitMinor?: string;
    creditCurrency?: string;
    paymentTermsDays?: number;
  };

  type PartnerShippingPartyPayload = {
    companyName?: string;
    address?: string;
    contactName?: string;
    phone?: string;
    email?: string;
    countryCode?: string;
    taxIdentifier?: string;
  };

  type PartnerShippingPreset = {
    id?: string;
    partnerId?: string;
    presetType?: number;
    title?: string;
    party?: PartnerShippingPartyPayload;
    text?: PartnerShippingTextPayload;
    isDefault?: boolean;
    sortOrder?: number;
    remark?: string;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerShippingPresetInput = {
    presetType: number;
    title: string;
    party?: PartnerShippingPartyPayload;
    text?: PartnerShippingTextPayload;
    isDefault?: boolean;
    sortOrder?: number;
    remark?: string;
    enabled?: boolean;
  };

  type PartnerShippingTextPayload = {
    content?: string;
    code?: string;
  };

  type Port = {
    id?: string;
    organizationId?: string;
    unLocode?: string;
    nameZh?: string;
    nameEn?: string;
    countryCode?: string;
    transportModes?: string[];
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
    sourceVersion?: string;
    sourceHash?: string;
  };

  type PrepareEnterpriseResourceImageUploadRequest = {
    fileName: string;
    mimeType: string;
    fileSize: string;
    checksum: string;
  };

  type PrepareEnterpriseResourceImageUploadResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    uploadUrl?: string;
    objectKey?: string;
    headers?: Record<string, any>;
    expiresAt?: string;
    traceId?: string;
  };

  type PreviewBillBatchRequest = {
    feeIds: string[];
    groupingPolicy: BillGroupingPolicy;
    organizationId: string;
    groupConfigs?: BillBatchPreviewGroupConfigInput[];
  };

  type PreviewBillBatchResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillBatchPreviewGroup[];
    previewToken?: string;
    traceId?: string;
    /** netting_pairs 仅在对冲建账模式下返回，按结算单位与账单币种给出抵销前后金额。 */
    nettingPairs?: BillBatchNettingPair[];
  };

  type PreviewChangeSeaDocumentModeRequest = {
    orderId: string;
    targetMode: number;
    newHouseBill?: SeaHouseBillInput;
    reason: string;
  };

  type PreviewChangeSeaDocumentModeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentModeChangePreview;
    traceId?: string;
  };

  type PreviewCommissionRequest = {
    verificationId?: string;
    employeeId: string;
    ruleId: string;
    nettingId?: string;
  };

  type PreviewCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CommissionCalculation;
    traceId?: string;
  };

  type PreviewEnterpriseResourceImportRequest = {
    resourceType: number;
    rows: EnterpriseResourceInput[];
  };

  type PreviewEnterpriseResourceImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    rows?: EnterpriseResourceImportRow[];
    validCount?: number;
    invalidCount?: number;
    createdCount?: number;
    traceId?: string;
    conflictCount?: number;
    overwriteAllowed?: boolean;
  };

  type PreviewExchangeRateImportRequest = {
    fileName: string;
    fileContent: string;
  };

  type PreviewExchangeRateImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateImportBatch;
    previewToken?: string;
    traceId?: string;
  };

  type PreviewNettingRequest = {
    organizationId: string;
    settlementPartyId: string;
    currency: string;
  };

  type PreviewNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNettingPreview;
    traceId?: string;
  };

  type PreviewSeaDocumentAmendmentRequest = {
    orderId: string;
    documentType: number;
    documentId: string;
    expectedOrderVersion: string;
    expectedDocumentVersion: string;
    expectedCurrentVersionId: string;
    reason: string;
    input: SeaDocumentAmendmentInput;
  };

  type PreviewSeaDocumentAmendmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentAmendmentPreview;
    traceId?: string;
  };

  type PreviewSeaDocumentVoidRequest = {
    orderId: string;
    documentType: number;
    documentId: string;
    expectedOrderVersion: string;
    expectedDocumentVersion: string;
    expectedCurrentVersionId: string;
    reason: string;
  };

  type PreviewSeaDocumentVoidResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaDocumentVoidPreview;
    traceId?: string;
  };

  type PreviewSeaOrderReassignmentRequest = {
    orderId: string;
    target: SeaOrderReassignmentTargetInput;
  };

  type PreviewSeaOrderReassignmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderReassignmentPreviewData;
    traceId?: string;
  };

  type PreviewSeaOrderSplitRequest = {
    orderId: string;
    note?: string;
    targets: SeaOrderSplitTargetInput[];
    results: SeaOrderSplitResultInput[];
    expectedVersions?: SeaOrderSplitExpectedVersions;
  };

  type PreviewSeaOrderSplitResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaOrderSplitPreviewData;
    traceId?: string;
  };

  type PreviewSeaTransportExecutionUpdateRequest = {
    orderId: string;
    expectedTransportExecutionVersion: string;
    input: SeaTransportExecutionUpdateInput;
    reason: string;
  };

  type PreviewSeaTransportExecutionUpdateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaTransportExecutionUpdatePreviewData;
    traceId?: string;
  };

  type RedFlushInvoiceRequest = {
    id: string;
    expectedVersion: string;
    redInvoiceNo: string;
    redInvoiceDate: string;
    reason: string;
  };

  type RedFlushInvoiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice;
    traceId?: string;
  };

  type RegisterAttachmentRequest = {
    orderId: string;
    docType: string;
    idempotencyKey: string;
    fileName: string;
    mimeType: string;
    fileSize: string;
    objectKey: string;
    checksum?: string;
  };

  type RegisterAttachmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAttachment;
    traceId?: string;
  };

  type RegisterDingTalkUserRequest = {
    /** 可选：专属邀请 Token；若提供则由服务端解析目标组织，杜绝客户端伪造 */
    invitationToken?: string;
    /** 可选：自选要加入的目标公司（公开注册通道用；若有 invitation_token 则由服务端优先从 Token 兑现） */
    organizationId?: string;
  };

  type RegisterDingTalkUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkRegistrationConfirmation;
    traceId?: string;
  };

  type RegisterPartnerAttachmentRequest = {
    partnerId: string;
    idempotencyKey: string;
    fileName: string;
    mimeType: string;
    fileSize: string;
    objectKey: string;
    checksum?: string;
  };

  type RegisterPartnerAttachmentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAttachment;
    traceId?: string;
  };

  type RejectDingTalkRegistrationRequest = {
    id: string;
    reason: string;
  };

  type RejectDingTalkRegistrationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveAbnormalCaseResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveAttachmentReferenceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveCargoItemResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveFeeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemovePersonnelResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveReleasePodResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RemoveShippingDocumentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type ReopenFeeRequest = {
    orderId: string;
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type ReopenFeeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderFee;
    traceId?: string;
  };

  type RequestOrderUnlockRequest = {
    orderId: string;
    expectedOrderVersion: string;
    idempotencyKey: string;
    reason?: string;
  };

  type RequestOrderUnlockResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderUnlockResultData;
    traceId?: string;
  };

  type RequeueBackgroundTaskRequest = {
    id: string;
  };

  type RequeueBackgroundTaskResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BackgroundTask;
    traceId?: string;
  };

  type ResetFeeLedgerPreferenceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerPreference;
    traceId?: string;
  };

  type ResetUserPasswordRequest = {
    id: string;
    password: string;
    username?: string;
  };

  type ResetUserPasswordResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type ResolveAbnormalCaseRequest = {
    orderId: string;
    id: string;
  };

  type ResolveAbnormalCaseResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAbnormalCase;
    traceId?: string;
  };

  type ResolveFeeExchangeRateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    exchangeRate?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    traceId?: string;
  };

  type ReverseNettingRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type ReverseNettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceNetting;
    traceId?: string;
  };

  type ReverseVerificationRequest = {
    id: string;
    expectedVersion: string;
    reason: string;
  };

  type ReverseVerificationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceVerification;
    traceId?: string;
  };

  type RevokeDingTalkInvitationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type RoleScope = {
    roleCode?: string;
    dataScope?: string;
  };

  type SameBatchOrderSummary = {
    orderId?: string;
    orderNo?: string;
    customerId?: string;
    customerReferenceNo?: string;
    bookingNo?: string;
    masterNo?: string;
    houseNo?: string;
    flowStatus?: number;
    matchSources?: string[];
    totalPackages?: number;
    totalGrossWeightKg?: number;
    totalVolumeCbm?: number;
    createdAt?: string;
  };

  type SaveSeaSharedContainerAllocationsDraftRequest = {
    id: string;
    expectedVersion: string;
    allocations?: SeaSharedContainerAllocationInput[];
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId: string;
  };

  type SaveSeaSharedContainerAllocationsDraftResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };

  type SeaBillContent = {
    shipperText?: string;
    consigneeText?: string;
    notifyPartyText?: string;
    secondNotifyPartyText?: string;
    marksText?: string;
    goodsDescriptionText?: string;
    packageCount?: number;
    packageUnit?: string;
    grossWeightKg?: number;
    volumeCbm?: number;
    freightTerms?: string;
    transportTerms?: string;
    billForm?: string;
    releaseType?: string;
    clauses?: string;
    foreignAgentText?: string;
  };

  type SeaDocumentAmendmentInput = {
    masterBillContent?: SeaBillContent;
    houseBill?: SeaHouseBillInput;
  };

  type SeaDocumentAmendmentPreview = {
    baseVersion?: SeaDocumentVersion;
    differences?: SeaDocumentFieldDifference[];
    impacts?: SeaDocumentDownstreamImpact[];
    executable?: boolean;
  };

  type SeaDocumentDownstreamImpact = {
    factType?: string;
    referenceId?: string;
    referenceNo?: string;
    message?: string;
    blocksExecution?: boolean;
  };

  type SeaDocumentEvent = {
    id?: string;
    eventType?: number;
    documentType?: number;
    documentId?: string;
    documentNo?: string;
    previousVersionId?: string;
    resultVersionId?: string;
    reason?: string;
    impactSummary?: string;
    createdBy?: string;
    createdAt?: string;
    previousMode?: number;
    targetMode?: number;
    confirmation?: SeaExternalConfirmationSummary;
  };

  type SeaDocumentFieldDifference = {
    field?: string;
    label?: string;
    beforeValue?: string;
    afterValue?: string;
  };

  type SeaDocumentModeChangePreview = {
    previousMode?: number;
    targetMode?: number;
    differences?: SeaDocumentFieldDifference[];
    impacts?: SeaDocumentDownstreamImpact[];
    executable?: boolean;
  };

  type SeaDocumentServiceExecuteChangeSeaDocumentModeParams = {
    orderId: string;
  };

  type SeaDocumentServiceExecuteSeaDocumentAmendmentParams = {
    orderId: string;
  };

  type SeaDocumentServiceExecuteSeaDocumentVoidParams = {
    orderId: string;
  };

  type SeaDocumentServiceGetSeaDocumentVersionParams = {
    orderId: string;
    versionId: string;
    documentType?: number;
  };

  type SeaDocumentServiceGetSeaOrderDocumentsParams = {
    orderId: string;
  };

  type SeaDocumentServiceListSeaDocumentEventsParams = {
    orderId: string;
    page?: number;
    pageSize?: number;
  };

  type SeaDocumentServiceListSeaHouseBillVersionsParams = {
    orderId: string;
    houseBillId: string;
    page?: number;
    pageSize?: number;
  };

  type SeaDocumentServiceListSeaMasterBillVersionsParams = {
    orderId: string;
    page?: number;
    pageSize?: number;
  };

  type SeaDocumentServicePreviewChangeSeaDocumentModeParams = {
    orderId: string;
  };

  type SeaDocumentServicePreviewSeaDocumentAmendmentParams = {
    orderId: string;
  };

  type SeaDocumentServicePreviewSeaDocumentVoidParams = {
    orderId: string;
  };

  type SeaDocumentServiceUpdateSeaHouseBillParams = {
    orderId: string;
    id: string;
  };

  type SeaDocumentServiceUpdateSeaMasterBillContentParams = {
    orderId: string;
  };

  type SeaDocumentVersion = {
    id?: string;
    documentType?: number;
    documentId?: string;
    orderId?: string;
    masterBillId?: string;
    versionNo?: string;
    sourceEntityVersion?: string;
    documentNo?: string;
    normalizedDocumentNo?: string;
    status?: string;
    source?: number;
    reason?: string;
    issuerPartnerId?: string;
    issuerOrganizationId?: string;
    issuerSource?: number;
    transportExecutionId?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    note?: string;
    content?: SeaBillContent;
    createdBy?: string;
    createdAt?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    confirmation?: SeaExternalConfirmationSummary;
  };

  type SeaDocumentVoidPreview = {
    baseVersion?: SeaDocumentVersion;
    differences?: SeaDocumentFieldDifference[];
    impacts?: SeaDocumentDownstreamImpact[];
    executable?: boolean;
  };

  type SeaExternalConfirmationInput = {
    confirmedByParty: string;
    confirmedAt: string;
    confirmationNote: string;
    confirmationAttachmentId?: string;
  };

  type SeaExternalConfirmationSummary = {
    confirmedByParty?: string;
    confirmedAt?: string;
    confirmationNote?: string;
    confirmationAttachmentId?: string;
    confirmationAttachmentName?: string;
  };

  type SeaHouseBill = {
    id?: string;
    organizationId?: string;
    orderId?: string;
    masterBillId?: string;
    houseNo?: string;
    issuerSource?: number;
    issuerOrganizationId?: string;
    issuerOrganizationName?: string;
    issuerPartnerId?: string;
    issuerPartnerName?: string;
    status?: number;
    version?: string;
    note?: string;
    content?: SeaBillContent;
    createdAt?: string;
    updatedAt?: string;
    currentVersionId?: string;
    immutableVersionCount?: string;
  };

  type SeaHouseBillInput = {
    id?: string;
    houseNo: string;
    issuerSource: number;
    issuerPartnerId?: string;
    note?: string;
    content?: SeaBillContent;
    expectedVersion?: string;
  };

  type SeaMasterBillCandidate = {
    id?: string;
    version?: string;
    masterNo?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    memberCount?: number;
    members?: SeaMasterBillMemberSummary[];
    transportExecutions?: SeaTransportExecution[];
  };

  type SeaMasterBillDetail = {
    id?: string;
    masterNo?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    status?: string;
    version?: string;
    content?: SeaBillContent;
    memberCount?: number;
    currentVersionId?: string;
    immutableVersionCount?: string;
  };

  type SeaMasterBillInput = {
    masterNo: string;
    candidateId?: string;
    expectedCandidateVersion?: string;
    correctionReason?: string;
    candidateTeId?: string;
    expectedCandidateTeVersion?: string;
  };

  type SeaMasterBillMemberSummary = {
    orderId?: string;
    orderNo?: string;
    customerReferenceNo?: string;
  };

  type SeaMasterBillSummary = {
    masterBillId?: string;
    masterNo?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    transportExecutionId?: string;
    originLocationId?: string;
    originLocationName?: string;
    dischargeLocationId?: string;
    dischargeLocationName?: string;
    transitLocationId?: string;
    transitLocationName?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    status?: string;
    version?: string;
    memberCount?: number;
    transportExecutionVersion?: string;
  };

  type SeaOrderChangeActionsData = {
    canSplit?: boolean;
    canReassign?: boolean;
    splitBlockedReasons?: string[];
    reassignBlockedReasons?: string[];
  };

  type SeaOrderChangeEventDetailData = {
    id?: string;
    eventType?: string;
    createdAt?: string;
    operatorId?: string;
    operatorName?: string;
    noteOrReason?: string;
    beforeSnapshotJson?: string;
    afterSnapshotJson?: string;
    conservationSnapshotJson?: string;
    splitSummary?: SeaOrderSplitEventSummary;
    reassignmentSummary?: SeaOrderReassignmentEventSummary;
  };

  type SeaOrderChangeEventSummary = {
    id?: string;
    eventType?: string;
    createdAt?: string;
    operatorId?: string;
    operatorName?: string;
    noteOrReason?: string;
    splitSummary?: SeaOrderSplitEventSummary;
    reassignmentSummary?: SeaOrderReassignmentEventSummary;
  };

  type SeaOrderChangeServiceExecuteSeaOrderReassignmentParams = {
    orderId: string;
  };

  type SeaOrderChangeServiceExecuteSeaOrderSplitParams = {
    orderId: string;
  };

  type SeaOrderChangeServiceExecuteSeaTransportExecutionUpdateParams = {
    orderId: string;
  };

  type SeaOrderChangeServiceGetSeaOrderChangeActionsParams = {
    orderId: string;
  };

  type SeaOrderChangeServiceGetSeaOrderChangeEventParams = {
    orderId: string;
    eventId: string;
    eventType?: string;
  };

  type SeaOrderChangeServiceGetSeaOrderSplitContextParams = {
    orderId: string;
  };

  type SeaOrderChangeServiceListSeaOrderChangeEventsParams = {
    orderId: string;
    page?: number;
    pageSize?: number;
  };

  type SeaOrderChangeServicePreviewSeaOrderReassignmentParams = {
    orderId: string;
  };

  type SeaOrderChangeServicePreviewSeaOrderSplitParams = {
    orderId: string;
  };

  type SeaOrderChangeServicePreviewSeaTransportExecutionUpdateParams = {
    orderId: string;
  };

  type SeaOrderDocumentInput = {
    documentStructure?: number;
    expectedLinkVersion?: string;
    expectedMblVersion?: string;
    masterBillContent?: SeaBillContent;
    houseBill?: SeaHouseBillInput;
  };

  type SeaOrderDocuments = {
    orderId?: string;
    documentStructure?: number;
    linkVersion?: string;
    masterBill?: SeaMasterBillDetail;
    allowedActions?: number[];
    houseBill?: SeaHouseBill;
  };

  type SeaOrderDocumentSummary = {
    documentStructure?: number;
    linkVersion?: string;
    houseNo?: string;
  };

  type SeaOrderReassignmentEventSummary = {
    orderId?: string;
    orderNo?: string;
    previousMasterNo?: string;
    targetMasterNo?: string;
    responsibilityType?: string;
    responsiblePartnerName?: string;
    reason?: string;
    confirmation?: SeaExternalConfirmationSummary;
  };

  type SeaOrderReassignmentPreviewData = {
    isValid?: boolean;
    errors?: string[];
    currentMasterBill?: SeaOrderSplitMasterBillSummary;
    targetMasterBill?: SeaOrderSplitMasterBillSummary;
    targetMemberCount?: number;
    differences?: VoyageDifferenceItem[];
    orderVersion?: string;
    currentLinkVersion?: string;
  };

  type SeaOrderReassignmentTargetInput = {
    targetType: string;
    candidateId?: string;
    candidateVersion?: string;
    masterNo?: string;
    shippingLineId?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    originLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    candidateTeId?: string;
    candidateTeVersion?: string;
  };

  type SeaOrderSplitAttachmentItem = {
    id?: string;
    assetId?: string;
    fileName?: string;
    mimeType?: string;
    fileSize?: string;
    docType?: string;
  };

  type SeaOrderSplitCargoAllocationInput = {
    cargoItemId: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
  };

  type SeaOrderSplitCargoItem = {
    id?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    version?: string;
  };

  type SeaOrderSplitContainerItem = {
    id?: string;
    containerNo?: string;
    containerSpecId?: string;
    containerSpecName?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    version?: string;
  };

  type SeaOrderSplitContainerPlanItem = {
    containerSpecId?: string;
    containerSpecName?: string;
    quantity?: number;
  };

  type SeaOrderSplitContextData = {
    orderId?: string;
    orderNo?: string;
    businessType?: string;
    shipmentType?: string;
    flowStatus?: string;
    orderVersion?: string;
    customerReferenceNo?: string;
    internalReferenceNo?: string;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    currentMasterBill?: SeaOrderSplitMasterBillSummary;
    currentLinkId?: string;
    currentLinkVersion?: string;
    documentStructure?: string;
    houseBills?: SeaOrderSplitHouseBillItem[];
    cargoItems?: SeaOrderSplitCargoItem[];
    containers?: SeaOrderSplitContainerItem[];
    draftFees?: SeaOrderSplitDraftFeeItem[];
    attachments?: SeaOrderSplitAttachmentItem[];
    containerPlans?: SeaOrderSplitContainerPlanItem[];
    attachmentReferenceFingerprint?: string;
    bookingNo?: string;
    currentHouseBill?: SeaOrderSplitHouseBillItem;
    sharedContainerAllocations?: SeaOrderSplitSharedContainerAllocationItem[];
  };

  type SeaOrderSplitCreatedOrder = {
    orderId?: string;
    orderNo?: string;
    clientResultKey?: string;
  };

  type SeaOrderSplitDraftFeeItem = {
    id?: string;
    feeCode?: string;
    feeName?: string;
    direction?: string;
    settlementPartyId?: string;
    settlementPartyName?: string;
    currency?: string;
    totalAmount?: string;
    baseCurrency?: string;
    baseCurrencyAmount?: string;
    version?: string;
  };

  type SeaOrderSplitEventSummary = {
    sourceOrderId?: string;
    sourceOrderNo?: string;
    resultCount?: number;
    results?: SeaOrderSplitResultSummaryItem[];
  };

  type SeaOrderSplitExpectedVersions = {
    orderVersion?: string;
    linkVersion?: string;
    cargoItemVersions?: Record<string, any>;
    containerVersions?: Record<string, any>;
    feeVersions?: Record<string, any>;
    candidateMblVersions?: Record<string, any>;
    attachmentReferenceFingerprint?: string;
    candidateTeVersions?: Record<string, any>;
    currentHblVersion?: string;
    sharedContainerVersions?: Record<string, any>;
  };

  type SeaOrderSplitHouseBillInput = {
    houseNo: string;
    issuerSource: string;
    issuerPartnerId?: string;
    note?: string;
  };

  type SeaOrderSplitHouseBillItem = {
    id?: string;
    houseNo?: string;
    status?: string;
    version?: string;
  };

  type SeaOrderSplitMasterBillSummary = {
    id?: string;
    masterNo?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    version?: string;
    originLocationId?: string;
    originLocationName?: string;
    dischargeLocationId?: string;
    dischargeLocationName?: string;
    transitLocationId?: string;
    transitLocationName?: string;
    transportExecutionId?: string;
    transportExecutionVersion?: string;
  };

  type SeaOrderSplitOrderReference = {
    orderId?: string;
    orderNo?: string;
  };

  type SeaOrderSplitPreviewData = {
    isValid?: boolean;
    conservationPassed?: boolean;
    validationErrors?: SeaOrderSplitValidationError[];
    baseline?: SeaOrderSplitQuantitySummary;
    allocated?: SeaOrderSplitQuantitySummary;
    remaining?: SeaOrderSplitQuantitySummary;
    results?: SeaOrderSplitPreviewResultItem[];
  };

  type SeaOrderSplitPreviewResultItem = {
    clientResultKey?: string;
    resultRole?: string;
    clientTargetKey?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    containerCount?: number;
    houseBillCount?: number;
    feeCount?: number;
    attachmentCount?: number;
    containerPlans?: SeaOrderSplitContainerPlanItem[];
    internalReferenceNo?: string;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    houseNo?: string;
  };

  type SeaOrderSplitQuantitySummary = {
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    containerCount?: number;
    houseBillCount?: number;
    feeCount?: number;
  };

  type SeaOrderSplitResultInput = {
    clientResultKey: string;
    resultRole: string;
    clientTargetKey: string;
    draftFeeIds?: string[];
    attachmentReferenceIds?: string[];
    internalReferenceNo?: string;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    houseBill?: SeaOrderSplitHouseBillInput;
    cargoAllocations?: SeaOrderSplitCargoAllocationInput[];
    containerIds?: string[];
    sharedContainerAllocations?: SeaOrderSplitSharedContainerAllocationInput[];
  };

  type SeaOrderSplitResultSummaryItem = {
    resultRole?: string;
    orderId?: string;
    orderNo?: string;
    finalMasterNo?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
  };

  type SeaOrderSplitSharedContainerAllocationInput = {
    allocationId: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
  };

  type SeaOrderSplitSharedContainerAllocationItem = {
    allocationId?: string;
    sharedContainerId?: string;
    containerNo?: string;
    containerSpecId?: string;
    containerSpecName?: string;
    cargoItemId?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    sharedContainerVersion?: string;
  };

  type SeaOrderSplitTargetInput = {
    clientTargetKey: string;
    targetType: string;
    candidateId?: string;
    candidateVersion?: string;
    masterNo?: string;
    shippingLineId?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    originLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    candidateTeId?: string;
    candidateTeVersion?: string;
  };

  type SeaOrderSplitValidationError = {
    reason?: string;
    message?: string;
    field?: string;
    clientResultKey?: string;
    houseBillId?: string;
    containerId?: string;
    cargoItemId?: string;
    feeId?: string;
    baselineValue?: string;
    allocatedValue?: string;
    diffValue?: string;
  };

  type SearchBillingUnitsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillingUnit[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type SearchCurrenciesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Currency[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type SearchEnterpriseResourceAssigneeOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResourceAssigneeOption[];
    total?: string;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type SearchEnterpriseResourcePartnerOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResourcePartnerOption[];
    total?: string;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type SearchFeeSettingsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeSetting[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type SearchPartnerAssignmentOptionsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAssignmentOption[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type SearchTaxableServicesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: TaxableService[];
    traceId?: string;
    total?: number;
    page?: number;
    pageSize?: number;
  };

  type SeaSharedContainer = {
    id?: string;
    organizationId?: string;
    transportExecutionId?: string;
    containerNo?: string;
    containerSpecId?: string;
    containerSpecName?: string;
    sealNo?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    status?: number;
    confirmedAt?: string;
    confirmedBy?: string;
    confirmedByName?: string;
    note?: string;
    version?: string;
    allocations?: SeaSharedContainerAllocation[];
    progress?: SeaSharedContainerProgress;
    createdAt?: string;
    updatedAt?: string;
  };

  type SeaSharedContainerAllocation = {
    id?: string;
    sharedContainerId?: string;
    orderId?: string;
    orderNo?: string;
    houseBillId?: string;
    houseNo?: string;
    cargoItemId?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    version?: string;
    createdAt?: string;
    updatedAt?: string;
    orderVersion?: string;
    linkVersion?: string;
    houseBillVersion?: string;
    cargoItemVersion?: string;
  };

  type SeaSharedContainerAllocationInput = {
    orderId: string;
    houseBillId: string;
    cargoItemId: string;
    packageCount: number;
    grossWeightKg: string;
    volumeCbm: string;
    expectedOrderVersion: string;
    expectedLinkVersion: string;
    expectedHouseBillVersion: string;
    expectedCargoItemVersion: string;
  };

  type SeaSharedContainerCandidateCargoItem = {
    id?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: string;
    volumeCbm?: string;
    version?: string;
  };

  type SeaSharedContainerCandidateOrder = {
    orderId?: string;
    orderNo?: string;
    houseBillId?: string;
    houseNo?: string;
    orderVersion?: string;
    linkVersion?: string;
    houseBillVersion?: string;
    cargoItems?: SeaSharedContainerCandidateCargoItem[];
  };

  type SeaSharedContainerInput = {
    transportExecutionId: string;
    containerNo: string;
    containerSpecId: string;
    sealNo?: string;
    packageCount: number;
    grossWeightKg: string;
    volumeCbm: string;
    note?: string;
  };

  type SeaSharedContainerProgress = {
    allocatedPackageCount?: number;
    allocatedGrossWeightKg?: string;
    allocatedVolumeCbm?: string;
    remainingPackageCount?: number;
    remainingGrossWeightKg?: string;
    remainingVolumeCbm?: string;
    containerBalanced?: boolean;
    cargoBalanced?: boolean;
  };

  type SeaSharedContainerServiceConfirmSeaSharedContainerParams = {
    id: string;
  };

  type SeaSharedContainerServiceDeleteSeaSharedContainerParams = {
    id: string;
    expectedVersion?: string;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId?: string;
  };

  type SeaSharedContainerServiceGetSeaSharedContainerParams = {
    id: string;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId?: string;
  };

  type SeaSharedContainerServiceListSeaSharedContainerCandidatesParams = {
    transportExecutionId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文 */
    orderId?: string;
  };

  type SeaSharedContainerServiceListSeaSharedContainersParams = {
    transportExecutionId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文 */
    orderId?: string;
  };

  type SeaSharedContainerServiceSaveSeaSharedContainerAllocationsDraftParams = {
    id: string;
  };

  type SeaSharedContainerServiceUpdateSeaSharedContainerParams = {
    id: string;
  };

  type SeaSharedContainerServiceWithdrawSeaSharedContainerParams = {
    id: string;
  };

  type SeaTransportExecution = {
    id?: string;
    shippingLineId?: string;
    shippingLineName?: string;
    originLocationId?: string;
    originLocationName?: string;
    dischargeLocationId?: string;
    dischargeLocationName?: string;
    transitLocationId?: string;
    transitLocationName?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
    version?: string;
  };

  type SeaTransportExecutionUpdateInput = {
    originLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    vesselName?: string;
    voyageNo?: string;
    etd?: string;
    eta?: string;
  };

  type SeaTransportExecutionUpdatePreviewData = {
    transportExecutionId?: string;
    transportExecutionVersion?: string;
    memberOrderIds?: string[];
    differences?: VoyageDifferenceItem[];
    impacts?: SeaDocumentDownstreamImpact[];
    executable?: boolean;
  };

  type SeaVoyageConflict = {
    field?: string;
    masterValue?: string;
    orderValue?: string;
    message?: string;
  };

  type SetMilestoneRequest = {
    orderId: string;
    type: string;
    expectedOrderVersion: string;
    occurredAt?: string;
    note?: string;
    clearOccurredAt?: boolean;
  };

  type SetMilestoneResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderMilestone;
    traceId?: string;
  };

  type SetSupplierBlacklistRequest = {
    id: string;
    blacklisted?: boolean;
    reason: string;
  };

  type SetSupplierBlacklistResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
    traceId?: string;
  };

  type SettlementServiceCancelBillParams = {
    id: string;
  };

  type SettlementServiceCancelCashflowParams = {
    id: string;
  };

  type SettlementServiceCancelCommissionAdjustmentParams = {
    id: string;
  };

  type SettlementServiceCancelCommissionParams = {
    id: string;
  };

  type SettlementServiceCancelInvoiceParams = {
    id: string;
  };

  type SettlementServiceCancelNettingParams = {
    id: string;
  };

  type SettlementServiceConfirmBillBatchParams = {
    id: string;
  };

  type SettlementServiceConfirmBillParams = {
    id: string;
  };

  type SettlementServiceConfirmCashflowParams = {
    id: string;
  };

  type SettlementServiceConfirmCommissionAdjustmentParams = {
    id: string;
  };

  type SettlementServiceConfirmCommissionParams = {
    id: string;
  };

  type SettlementServiceConfirmNettingParams = {
    id: string;
  };

  type SettlementServiceCreateCommissionAdjustmentParams = {
    commissionId: string;
  };

  type SettlementServiceExportCommissionsParams = {
    keyword?: string;
    status?: number;
    commissionDateFrom?: string;
    commissionDateTo?: string;
    organizationId?: string;
  };

  type SettlementServiceGetBillParams = {
    id: string;
  };

  type SettlementServiceGetCommissionParams = {
    id: string;
  };

  type SettlementServiceGetCreditLimitControlPolicyParams = {
    organizationId?: string;
  };

  type SettlementServiceGetFeeLedgerOrderDetailParams = {
    orderId: string;
  };

  type SettlementServiceGetInvoiceParams = {
    id: string;
  };

  type SettlementServiceGetNettingParams = {
    id: string;
  };

  type SettlementServiceIssueInvoiceParams = {
    id: string;
  };

  type SettlementServiceListBillCreationCandidatesParams = {
    organizationId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
  };

  type SettlementServiceListBillSettlementAccountCandidatesParams = {
    organizationId?: string;
    settlementPartyId?: string;
    direction?: string;
    currency?: string;
  };

  type SettlementServiceListBillSettlementAccountUpdateCandidatesParams = {
    billId?: string;
  };

  type SettlementServiceListBillsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    currency?: string;
    billDateFrom?: string;
    billDateTo?: string;
    tagIds?: string[];
    organizationId?: string;
    dueDateFrom?: string;
    dueDateTo?: string;
    onlyUnsettled?: boolean;
    onlyOverdue?: boolean;
  };

  type SettlementServiceListCashflowsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    currency?: string;
    organizationId?: string;
  };

  type SettlementServiceListCommissionCandidatesParams = {
    verificationId?: string;
    ruleId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    organizationId?: string;
  };

  type SettlementServiceListCommissionEmployeesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    organizationId?: string;
  };

  type SettlementServiceListCommissionNettingCandidatesParams = {
    organizationId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type SettlementServiceListCommissionRuleCandidatesParams = {
    organizationId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    personnelRole?: string;
  };

  type SettlementServiceListCommissionRulesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    personnelRole?: string;
    enabled?: boolean;
    organizationId?: string;
  };

  type SettlementServiceListCommissionsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: number;
    commissionDateFrom?: string;
    commissionDateTo?: string;
    organizationId?: string;
  };

  type SettlementServiceListCommissionVerificationCandidatesParams = {
    organizationId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type SettlementServiceListFeeLedgerParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    businessType?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    currency?: string;
    expenseDateFrom?: string;
    expenseDateTo?: string;
    customerId?: string;
    financialProgress?: number;
    billNo?: string;
    financeLocked?: boolean;
    tagIds?: string[];
    organizationId?: string;
  };

  type SettlementServiceListFinanceBillTagAssignmentOptionsParams = {
    organizationId?: string;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type SettlementServiceListFinanceBillTagOptionsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
    organizationId?: string;
  };

  type SettlementServiceListFinanceFeeTagAssignmentOptionsParams = {
    organizationId?: string;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type SettlementServiceListFinanceFeeTagOptionsParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
    organizationId?: string;
  };

  type SettlementServiceListFinanceOrganizationOptionsParams = {
    purpose?: number;
    keyword?: string;
  };

  type SettlementServiceListFinanceSettlementPartyOptionsParams = {
    purpose?: number;
    organizationId?: string;
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type SettlementServiceListInvoiceCreationBillsParams = {
    organizationId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    settlementPartyId?: string;
    currency?: string;
  };

  type SettlementServiceListInvoiceProfilesForBillParams = {
    billId?: string;
  };

  type SettlementServiceListInvoicesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: number;
    organizationId?: string;
  };

  type SettlementServiceListNettingsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: number;
    settlementPartyId?: string;
    currency?: string;
    organizationId?: string;
  };

  type SettlementServiceListVerificationCreationCandidatesParams = {
    organizationId?: string;
    direction?: string;
    settlementPartyId?: string;
    currency?: string;
  };

  type SettlementServiceListVerificationsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: number;
    organizationId?: string;
  };

  type SettlementServiceMarkCommissionAdjustmentPaidParams = {
    id: string;
  };

  type SettlementServiceMarkCommissionPaidParams = {
    id: string;
  };

  type SettlementServiceRedFlushInvoiceParams = {
    id: string;
  };

  type SettlementServiceResetFeeLedgerPreferenceParams = {
    version?: string;
  };

  type SettlementServiceReverseNettingParams = {
    id: string;
  };

  type SettlementServiceReverseVerificationParams = {
    id: string;
  };

  type SettlementServiceUpdateBillParams = {
    id: string;
  };

  type SettlementServiceUpdateCommissionRuleParams = {
    id: string;
  };

  type ShippingLine = {
    id?: string;
    organizationId?: string;
    scacCode?: string;
    nameZh?: string;
    nameEn?: string;
    countryCode?: string;
    trackingUrl?: string;
    alliance?: string;
    containerPrefixes?: string[];
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type SwitchOrganizationRequest = {
    organizationId: string;
  };

  type SwitchOrganizationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
    /** 与登录响应同构：切换后的成员资格组织候选列表。 */
    organizationChoices?: OrganizationChoice[];
  };

  type TaxableService = {
    id?: string;
    organizationId?: string;
    name?: string;
    shortName?: string;
    goodsCode?: string;
    defaultTaxRate?: string;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type TerminateUserRequest = {
    id: string;
  };

  type TerminateUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type TransferDingTalkRegistrationRequest = {
    userId: string;
    targetOrganizationId: string;
    reason: string;
  };

  type TransferDingTalkRegistrationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type TransitionOrderClosureRequest = {
    id: string;
    expectedVersion: string;
    targetStatus: number;
    reason: string;
  };

  type TransitionOrderClosureResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type TransitionOrderStatusRequest = {
    id: string;
    expectedVersion: string;
    targetFlowStatus: number;
    reason?: string;
  };

  type TransitionOrderStatusResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type TransitionOrderTerminationRequest = {
    id: string;
    expectedVersion: string;
    targetStatus: number;
    terminationType?: number;
    reason: string;
  };

  type TransitionOrderTerminationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type TransitionReleasePodStatusRequest = {
    orderId: string;
    id: string;
    expectedStatus: number;
    toStatus: number;
  };

  type TransitionReleasePodStatusResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod;
    traceId?: string;
  };

  type TransitionShippingDocumentStatusRequest = {
    orderId: string;
    id: string;
    expectedStatus: number;
    toStatus: number;
  };

  type TransitionShippingDocumentStatusResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument;
    traceId?: string;
  };

  type UpdateAirlineRequest = {
    id: string;
    icaoCode?: string;
    awbPrefix?: string;
    nameZh?: string;
    nameEn: string;
    countryCode: string;
    cargoOnly?: boolean;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
  };

  type UpdateAirlineResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airline;
    traceId?: string;
  };

  type UpdateAirportRequest = {
    id: string;
    icaoCode?: string;
    nameZh: string;
    nameEn: string;
    cityNameZh: string;
    cityNameEn?: string;
    countryCode: string;
    sortOrder?: number;
    enabled?: boolean;
  };

  type UpdateAirportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airport;
    traceId?: string;
  };

  type UpdateBilledFeeEditPolicyRequest = {
    enabled?: boolean;
    editableFields?: number[];
    expectedVersion: string;
  };

  type UpdateBilledFeeEditPolicyResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BilledFeeEditPolicy;
    traceId?: string;
  };

  type UpdateBillingUnitRequest = {
    id: string;
    code: string;
    name: string;
    sortOrder?: number;
    enabled?: boolean;
    isContainerUnit?: boolean;
  };

  type UpdateBillingUnitResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillingUnit;
    traceId?: string;
  };

  type UpdateBillRequest = {
    id: string;
    billDate: string;
    dueDate?: string;
    note?: string;
    expectedVersion: string;
    statementTitle?: string;
    paymentTermsDays?: number;
    settlementAccountId: string;
    estimatedInvoiceCurrency?: string;
    estimatedInvoiceRate?: string;
  };

  type UpdateBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill;
    traceId?: string;
  };

  type UpdateCargoItemRequest = {
    orderId: string;
    id: string;
    cargoName: string;
    packageCount: number;
    grossWeightKg: number;
    volumeCbm: number;
    netWeightKg?: number;
    note?: string;
    expectedVersion: string;
  };

  type UpdateCargoItemResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderCargoItem;
    traceId?: string;
  };

  type UpdateCommissionRuleRequest = {
    id: string;
    rule: CommissionRuleInput;
    expectedVersion: string;
  };

  type UpdateCommissionRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionRule;
    traceId?: string;
  };

  type UpdateContainerRequest = {
    orderId: string;
    id: string;
    containerNo: string;
    containerSpecId: string;
    sealNo?: string;
    grossWeightKg: number;
    volumeCbm: number;
    note?: string;
    packageCount: number;
    expectedVersion: string;
  };

  type UpdateContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer;
    traceId?: string;
  };

  type UpdateCreditLimitControlPolicyRequest = {
    organizationId?: string;
    allowSelectionWhenCreditExceeded?: boolean;
    expectedVersion: string;
  };

  type UpdateCreditLimitControlPolicyResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CreditLimitControlPolicy;
    traceId?: string;
  };

  type UpdateEnterpriseResourceRequest = {
    id: string;
    resource: EnterpriseResourceInput;
  };

  type UpdateEnterpriseResourceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseResource;
    traceId?: string;
  };

  type UpdateEnterpriseTagGroupRequest = {
    id: string;
    group: EnterpriseTagGroupInput;
  };

  type UpdateEnterpriseTagGroupResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: EnterpriseTagGroup;
    traceId?: string;
  };

  type UpdateExchangeRateSettingRequest = {
    id: string;
    fromCurrency: string;
    toCurrency: string;
    /** 示例：2026-08-27T09:30:00+08:00。 */
    effectiveFrom: string;
    effectiveTo?: string;
    rate: string;
  };

  type UpdateExchangeRateSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateSetting;
    traceId?: string;
  };

  type UpdateFeeLedgerPreferenceRequest = {
    columns: FeeLedgerColumnPreference[];
    pageSize: number;
    sortField?: string;
    sortDirection?: string;
    rowColors: FeeLedgerRowColors;
    version?: string;
  };

  type UpdateFeeLedgerPreferenceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeLedgerPreference;
    traceId?: string;
  };

  type UpdateFeeRequest = {
    orderId: string;
    id: string;
    direction: number;
    settlementPartyId: string;
    quantity: string;
    unitPrice: string;
    currency: string;
    expenseDate: string;
    note?: string;
    exchangeRateOverride?: string;
    feeSettingId: string;
    billingUnitId: string;
    expectedVersion: string;
    taxInclusive?: boolean;
    /** tax_rate 仅在修改已进入草稿账单的费用时作为目标税率；不传时沿用费用项目默认或现有税率。 */
    taxRate?: string;
    /** fee_name 仅覆盖当前费用及其草稿账单行快照，不修改费用设置主数据。 */
    feeName?: string;
  };

  type UpdateFeeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderFee;
    traceId?: string;
  };

  type UpdateFeeSettingRequest = {
    id: string;
    feeCode: string;
    nameZh: string;
    nameEn?: string;
    aliasName?: string;
    serviceTypeId?: string;
    defaultCurrency: string;
    billingUnitId: string;
    abnormalCaseId?: string;
    taxRate: string;
    taxableServiceId: string;
    enabled?: boolean;
    sortOrder?: number;
  };

  type UpdateFeeSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FeeSetting;
    traceId?: string;
  };

  type UpdateItemRequest = {
    id: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    kind: number;
    attributes?: MasterDataAttributes;
  };

  type UpdateItemResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem;
    traceId?: string;
  };

  type UpdateNumberRuleRequest = {
    id: string;
    prefix?: string;
    dateFormat: number;
    sequenceLength: number;
    resetPolicy: number;
    enabled?: boolean;
  };

  type UpdateNumberRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: NumberRule;
    traceId?: string;
  };

  type UpdateOrderRequest = {
    id: string;
    expectedVersion: string;
    customerId?: string;
    businessType?: number;
    tradeDirection?: number;
    tradeTerm?: number;
    paymentTerm?: number;
    shippingLineId?: string;
    bookingAgentId?: string;
    shipmentType?: number;
    containerOwnership?: number;
    shipmentMode?: number;
    serviceTypeIds?: string[];
    cargoCategoryIds?: string[];
    originLocationId?: string;
    destinationLocationId?: string;
    dischargeLocationId?: string;
    transitLocationId?: string;
    vesselVoyage?: string;
    etd?: string;
    eta?: string;
    siCutoff?: string;
    docCutoff?: string;
    customsCutoff?: string;
    vgmCutoff?: string;
    goodsDescription?: string;
    totalPackages?: number;
    totalPackageUnit?: string;
    specialRequirements?: string;
    orderDate?: string;
    notes?: string;
    customerReferenceNo?: string;
    foreignAgentId?: string;
    contractNo?: string;
    cargoValue?: string;
    cargoCurrency?: string;
    internalReferenceNo?: string;
    shippingAgentId?: string;
    insurancePremium?: string;
    insuranceCurrency?: string;
    unNumber?: string;
    hazardClass?: string;
    factoryName?: string;
    cargoReadyAt?: string;
    receivedAt?: string;
    bookingNotes?: string;
    allocationNotes?: string;
    operationNotes?: string;
    shippingDocuments?: OrderShippingDocumentInput[];
    containerRequests?: OrderContainerRequestInput[];
    declarationCutoffAt?: string;
    totalGrossWeightKg?: number;
    totalVolumeCbm?: number;
    shipperShortName?: string;
    consigneeShortName?: string;
    seaMasterBill?: SeaMasterBillInput;
    seaDocument?: SeaOrderDocumentInput;
    bookingNo?: string;
    /** idempotency_key 草稿更新幂等键：可选，传入即启用；同键 + 同 expected_version
 的重放返回当前草稿，否则走既有乐观锁冲突。 */
    idempotencyKey?: string;
  };

  type UpdateOrderResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type UpdateOrganizationRequest = {
    id: string;
    name: string;
    enabled?: boolean;
    baseCurrency?: string;
  };

  type UpdateOrganizationResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization;
    traceId?: string;
  };

  type UpdatePartnerAccountRequest = {
    partnerId: string;
    id: string;
    account: PartnerAccountInput;
  };

  type UpdatePartnerAccountResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAccount;
    traceId?: string;
  };

  type UpdatePartnerContractInput = {
    name: string;
    status: number;
    startDate: string;
    endDate: string;
    paymentTerms?: string;
    disputeResolution?: string;
    otherNotes?: string;
  };

  type UpdatePartnerContractRequest = {
    partnerId: string;
    id: string;
    contract: UpdatePartnerContractInput;
  };

  type UpdatePartnerContractResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerContract;
    traceId?: string;
  };

  type UpdatePartnerInvoiceProfileRequest = {
    partnerId: string;
    id: string;
    invoiceTitle: string;
    taxpayerIdentificationNo: string;
    registeredAddress?: string;
    registeredPhone?: string;
    bankName?: string;
    bankAccount?: string;
    defaultInvoiceType: string;
    isDefault?: boolean;
    enabled?: boolean;
    expectedVersion: string;
  };

  type UpdatePartnerInvoiceProfileResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerInvoiceProfile;
    traceId?: string;
  };

  type UpdatePartnerRequest = {
    id: string;
    legalName: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    enabled?: boolean;
    roles?: PartnerRoleInput[];
    contacts?: PartnerContactInput[];
    aliases?: PartnerAliasInput[];
    profile?: PartnerProfile;
    assignments?: PartnerAssignmentInput[];
    isCasual?: boolean;
  };

  type UpdatePartnerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
    traceId?: string;
  };

  type UpdatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    id: string;
    rule: PartnerSettlementRuleInput;
  };

  type UpdatePartnerSettlementRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerSettlementRule;
    traceId?: string;
  };

  type UpdatePartnerShippingPresetRequest = {
    partnerId: string;
    id: string;
    preset: PartnerShippingPresetInput;
  };

  type UpdatePartnerShippingPresetResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerShippingPreset;
    traceId?: string;
  };

  type UpdatePortRequest = {
    id: string;
    nameZh: string;
    nameEn: string;
    countryCode: string;
    transportModes?: string[];
    sortOrder?: number;
    enabled?: boolean;
  };

  type UpdatePortResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Port;
    traceId?: string;
  };

  type UpdateReleasePodRequest = {
    orderId: string;
    id: string;
    shippingDocumentId?: string;
    releaseNo?: string;
    podNo?: string;
    note?: string;
    seaDocumentType?: number;
    seaDocumentId?: string;
  };

  type UpdateReleasePodResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod;
    traceId?: string;
  };

  type UpdateRoleRequest = {
    id: string;
    name: string;
    dataScope: number;
    enabled?: boolean;
    permissionKeys?: string[];
  };

  type UpdateRoleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole;
    traceId?: string;
  };

  type UpdateSeaHouseBillRequest = {
    orderId: string;
    id: string;
    expectedVersion: string;
    expectedLinkVersion: string;
    houseBill: SeaHouseBillInput;
  };

  type UpdateSeaHouseBillResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaHouseBill;
    traceId?: string;
  };

  type UpdateSeaMasterBillContentRequest = {
    orderId: string;
    expectedMblVersion: string;
    content: SeaBillContent;
  };

  type UpdateSeaMasterBillContentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaMasterBillDetail;
    traceId?: string;
  };

  type UpdateSeaSharedContainerRequest = {
    id: string;
    expectedVersion: string;
    input: SeaSharedContainerInput;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId: string;
  };

  type UpdateSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };

  type UpdateShippingDocumentRequest = {
    orderId: string;
    id: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
  };

  type UpdateShippingDocumentResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument;
    traceId?: string;
  };

  type UpdateShippingLineRequest = {
    id: string;
    nameZh: string;
    nameEn: string;
    countryCode: string;
    trackingUrl?: string;
    alliance?: string;
    containerPrefixes?: string[];
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
  };

  type UpdateShippingLineResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ShippingLine;
    traceId?: string;
  };

  type UpdateTaxableServiceRequest = {
    id: string;
    name: string;
    shortName?: string;
    goodsCode?: string;
    defaultTaxRate: string;
    enabled?: boolean;
  };

  type UpdateTaxableServiceResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: TaxableService;
    traceId?: string;
  };

  type UpdateUserMembershipRequest = {
    userId: string;
    id: string;
    roleIds?: string[];
    enabled?: boolean;
    primary?: boolean;
  };

  type UpdateUserMembershipResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUserMembership;
    traceId?: string;
  };

  type UpdateUserRequest = {
    id: string;
    displayName: string;
    email?: string;
    enabled?: boolean;
    roleIds?: string[];
  };

  type UpdateUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser;
    traceId?: string;
  };

  type VerificationAllocationInput = {
    cashflowId: string;
    billId: string;
    amount: string;
  };

  type VerificationCreationCandidates = {
    cashflows?: FinanceCashflow[];
    bills?: FinanceBill[];
  };

  type VoyageDifferenceItem = {
    fieldName?: string;
    label?: string;
    currentValue?: string;
    targetValue?: string;
    isDifferent?: boolean;
  };

  type WeComLoginConfig = {
    enabled?: boolean;
    authorizeUrl?: string;
  };

  type WeComLoginRequest = {
    code: string;
    state: string;
  };

  type WeComLoginResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
  };

  type WithdrawSeaSharedContainerRequest = {
    id: string;
    expectedVersion: string;
    /** 授权锚点：中间件按该订单确定业务类型与组织上下文（id 始终是共享箱 ID） */
    orderId: string;
  };

  type WithdrawSeaSharedContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: SeaSharedContainer;
    traceId?: string;
  };
}
