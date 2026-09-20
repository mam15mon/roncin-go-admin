import type { RequestOptions } from '@/utils/requestClient';
import * as Sentry from '@sentry/react';
import { history } from '@/router/history';
import { showErrorMessage, showErrorNotification } from '@/utils/appFeedback';
import { generateUUID } from '@/utils/uuid';

interface ErrorEnvelope {
  success: false;
  code: number;
  message: string;
  reason?: string;
  traceId?: string;
}

export interface RequestError extends Error {
  code?: string;
  response?: { status?: number; data?: ErrorEnvelope };
  data?: ErrorEnvelope;
}

// 防重守卫附加在 axios config 上的标记，用于请求完成后释放 in-flight 记录。
interface InflightConfig extends RequestOptions {
  inflightWriteKey?: string;
}

const loginPath = '/user/login';

function redirectToLogin() {
  if (history.location.pathname === loginPath) return;
  const { pathname, search, hash } = history.location;
  history.replace(
    `${loginPath}?redirect=${encodeURIComponent(pathname + search + hash)}`,
  );
}

export function getRequestErrorStatus(rawError: unknown): number | undefined {
  const error = rawError as RequestError;
  return (
    error.response?.status ?? error.data?.code ?? error.response?.data?.code
  );
}

// 提取服务端业务报文中的用户可读提示。skipErrorHandler 的调用方（如登录页）
// 用它展示后端原因，避免把 Axios 的英文默认文案直接抛给用户。
export function getRequestErrorMessage(
  rawError: unknown,
  fallback: string,
): string {
  const error = rawError as RequestError;
  return error.data?.message ?? error.response?.data?.message ?? fallback;
}

export function isRequestTimeoutError(rawError: unknown): boolean {
  const error = rawError as RequestError;
  return (
    error.code === 'ECONNABORTED' ||
    error.code === 'ETIMEDOUT' ||
    error.name === 'TimeoutError'
  );
}

// 请求错误统一上报 Sentry（401 跳登录与防重守卫拦截除外）。
// 未配置 SENTRY_DSN 时 Sentry 处于 no-op 客户端，该调用零开销。
function captureRequestError(error: Error, status?: number) {
  Sentry.captureException(error, {
    tags: {
      kind: 'request',
      ...(status !== undefined ? { http_status: String(status) } : {}),
    },
  });
}

function isDuplicateSubmitError(rawError: unknown): boolean {
  return (rawError as RequestError)?.name === 'DuplicateSubmitError';
}

// ---------------------------------------------------------------------------
// 请求层防重守卫：对「同 method + URL + 序列化请求体」的写操作做 in-flight 去重。
// 命中时第二个请求不再发往后端，直接抛业务错误；请求完成（成功或失败）即移除
// 记录，不做完成后的时间窗缓存，合法重提不受阻。
// ---------------------------------------------------------------------------
const WRITE_METHODS = new Set(['POST', 'PUT', 'DELETE']);
const inflightWriteKeys = new Set<string>();

function serializeWriteBody(data: unknown): string {
  if (data === null || data === undefined) return '';
  if (typeof data === 'string') return data;
  try {
    return JSON.stringify(data) ?? '';
  } catch {
    return String(data);
  }
}

function requestWriteKey(config: RequestOptions): string | null {
  const method = (config.method ?? 'GET').toUpperCase();
  if (!WRITE_METHODS.has(method)) return null;
  return `${method} ${config.url ?? ''} ${serializeWriteBody(config.data)}`;
}

function duplicateSubmitError(): RequestError {
  const envelope: ErrorEnvelope = {
    success: false,
    code: 409,
    message: '操作正在提交中，请勿重复提交',
    reason: 'DUPLICATE_SUBMIT',
  };
  const error = new Error(envelope.message) as RequestError;
  error.name = 'DuplicateSubmitError';
  error.data = envelope;
  return error;
}

// 守卫拦截也覆盖 skipErrorHandler 调用方：防重是正确性底线而非提示策略。
function guardInflightWrite(config: RequestOptions): RequestOptions {
  const key = requestWriteKey(config);
  if (key === null) return config;
  if (inflightWriteKeys.has(key)) {
    throw duplicateSubmitError();
  }
  inflightWriteKeys.add(key);
  return { ...config, inflightWriteKey: key } as InflightConfig;
}

function releaseInflightWrite(config: unknown) {
  const key = (config as InflightConfig | undefined)?.inflightWriteKey;
  if (key) inflightWriteKeys.delete(key);
}

// 原 @umijs/max RequestConfig 的等价本地类型：拦截器与错误处理约定的挂载结构，
// 由 @/utils/requestClient 消费。
export interface AxiosResponseLike {
  config?: unknown;
  [key: string]: unknown;
}

export interface AppRequestConfig {
  errorConfig?: {
    errorThrower?: (response: unknown) => void;
    errorHandler?: (
      error: unknown,
      options?: { skipErrorHandler?: boolean },
    ) => void;
  };
  requestInterceptors?: Array<(config: RequestOptions) => RequestOptions>;
  responseInterceptors?: Array<[
    (
      response: AxiosResponseLike,
    ) => AxiosResponseLike | Promise<AxiosResponseLike>,
    (error: unknown) => unknown,
  ]>;
}

export const errorConfig: AppRequestConfig = {
  errorConfig: {
    errorThrower: (response) => {
      const envelope = response as ErrorEnvelope;
      if (!envelope.success) {
        const error = new Error(envelope.message) as RequestError;
        error.name = 'BusinessError';
        error.data = envelope;
        throw error;
      }
    },
    errorHandler: (rawError, options) => {
      if (options?.skipErrorHandler) throw rawError;
      const error = rawError as RequestError;
      const envelope = error.data ?? error.response?.data;
      const status = getRequestErrorStatus(error);

      if (isRequestTimeoutError(error)) {
        captureRequestError(error, status);
        showErrorNotification({
          title: '请求超时',
          description: '请确认操作结果后再重试，避免重复提交。',
        });
        return;
      }
      if (status === 401) {
        // 预期流：跳转登录页，不上报 Sentry。
        redirectToLogin();
        return;
      }
      if (!isDuplicateSubmitError(error)) {
        // 防重拦截属于预期交互反馈，不作为异常上报。
        captureRequestError(error, status);
      }
      if (status === 403) {
        showErrorMessage(envelope?.message ?? '无权执行此操作');
        return;
      }
      showErrorNotification({
        title: envelope?.message ?? '请求失败',
        description: envelope?.traceId
          ? `追踪编号：${envelope.traceId}`
          : '请联系系统管理员查看服务日志。',
      });
    },
  },
  requestInterceptors: [
    (config: RequestOptions) => ({
      ...config,
      headers: {
        ...config.headers,
        'X-Request-ID': generateUUID(),
      },
    }),
    guardInflightWrite,
  ],
  responseInterceptors: [
    [
      (response) => {
        releaseInflightWrite(response?.config);
        return response;
      },
      (error) => {
        releaseInflightWrite((error as { config?: unknown })?.config);
        return Promise.reject(error);
      },
    ],
  ],
};
