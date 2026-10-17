import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider, useQuery } from '@tanstack/react-query';
import { act, cleanup, render, screen } from '@testing-library/react';
import type {
  AxiosAdapter,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios';
import { StrictMode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import { OrganizationWorkspace } from '@/components/layout/OrganizationWorkspace';
import { activateWorkspaceRequestScope, request } from './requestClient';

const { transport, replace, showErrorMessage, showErrorNotification } =
  vi.hoisted(() => ({
    transport: vi.fn<AxiosAdapter>(),
    replace: vi.fn(),
    showErrorMessage: vi.fn(),
    showErrorNotification: vi.fn(),
  }));
vi.mock('axios', async (original) => {
  const actual = await original<typeof import('axios')>();
  return {
    ...actual,
    default: {
      ...actual.default,
      create: (config: Parameters<typeof actual.default.create>[0]) =>
        actual.default.create({ ...config, adapter: transport }),
    },
  };
});
vi.mock('@/components/layout/TagsView', () => ({ TagsView: () => null }));
vi.mock('@sentry/react', () => ({ captureException: vi.fn() }));
vi.mock('@/utils/appFeedback', () => ({
  showErrorMessage,
  showErrorNotification,
}));
vi.mock('@/router/history', () => ({
  history: {
    location: { pathname: '/welcome', search: '', hash: '' },
    replace,
  },
}));

function pendingRequest() {
  let resolve!: (response: AxiosResponse) => void;
  let reject!: (error: unknown) => void;
  let config!: InternalAxiosRequestConfig;
  transport.mockImplementationOnce((options) => {
    config = options;
    return new Promise<AxiosResponse>((yes, no) => {
      resolve = yes;
      reject = no;
    });
  });
  return {
    success: (data: unknown) =>
      resolve({ data, config, status: 200, statusText: 'OK', headers: {} }),
    fail: (error: unknown) => reject(error),
    getConfig: () => config,
  };
}

const scopes: Array<() => void> = [];
function beginScope() {
  const release = activateWorkspaceRequestScope();
  scopes.push(release);
  return release;
}

beforeEach(() => vi.clearAllMocks());
afterEach(() => {
  cleanup();
  for (const release of scopes.splice(0)) release();
});

describe('认证工作区统一请求提示隔离', () => {
  it('StrictMode模拟卸载后的真实请求可完成，旧传输迟到失败不干扰重挂载', async () => {
    const oldRead = pendingRequest();
    const currentRead = pendingRequest();
    function BusinessQuery() {
      const { data } = useQuery({
        queryKey: ['strict-workspace-request'],
        queryFn: () => request<{ data: string }>('/api/v1/strict-workspace'),
        meta: { silent: true },
      });
      return <output>{data?.data ?? '正在加载'}</output>;
    }
    render(
      <StrictMode>
        <appInitialStateContext.Provider
          value={{
            initialState: {
              currentUser: {
                id: 'user-a',
                currentOrganization: { id: 'company-a' },
              },
            },
            setInitialState: vi.fn(),
            refresh: vi.fn(),
            loading: false,
          }}
        >
          <OrganizationWorkspace>
            <BusinessQuery />
          </OrganizationWorkspace>
        </appInitialStateContext.Provider>
      </StrictMode>,
    );
    await vi.waitFor(() => expect(transport).toHaveBeenCalledTimes(2));
    await act(async () => currentRead.success({ data: '重挂载工作区数据' }));
    expect(await screen.findByText('重挂载工作区数据')).toBeVisible();
    await act(async () => oldRead.fail({ response: { status: 401 } }));
    expect(screen.getByText('重挂载工作区数据')).toBeVisible();
    expect(replace).not.toHaveBeenCalled();
    expect(showErrorNotification).not.toHaveBeenCalled();
  });

  it.each([
    ['401', { response: { status: 401 } }],
    ['500', { response: { status: 500, data: { message: '旧工作区失败' } } }],
    ['超时', { code: 'ECONNABORTED' }],
  ])(
    '旧工作区%s迟到不导航或通知新身份，仍返回undefined',
    async (_name, error) => {
      const releaseOld = beginScope();
      const pending = pendingRequest();
      const response = request('/api/v1/workbench/my-commissions');
      await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce());
      releaseOld();
      beginScope();
      pending.fail(error);
      await expect(response).resolves.toBeUndefined();
      expect(replace).not.toHaveBeenCalled();
      expect(showErrorMessage).not.toHaveBeenCalled();
      expect(showErrorNotification).not.toHaveBeenCalled();

      transport.mockRejectedValueOnce({
        response: { status: 403, data: { message: '当前身份无权访问' } },
      });
      await expect(
        request('/api/v1/workbench/my-receivables'),
      ).resolves.toBeUndefined();
      expect(showErrorMessage).toHaveBeenCalledWith('当前身份无权访问');
    },
  );

  it('业务迟到成功按undefined返回，当前工作区成功仍保留原报文', async () => {
    const release = beginScope();
    const pending = pendingRequest();
    const oldSave = request('/api/v1/orders/1', {
      method: 'PUT',
      data: { expectedVersion: '1' },
    });
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce());
    release();
    beginScope();
    const success = { success: true, data: { id: 'order-1' } };
    pending.success(success);
    await expect(oldSave).resolves.toBeUndefined();
    const currentRead = pendingRequest();
    const currentResponse = request('/api/v1/orders/1');
    await vi.waitFor(() => expect(transport).toHaveBeenCalledTimes(2));
    currentRead.success(success);
    await expect(currentResponse).resolves.toEqual(success);
    expect(showErrorNotification).not.toHaveBeenCalled();
  });

  it('真实工作区Provider切换自动保护统一请求，匿名后不遗留作用域', async () => {
    const oldRead = pendingRequest();
    const newRead = pendingRequest();
    function BusinessQuery() {
      const { data } = useQuery({
        queryKey: ['workbench', 'my-commissions'],
        queryFn: () =>
          request<{ data: string }>('/api/v1/workbench/my-commissions'),
        meta: { silent: true },
      });
      return <output>{data?.data ?? '正在加载'}</output>;
    }
    const entryClient = createTestQueryClient();
    function tree(userId?: string) {
      return (
        <QueryClientProvider client={entryClient}>
          <appInitialStateContext.Provider
            value={{
              initialState: {
                currentUser: userId
                  ? { id: userId, currentOrganization: { id: 'company' } }
                  : undefined,
              },
              setInitialState: vi.fn(),
              refresh: vi.fn(),
              loading: false,
            }}
          >
            <OrganizationWorkspace>
              <BusinessQuery />
            </OrganizationWorkspace>
          </appInitialStateContext.Provider>
        </QueryClientProvider>
      );
    }
    const view = render(tree('user-a'));
    await vi.waitFor(() => expect(transport).toHaveBeenCalledTimes(1));
    view.rerender(tree('user-b'));
    await vi.waitFor(() => expect(transport).toHaveBeenCalledTimes(2));
    await act(async () => oldRead.fail({ response: { status: 401 } }));
    expect(replace).not.toHaveBeenCalled();
    expect(showErrorNotification).not.toHaveBeenCalled();
    expect(screen.getByText('正在加载')).toBeVisible();
    await act(async () => newRead.success({ data: 'B业务数据' }));
    expect(await screen.findByText('B业务数据')).toBeVisible();
    view.rerender(tree());
    transport.mockRejectedValueOnce({ response: { status: 401 } });
    await expect(request('/api/v1/auth/me')).resolves.toBeUndefined();
    expect(replace).toHaveBeenCalledWith('/user/login?redirect=%2Fwelcome');
  });

  it('当前工作区和无工作区请求保持原401跳转与失败返回契约', async () => {
    const release = beginScope();
    transport.mockRejectedValueOnce({ response: { status: 401 } });
    await expect(request('/api/v1/orders')).resolves.toBeUndefined();
    expect(replace).toHaveBeenCalledWith('/user/login?redirect=%2Fwelcome');
    release();
    transport.mockRejectedValueOnce(new Error('无工作区请求失败'));
    await expect(request('/api/v1/auth/me')).resolves.toBeUndefined();
    expect(showErrorNotification).toHaveBeenCalledWith(
      expect.objectContaining({ title: '请求失败' }),
    );
  });

  it('skipErrorHandler继续原样外抛旧请求错误，查询缓存负责丢弃取消结果', async () => {
    const release = beginScope();
    const pending = pendingRequest();
    const error = new Error('原始失败');
    const response = request('/api/v1/orders/1', { skipErrorHandler: true });
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce());
    release();
    pending.fail(error);
    await expect(response).rejects.toBe(error);
    expect(showErrorNotification).not.toHaveBeenCalled();
  });

  it.each([
    '/api/v1/auth/dingtalk/login-config',
    '/api/v1/auth/dingtalk/login',
    '/api/v1/auth/login',
  ])('公开协议%s的传输和成功报文不因工作区卸载丢失', async (url) => {
    const release = beginScope();
    const pending = pendingRequest();
    const response = request(url);
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce());
    release();
    beginScope();
    expect(pending.getConfig().signal).toBeUndefined();
    pending.success({
      success: true,
      data: { state: 'cookie对应的原始state' },
    });
    await expect(response).resolves.toEqual({
      success: true,
      data: { state: 'cookie对应的原始state' },
    });
  });

  it('公开登录失败不被旧业务作用域静默吞掉', async () => {
    const release = beginScope();
    const pending = pendingRequest();
    const response = request('/api/v1/auth/login');
    await vi.waitFor(() => expect(transport).toHaveBeenCalledOnce());
    release();
    pending.fail({ response: { status: 403, data: { message: '登录失败' } } });
    await expect(response).resolves.toBeUndefined();
    expect(showErrorMessage).toHaveBeenCalledWith('登录失败');
  });
});
