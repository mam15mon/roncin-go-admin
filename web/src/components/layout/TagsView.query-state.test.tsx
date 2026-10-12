import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from '@testing-library/react';
import type { Modal } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { setAppFeedback } from '@/utils/appFeedback';
import { TagsView } from './TagsView';
import {
  _clearAllTabCloseGuards,
  registerTabCloseGuard,
} from './tabCloseGuard';

const clearQueries = vi.fn();
let pathname = '/welcome';
const confirm = vi.fn();
vi.mock('@/components/ui/list-query-state', () => ({
  useClearListQueries: () => clearQueries,
}));
vi.mock('@/router/history', () => ({ history: { push: vi.fn() } }));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useLocation: () => ({ pathname, search: '', hash: '' }),
}));
// 菜单采用同步渲染；回归聚焦真实页签操作与dirty确认，不依赖弹层动画。
vi.mock('antd', async (original) => ({
  ...(await original<typeof import('antd')>()),
  Dropdown: ({
    children,
    menu,
  }: {
    children: React.ReactNode;
    menu: {
      items: Array<{
        key?: string;
        label?: React.ReactNode;
        disabled?: boolean;
        onClick?: () => void;
      }>;
    };
  }) => (
    <section>
      {children}
      <div>
        {menu.items
          .filter((item) => item?.label)
          .map((item) => (
            <button
              type="button"
              key={item.key}
              disabled={item.disabled}
              onClick={item.onClick}
            >
              {item.label}
            </button>
          ))}
      </div>
    </section>
  ),
}));
function visitTabs() {
  const view = render(<TagsView />);
  for (const path of [
    '/partners/customers',
    '/partners/suppliers',
    '/finance/fees',
  ]) {
    pathname = path;
    view.rerender(<TagsView />);
  }
  return view;
}
function menu(title: string, action: string) {
  const section = screen.getByText(title).closest('section');
  if (!section) throw new Error('页签菜单未找到');
  fireEvent.click(within(section).getByRole('button', { name: action }));
}
beforeEach(() => {
  pathname = '/welcome';
  vi.clearAllMocks();
  _clearAllTabCloseGuards();
  confirm.mockReturnValue({ destroy: vi.fn(), update: vi.fn() });
  setAppFeedback({ modal: { confirm } } as unknown as Parameters<
    typeof setAppFeedback
  >[0]);
});
afterEach(() => {
  cleanup();
  _clearAllTabCloseGuards();
});
describe('页签关闭查询清理', () => {
  it.each([
    ['客户', '关闭标签页', ['/partners/customers']],
    ['客户', '关闭其他标签页', ['/partners/suppliers', '/finance/fees']],
    ['客户', '关闭右侧标签页', ['/partners/suppliers', '/finance/fees']],
    [
      '集运费用明细',
      '关闭左侧标签页',
      ['/partners/customers', '/partners/suppliers'],
    ],
    [
      '工作台',
      '关闭其他标签页',
      ['/partners/customers', '/partners/suppliers', '/finance/fees'],
    ],
  ])('%s执行%s仅清理实际关闭页签', (title, action, expected) => {
    visitTabs();
    menu(title, action);
    expect(clearQueries).toHaveBeenCalledExactlyOnceWith(expected);
  });
  it('未确认dirty关闭保留查询，确认后才清理；普通切页签不清理', async () => {
    visitTabs();
    registerTabCloseGuard('/partners/customers', {
      isDirty: () => true,
      message: '尚未保存',
    });
    fireEvent.click(screen.getByText('客户'));
    expect(clearQueries).not.toHaveBeenCalled();
    fireEvent.click(screen.getByLabelText('关闭 客户'));
    expect(confirm).toHaveBeenCalledOnce();
    expect(clearQueries).not.toHaveBeenCalled();
    const props = confirm.mock.calls[0][0] as Parameters<
      typeof Modal.confirm
    >[0];
    await act(async () => {
      await props.onCancel?.();
    });
    expect(clearQueries).not.toHaveBeenCalled();
    expect(screen.getByText('客户')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('关闭 客户'));
    expect(confirm).toHaveBeenCalledTimes(2);
    const confirmedProps = confirm.mock.calls[1][0] as Parameters<
      typeof Modal.confirm
    >[0];
    await act(async () => {
      await confirmedProps.onOk?.();
    });
    expect(clearQueries).toHaveBeenCalledExactlyOnceWith([
      '/partners/customers',
    ]);
  });
});
