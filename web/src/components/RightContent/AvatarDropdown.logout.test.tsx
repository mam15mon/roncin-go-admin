import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { type ReactNode, useMemo, useRef, useState } from 'react';
import { createMemoryRouter, Outlet, RouterProvider } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useInitialState } from '@/app/AppProvider';
import {
  appInitialStateContext,
  type InitialState,
} from '@/app/initialStateContext';
import {
  _clearAllTabCloseGuards,
  getAnyDirtyTabKey,
  useTabCloseGuard,
} from '@/components/layout/tabCloseGuard';
import { useRouteLeaveGuard } from '@/components/layout/useRouteLeaveGuard';
import { bindRouter, history } from '@/router/history';
import { AvatarDropdown } from './AvatarDropdown';

const { logout, clearCache } = vi.hoisted(() => ({
  logout: vi.fn<() => Promise<API.LogoutResponse | undefined>>(),
  clearCache: vi.fn(),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceLogout: logout,
}));
vi.mock('@/features/orders/options', () => ({
  clearOrderMasterDataCache: clearCache,
}));
vi.mock('@/features/auth/passkeys', () => ({
  PasskeyManagerModal: () => null,
}));

function Provider({ children }: { children: ReactNode }) {
  const [initialState, setInitialState] = useState<InitialState | undefined>({
    currentUser: {
      id: 'user-a',
      displayName: '测试员工',
      currentOrganization: { id: 'company-a' },
    },
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
    <ConfigProvider theme={{ token: { motion: false } }}>
      <App>
        <appInitialStateContext.Provider value={value}>
          {children}
        </appInitialStateContext.Provider>
      </App>
    </ConfigProvider>
  );
}

function Shell() {
  const { initialState } = useInitialState();
  return (
    <>
      <output data-testid="current-user">
        {initialState?.currentUser?.id ?? 'anonymous'}
      </output>
      <AvatarDropdown />
      <Outlet />
    </>
  );
}

function Editor() {
  const dirty = useRef(false);
  useTabCloseGuard({
    tabKey: '/orders/sea-export',
    isDirty: () => dirty.current,
  });
  useRouteLeaveGuard('/orders/sea-export', () => dirty.current);
  return (
    <input
      aria-label="未保存订单输入"
      onChange={() => {
        dirty.current = true;
      }}
    />
  );
}

let router: ReturnType<typeof createMemoryRouter> | undefined;
function mountEditor() {
  router = createMemoryRouter(
    [
      {
        element: <Shell />,
        children: [
          { path: '/orders/sea-export/order-a', element: <Editor /> },
          { path: '/user/login', element: <div>登录页面</div> },
        ],
      },
    ],
    { initialEntries: ['/orders/sea-export/order-a'] },
  );
  bindRouter(router);
  const view = render(
    <Provider>
      <RouterProvider router={router} />
    </Provider>,
  );
  fireEvent.change(screen.getByLabelText('未保存订单输入'), {
    target: { value: '未保存编号' },
  });
  return view;
}

async function requestLogout() {
  const trigger = document.querySelector('.roncin-avatar-trigger');
  if (!trigger) throw new Error('缺少头像入口');
  fireEvent.mouseEnter(trigger);
  fireEvent.click(await screen.findByRole('button', { name: /退出登录/ }));
  return screen.findByRole('dialog');
}

beforeEach(() => {
  logout.mockReset().mockResolvedValue({ success: true });
  clearCache.mockReset();
});
afterEach(() => {
  cleanup();
  router?.dispose();
  _clearAllTabCloseGuards();
  vi.restoreAllMocks();
});

describe('主动退出登录的未保存保护', () => {
  it('取消退出不发请求、不清身份，保留输入和dirty', async () => {
    mountEditor();
    const dialog = await requestLogout();
    expect(dialog).toHaveTextContent('退出登录将放弃当前编辑');
    fireEvent.click(within(dialog).getByRole('button', { name: '继续编辑' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(logout).not.toHaveBeenCalled();
    expect(clearCache).not.toHaveBeenCalled();
    expect(screen.getByTestId('current-user')).toHaveTextContent('user-a');
    expect(screen.getByLabelText('未保存订单输入')).toHaveValue('未保存编号');
    expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  });

  it('确认后成功退出一次，复用已确认动作而不触发路由二次提示', async () => {
    mountEditor();
    const replace = vi.spyOn(history, 'replace');
    const dialog = await requestLogout();
    fireEvent.click(within(dialog).getByRole('button', { name: '放弃并离开' }));

    expect(await screen.findByText('登录页面')).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(logout).toHaveBeenCalledOnce();
    expect(clearCache).toHaveBeenCalledOnce();
    expect(replace).toHaveBeenCalledExactlyOnceWith('/user/login');
    expect(screen.getByTestId('current-user')).toHaveTextContent('anonymous');
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });

  it.each(['reject', 'undefined', 'success:false'] as const)(
    '退出接口返回 %s 时保持输入与身份，下一次退出仍需确认',
    async (failure) => {
      if (failure === 'reject') {
        logout.mockRejectedValueOnce(new Error('退出失败'));
      } else {
        logout.mockResolvedValueOnce(
          failure === 'undefined' ? undefined : { success: false },
        );
      }
      mountEditor();
      const dialog = await requestLogout();
      fireEvent.click(
        within(dialog).getByRole('button', { name: '放弃并离开' }),
      );

      await waitFor(() => expect(logout).toHaveBeenCalledOnce());
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
      expect(clearCache).not.toHaveBeenCalled();
      expect(screen.getByTestId('current-user')).toHaveTextContent('user-a');
      expect(router?.state.location.pathname).toBe(
        '/orders/sea-export/order-a',
      );
      expect(screen.getByLabelText('未保存订单输入')).toHaveValue('未保存编号');
      expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');

      const retryDialog = await requestLogout();
      expect(retryDialog).toHaveTextContent('退出登录将放弃当前编辑');
      expect(logout).toHaveBeenCalledOnce();
      fireEvent.click(
        within(retryDialog).getByRole('button', { name: '继续编辑' }),
      );
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    },
  );
});
