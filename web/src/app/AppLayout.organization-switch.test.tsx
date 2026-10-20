import type { ProLayoutProps } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React, { useMemo, useState } from 'react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import { bindRouter, history } from '@/router/history';
import { AppLayout } from './AppLayout';
import {
  appInitialStateContext,
  type InitialState,
  useInitialState,
} from './AppProvider';

const { switchOrganization, cacheClear } = vi.hoisted(() => ({
  switchOrganization: vi.fn(),
  cacheClear: vi.fn(),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceSwitchOrganization: switchOrganization,
}));
vi.mock('@/features/orders/options', () => ({
  clearOrderMasterDataCache: cacheClear,
}));
vi.mock('@ant-design/pro-components', () => ({
  ProLayout: (props: ProLayoutProps) => (
    <div>
      {typeof props.actionsRender === 'function'
        ? props.actionsRender({})
        : null}
      {props.children}
    </div>
  ),
}));
vi.mock('@/components/layout/HeaderMenus', () => ({ HeaderMenus: () => null }));
vi.mock('@/components/layout/HeaderTitle', () => ({ HeaderTitle: () => null }));
vi.mock('@/components/layout/TagsView', () => ({ TagsView: () => null }));
vi.mock('@/components/RightContent/AvatarDropdown', () => ({
  AvatarDropdown: () => null,
}));
vi.mock('@/components/HeaderDropdown', () => ({
  default: ({
    menu,
    children,
  }: {
    children: React.ReactNode;
    menu: { onClick: (item: { key: string }) => void };
  }) => (
    <div>
      {children}
      <button type="button" onClick={() => menu.onClick({ key: 'chengdu' })}>
        切成都
      </button>
    </div>
  ),
}));

function user(organizationId: string): API.CurrentUser {
  return {
    id: 'admin',
    currentOrganization: {
      id: organizationId,
      kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      name: organizationId === 'beijing' ? '北京公司' : '成都公司',
    },
    organizations: [
      { id: 'beijing', name: '北京公司' },
      { id: 'chengdu', name: '成都公司' },
    ],
    permissionCapabilities: [
      'system.platform.access',
      'system.role.read',
      'business.order.se.read',
    ].map((key) => ({ key, dataScope: 'organization' })),
  };
}

function Provider({ children }: { children: React.ReactNode }) {
  const [initialState, setInitialState] = useState<InitialState | undefined>({
    currentUser: user('beijing'),
  });
  const value = useMemo(
    () => ({
      initialState,
      setInitialState,
      refresh: () => {},
      loading: false,
    }),
    [initialState],
  );
  return (
    <App>
      <appInitialStateContext.Provider value={value}>
        {children}
      </appInitialStateContext.Provider>
    </App>
  );
}

const requests: string[] = [];
function QueryPage({ record }: { record: boolean }) {
  const { initialState } = useInitialState();
  const org = initialState?.currentUser?.currentOrganization?.id;
  useQuery({
    queryKey: ['switch-regression', record, org],
    queryFn: async () => {
      requests.push(`${org}:${record ? 'old-record' : 'list'}`);
      return org;
    },
  });
  return (
    <div>
      {org}:{record ? '旧订单' : '列表'}
      <input aria-label="当前输入" defaultValue="原输入" />
    </div>
  );
}

let router: ReturnType<typeof createMemoryRouter> | undefined;
afterEach(() => {
  cleanup();
  router?.dispose();
  vi.restoreAllMocks();
  vi.clearAllMocks();
  requests.length = 0;
});

function mount(
  initialPath: string,
  children: Parameters<typeof createMemoryRouter>[0],
) {
  router = createMemoryRouter([{ Component: AppLayout, children }], {
    initialEntries: [initialPath],
  });
  bindRouter(router);
  render(
    <Provider>
      <RouterProvider router={router} />
    </Provider>,
  );
  return router;
}

