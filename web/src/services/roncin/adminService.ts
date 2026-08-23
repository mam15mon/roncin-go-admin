// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/admin/audit-logs */
export async function adminServiceListAuditLogs(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceListAuditLogsParams,
  options?: { [key: string]: any }
) {
  return request<API.AdminAuditLogListReply>("/api/v1/admin/audit-logs", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/admin/organizations */
export async function adminServiceListOrganizations(options?: {
  [key: string]: any;
}) {
  return request<API.AdminOrganizationListReply>(
    "/api/v1/admin/organizations",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/admin/organizations */
export async function adminServiceCreateOrganization(
  body: API.CreateOrganizationRequest,
  options?: { [key: string]: any }
) {
  return request<API.AdminOrganizationReply>("/api/v1/admin/organizations", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/admin/organizations/${param0} */
export async function adminServiceUpdateOrganization(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceUpdateOrganizationParams,
  body: API.UpdateOrganizationRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AdminOrganizationReply>(
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
  return request<API.AdminRoleListReply>(
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
  return request<API.AdminPermissionListReply>("/api/v1/admin/permissions", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/admin/roles */
export async function adminServiceListRoles(options?: { [key: string]: any }) {
  return request<API.AdminRoleListReply>("/api/v1/admin/roles", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/admin/roles */
export async function adminServiceCreateRole(
  body: API.CreateRoleRequest,
  options?: { [key: string]: any }
) {
  return request<API.AdminRoleReply>("/api/v1/admin/roles", {
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
  return request<API.AdminRoleReply>(`/api/v1/admin/roles/${param0}`, {
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
  return request<API.AdminUserListReply>("/api/v1/admin/users", {
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
  return request<API.AdminUserReply>("/api/v1/admin/users", {
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
  return request<API.AdminUserReply>(`/api/v1/admin/users/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/admin/users/${param0}/password */
export async function adminServiceResetUserPassword(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceResetUserPasswordParams,
  body: API.ResetUserPasswordRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AdminOperationReply>(
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

/** 此处后端没有提供注释 POST /api/v1/admin/users/${param0}/wecom-authorization */
export async function adminServiceAuthorizeWeComUser(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AdminServiceAuthorizeWeComUserParams,
  body: API.AuthorizeWeComUserRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AdminUserReply>(
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
