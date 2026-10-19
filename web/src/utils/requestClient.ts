import type {
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios';
import axios from 'axios';
import { errorConfig } from '@/requestErrorConfig';
import { DEFAULT_REQUEST_TIMEOUT } from '@/utils/requestTimeout';

// 与 OpenAPI 生成客户端约定的请求选项：method/params/data/headers 与透传键
// （skipErrorHandler、覆盖 timeout 等）均为顶层字段，结构对齐 umi-request。
export interface RequestOptions {
  method?: AxiosRequestConfig['method'];
  params?: Record<string, unknown>;
  data?: unknown;
  headers?: Record<string, string>;
  timeout?: number;
  signal?: AbortSignal;
  responseType?: AxiosRequestConfig['responseType'];
  skipErrorHandler?: boolean;
  [key: string]: unknown;
}

interface ResponseEnvelope {
  success?: boolean;
  code?: number;
  message?: string;
  reason?: string;
  traceId?: string;
}

export interface RequestError extends Error {
  code?: string;
  response?: { status?: number; data?: ResponseEnvelope };
  data?: ResponseEnvelope;
}

// 只标记认证工作区请求的展示生命周期，不中止传输或改变 Cookie 响应顺序。
let workspaceRequestScope: { active: boolean } | undefined;

export function activateWorkspaceRequestScope(): () => void {
  const scope = { active: true };
  workspaceRequestScope = scope;
  return () => {
    scope.active = false;
    if (workspaceRequestScope === scope) workspaceRequestScope = undefined;
  };
}

const instance = axios.create({
  baseURL: '',
  timeout: DEFAULT_REQUEST_TIMEOUT,
  withCredentials: true,
});

// X-Request-ID、写操作防重守卫与 in-flight 释放复用既有实现，不另起一套。
for (const interceptor of errorConfig.requestInterceptors ?? []) {
  instance.interceptors.request.use(
    interceptor as unknown as (
      config: InternalAxiosRequestConfig,
    ) => InternalAxiosRequestConfig,
  );
}
for (const pair of errorConfig.responseInterceptors ?? []) {
  const [onFulfilled, onRejected] = pair as unknown as [
    (response: AxiosResponse) => AxiosResponse | Promise<AxiosResponse>,
    (error: unknown) => unknown,
  ];
  instance.interceptors.response.use(onFulfilled, onRejected);
}

/**
 * 统一请求入口，复刻 umi-request 契约：
 * - 返回完整响应体（success/data 报文不解包，由调用方取 .data）；
 * - errorThrower：success:false 报文转 BusinessError（复用 requestErrorConfig）；
 * - errorHandler 消费掉的错误（401 跳转、防重拦截、超时/失败通知）resolve 为
 *   undefined，与 umi-request「错误被处理后不再 reject」的行为一致；
 * - skipErrorHandler: true 时原始错误直接外抛，由调用方自行处理。
 */
export async function request<T>(
  url: string,
  options: RequestOptions = {},
): Promise<T> {
  // 登录配置/换码属于公开会话协议，不参与业务工作区清理。
  const requestScope =
    url === '/api/v1/auth/dingtalk/embedded-config' ||
    url === '/api/v1/auth/dingtalk/embedded-login' ||
    url === '/api/v1/auth/dingtalk/login-config' ||
    url === '/api/v1/auth/dingtalk/login' ||
    url === '/api/v1/auth/login' ||
    url === '/api/v1/auth/password-reset/code' ||
    url === '/api/v1/auth/password-reset/confirm'
      ? undefined
      : workspaceRequestScope;
  const {
    method = 'GET',
    params,
    data,
    headers,
    timeout,
    signal,
    responseType,
  } = options;
  try {
    const response = await instance.request<T>({
      url,
      method,
      params,
      data,
      headers,
      timeout,
      signal,
      responseType,
    });
    // 已退出的工作区结果按消费失败处理，阻止旧保存回调导航、提示或清草稿。
    if (requestScope && !requestScope.active) return undefined as T;
    const body = response.data as T & ResponseEnvelope;
    // 仅对携带 success 字段的 JSON 报文做业务校验；blob / 数组等形态跳过。
    if (body && typeof body === 'object' && 'success' in body) {
      errorConfig.errorConfig?.errorThrower?.(body);
    }
    return body;
  } catch (rawError) {
    if (options.skipErrorHandler) throw rawError;
    // 旧工作区的 401、超时等迟到失败不能导航或通知新身份。
    if (requestScope && !requestScope.active) return undefined as T;
    const errorHandler = errorConfig.errorConfig?.errorHandler as
      | ((error: unknown, options?: { skipErrorHandler?: boolean }) => void)
      | undefined;
    errorHandler?.(rawError, { skipErrorHandler: options.skipErrorHandler });
    return undefined as T;
  }
}
