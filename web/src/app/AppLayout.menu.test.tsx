import type { ProLayoutProps } from '@ant-design/pro-components';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { MemoryRouter } from 'react-router';
import { afterEach, expect, it, vi } from 'vitest';
import access from '@/access';
import { AppLayout } from './AppLayout';

const state = vi.hoisted(() => ({
  md: false,
  platform: false,
  permissions: [
    'business.partner.read',
    'business.enterprise_resource.read',
    'system.user.update',
  ],
}));
const closeDrawer = vi.fn();
let captured: ProLayoutProps;
vi.mock('antd', async (original) => ({
  ...(await original<typeof import('antd')>()),
  Grid: { useBreakpoint: () => ({ md: state.md }) },
}));
vi.mock('./AppProvider', () => ({
  LOGIN_PATH: '/user/login',
  PUBLIC_AUTH_PATHS: new Set(['/user/login']),
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user', currentOrganization: { id: 'org' } },
    },
  }),
}));
vi.mock('./access', () => ({
  useAccess: () =>
    access({
      currentUser: {
        id: 'u',
        currentOrganization: { id: 'o', kind: state.platform ? 1 : 2 },
        permissionCapabilities: state.permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
        roleScopes: [{ roleCode: 'r' }],
      },
    }),
}));
vi.mock('@/router/adaptRoutes', () => ({
  buildMenuData: () => ({ path: '/', routes: [] }),
}));
vi.mock('@/router/history', () => ({ history: { replace: vi.fn() } }));
vi.mock('@ant-design/pro-components', () => ({
  ProLayout: (props: ProLayoutProps) => {
    captured = props;
    const groups = props.menuDataRender?.([]) ?? [];
    return (
      <div>
        {groups.map((group) => (
          <div key={group.key}>
            {group.name}
            {group.children?.map((item) => (
              <React.Fragment key={item.key}>
                {typeof props.menuItemRender === 'function'
                  ? props.menuItemRender(
                      { ...item, isUrl: false, onClick: closeDrawer },
                      <span>{item.name}</span>,
                      {},
                    )
                  : null}
              </React.Fragment>
            ))}
          </div>
        ))}
      </div>
    );
  },
}));
afterEach(() => {
  cleanup();
  closeDrawer.mockClear();
  state.md = false;
  state.platform = false;
  state.permissions = [
    'business.partner.read',
    'business.enterprise_resource.read',
    'system.user.update',
  ];
});
it('移动Drawer共享权限入口保留query并调用官方关闭回调；设置query精准高亮', () => {
  render(
    <MemoryRouter initialEntries={['/admin?tab=users']}>
      <AppLayout />
    </MemoryRouter>,
  );
  expect(screen.getByRole('link', { name: '供应商' })).toHaveAttribute(
    'href',
    '/partners/suppliers',
  );
  expect(screen.getByRole('link', { name: '国外代理' })).toHaveAttribute(
    'href',
    '/partners/foreign-agents',
  );
  const users = screen.getByRole('link', { name: '用户人员' });
  expect(users).toHaveAttribute('href', '/admin?tab=users');
  expect(captured.menuProps?.selectedKeys).toEqual(['/admin?tab=users']);
  fireEvent.click(users);
  expect(closeDrawer).toHaveBeenCalledOnce();
});
it('md桌面不追加移动组，无权限移动不输出空组', () => {
  state.md = true;
  const view = render(
    <MemoryRouter>
      <AppLayout />
    </MemoryRouter>,
  );
  expect(captured.menuDataRender?.([])).toEqual([]);
  view.unmount();
  state.md = false;
  state.permissions = [];
  render(
    <MemoryRouter>
      <AppLayout />
    </MemoryRouter>,
  );
  expect(captured.menuDataRender?.([])).toEqual([]);
});