describe('真实异步路由中的工作区转换', () => {
  it('懒加载未完成期间不以新身份挂载旧记录，也不以旧身份挂载新列表', async () => {
    let ready:
      | ((value: { Component: () => React.ReactNode }) => void)
      | undefined;
    const lazyTarget = new Promise<{ Component: () => React.ReactNode }>(
      (resolve) => {
        ready = resolve;
      },
    );
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    const activeRouter = mount('/orders/sea-export/old?tab=fees#record', [
      { path: '/orders/sea-export/:id', Component: () => <QueryPage record /> },
      { path: '/orders/sea-export', lazy: () => lazyTarget },
    ]);
    expect(await screen.findByText('beijing:旧订单')).toBeInTheDocument();
    await waitFor(() => expect(requests).toEqual(['beijing:old-record']));
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    await waitFor(() =>
      expect(activeRouter.state.navigation.state).toBe('loading'),
    );
    expect(screen.queryByText('chengdu:旧订单')).not.toBeInTheDocument();
    expect(screen.queryByText('beijing:旧订单')).not.toBeInTheDocument();
    expect(screen.queryByText('chengdu:列表')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '切换当前组织' })).toBeDisabled();
    expect(requests).toEqual(['beijing:old-record']);
    await act(async () => {
      ready?.({ Component: () => <QueryPage record={false} /> });
      await lazyTarget;
    });
    expect(await screen.findByText('chengdu:列表')).toBeInTheDocument();
    await waitFor(() =>
      expect(requests).toEqual(['beijing:old-record', 'chengdu:list']),
    );
    expect(activeRouter.state.location.pathname).toBe('/orders/sea-export');
    expect(activeRouter.state.location.search).toBe('');
    expect(activeRouter.state.location.hash).toBe('');
    expect(cacheClear).toHaveBeenCalledOnce();
  });

  it('同路径同子页签切换也提交新location，重建列表并清理旧输入', async () => {
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    const activeRouter = mount('/admin?tab=roles', [
      { path: '/admin', Component: () => <QueryPage record={false} /> },
    ]);
    const previousKey = activeRouter.state.location.key;
    fireEvent.change(screen.getByLabelText('当前输入'), {
      target: { value: '北京尚未保存' },
    });
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    expect(await screen.findByText('chengdu:列表')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(
      activeRouter.state.location.pathname + activeRouter.state.location.search,
    ).toBe('/admin?tab=roles');
    expect(activeRouter.state.location.key).not.toBe(previousKey);
    expect(screen.getByLabelText('当前输入')).toHaveValue('原输入');
    expect(requests).toEqual(['beijing:list', 'chengdu:list']);
  });

  it('切换API失败时原页面和用户输入保持，工作区缓存不清理', async () => {
    switchOrganization.mockRejectedValue(new Error('network'));
    mount('/admin?tab=roles', [
      { path: '/admin', Component: () => <QueryPage record={false} /> },
    ]);
    fireEvent.change(screen.getByLabelText('当前输入'), {
      target: { value: '北京尚未保存' },
    });
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    await waitFor(() => expect(switchOrganization).toHaveBeenCalledOnce());
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(screen.getByText('beijing:列表')).toBeInTheDocument();
    expect(screen.getByLabelText('当前输入')).toHaveValue('北京尚未保存');
    expect(cacheClear).not.toHaveBeenCalled();
    expect(requests).toEqual(['beijing:list']);
  });

  it('目标懒加载报错后展示新组织的路由错误，不重新挂旧记录或一直挂起', async () => {
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    mount('/orders/sea-export/old', [
      { path: '/orders/sea-export/:id', Component: () => <QueryPage record /> },
      {
        path: '/orders/sea-export',
        lazy: async () => {
          throw new Error('页面加载失败');
        },
        errorElement: <div>目标页面加载失败</div>,
      },
    ]);
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    await waitFor(() => expect(router?.state.navigation.state).toBe('idle'));
    expect(await screen.findByText('页面暂时无法打开')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(requests).toEqual(['beijing:old-record']);
    expect(screen.queryByText('chengdu:旧订单')).not.toBeInTheDocument();
  });
  it('目标仍在懒加载时另一导航取消它，转到安全工作台且不等待旧lazy结束', async () => {
    let releaseTarget:
      | ((value: { Component: () => React.ReactNode }) => void)
      | undefined;
    const lazyTarget = new Promise<{ Component: () => React.ReactNode }>(
      (resolve) => {
        releaseTarget = resolve;
      },
    );
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    const activeRouter = mount('/orders/sea-export/old', [
      { path: '/orders/sea-export/:id', Component: () => <QueryPage record /> },
      { path: '/orders/sea-export', lazy: () => lazyTarget },
      { path: '/interrupted', Component: () => <QueryPage record /> },
      { path: '/welcome', Component: () => <QueryPage record={false} /> },
    ]);
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    await waitFor(() =>
      expect(activeRouter.state.navigation.state).toBe('loading'),
    );
    await act(async () => {
      await activeRouter.navigate('/interrupted');
    });
    expect(await screen.findByText('chengdu:列表')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(activeRouter.state.location.pathname).toBe('/welcome');
    expect(requests).toEqual(['beijing:old-record', 'chengdu:list']);
    await act(async () => {
      releaseTarget?.({ Component: () => <QueryPage record={false} /> });
      await lazyTarget;
    });
    expect(activeRouter.state.location.pathname).toBe('/welcome');
    expect(requests).toEqual(['beijing:old-record', 'chengdu:list']);
  });

  it('导航器连续拒绝时结束切换并展示可重新加载的错误页，保留新身份', async () => {
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    mount('/orders/sea-export/old', [
      { path: '/orders/sea-export/:id', Component: () => <QueryPage record /> },
    ]);
    const navigation = vi
      .spyOn(history, 'replace')
      .mockRejectedValue(new Error('导航器拒绝'));
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    expect(await screen.findByText('页面暂时无法打开')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(navigation).toHaveBeenCalledTimes(2);
    expect(
      screen.getByRole('button', { name: '切换当前组织' }),
    ).toHaveTextContent('成都公司');
    expect(requests).toEqual(['beijing:old-record']);
    expect(screen.getByRole('link', { name: '重新加载' })).toBeInTheDocument();
  });

  it('浏览器后退取消懒加载时恢复安全工作台，迟到目标不重新挂载', async () => {
    let ready:
      | ((value: { Component: () => React.ReactNode }) => void)
      | undefined;
    const lazyTarget = new Promise<{ Component: () => React.ReactNode }>(
      (resolve) => {
        ready = resolve;
      },
    );
    switchOrganization.mockResolvedValue({ data: user('chengdu') });
    router = createMemoryRouter(
      [
        {
          Component: AppLayout,
          children: [
            { path: '/welcome', Component: () => <QueryPage record={false} /> },
            {
              path: '/orders/sea-export/:id',
              Component: () => <QueryPage record />,
            },
            { path: '/orders/sea-export', lazy: () => lazyTarget },
          ],
        },
      ],
      {
        initialEntries: ['/welcome', '/orders/sea-export/old'],
        initialIndex: 1,
      },
    );
    const activeRouter = router;
    bindRouter(activeRouter);
    render(
      <Provider>
        <RouterProvider router={activeRouter} />
      </Provider>,
    );
    await waitFor(() => expect(requests).toEqual(['beijing:old-record']));
    fireEvent.click(screen.getByRole('button', { name: '切成都' }));
    await waitFor(() =>
      expect(activeRouter.state.navigation.state).toBe('loading'),
    );
    await act(async () => {
      await activeRouter.navigate(-1);
    });
    expect(await screen.findByText('chengdu:列表')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '切换当前组织' }),
      ).toBeEnabled(),
    );
    expect(activeRouter.state.location.pathname).toBe('/welcome');
    expect(requests).toEqual(['beijing:old-record', 'chengdu:list']);
    await act(async () => {
      ready?.({ Component: () => <QueryPage record={false} /> });
      await lazyTarget;
    });
    expect(activeRouter.state.location.pathname).toBe('/welcome');
    expect(requests).toEqual(['beijing:old-record', 'chengdu:list']);
  });
});
