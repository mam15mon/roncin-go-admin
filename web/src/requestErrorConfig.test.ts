import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  errorConfig,
  getRequestErrorStatus,
  isRequestTimeoutError,
} from './requestErrorConfig';
import { showErrorMessage, showErrorNotification } from './utils/appFeedback';

const replace = vi.hoisted(() => vi.fn());
const captureException = vi.hoisted(() => vi.fn());

vi.mock('@sentry/react', () => ({
  captureException,
}));

vi.mock('@/utils/appFeedback', () => ({
  showErrorMessage: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock('@/router/history', () => ({
  history: {
    location: { pathname: '/welcome', search: '', hash: '' },
    replace,
  },
}));

interface RequestOptionsLike {
  method?: string;
  url?: string;
  data?: unknown;
  headers?: Record<string, string>;
  skipErrorHandler?: boolean;
  inflightWriteKey?: string;
}

describe('requestErrorConfig', () => {
  // biome-ignore lint/style/noNonNullAssertion: config handlers are always defined
  const errorThrower = errorConfig.errorConfig!.errorThrower!;
  // biome-ignore lint/style/noNonNullAssertion: config handlers are always defined
  const errorHandler = errorConfig.errorConfig!.errorHandler!;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should throw the backend business envelope', () => {
    expect(() => {
      errorThrower({
        success: false,
        code: 400,
        message: 'Bad Request',
        reason: 'AUTH_INVALID_CREDENTIALS',
      });
    }).toThrow('Bad Request');

    try {
      errorThrower({ success: false, code: 403, message: 'Forbidden' });
    } catch (error: any) {
      expect(error.name).toBe('BusinessError');
      expect(error.data).toEqual({
        success: false,
        code: 403,
        message: 'Forbidden',
      });
    }
  });

  it('should not throw for a successful envelope', () => {
    expect(() =>
      errorThrower({ success: true, data: { id: 1 } }),
    ).not.toThrow();
  });

  it('should redirect unauthorized requests to login', () => {
    errorHandler({ response: { status: 401 } } as any, {});

    expect(replace).toHaveBeenCalledWith('/user/login?redirect=%2Fwelcome');
    expect(showErrorMessage).not.toHaveBeenCalled();
    expect(showErrorNotification).not.toHaveBeenCalled();
    // 401 跳登录是预期流，不上报 Sentry。
    expect(captureException).not.toHaveBeenCalled();
  });

  it('只把明确的未认证错误识别为 401', () => {
    expect(getRequestErrorStatus({ response: { status: 401 } })).toBe(401);
    expect(
      getRequestErrorStatus({
        data: { success: false, code: 401, message: '未登录' },
      }),
    ).toBe(401);
    expect(getRequestErrorStatus({ response: { status: 500 } })).toBe(500);
    expect(getRequestErrorStatus(new Error('Network error'))).toBeUndefined();
  });

  it('should show a direct error for forbidden requests', () => {
    errorHandler(
      {
        response: {
          status: 403,
          data: { success: false, code: 403, message: '无权执行此操作' },
        },
      } as any,
      {},
    );

    expect(showErrorMessage).toHaveBeenCalledWith('无权执行此操作');
    expect(showErrorNotification).not.toHaveBeenCalled();
    expect(captureException).toHaveBeenCalledWith(expect.anything(), {
      tags: { kind: 'request', http_status: '403' },
    });
  });

  it('should include trace id in generic error notification', () => {
    errorHandler(
      {
        response: {
          status: 500,
          data: {
            success: false,
            code: 500,
            message: '服务暂不可用',
            traceId: 'trace-123',
          },
        },
      } as any,
      {},
    );

    expect(showErrorNotification).toHaveBeenCalledWith({
      title: '服务暂不可用',
      description: '追踪编号：trace-123',
    });
    expect(captureException).toHaveBeenCalledWith(expect.anything(), {
      tags: { kind: 'request', http_status: '500' },
    });
  });

  it('should handle a generic network error', () => {
    errorHandler(new Error('Network error'), {});

    expect(showErrorNotification).toHaveBeenCalledWith({
      title: '请求失败',
      description: '请联系系统管理员查看服务日志。',
    });
    expect(captureException).toHaveBeenCalledWith(expect.anything(), {
      tags: { kind: 'request' },
    });
  });

  it('明确识别请求库和 Fetch 的超时错误', () => {
    expect(isRequestTimeoutError({ code: 'ECONNABORTED' })).toBe(true);
    expect(isRequestTimeoutError({ code: 'ETIMEDOUT' })).toBe(true);
    expect(isRequestTimeoutError({ name: 'TimeoutError' })).toBe(true);
    expect(isRequestTimeoutError(new Error('Network error'))).toBe(false);

    errorHandler({ code: 'ECONNABORTED' } as any, {});

    expect(showErrorNotification).toHaveBeenCalledWith({
      title: '请求超时',
      description: '请确认操作结果后再重试，避免重复提交。',
    });
  });

  it('should rethrow when skipErrorHandler is true', () => {
    const error = new Error('Test error');

    expect(() => errorHandler(error, { skipErrorHandler: true })).toThrow(
      'Test error',
    );
  });

  it('should add a request id in the interceptor', () => {
    const interceptor = errorConfig.requestInterceptors?.[0] as (config: {
      headers?: Record<string, string>;
    }) => { headers?: Record<string, string> };
    const result = interceptor({ headers: { 'X-Client': 'web' } });

    expect(result.headers?.['X-Client']).toBe('web');
    expect(result.headers?.['X-Request-ID']).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
    );
  });

  it('请求错误以外的防重拦截反馈不上报 Sentry', () => {
    const duplicate = new Error('操作正在提交中，请勿重复提交') as any;
    duplicate.name = 'DuplicateSubmitError';
    duplicate.data = {
      success: false,
      code: 409,
      message: '操作正在提交中，请勿重复提交',
      reason: 'DUPLICATE_SUBMIT',
    };
    errorHandler(duplicate, {});

    expect(captureException).not.toHaveBeenCalled();
    expect(showErrorNotification).toHaveBeenCalledWith({
      title: '操作正在提交中，请勿重复提交',
      description: '请联系系统管理员查看服务日志。',
    });
  });
});

