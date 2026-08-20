// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 POST /api/v1/auth/login */
export async function authServiceLogin(
  body: API.LoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.LoginReply>("/api/v1/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/auth/logout */
export async function authServiceLogout(
  body: {
    id?: number;
  },
  options?: { [key: string]: any }
) {
  return request<API.OperationReply>("/api/v1/auth/logout", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/auth/me */
export async function authServiceMe(options?: { [key: string]: any }) {
  return request<API.MeReply>("/api/v1/auth/me", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/auth/switch-organization */
export async function authServiceSwitchOrganization(
  body: API.SwitchOrganizationRequest,
  options?: { [key: string]: any }
) {
  return request<API.MeReply>("/api/v1/auth/switch-organization", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}
