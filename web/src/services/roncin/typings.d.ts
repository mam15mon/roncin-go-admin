declare namespace API {
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

  type AdminServiceListUsersParams = {
    page?: number;
    pageSize?: number;
    keyword?: string;
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

  type MeReply = {
    success?: boolean;
    code?: number;
    message?: string;
    data?: CurrentUser;
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

  type RoleScope = {
    roleCode?: string;
    dataScope?: string;
  };

  type SwitchOrganizationRequest = {
    organizationId: string;
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
