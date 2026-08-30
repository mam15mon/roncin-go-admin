import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  errorConfig,
  getRequestErrorStatus,
  isRequestTimeoutError,
} from './requestErrorConfig';
import { showErrorMessage, showErrorNotification } from './utils/appFeedback';

const replace = vi.hoisted(() => vi.fn());

vi.mock('@/utils/appFeedback', () => ({
  showErrorMessage: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  history: {
    location: { pathname: '/welcome', search: '', hash: '' },
    replace,
  },
}));

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
  });

  it('should handle a generic network error', () => {
    errorHandler(new Error('Network error'), {});

    expect(showErrorNotification).toHaveBeenCalledWith({
      title: '请求失败',
      description: '请联系系统管理员查看服务日志。',
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
});
