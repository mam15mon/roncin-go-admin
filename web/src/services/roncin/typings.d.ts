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
    shippingDocumentId?: string;
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
    masterNo: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
    masterDocumentType?: string;
    masterReleaseMethod?: string;
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
    orderOrganizationAccesses?: OrderOrganizationAccess[];
  };

  type AdminServiceAuthorizeDingTalkUserParams = {
    id: string;
  };

  type AdminServiceAuthorizeWeComUserParams = {
    id: string;
  };

  type AdminServiceDeleteUserParams = {
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

  type AdminServiceListOrganizationRolesParams = {
    organizationId: string;
  };

  type AdminServiceListUsersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type AdminServiceResetUserPasswordParams = {
    id: string;
  };

  type AdminServiceUpdateOrganizationParams = {
    id: string;
  };

  type AdminServiceUpdateRoleParams = {
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
  };

  type BackgroundTaskServiceRequeueBackgroundTaskParams = {
    id: string;
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
    status?: string;
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

  type CreateAirlineRequest = {
    iataCode: string;
    icaoCode?: string;
    awbPrefix: string;
    nameZh: string;
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
  };

  type CreateBillBatchRequest = {
    feeIds: string[];
    groupingPolicy: BillGroupingPolicy;
    groups: CreateBillBatchGroupInput[];
    previewToken: string;
    idempotencyKey: string;
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
    verificationId: string;
    employeeId: string;
    note?: string;
    idempotencyKey: string;
    ruleId: string;
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
  };

  type CreateCommissionRuleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceCommissionRule;
    traceId?: string;
  };

  type CreateExchangeRateSettingRequest = {
    rateType: string;
    fromCurrency: string;
    toCurrency: string;
    /** 示例：2026-08-27T09:30:00+08:00。 */
    effectiveFrom: string;
    effectiveTo?: string;
    receivableRate: string;
    payableRate: string;
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

  type CreateMilestoneTemplateRequest = {
    code: string;
    name: string;
    businessType: number;
    tradeTerm?: string;
    version: number;
    items: MilestoneTemplateItemInput[];
  };

  type CreateMilestoneTemplateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate;
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
    tradeTerm: number;
    paymentTerm: number;
    carrierId?: string;
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
    loadingTerms?: string;
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
    tags?: OrderTagsInput;
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
    code: string;
    name: string;
    dataScope: number;
    permissionKeys?: string[];
    orderOrganizationAccesses?: OrderOrganizationAccess[];
  };

  type CreateRoleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole;
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

  type DeleteUserResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
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
    data?: CurrentUser;
    traceId?: string;
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

  type ExchangeRateCustomSetting = {
    organizationId?: string;
    inheritBaseCurrencyRate?: boolean;
    /** 未保存过自定义设置时为 0；首次保存需携带 expected_version=0。 */
    version?: string;
    updatedAt?: string;
    updatedBy?: string;
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
    rateType?: string;
    fromCurrency?: string;
    toCurrency?: string;
    receivableRate?: string;
    payableRate?: string;
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
    rateType?: string;
    fromCurrency?: string;
    toCurrency?: string;
    /** effective_from 为带时区且精确到秒的 RFC 3339 时间，区间左边界包含该时刻。 */
    effectiveFrom?: string;
    /** effective_to 为带时区且精确到秒的 RFC 3339 时间，区间右边界不包含该时刻；空表示长期有效。 */
    effectiveTo?: string;
    receivableRate?: string;
    payableRate?: string;
    isActive?: boolean;
    createdAt?: string;
    updatedAt?: string;
  };

  type ExchangeRateTimeStandardSetting = {
    rateType?: string;
    timeStandards?: string[];
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
    status?: string;
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
    financialProgress?: string;
    billNo?: string;
    financeLocked?: boolean;
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
    receivableBaseAmount?: string;
    payableBaseAmount?: string;
    profitBaseAmount?: string;
    baseCurrency?: string;
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

  type FinanceBill = {
    id?: string;
    billNo?: string;
    direction?: string;
    status?: string;
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

  type FinanceCashflow = {
    id?: string;
    flowNo?: string;
    direction?: string;
    status?: string;
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
  };

  type FinanceCommission = {
    id?: string;
    commissionNo?: string;
    verificationId?: string;
    verificationNo?: string;
    employeeId?: string;
    employeeName?: string;
    status?: string;
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
    status?: string;
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
  };

  type FinanceInvoice = {
    id?: string;
    recordNo?: string;
    direction?: string;
    status?: string;
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

  type FinanceVerification = {
    id?: string;
    verificationNo?: string;
    status?: string;
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
    exchangeRate?: string;
    exchangeRateSource?: string;
    exchangeRateDate?: string;
    exchangeRateSettingId?: string;
    baseAmount?: string;
    billBaseAmount?: string;
    cashflowBaseAmount?: string;
    exchangeGainLoss?: string;
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
    writeOffBaseAmount?: string;
    exchangeGainLoss?: string;
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

  type GetDingTalkLoginConfigResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: DingTalkLoginConfig;
    traceId?: string;
  };

  type GetExchangeRateCustomSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateCustomSetting;
    traceId?: string;
  };

  type GetExchangeRateImportResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateImportBatch;
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

  type GetOrderResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
  };

  type GetPartnerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
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

  type ListBillsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceBill[];
    total?: string;
    traceId?: string;
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

  type ListExchangeRateSettingsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateSetting[];
    traceId?: string;
    baseCurrency?: string;
  };

  type ListExchangeRateTimeStandardsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateTimeStandardSetting[];
    traceId?: string;
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

  type ListInvoicesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceInvoice[];
    total?: string;
    traceId?: string;
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

  type ListMilestoneTemplatesResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate[];
    traceId?: string;
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

  type ListVerificationsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: FinanceVerification[];
    total?: string;
    traceId?: string;
  };

  type LoginRequest = {
    username: string;
    password: string;
  };

  type LoginResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
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

  type MasterDataServiceListMilestoneTemplatesParams = {
    businessType?: number;
    tradeTerm?: string;
    published?: boolean;
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

  type MasterDataServicePublishMilestoneTemplateParams = {
    id: string;
  };

  type MasterDataServiceSearchCurrenciesParams = {
    keyword?: string;
    page?: number;
    pageSize?: number;
  };

  type MasterDataServiceSetDefaultMilestoneTemplateParams = {
    id: string;
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

  type MeResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
  };

  type MilestoneTemplate = {
    id?: string;
    organizationId?: string;
    code?: string;
    name?: string;
    businessType?: number;
    tradeTerm?: string;
    version?: number;
    isDefault?: boolean;
    publishedAt?: string;
    enabled?: boolean;
    items?: MilestoneTemplateItem[];
    createdAt?: string;
    updatedAt?: string;
  };

  type MilestoneTemplateItem = {
    id?: string;
    code?: string;
    label?: string;
    description?: string;
    category?: string;
    sortOrder?: number;
    enabled?: boolean;
    dependsOn?: string[];
  };

  type MilestoneTemplateItemInput = {
    code: string;
    label: string;
    description?: string;
    category?: string;
    sortOrder?: number;
    enabled?: boolean;
    dependsOn?: string[];
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
    carrierId?: string;
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
    loadingTerms?: string;
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
    tags?: string[];
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
  };

  type OrderAttachmentServiceListAttachmentsParams = {
    orderId: string;
  };

  type OrderAttachmentServiceRegisterAttachmentParams = {
    orderId: string;
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
    shippingDocumentId?: string;
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

  type OrderOrganizationAccess = {
    organizationId: string;
    writable?: boolean;
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
    carrierId?: string;
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
    tags?: string[];
    tagMatchMode?: number;
    isLocked?: boolean;
    isShared?: boolean;
  };

  type OrderServiceListPersonnelOptionsParams = {
    businessType?: number;
    keyword?: string;
    page?: number;
    pageSize?: number;
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
    masterNo?: string;
    houseNo?: string;
    releaseType?: string;
    status?: number;
    note?: string;
    createdAt?: string;
    updatedAt?: string;
    consolidationId?: string;
    masterDocumentType?: string;
    masterReleaseMethod?: string;
  };

  type OrderShippingDocumentInput = {
    id?: string;
    masterNo: string;
    houseNo: string;
    /** release_type 是分单（HBL）签放方式。 */
    releaseType?: string;
    note?: string;
    /** 主单属性存储在共享的主单批次，同一主单组内必须一致。 */
    masterDocumentType?: string;
    masterReleaseMethod?: string;
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

  type OrderTagsInput = {
    values?: string[];
  };

  type Organization = {
    id?: string;
    code?: string;
    name?: string;
    baseCurrency?: string;
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
  };

  type PartnerAccount = {
    id?: string;
    partnerRoleId?: string;
    accountType?: string;
    currency?: string;
    bankName?: string;
    bankAccount?: string;
    swiftCode?: string;
    isDefault?: boolean;
    status?: number;
    remark?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  type PartnerAccountInput = {
    currency: string;
    bankName?: string;
    bankAccount?: string;
    swiftCode?: string;
    isDefault?: boolean;
    status: number;
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

  type PreviewBillBatchRequest = {
    feeIds: string[];
    groupingPolicy: BillGroupingPolicy;
  };

  type PreviewBillBatchResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BillBatchPreviewGroup[];
    previewToken?: string;
    traceId?: string;
  };

  type PreviewCommissionRequest = {
    verificationId: string;
    employeeId: string;
    ruleId: string;
  };

  type PreviewCommissionResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CommissionCalculation;
    traceId?: string;
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

  type PublishMilestoneTemplateRequest = {
    id: string;
    isDefault?: boolean;
  };

  type PublishMilestoneTemplateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate;
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

  type RemoveAbnormalCaseResponse = {
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

  type RoleScope = {
    roleCode?: string;
    dataScope?: string;
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

  type SetDefaultMilestoneTemplateRequest = {
    id: string;
  };

  type SetDefaultMilestoneTemplateResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate;
    traceId?: string;
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

  type SettlementServiceCreateCommissionAdjustmentParams = {
    commissionId: string;
  };

  type SettlementServiceGetBillParams = {
    id: string;
  };

  type SettlementServiceGetCommissionParams = {
    id: string;
  };

  type SettlementServiceGetInvoiceParams = {
    id: string;
  };

  type SettlementServiceIssueInvoiceParams = {
    id: string;
  };

  type SettlementServiceListBillsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: string;
    settlementPartyId?: string;
    currency?: string;
    billDateFrom?: string;
    billDateTo?: string;
  };

  type SettlementServiceListCashflowsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: string;
    settlementPartyId?: string;
    currency?: string;
  };

  type SettlementServiceListCommissionCandidatesParams = {
    verificationId?: string;
    ruleId?: string;
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type SettlementServiceListCommissionEmployeesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
  };

  type SettlementServiceListCommissionRulesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    personnelRole?: string;
    enabled?: boolean;
  };

  type SettlementServiceListCommissionsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: string;
  };

  type SettlementServiceListFeeLedgerParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    businessType?: string;
    direction?: string;
    status?: string;
    settlementPartyId?: string;
    currency?: string;
    expenseDateFrom?: string;
    expenseDateTo?: string;
    customerId?: string;
    financialProgress?: string;
    billNo?: string;
    financeLocked?: boolean;
  };

  type SettlementServiceListInvoicesParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    direction?: string;
    status?: string;
  };

  type SettlementServiceListVerificationsParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: string;
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
    awbPrefix: string;
    nameZh: string;
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
    shippingDocumentId?: string;
  };

  type UpdateContainerResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer;
    traceId?: string;
  };

  type UpdateExchangeRateCustomSettingRequest = {
    inheritBaseCurrencyRate?: boolean;
    expectedVersion: string;
  };

  type UpdateExchangeRateCustomSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateCustomSetting;
    traceId?: string;
  };

  type UpdateExchangeRateSettingRequest = {
    id: string;
    rateType: string;
    fromCurrency: string;
    toCurrency: string;
    /** 示例：2026-08-27T09:30:00+08:00。 */
    effectiveFrom: string;
    effectiveTo?: string;
    receivableRate: string;
    payableRate: string;
  };

  type UpdateExchangeRateSettingResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateSetting;
    traceId?: string;
  };

  type UpdateExchangeRateTimeStandardsRequest = {
    data: ExchangeRateTimeStandardSetting[];
  };

  type UpdateExchangeRateTimeStandardsResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ExchangeRateTimeStandardSetting[];
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
    carrierId?: string;
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
    loadingTerms?: string;
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
    tags?: OrderTagsInput;
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
    orderOrganizationAccesses?: OrderOrganizationAccess[];
  };

  type UpdateRoleResponse = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole;
    traceId?: string;
  };

  type UpdateShippingDocumentRequest = {
    orderId: string;
    id: string;
    masterNo: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
    masterDocumentType?: string;
    masterReleaseMethod?: string;
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
}
