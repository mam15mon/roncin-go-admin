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

  type CreatePartnerRequest = {
    code: string;
    name: string;
    type: number;
    contactName?: string;
    phone?: string;
    email?: string;
    address?: string;
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

  type MasterDataServiceListStatusTemplatesParams = {
    businessType?: number;
    published?: boolean;
  };

  type MasterDataServicePublishStatusTemplateParams = {
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
    name?: string;
    type?: number;
    contactName?: string;
    phone?: string;
    email?: string;
    address?: string;
    enabled?: boolean;
    createdAt?: string;
    updatedAt?: string;
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

  type PartnerServiceListPartnersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
    type?: number;
    enabled?: boolean;
  };

  type PartnerServiceUpdatePartnerParams = {
    id: string;
  };

  type PublishStatusTemplateRequest = {
    id: string;
    isDefault?: boolean;
  };

  type ResetUserPasswordRequest = {
    id: string;
    password: string;
  };

  type RoleScope = {
    roleCode?: string;
    dataScope?: string;
  };

  type SetDefaultStatusTemplateRequest = {
    id: string;
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

  type UpdatePartnerRequest = {
    id: string;
    name: string;
    type: number;
    contactName?: string;
    phone?: string;
    email?: string;
    address?: string;
    enabled?: boolean;
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
