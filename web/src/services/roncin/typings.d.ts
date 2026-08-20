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

  type RegisterPartnerAttachmentRequest = {
    partnerId: string;
    idempotencyKey: string;
    fileName: string;
    mimeType: string;
    fileSize: string;
    objectKey: string;
    checksum?: string;
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
