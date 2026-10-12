import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { useRef } from 'react';
import {
  createMemoryRouter,
  Link,
  Outlet,
  RouterProvider,
  useLocation,
} from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  clearTabDrafts,
  getFormDraftKey,
  getFormDraftScope,
} from './formDraft';
import {
  _clearAllTabCloseGuards,
  confirmIfTabsDirty,
  useTabCloseGuard,
} from './tabCloseGuard';
import { useRouteLeaveGuard } from './useRouteLeaveGuard';

function Editor() {
  const dirty = useRef(false);
  const location = useLocation();
  useTabCloseGuard({
    tabKey: '/partners/customers',
    isDirty: () => dirty.current,
  });
  useRouteLeaveGuard('/partners/customers', () => dirty.current);
  return (
    <>
      <input
        aria-label="未保存内容"
        onChange={() => {
          dirty.current = true;
        }}
      />
      <span>{location.pathname}</span>
      <Link to="/partners/customers/B">另一个客商</Link>
      <Link to="/welcome">工作台</Link>
    </>
  );
}
function mount() {
  const router = createMemoryRouter(
    [
      {
        element: (
          <ConfigProvider theme={{ token: { motion: false } }}>
            <App>
              <Outlet />
            </App>
          </ConfigProvider>
        ),
        children: [
          { path: '/partners/customers/:id', element: <Editor /> },
          { path: '/welcome', element: <div>工作台页面</div> },
        ],
      },
    ],
    {
      initialEntries: [
        '/welcome',
        '/partners/customers/A',
        '/partners/customers/B',
      ],
      initialIndex: 1,
    },
  );
  render(<RouterProvider router={router} />);
  return router;
}
afterEach(() => {
  _clearAllTabCloseGuards();
  sessionStorage.clear();
});

describe('客商真实路由离开保护', () => {
  it('批量关闭首个为后台草稿时，当前客商复用同一次确认', async () => {
    const router = mount();
    const draftScope = getFormDraftScope('user', 'org');
    const backgroundTab = '/orders/sea-export';
    const draftKey = getFormDraftKey(
      backgroundTab,
      '/orders/sea-export/new',
      draftScope,
    );
    sessionStorage.setItem(draftKey, JSON.stringify({ note: '后台草稿' }));
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '客商编辑值' },
    });
    const modal = vi.fn();
    confirmIfTabsDirty(
      [backgroundTab, '/partners/customers'],
      () => {
        clearTabDrafts(backgroundTab, draftScope);
        void router.navigate('/welcome');
      },
      modal,
      draftScope,
    );
    expect(modal).toHaveBeenCalledTimes(1);
    expect(sessionStorage.getItem(draftKey)).not.toBeNull();
    await act(async () => {
      modal.mock.calls[0][0].onOk();
    });
    await screen.findByText('工作台页面');
    expect(sessionStorage.getItem(draftKey)).toBeNull();
    expect(screen.queryByText('确定离开')).not.toBeInTheDocument();
  });

  it('同菜单身份切换取消时保留输入，确认后才导航', async () => {
    const router = mount();
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '编辑值' },
    });
    fireEvent.click(screen.getByText('另一个客商'));
    expect(await screen.findByText('确定离开')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }));
    await waitFor(() =>
      expect(screen.queryByText('确定离开')).not.toBeInTheDocument(),
    );
    await waitFor(() => expect(router.state.blockers.size).toBe(1));
    expect(router.state.location.pathname).toBe('/partners/customers/A');
    expect(screen.getByLabelText('未保存内容')).toHaveValue('编辑值');
    fireEvent.click(screen.getByText('另一个客商'));
    fireEvent.click(await screen.findByText('确定离开'));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/partners/customers/B'),
    );
  });

  it('浏览器后退及前进均阻断，取消后仍留在编辑页', async () => {
    const router = mount();
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '修改' },
    });
    await act(async () => {
      await router.navigate(-1);
    });
    fireEvent.click(await screen.findByRole('button', { name: /取\s*消/ }));
    await waitFor(() =>
      expect(screen.queryByText('确定离开')).not.toBeInTheDocument(),
    );
    expect(router.state.location.pathname).toBe('/partners/customers/A');
    await act(async () => {
      await router.navigate(1);
    });
    fireEvent.click(await screen.findByRole('button', { name: /取\s*消/ }));
    await waitFor(() =>
      expect(screen.queryByText('确定离开')).not.toBeInTheDocument(),
    );
    expect(router.state.location.pathname).toBe('/partners/customers/A');
    await act(async () => {
      await router.navigate(-1);
    });
    fireEvent.click(await screen.findByText('确定离开'));
    await screen.findByText('工作台页面');
    await act(async () => {
      await router.navigate(1);
    });
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '新修改' },
    });
    await act(async () => {
      await router.navigate(-1);
    });
    fireEvent.click(await screen.findByRole('button', { name: /取\s*消/ }));
    await waitFor(() =>
      expect(screen.queryByText('确定离开')).not.toBeInTheDocument(),
    );
    expect(router.state.location.pathname).toBe('/partners/customers/A');
    await act(async () => {
      await router.navigate('/partners/customers/B', { replace: true });
    });
    fireEvent.click(await screen.findByText('确定离开'));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/partners/customers/B'),
    );
  });

  it('页签关闭已有确认后路由直接放行，异步动作失败后保护仍生效', async () => {
    const router = mount();
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '修改' },
    });
    const modal = vi.fn();
    confirmIfTabsDirty(
      ['/partners/customers'],
      () => {
        void router.navigate('/welcome');
      },
      modal,
    );
    await act(async () => {
      modal.mock.calls[0][0].onOk();
    });
    await screen.findByText('工作台页面');
    expect(screen.queryByText('确定离开')).not.toBeInTheDocument();
    await act(async () => {
      await router.navigate(-1);
    });
    fireEvent.change(screen.getByLabelText('未保存内容'), {
      target: { value: '再修改' },
    });
    confirmIfTabsDirty(
      ['/partners/customers'],
      async () => {
        throw new Error('模拟切换失败');
      },
      modal,
    );
    await expect(modal.mock.calls[1][0].onOk()).rejects.toThrow('模拟切换失败');
    fireEvent.click(screen.getByText('工作台'));
    expect(await screen.findByText('确定离开')).toBeInTheDocument();
  });
});
