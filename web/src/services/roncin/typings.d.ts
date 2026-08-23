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

  type AddReleasePodRequest = {
    orderId: string;
    shippingDocumentId?: string;
    releaseNo?: string;
    podNo?: string;
    note?: string;
  };

  type AddShippingDocumentRequest = {
    orderId: string;
    masterNo: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
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

  type AdminAuditLogListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminAuditLog[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
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

  type AdministrativeRegionListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdministrativeRegion[];
    traceId?: string;
  };

  type AdminOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type AdminOrganization = {
    id?: string;
    code?: string;
    name?: string;
    parentId?: string;
    enabled?: boolean;
    kind?: number;
  };

  type AdminOrganizationListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization[];
    traceId?: string;
  };

  type AdminOrganizationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminOrganization;
    traceId?: string;
  };

  type AdminPermission = {
    key?: string;
    name?: string;
    group?: string;
    description?: string;
  };

  type AdminPermissionListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminPermission[];
    traceId?: string;
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

  type AdminRoleListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole[];
    traceId?: string;
  };

  type AdminRoleReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminRole;
    traceId?: string;
  };

  type AdminServiceAuthorizeWeComUserParams = {
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
  };

  type AdminUserListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type AdminUserReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: AdminUser;
    traceId?: string;
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

  type AirlineListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airline[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type AirlineReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airline;
    traceId?: string;
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

  type AirportListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airport[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type AirportReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Airport;
    traceId?: string;
  };

  type AssignPersonnelRequest = {
    orderId: string;
    userId: string;
    role: number;
  };

  type AuthorizeWeComUserRequest = {
    id: string;
    organizationId: string;
    displayName: string;
    email?: string;
    roleIds: string[];
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

  type BackgroundTaskListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BackgroundTask[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type BackgroundTaskReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: BackgroundTask;
    traceId?: string;
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

  type CreateMasterDataItemRequest = {
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

  type CreateMilestoneTemplateRequest = {
    code: string;
    name: string;
    businessType: number;
    tradeTerm?: string;
    version: number;
    items: MilestoneTemplateItemInput[];
  };

  type CreateNumberRuleRequest = {
    documentType: number;
    prefix?: string;
    dateFormat: number;
    sequenceLength: number;
    resetPolicy: number;
  };

  type CreateOrderRequest = {
    customerId: string;
    businessType: number;
    tradeDirection: number;
    tradeTerm: number;
    paymentTerm: number;
    statusTemplateId: string;
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
  };

  type CreateOrganizationRequest = {
    code: string;
    name: string;
    parentId: string;
    kind: number;
  };

  type CreatePartnerAccountRequest = {
    partnerId: string;
    account: PartnerAccountInput;
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

  type CreatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    rule: PartnerSettlementRuleInput;
  };

  type CreatePartnerShippingPresetRequest = {
    partnerId: string;
    preset: PartnerShippingPresetInput;
  };

  type CreatePortRequest = {
    unLocode: string;
    nameZh: string;
    nameEn: string;
    countryCode: string;
    transportModes?: string[];
    sortOrder?: number;
  };

  type CreateRoleRequest = {
    code: string;
    name: string;
    dataScope: number;
    permissionKeys?: string[];
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

  type CreateStatusTemplateRequest = {
    code: string;
    name: string;
    businessType: number;
    version: number;
    items: StatusTemplateItemInput[];
  };

  type CreateUserRequest = {
    username: string;
    displayName: string;
    password: string;
    email?: string;
    roleIds?: string[];
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

  type CurrencyListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Currency[];
    traceId?: string;
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
  };

  type ImportMasterDataItemsRequest = {
    kind: number;
    source: string;
    mode: number;
    items: MasterDataImportItemInput[];
  };

  type ImportPartnersRequest = {
    source: string;
    mode: number;
    items: PartnerImportItemInput[];
  };

  type LoginReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
    traceId?: string;
  };

  type LoginRequest = {
    username: string;
    password: string;
  };

  type MarkAbnormalCaseRequest = {
    orderId: string;
    abnormalCaseId: string;
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

  type MasterDataImportReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    createdCount?: number;
    updatedCount?: number;
    traceId?: string;
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

  type MasterDataItemListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type MasterDataItemReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem;
    traceId?: string;
  };

  type MasterDataOptionsReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MasterDataItem[];
    traceId?: string;
  };

  type MasterDataServiceListAdministrativeRegionsParams = {
    level?: number;
    parentCode?: string;
    keyword?: string;
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

  type MasterDataServiceListStatusTemplatesParams = {
    businessType?: number;
    published?: boolean;
  };

  type MasterDataServicePublishMilestoneTemplateParams = {
    id: string;
  };

  type MasterDataServicePublishStatusTemplateParams = {
    id: string;
  };

  type MasterDataServiceSetDefaultMilestoneTemplateParams = {
    id: string;
  };

  type MasterDataServiceSetDefaultStatusTemplateParams = {
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

  type MeReply = {
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

  type MilestoneTemplateListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate[];
    traceId?: string;
  };

  type MilestoneTemplateReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: MilestoneTemplate;
    traceId?: string;
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

  type NumberRuleListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: NumberRule[];
    traceId?: string;
  };

  type NumberRuleReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: NumberRule;
    traceId?: string;
  };

  type OperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
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
    status?: string;
    statusTemplateId?: string;
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

  type OrderAbnormalCaseListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAbnormalCase[];
    traceId?: string;
  };

  type OrderAbnormalCaseOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderAbnormalCaseReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAbnormalCase;
    traceId?: string;
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

  type OrderAttachmentListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAttachment[];
    traceId?: string;
  };

  type OrderAttachmentReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderAttachment;
    traceId?: string;
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

  type OrderCargoItemListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderCargoItem[];
    traceId?: string;
  };

  type OrderCargoItemOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderCargoItemReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderCargoItem;
    traceId?: string;
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

  type OrderContainerListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer[];
    traceId?: string;
  };

  type OrderContainerOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderContainerReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderContainer;
    traceId?: string;
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

  type OrderListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
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

  type OrderMilestoneListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderMilestone[];
    traceId?: string;
  };

  type OrderMilestoneReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderMilestone;
    traceId?: string;
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
  };

  type OrderPersonnelListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderPersonnel[];
    traceId?: string;
  };

  type OrderPersonnelOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderPersonnelReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderPersonnel;
    traceId?: string;
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

  type OrderReferenceCheckReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReferenceCheck;
    traceId?: string;
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

  type OrderReleasePodListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod[];
    traceId?: string;
  };

  type OrderReleasePodOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderReleasePodReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderReleasePod;
    traceId?: string;
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

  type OrderReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
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

  type OrderServiceListOrdersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    status?: string;
    businessType?: number;
    customerId?: string;
  };

  type OrderServiceTransitionOrderStatusParams = {
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
  };

  type OrderShippingDocumentListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument[];
    traceId?: string;
  };

  type OrderShippingDocumentOperationReply = {
    success?: boolean;
    code?: number;
    message?: string;
    traceId?: string;
  };

  type OrderShippingDocumentReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: OrderShippingDocument;
    traceId?: string;
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

  type Organization = {
    id?: string;
    code?: string;
    name?: string;
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
    invoiceTitle?: string;
    unifiedSocialCreditCode?: string;
    billingAddress?: string;
    billingPhone?: string;
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
    invoiceTitle: string;
    unifiedSocialCreditCode?: string;
    billingAddress?: string;
    billingPhone?: string;
    bankName?: string;
    bankAccount?: string;
    swiftCode?: string;
    isDefault?: boolean;
    status: number;
    remark?: string;
  };

  type PartnerAccountListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAccount[];
    traceId?: string;
  };

  type PartnerAccountReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAccount;
    traceId?: string;
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

  type PartnerAssignmentOptionListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAssignmentOption[];
    traceId?: string;
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

  type PartnerAttachmentListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAttachment[];
    traceId?: string;
  };

  type PartnerAttachmentReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAttachment;
    traceId?: string;
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

  type PartnerAuditLogListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerAuditLog[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
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

  type PartnerContractListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerContract[];
    traceId?: string;
  };

  type PartnerContractReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerContract;
    traceId?: string;
  };

  type PartnerExportItem = {
    code?: string;
    legalName?: string;
    unifiedSocialCreditCode?: string;
    registeredAddress?: string;
    enabled?: boolean;
    roles?: number[];
  };

  type PartnerExportReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerExportItem[];
    traceId?: string;
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

  type PartnerImportReply = {
    success?: boolean;
    code?: number;
    message?: string;
    createdCount?: number;
    updatedCount?: number;
    traceId?: string;
  };

  type PartnerListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
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

  type PartnerReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Partner;
    traceId?: string;
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

  type PartnerSettlementRuleListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerSettlementRule[];
    traceId?: string;
  };

  type PartnerSettlementRuleReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerSettlementRule;
    traceId?: string;
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

  type PartnerShippingPresetListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerShippingPreset[];
    traceId?: string;
  };

  type PartnerShippingPresetReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: PartnerShippingPreset;
    traceId?: string;
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

  type PortListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Port[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type PortReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Port;
    traceId?: string;
  };

  type PublishMilestoneTemplateRequest = {
    id: string;
    isDefault?: boolean;
  };

  type PublishStatusTemplateRequest = {
    id: string;
    isDefault?: boolean;
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

  type RegisterPartnerAttachmentRequest = {
    partnerId: string;
    idempotencyKey: string;
    fileName: string;
    mimeType: string;
    fileSize: string;
    objectKey: string;
    checksum?: string;
  };

  type RequeueBackgroundTaskRequest = {
    id: string;
  };

  type ResetUserPasswordRequest = {
    id: string;
    password: string;
  };

  type ResolveAbnormalCaseRequest = {
    orderId: string;
    id: string;
  };

  type RoleScope = {
    roleCode?: string;
    dataScope?: string;
  };

  type SetDefaultMilestoneTemplateRequest = {
    id: string;
  };

  type SetDefaultStatusTemplateRequest = {
    id: string;
  };

  type SetMilestoneRequest = {
    orderId: string;
    type: string;
    expectedOrderStatus: string;
    occurredAt?: string;
    note?: string;
    clearOccurredAt?: boolean;
  };

  type SetSupplierBlacklistRequest = {
    id: string;
    blacklisted?: boolean;
    reason: string;
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

  type ShippingLineListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ShippingLine[];
    total?: number;
    page?: number;
    pageSize?: number;
    traceId?: string;
  };

  type ShippingLineReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: ShippingLine;
    traceId?: string;
  };

  type StatusTemplate = {
    id?: string;
    organizationId?: string;
    code?: string;
    name?: string;
    businessType?: number;
    version?: number;
    isDefault?: boolean;
    publishedAt?: string;
    enabled?: boolean;
    items?: StatusTemplateItem[];
    createdAt?: string;
    updatedAt?: string;
  };

  type StatusTemplateItem = {
    id?: string;
    code?: string;
    label?: string;
    sortOrder?: number;
    enabled?: boolean;
    colorToken?: string;
    system?: boolean;
  };

  type StatusTemplateItemInput = {
    code: string;
    label: string;
    sortOrder?: number;
    enabled?: boolean;
    colorToken?: string;
    system?: boolean;
  };

  type StatusTemplateListReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: StatusTemplate[];
    traceId?: string;
  };

  type StatusTemplateReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: StatusTemplate;
    traceId?: string;
  };

  type SwitchOrganizationRequest = {
    organizationId: string;
  };

  type TransitionOrderStatusRequest = {
    id: string;
    expectedStatus: string;
    targetStatus: string;
    reason?: string;
  };

  type TransitionReleasePodStatusRequest = {
    orderId: string;
    id: string;
    expectedStatus: number;
    toStatus: number;
  };

  type TransitionShippingDocumentStatusRequest = {
    orderId: string;
    id: string;
    expectedStatus: number;
    toStatus: number;
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

  type UpdateMasterDataItemRequest = {
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

  type UpdateNumberRuleRequest = {
    id: string;
    prefix?: string;
    dateFormat: number;
    sequenceLength: number;
    resetPolicy: number;
    enabled?: boolean;
  };

  type UpdateOrderRequest = {
    id: string;
    expectedStatus: string;
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
  };

  type UpdateOrganizationRequest = {
    id: string;
    name: string;
    enabled?: boolean;
  };

  type UpdatePartnerAccountRequest = {
    partnerId: string;
    id: string;
    account: PartnerAccountInput;
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

  type UpdatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    id: string;
    rule: PartnerSettlementRuleInput;
  };

  type UpdatePartnerShippingPresetRequest = {
    partnerId: string;
    id: string;
    preset: PartnerShippingPresetInput;
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

  type UpdateReleasePodRequest = {
    orderId: string;
    id: string;
    shippingDocumentId?: string;
    releaseNo?: string;
    podNo?: string;
    note?: string;
  };

  type UpdateRoleRequest = {
    id: string;
    name: string;
    dataScope: number;
    enabled?: boolean;
    permissionKeys?: string[];
  };

  type UpdateShippingDocumentRequest = {
    orderId: string;
    id: string;
    masterNo: string;
    houseNo: string;
    releaseType?: string;
    note?: string;
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

  type UpdateUserRequest = {
    id: string;
    displayName: string;
    email?: string;
    enabled?: boolean;
    roleIds?: string[];
  };

  type WeComLoginConfig = {
    enabled?: boolean;
    authorizeUrl?: string;
  };

  type WeComLoginConfigReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: WeComLoginConfig;
    traceId?: string;
  };

  type WeComLoginRequest = {
    code: string;
    state: string;
  };
}
