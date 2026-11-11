import { ProFormText } from '@ant-design/pro-components';
import { renderWithDataRouter } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import type { ReactNode } from 'react';
import { useLocation } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import { bindRouter } from '@/router/history';
import { setAppFeedback } from '@/utils/appFeedback';
import { getFormDraftKey, hasFormDraft, saveFormDraft } from './formDraft';
import { TagsView } from './TagsView';
import { _clearAllTabCloseGuards, isTabDirty } from './tabCloseGuard';
import { useRouteLeaveGuard } from './useRouteLeaveGuard';

const clearQueries = vi.fn();
vi.mock('@/components/ui/list-query-state', () => ({
  useClearListQueries: () => clearQueries,
}));
vi.mock('antd', async (original) => ({
  ...(await original<typeof import('antd')>()),
  Dropdown: ({
    children,
    menu,
  }: {
    children: ReactNode;
    menu: {
      items?: Array<{
        key?: string;
        label?: ReactNode;
        disabled?: boolean;
        onClick?: () => void;
      }>;
    };
  }) => (
    <section>
      {children}
      <div>
        {menu.items?.map(
          (item) =>
            item?.label && (
              <button
                type="button"
                key={item.key}
                disabled={item.disabled}
                onClick={item.onClick}
              >
                {item.label}
              </button>
            ),
        )}
      </div>
    </section>
  ),
}));

function Editor() {
  useRouteLeaveGuard('/orders/sea-export/order-1', () =>
    isTabDirty('/orders/sea-export/order-1'),
  );
  return (
    <OrderFormTemplate
      tabKey="/orders/sea-export/order-1"
      initialValues={{ customerReferenceNo: '原编号' }}
      sections={[
        {
          key: 'reference',
          title: '订单资料',
          content: (
            <ProFormText name="customerReferenceNo" label="客户业务编号" />
          ),
        },
      ]}
      submitter={false}
    />
  );
}
function Workspace() {
  const app = App.useApp();
  setAppFeedback(app);
  const location = useLocation();
  return (
    <>
      <TagsView />
      {location.pathname.startsWith('/orders/sea-export/') ? (
        <Editor />
      ) : (
        <div>工作台及列表</div>
      )}
    </>
  );
}
afterEach(() => {
  sessionStorage.clear();
  _clearAllTabCloseGuards();
  vi.clearAllMocks();
});

describe('关闭全部标签页的真实路由确认', () => {
  it('多个待关闭页签只确认一次，取消保留全部，确认才清理并离开', async () => {
    const view = renderWithDataRouter(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <Workspace />
        </App>
      </ConfigProvider>,
      {
        user: { id: 'user', currentOrganization: { id: 'company' } },
        routerProps: { initialEntries: ['/welcome'] },
      },
    );
    const router = view.router;
    if (!router) throw new Error('需要真实数据路由');
    bindRouter(router);
    await act(async () => router.navigate('/orders/air-export/new'));
    const newDraft = getFormDraftKey(
      '/orders/air-export/new',
      '/orders/air-export/new',
      'user:company',
    );
    saveFormDraft(newDraft, { customerReferenceNo: '后台新建草稿' });
    await act(async () => router.navigate('/orders/sea-export/order-1'));
    fireEvent.change(screen.getByLabelText('客户业务编号'), {
      target: { value: '未保存编号' },
    });
    const workbenchMenu = screen.getByText('工作台').closest('section');
    if (!workbenchMenu) throw new Error('缺少工作台菜单');
    const closeAll = within(workbenchMenu).getByRole('button', {
      name: '关闭全部标签页',
    });
    fireEvent.click(closeAll);
    const dialog = await screen.findByRole('dialog');
    expect(screen.getAllByRole('dialog')).toHaveLength(1);
    fireEvent.click(within(dialog).getByRole('button', { name: '继续编辑' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.getAllByRole('tab')).toHaveLength(3);
    expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存编号');
    expect(hasFormDraft(newDraft)).toBe(true);
    expect(clearQueries).not.toHaveBeenCalled();

    fireEvent.click(closeAll);
    fireEvent.click(await screen.findByRole('button', { name: '放弃并关闭' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/welcome'),
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(screen.getAllByRole('tab')).toHaveLength(1);
    expect(hasFormDraft(newDraft)).toBe(false);
    expect(clearQueries).toHaveBeenCalledExactlyOnceWith([
      '/orders/air-export/new',
      '/orders/sea-export/order-1',
    ]);
    expect(screen.queryByText('有未保存的修改，是否放弃并离开？')).toBeNull();
  });
});