describe('请求层防重守卫', () => {
  // biome-ignore lint/style/noNonNullAssertion: guard interceptor is always defined
  const guard = errorConfig.requestInterceptors![1] as unknown as (
    config: RequestOptionsLike,
  ) => RequestOptionsLike;
  const responseInterceptorTuple = errorConfig.responseInterceptors?.[0] as
    | unknown
    | undefined;
  const [releaseOnSuccess, releaseOnError] = (responseInterceptorTuple ??
    []) as unknown as [
    (response: { config?: unknown }) => unknown,
    (error: { config?: unknown }) => Promise<unknown>,
  ];

  const write = (overrides: RequestOptionsLike = {}): RequestOptionsLike =>
    guard({
      method: 'POST',
      url: '/api/v1/orders',
      data: { customerId: 'c-1' },
      ...overrides,
    });

  const release = (config: RequestOptionsLike | undefined) => {
    releaseOnSuccess({ config });
  };

  afterEach(() => {
    // 清空在途记录：用一次 release 触达所有可能的遗留键。
    releaseOnSuccess({ config: undefined });
    releaseOnSuccess({
      config: {
        inflightWriteKey: 'POST /api/v1/orders {"customerId":"c-1"}',
      },
    });
    releaseOnSuccess({
      config: {
        inflightWriteKey: 'POST /api/v1/orders {"customerId":"c-2"}',
      },
    });
    releaseOnSuccess({
      config: { inflightWriteKey: 'PUT /api/v1/draft {"v":1}' },
    });
    releaseOnSuccess({
      config: { inflightWriteKey: 'DELETE /api/v1/items/1 ' },
    });
  });

  it('inflight 期间同 method+URL+体的第二次请求被拦截且不发后端', () => {
    const first = write();
    expect(first.inflightWriteKey).toBeTruthy();

    expect(() => write()).toThrow('操作正在提交中，请勿重复提交');

    try {
      write();
    } catch (error: any) {
      expect(error.name).toBe('DuplicateSubmitError');
      expect(error.data).toEqual({
        success: false,
        code: 409,
        message: '操作正在提交中，请勿重复提交',
        reason: 'DUPLICATE_SUBMIT',
      });
    }

    release(first);
  });

  it('请求完成（成功或失败）后立即放行合法重提', async () => {
    const first = write();
    expect(() => write()).toThrow();

    release(first);
    const retried = write();
    expect(retried.inflightWriteKey).toBeTruthy();
    release(retried);

    // 失败路径同样释放。
    const failed = write();
    await expect(releaseOnError({ config: failed })).rejects.toEqual({
      config: failed,
    });
    const retriedAgain = write();
    expect(retriedAgain.inflightWriteKey).toBeTruthy();
    release(retriedAgain);
  });

  it('请求体不同时不视为重复提交', () => {
    const first = write();
    const otherBody = write({ data: { customerId: 'c-2' } });
    expect(otherBody.inflightWriteKey).toBeTruthy();

    release(first);
    release(otherBody);
  });

  it('PUT 与 DELETE 同样受守卫', () => {
    const putFirst = write({
      method: 'PUT',
      url: '/api/v1/draft',
      data: { v: 1 },
    });
    expect(() =>
      write({ method: 'PUT', url: '/api/v1/draft', data: { v: 1 } }),
    ).toThrow();
    release(putFirst);

    const deleteFirst = guard({ method: 'DELETE', url: '/api/v1/items/1' });
    expect(() => guard({ method: 'DELETE', url: '/api/v1/items/1' })).toThrow();
    release(deleteFirst);
  });

  it('GET 请求不受守卫影响', () => {
    expect(() =>
      guard({ method: 'GET', url: '/api/v1/orders', data: undefined }),
    ).not.toThrow();
    expect(() =>
      guard({ method: 'GET', url: '/api/v1/orders', data: undefined }),
    ).not.toThrow();
  });

  it('skipErrorHandler 调用方仍受守卫', () => {
    const first = write({ skipErrorHandler: true });
    expect(() => write({ skipErrorHandler: true })).toThrow(
      '操作正在提交中，请勿重复提交',
    );
    release(first);
  });
});
