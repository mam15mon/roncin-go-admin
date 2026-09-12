// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/admin/audit-logs */
export async function adminServiceListAuditLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListAuditLogsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAuditLogsResponse>("/api/v1/admin/audit-logs", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/admin/dingtalk/invitations */
export async function adminServiceListDingTalkInvitations(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListDingTalkInvitationsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListDingTalkInvitationsResponse>(
    "/api/v1/admin/dingtalk/invitations",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** CreateDingTalkInvitation 为目标手机号预建扫码邀请（通道 A：员工扫码自动激活）。 POST /api/v1/admin/dingtalk/invitations */
export async function adminServiceCreateDingTalkInvitation(
  body: API.CreateDingTalkInvitationRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateDingTalkInvitationResponse>(
    "/api/v1/admin/dingtalk/invitations",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 DELETE /api/v1/admin/dingtalk/invitations/${param0} */
export async function adminServiceRevokeDingTalkInvitation(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceRevokeDingTalkInvitationParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.RevokeDingTalkInvitationResponse>(
    `/api/v1/admin/dingtalk/invitations/${param0}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ListDingTalkRegistrations 返回当前组织范围内待审批的钉钉扫码注册队列。 GET /api/v1/admin/dingtalk/registrations */
export async function adminServiceListDingTalkRegistrations(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListDingTalkRegistrationsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListDingTalkRegistrationsResponse>(
    "/api/v1/admin/dingtalk/registrations",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ApproveDingTalkRegistration 一站式同意：启用账号 + 建目标组织成员资格 + 授予初始角色 + 通知本人。 POST /api/v1/admin/dingtalk/registrations/${param0}/approval */
export async function adminServiceApproveDingTalkRegistration(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceApproveDingTalkRegistrationParams,
  body: API.ApproveDingTalkRegistrationRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ApproveDingTalkRegistrationResponse>(
    `/api/v1/admin/dingtalk/registrations/${param0}/approval`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/admin/dingtalk/registrations/${param0}/rejection */
export async function adminServiceRejectDingTalkRegistration(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceRejectDingTalkRegistrationParams,
  body: API.RejectDingTalkRegistrationRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.RejectDingTalkRegistrationResponse>(
    `/api/v1/admin/dingtalk/registrations/${param0}/rejection`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** TransferDingTalkRegistration 将待审批注册转派至兄弟分公司。 POST /api/v1/admin/dingtalk/registrations/${param0}/transfer */
export async function adminServiceTransferDingTalkRegistration(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceTransferDingTalkRegistrationParams,
  body: API.TransferDingTalkRegistrationRequest,
  options?: { [key: string]: any }
) {
  const { userId: param0, ...queryParams } = params;
  return request<API.TransferDingTalkRegistrationResponse>(
    `/api/v1/admin/dingtalk/registrations/${param0}/transfer`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/admin/organizations */
export async function adminServiceListOrganizations(options?: {
  [key: string]: any;
}) {
  return request<API.ListOrganizationsResponse>("/api/v1/admin/organizations", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/admin/organizations */
export async function adminServiceCreateOrganization(
  body: API.CreateOrganizationRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateOrganizationResponse>(
    "/api/v1/admin/organizations",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/admin/organizations/${param0} */
export async function adminServiceUpdateOrganization(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceUpdateOrganizationParams,
  body: API.UpdateOrganizationRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateOrganizationResponse>(
    `/api/v1/admin/organizations/${param0}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/admin/organizations/${param0}/roles */
export async function adminServiceListOrganizationRoles(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListOrganizationRolesParams,
  options?: { [key: string]: any }
) {
  const { organizationId: param0, ...queryParams } = params;
  return request<API.ListOrganizationRolesResponse>(
    `/api/v1/admin/organizations/${param0}/roles`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/admin/permissions */
export async function adminServiceListPermissions(options?: {
  [key: string]: any;
}) {
  return request<API.ListPermissionsResponse>("/api/v1/admin/permissions", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/admin/roles */
export async function adminServiceListRoles(options?: { [key: string]: any }) {
  return request<API.ListRolesResponse>("/api/v1/admin/roles", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/admin/roles */
export async function adminServiceCreateRole(
  body: API.CreateRoleRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateRoleResponse>("/api/v1/admin/roles", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/admin/roles/${param0} */
export async function adminServiceUpdateRole(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceUpdateRoleParams,
  body: API.UpdateRoleRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateRoleResponse>(`/api/v1/admin/roles/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/admin/users */
export async function adminServiceListUsers(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListUsersParams,
  options?: { [key: string]: any }
) {
  return request<API.ListUsersResponse>("/api/v1/admin/users", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/admin/users */
export async function adminServiceCreateUser(
  body: API.CreateUserRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateUserResponse>("/api/v1/admin/users", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/admin/users/${param0} */
export async function adminServiceUpdateUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceUpdateUserParams,
  body: API.UpdateUserRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateUserResponse>(`/api/v1/admin/users/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/admin/users/${param0}/dingtalk-authorization */
export async function adminServiceAuthorizeDingTalkUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceAuthorizeDingTalkUserParams,
  body: API.AuthorizeDingTalkUserRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AuthorizeDingTalkUserResponse>(
    `/api/v1/admin/users/${param0}/dingtalk-authorization`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/admin/users/${param0}/memberships */
export async function adminServiceListUserMemberships(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListUserMembershipsParams,
  options?: { [key: string]: any }
) {
  const { userId: param0, ...queryParams } = params;
  return request<API.ListUserMembershipsResponse>(
    `/api/v1/admin/users/${param0}/memberships`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/admin/users/${param0}/memberships */
export async function adminServiceCreateUserMembership(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceCreateUserMembershipParams,
  body: API.CreateUserMembershipRequest,
  options?: { [key: string]: any }
) {
  const { userId: param0, ...queryParams } = params;
  return request<API.CreateUserMembershipResponse>(
    `/api/v1/admin/users/${param0}/memberships`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/admin/users/${param0}/memberships/${param1} */
export async function adminServiceUpdateUserMembership(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceUpdateUserMembershipParams,
  body: API.UpdateUserMembershipRequest,
  options?: { [key: string]: any }
) {
  const { userId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateUserMembershipResponse>(
    `/api/v1/admin/users/${param0}/memberships/${param1}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 DELETE /api/v1/admin/users/${param0}/memberships/${param1} */
export async function adminServiceDeleteUserMembership(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceDeleteUserMembershipParams,
  options?: { [key: string]: any }
) {
  const { userId: param0, id: param1, ...queryParams } = params;
  return request<API.DeleteUserMembershipResponse>(
    `/api/v1/admin/users/${param0}/memberships/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/admin/users/${param0}/password */
export async function adminServiceResetUserPassword(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceResetUserPasswordParams,
  body: API.ResetUserPasswordRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ResetUserPasswordResponse>(
    `/api/v1/admin/users/${param0}/password`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** TerminateUser 办理员工离职，保留全局账号、外部身份和历史业务记录。 POST /api/v1/admin/users/${param0}/termination */
export async function adminServiceTerminateUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceTerminateUserParams,
  body: API.TerminateUserRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.TerminateUserResponse>(
    `/api/v1/admin/users/${param0}/termination`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/admin/users/${param0}/wecom-authorization */
export async function adminServiceAuthorizeWeComUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceAuthorizeWeComUserParams,
  body: API.AuthorizeWeComUserRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AuthorizeWeComUserResponse>(
    `/api/v1/admin/users/${param0}/wecom-authorization`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}
