// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

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

/** 此处后端没有提供注释 POST /api/v1/auth/password-reset/code */
export async function authServiceRequestPasswordResetCode(
  body: API.RequestPasswordResetCodeRequest,
  options?: { [key: string]: any }
) {
  return request<API.RequestPasswordResetCodeResponse>(
    "/api/v1/auth/password-reset/code",
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

/** 此处后端没有提供注释 POST /api/v1/auth/password-reset/confirm */
export async function authServiceConfirmPasswordReset(
  body: API.ConfirmPasswordResetRequest,
  options?: { [key: string]: any }
) {
  return request<API.ConfirmPasswordResetResponse>(
    "/api/v1/auth/password-reset/confirm",
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
