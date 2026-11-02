// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** 此处后端没有提供注释 GET /api/v1/auth/passkey/config */
export async function passkeyServiceGetPasskeyConfig(options?: {
  [key: string]: any;
}) {
  return request<API.GetPasskeyConfigResponse>("/api/v1/auth/passkey/config", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/auth/passkey/login/begin */
export async function passkeyServiceBeginPasskeyLogin(
  body: API.BeginPasskeyLoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.BeginPasskeyLoginResponse>(
    "/api/v1/auth/passkey/login/begin",
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

/** 此处后端没有提供注释 POST /api/v1/auth/passkey/login/finish */
export async function passkeyServiceFinishPasskeyLogin(
  body: API.FinishPasskeyLoginRequest,
  options?: { [key: string]: any }
) {
  return request<API.FinishPasskeyLoginResponse>(
    "/api/v1/auth/passkey/login/finish",
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

/** 此处后端没有提供注释 POST /api/v1/auth/passkey/registration/begin */
export async function passkeyServiceBeginPasskeyRegistration(
  body: API.BeginPasskeyRegistrationRequest,
  options?: { [key: string]: any }
) {
  return request<API.BeginPasskeyRegistrationResponse>(
    "/api/v1/auth/passkey/registration/begin",
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

/** 此处后端没有提供注释 POST /api/v1/auth/passkey/registration/finish */
export async function passkeyServiceFinishPasskeyRegistration(
  body: API.FinishPasskeyRegistrationRequest,
  options?: { [key: string]: any }
) {
  return request<API.FinishPasskeyRegistrationResponse>(
    "/api/v1/auth/passkey/registration/finish",
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

/** 此处后端没有提供注释 GET /api/v1/auth/passkeys */
export async function passkeyServiceListPasskeys(options?: {
  [key: string]: any;
}) {
  return request<API.ListPasskeysResponse>("/api/v1/auth/passkeys", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 DELETE /api/v1/auth/passkeys/${param0} */
export async function passkeyServiceDeletePasskey(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PasskeyServiceDeletePasskeyParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.DeletePasskeyResponse>(`/api/v1/auth/passkeys/${param0}`, {
    method: "DELETE",
    params: { ...queryParams },
    ...(options || {}),
  });
}
