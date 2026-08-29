import type { RequestOptions } from '@@/plugin-request/request';
import type { RequestConfig } from '@umijs/max';
import { history } from '@umijs/max';
import { showErrorMessage, showErrorNotification } from '@/utils/appFeedback';

interface ErrorEnvelope {
  success: false;
  code: number;
  message: string;
  reason?: string;
  traceId?: string;
}

export interface RequestError extends Error {
  response?: { status?: number; data?: ErrorEnvelope };
  data?: ErrorEnvelope;
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

export const errorConfig: RequestConfig = {
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

      if (status === 401) {
        redirectToLogin();
        return;
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
        'X-Request-ID': crypto.randomUUID(),
      },
    }),
  ],
};
