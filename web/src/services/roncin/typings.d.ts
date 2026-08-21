declare namespace API {
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

  type AdminServiceListAuditLogsParams = {
    page?: number;
    pageSize?: number;
    action?: string;
    userId?: string;
    startTime?: string;
    endTime?: string;
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

  type AssignPersonnelRequest = {
    orderId: string;
    userId: string;
    role: number;
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

  type CreateMasterDataItemRequest = {
    kind: number;
    code: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    transportMode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
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
  };

  type CreateOrganizationRequest = {
    code: string;
    name: string;
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
  };

  type CreatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    rule: PartnerSettlementRuleInput;
  };

  type CreateRoleRequest = {
    code: string;
    name: string;
    dataScope: number;
    permissionKeys?: string[];
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

  type MasterDataImportItemInput = {
    code: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    transportMode?: string;
    teuFactor?: string;
    sortOrder?: number;
    enabled?: boolean;
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
    transportMode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
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

  type MasterDataServiceUpdateItemParams = {
    id: string;
  };

  type MasterDataServiceUpdateNumberRuleParams = {
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

  type OrderReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: Order;
    traceId?: string;
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
  };

  type PartnerRoleInput = {
    type?: number;
    enabled?: boolean;
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

  type PartnerServiceListPartnerContractsParams = {
    partnerId: string;
    status?: number;
  };

  type PartnerServiceListPartnerSettlementRulesParams = {
    partnerId: string;
    roleType: number;
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
  };

  type PartnerSettlementRuleInput = {
    statementMode: number;
    settlementMethod: number;
    settlementDay?: number;
    settlementCycleDays?: number;
    settlementBase?: number;
    settlementCurrency: string;
    isActive?: boolean;
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

  type UpdateMasterDataItemRequest = {
    id: string;
    name: string;
    nameEn?: string;
    parentCode?: string;
    transportMode?: string;
    teuFactor?: string;
    source?: string;
    sortOrder?: number;
    enabled?: boolean;
    kind: number;
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
  };

  type UpdatePartnerSettlementRuleRequest = {
    partnerId: string;
    roleType: number;
    id: string;
    rule: PartnerSettlementRuleInput;
  };

  type UpdateRoleRequest = {
    id: string;
    name: string;
    dataScope: number;
    enabled?: boolean;
    permissionKeys?: string[];
  };

  type UpdateUserRequest = {
    id: string;
    displayName: string;
    email?: string;
    enabled?: boolean;
    roleIds?: string[];
  };
}
