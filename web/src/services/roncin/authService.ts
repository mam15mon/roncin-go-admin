// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 POST /api/v1/auth/dingtalk/login */
export async function authServiceDingTalkLogin(
  body: API.DingTalkLoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.DingTalkLoginResponse>("/api/v1/auth/dingtalk/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/auth/dingtalk/login-config */
export async function authServiceGetDingTalkLoginConfig(options?: {
  [key: string]: any;
}) {
  return request<API.GetDingTalkLoginConfigResponse>(
    "/api/v1/auth/dingtalk/login-config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/auth/login */
export async function authServiceLogin(
  body: API.LoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.LoginResponse>("/api/v1/auth/login", {
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
  body: API.LogoutRequest,
  options?: { [key: string]: any }
) {
  return request<API.LogoutResponse>("/api/v1/auth/logout", {
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
  return request<API.MeResponse>("/api/v1/auth/me", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/auth/switch-organization */
export async function authServiceSwitchOrganization(
  body: API.SwitchOrganizationRequest,
  options?: { [key: string]: any }
) {
  return request<API.SwitchOrganizationResponse>(
    "/api/v1/auth/switch-organization",
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

/** 此处后端没有提供注释 POST /api/v1/auth/wecom/login */
export async function authServiceWeComLogin(
  body: API.WeComLoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.WeComLoginResponse>("/api/v1/auth/wecom/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/auth/wecom/login-config */
export async function authServiceGetWeComLoginConfig(options?: {
  [key: string]: any;
}) {
  return request<API.GetWeComLoginConfigResponse>(
    "/api/v1/auth/wecom/login-config",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}
