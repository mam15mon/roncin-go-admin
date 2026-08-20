declare namespace API {
  type CreatePartnerRequest = {
    code: string;
    name: string;
    type: number;
    contactName?: string;
    phone?: string;
    email?: string;
    address?: string;
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
}
