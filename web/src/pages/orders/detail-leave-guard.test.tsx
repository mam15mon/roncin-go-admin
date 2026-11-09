import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { Link, useLocation } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getFormDraftKey,
  hasFormDraft,
  saveFormDraft,
} from '@/components/layout/formDraft';
import { resolveTabKey } from '@/components/layout/routeUtils';
import {
  _clearAllTabCloseGuards,
  confirmIfTabsDirty,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { bindRouter } from '@/router/history';
import {
  getOrderUpdateMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
  save,
} from './detail-port-save.fixtures';
import type { OrderDetailFeaturesProps } from './order-kinds/types';

vi.mock('react-router', async (original) => {
  const actual = await original<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => {
      const parts = actual.useLocation().pathname.split('/');
      return { kind: parts[2], id: parts[3] };
    },
  };
});

vi.mock('./order-kinds/air-export/AirExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./templates/air-template', async () => {
  const { Form, Input } = await import('antd');
  return {
    getAirTemplateSections: () => [
      {
        key: 'basic',
        title: '空运订单资料',
        content: (
          <Form.Item name="customerReferenceNo" label="客户业务编号">
            <Input />
          </Form.Item>
        ),
      },
    ],
  };
});

let OrderDetailPage: typeof import('./detail')['default'];
const state = getPortSaveState();
function RoutePages() {
  const location = useLocation();
  return location.pathname === '/welcome' ? (
    <>
      <div>工作台页面</div>
      <Link to={`/orders/${state.routeKind}/${state.routeId}`}>
        重新打开订单
      </Link>
    </>
  ) : (
    <>
      <Link to="/welcome">前往工作台</Link>
      <OrderDetailPage />
    </>
  );
}

function mount() {
  const view = renderDetail(<RoutePages />);
  if (!view.router) throw new Error('测试必须使用真实数据路由');
  bindRouter(view.router);
  return { ...view, router: view.router };
}
function editReference() {
  fireEvent.change(screen.getByLabelText('客户业务编号'), {
    target: { value: '未保存业务编号' },
  });
}
beforeEach(async () => {
  OrderDetailPage = (await import('./detail')).default;
  resetPortSaveState();
  _clearAllTabCloseGuards();
  state.order.customerReferenceNo = '服务端业务编号';
});
afterEach(() => _clearAllTabCloseGuards());

describe('订单详情真实路由离开保护', () => {
  it.each(['sea-export', 'air-export'])(
    '%s 切换前确认，取消保留输入，确认后返回不恢复草稿',
    async (kind) => {
      state.routeKind = kind;
      const path = `/orders/${kind}/${state.routeId}`;
      const key = getFormDraftKey(resolveTabKey(path), path, 'user-1:org-1');
      saveFormDraft(key, {
        version: '1',
        customerReferenceNo: '退役详情草稿',
        containerRequests: null,
      });
      const { router } = mount();
      expect(screen.getByLabelText('客户业务编号')).toHaveValue(
        '服务端业务编号',
      );
      expect(isTabDirty(resolveTabKey(path), 'user-1:org-1')).toBe(false);
      editReference();
      fireEvent.click(screen.getByText('前往工作台'));
      const dialog = await screen.findByRole('dialog');
      expect(dialog).toHaveTextContent('有未保存的修改，是否放弃并离开？');
      fireEvent.click(within(dialog).getByRole('button', { name: '继续编辑' }));
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
      expect(router.state.location.pathname).toBe(path);
      expect(screen.getByLabelText('客户业务编号')).toHaveValue(
        '未保存业务编号',
      );
      expect(state.form?.getFieldValue('version')).toBe(
        kind === 'sea-export' ? '4' : undefined,
      );
      fireEvent.click(screen.getByText('前往工作台'));
      fireEvent.click(
        await screen.findByRole('button', { name: '放弃并离开' }),
      );
      await screen.findByText('工作台页面');
      fireEvent.click(screen.getByText('重新打开订单'));
      expect(await screen.findByLabelText('客户业务编号')).toHaveValue(
        '服务端业务编号',
      );
      // 遗留记录既不读取也不更新，更不会转换成新的恢复会话。
      expect(
        JSON.parse(sessionStorage.getItem(key) ?? '{}').customerReferenceNo,
      ).toBe('退役详情草稿');
    },
  );

  it('修改后立即关闭复用同一次确认，取消不导航，确认才关闭', async () => {
    const { router } = mount();
    editReference();
    const modal = vi.fn();
    const close = () =>
      confirmIfTabsDirty(
        ['/orders/sea-export'],
        () => router.navigate('/welcome'),
        modal,
      );
    close();
    expect(modal).toHaveBeenCalledOnce();
    expect(modal.mock.calls[0][0].content).toBe(
      '有未保存的修改，是否放弃并关闭？',
    );
    modal.mock.calls[0][0].onCancel?.();
    expect(router.state.location.pathname).toBe('/orders/sea-export/order-1');
    expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存业务编号');
    close();
    await act(async () => modal.mock.calls[1][0].onOk());
    await screen.findByText('工作台页面');
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('保存失败仍保护输入，保存成功后离开无需确认且不产生持久草稿', async () => {
    const { router } = mount();
    editReference();
    getOrderUpdateMock().mockRejectedValueOnce({
      response: { status: 409, data: { message: '已被更新' } },
    });
    await save();
    expect(isTabDirty('/orders/sea-export')).toBe(true);
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(screen.getByLabelText('客户业务编号')).toHaveValue('未保存业务编号');
    fireEvent.click(screen.getByText('前往工作台'));
    fireEvent.click(await screen.findByRole('button', { name: '继续编辑' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    await save();
    expect(isTabDirty('/orders/sea-export')).toBe(false);
    expect(
      hasFormDraft(
        getFormDraftKey(
          '/orders/sea-export',
          '/orders/sea-export/order-1',
          'user-1:org-1',
        ),
      ),
    ).toBe(false);
    fireEvent.click(screen.getByText('前往工作台'));
    await screen.findByText('工作台页面');
    expect(router.state.location.pathname).toBe('/welcome');
    expect(screen.queryByRole('dialog')).toBeNull();
  });
});
