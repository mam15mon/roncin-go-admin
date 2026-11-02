import {
  browserSupportsWebAuthn,
  type PublicKeyCredentialCreationOptionsJSON,
  type PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser';
import { queryOptions } from '@tanstack/react-query';
import { getRequestErrorMessage } from '@/requestErrorConfig';
import {
  passkeyServiceBeginPasskeyLogin,
  passkeyServiceBeginPasskeyRegistration,
  passkeyServiceGetPasskeyConfig,
} from '@/services/roncin/passkeyService';

export class PasskeyFlowError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'PasskeyFlowError';
  }
}

export const passkeyKeys = {
  all: ['auth', 'passkeys'] as const,
  config: ['auth', 'passkeys', 'config'] as const,
  login: ['auth', 'passkeys', 'login-options'] as const,
  registration: (userId: string) =>
    ['auth', 'passkeys', 'registration-options', userId] as const,
  credentials: (userId: string) =>
    ['auth', 'passkeys', 'credentials', userId] as const,
};

export type PasskeyConfiguration = {
  enabled: boolean;
  rpId: string;
  rpName: string;
  origins: string[];
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export const passkeyConfigOptions = queryOptions({
  queryKey: passkeyKeys.config,
  queryFn: async (): Promise<PasskeyConfiguration> => {
    const response = await passkeyServiceGetPasskeyConfig({
      skipErrorHandler: true,
    });
    if (!response?.data || response.success === false)
      throw new PasskeyFlowError('通行密钥配置获取失败');
    const data = response.data;
    if (
      data.enabled &&
      (!data.rpId ||
        !data.rpName ||
        !Array.isArray(data.origins) ||
        !data.origins.every((origin) => typeof origin === 'string'))
    )
      throw new PasskeyFlowError('通行密钥配置不完整');
    return {
      enabled: data.enabled === true,
      rpId: data.rpId ?? '',
      rpName: data.rpName ?? '',
      origins: data.origins ?? [],
    };
  },
  staleTime: 60_000,
  gcTime: 5 * 60_000,
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
  meta: { silent: true },
});

export function passkeyUnavailableReason(
  config?: PasskeyConfiguration,
): string {
  if (!window.isSecureContext) return '请使用 HTTPS 域名开启通行密钥';
  if (!browserSupportsWebAuthn()) return '当前浏览器不支持通行密钥';
  if (!config) return '正在确认通行密钥支持';
  if (!config.enabled) return '通行密钥尚未启用，请联系管理员';
  if (!config.origins.includes(window.location.origin))
    return '当前登录地址尚未启用通行密钥';
  return '';
}

export type PreparedCeremony = {
  options: Record<string, unknown>;
  expiresAt: number;
};

function parseCeremony(data?: API.PasskeyCeremony): PreparedCeremony {
  if (!data?.optionsJson || !data.expiresAt)
    throw new PasskeyFlowError('通行密钥验证信息不完整');
  const options: unknown = JSON.parse(data.optionsJson);
  const expiresAt = Date.parse(data.expiresAt);
  if (
    !isRecord(options) ||
    typeof options.challenge !== 'string' ||
    !options.challenge ||
    !Number.isFinite(expiresAt) ||
    expiresAt <= Date.now()
  )
    throw new PasskeyFlowError('通行密钥验证已过期，请重试');
  return { options, expiresAt };
}

export const passkeyLoginOptions = queryOptions({
  queryKey: passkeyKeys.login,
  queryFn: async () => {
    // Begin 写 HttpOnly challenge Cookie，不中止在途响应，避免后到 Cookie 覆盖新仪式。
    const response = await passkeyServiceBeginPasskeyLogin(
      {},
      {
        skipErrorHandler: true,
      },
    );
    if (!response || response.success === false)
      throw new PasskeyFlowError('通行密钥验证准备失败');
    return parseCeremony(response.data);
  },
  staleTime: 0,
  gcTime: 5 * 60_000,
  refetchOnMount: 'always',
  retryOnMount: false,
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
  meta: { silent: true },
});

export function passkeyRegistrationOptions(userId: string) {
  return queryOptions({
    queryKey: passkeyKeys.registration(userId),
    queryFn: async () => {
      const response = await passkeyServiceBeginPasskeyRegistration(
        {},
        {
          skipErrorHandler: true,
        },
      );
      if (!response || response.success === false)
        throw new PasskeyFlowError('通行密钥注册准备失败');
      return parseCeremony(response.data);
    },
    staleTime: 0,
    gcTime: 0,
    retryOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    meta: { silent: true },
  });
}

export function authenticationOptions(
  ceremony: PreparedCeremony | undefined,
  rpId: string,
): PublicKeyCredentialRequestOptionsJSON {
  if (!ceremony || ceremony.expiresAt <= Date.now())
    throw new PasskeyFlowError('通行密钥验证已过期，请重试');
  if (ceremony.options.rpId !== rpId)
    throw new PasskeyFlowError('通行密钥验证域名不正确');
  if (!isRequestOptions(ceremony.options))
    throw new PasskeyFlowError('通行密钥验证信息不正确');
  return ceremony.options;
}

export function registrationOptions(
  ceremony: PreparedCeremony | undefined,
  rpId: string,
): PublicKeyCredentialCreationOptionsJSON {
  if (!ceremony || ceremony.expiresAt <= Date.now())
    throw new PasskeyFlowError('通行密钥注册已过期，请重试');
  const { options } = ceremony;
  if (
    !isRecord(options.rp) ||
    options.rp.id !== rpId ||
    !isRecord(options.user) ||
    typeof options.user.id !== 'string' ||
    !options.user.id ||
    typeof options.user.name !== 'string' ||
    typeof options.user.displayName !== 'string' ||
    !Array.isArray(options.pubKeyCredParams) ||
    !options.pubKeyCredParams.length
  )
    throw new PasskeyFlowError('通行密钥注册信息不正确');
  if (!isCreationOptions(options))
    throw new PasskeyFlowError('通行密钥注册信息不正确');
  return options;
}

export function passkeyErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof PasskeyFlowError) return error.message;
  if (!error) return fallback;
  if (
    error instanceof Error &&
    (error.name === 'NotAllowedError' ||
      error.name === 'AbortError' ||
      (error.cause instanceof Error &&
        ['NotAllowedError', 'AbortError'].includes(error.cause.name)))
  )
    return '未完成通行密钥验证，可以重试或选择其他登录方式';
  return getRequestErrorMessage(error, fallback);
}

function isRequestOptions(
  value: Record<string, unknown>,
): value is Record<string, unknown> & PublicKeyCredentialRequestOptionsJSON {
  return typeof value.challenge === 'string' && typeof value.rpId === 'string';
}

function isCreationOptions(
  value: Record<string, unknown>,
): value is Record<string, unknown> & PublicKeyCredentialCreationOptionsJSON {
  return (
    typeof value.challenge === 'string' &&
    isRecord(value.rp) &&
    typeof value.rp.name === 'string' &&
    typeof value.rp.id === 'string' &&
    isRecord(value.user) &&
    typeof value.user.id === 'string' &&
    typeof value.user.name === 'string' &&
    typeof value.user.displayName === 'string' &&
    Array.isArray(value.pubKeyCredParams) &&
    value.pubKeyCredParams.every(
      (parameter) =>
        isRecord(parameter) &&
        parameter.type === 'public-key' &&
        typeof parameter.alg === 'number',
    )
  );
}
