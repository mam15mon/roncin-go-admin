import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { expect, it, vi } from 'vitest';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import OrderFeesPage from './fees';

const state = vi.hoisted(() => ({ pending: true }));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'u1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
    canReviewOrderFees: true,
    canReadFinanceFees: false,
    canCreateFinanceBills: true,
  }),
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useParams: () => ({ kind: 'sea-export', id: 'o1' }),
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  FinanceSummaryBoard: () => null,
}));
vi.mock('./components/OrderPageHeader', () => ({
  default: () => <div>费用工作台</div>,
}));
vi.mock('./components/fees/OrderFeeHeader', () => ({ default: () => null }));
vi.mock('./components/fees/OrderFeeTableTabs', () => ({ default: () => null }));
vi.mock('./components/fees/FeeFormModal', () => ({ default: () => null }));
vi.mock('./components/fees/QuickAddFeeModal', () => ({ default: () => null }));
vi.mock('./components/fees/QuickAddPartnerModal', () => ({
  default: () => null,
}));
vi.mock('./components/fees/fee-review/OrderFeeReview', () => ({
  OrderFeeReviewSection: () => null,
}));
vi.mock('./use-order-lock-state', async (original) => ({
  ...(await original<typeof import('./use-order-lock-state')>()),
  useOrderLockState: () => ({
    state: { isLocked: true },
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));
vi.mock('@/services/roncin/orderFeeService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/orderFeeService')>()),
  orderFeeServiceListFeeOptions: vi.fn().mockResolvedValue({
    currencies: [{ code: 'CNY', name: '人民币' }],
    settlementParties: [{ id: 'p1', name: '供应商' }],
    feeSettings: [{ id: 'f1', feeCode: 'TRUCK', nameZh: '拖车费' }],
    billingUnits: [{ id: 'b1', name: '票' }],
  }),
  orderFeeServicePreviewOrderFeeReview: vi
    .fn()
    .mockImplementation(async () => ({
      data: {
        orderId: 'o1',
        status: 'APPROVED',
        hasPendingSupplement: state.pending,
        canBill: !state.pending,
      },
    })),
  orderFeeServiceListOrderFeeSupplementRequests: vi
    .fn()
    .mockImplementation(async () => ({
      success: true,
      data: {
        items: [
          {
            id: 's1',
            orderId: 'o1',
            version: '1',
            status: state.pending ? 'PENDING' : 'WITHDRAWN',
            canWithdraw: state.pending,
            feeSettingId: 'f1',
            settlementPartyId: 'p1',
            billingUnitId: 'b1',
            currency: 'CNY',
            quantity: '2',
            unitPrice: '50',
            expenseDate: '2026-10-12',
            reason: '原补录原因',
          },
        ],
        total: 1,
      },
    })),
  orderFeeServiceWithdrawOrderFeeSupplement: vi
    .fn()
    .mockImplementation(async () => {
      state.pending = false;
      return { success: true, data: { id: 's1' } };
    }),
}));

it('真实费用页撤回重提后，后台重查订单期间及完成后保留预填与新输入', async () => {
  const order = {
    id: 'o1',
    orderNo: 'SE001',
    organizationId: 'org-1',
    businessType: 1,
    version: '1',
  };
  const getOrder = vi.mocked(orderServiceGetOrder);
  getOrder.mockResolvedValueOnce({ success: true, data: order });
  let resolveRefresh!: (value: API.GetOrderResponse) => void;
  getOrder.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
  );
  const { queryClient } = renderWithClient(
    <App>
      <OrderFeesPage />
    </App>,
  );
  // 先等待父层查询完成并冲刷挂载更新，避免并发门禁时把请求链耗时
  // 全算进表格元素的默认查找窗口；第二次订单请求仍专用于后台刷新。
  await act(async () => {
    await Promise.all([
      queryClient.ensureQueryData({
        queryKey: ['orders', 'fee-options', { orderId: 'o1' }],
      }),
      queryClient.ensureQueryData({
        queryKey: ['order-fee-review', 'org-1', 'o1'],
      }),
    ]);
  });
  // 真实 ProTable 在父查询之后异步调度列表，允许全量并发下的组件调度耗时。
  fireEvent.click(await screen.findByText('撤回并重提', {}, { timeout: 5000 }));
  fireEvent.click(screen.getByRole('button', { name: '撤回并重提' }));
  await waitFor(() => expect(getOrder).toHaveBeenCalledTimes(2));
  await waitFor(() =>
    expect(
      queryClient.isFetching({ queryKey: ['orders', 'fee-options'] }),
    ).toBe(1),
  );
  const prefill = await screen.findByText(
    '已预填撤回申请，请核对后作为新申请提交',
  );
  const dialog = prefill.closest<HTMLElement>('[role=dialog]');
  if (!dialog) throw new Error('预填内容未挂载在补录弹窗中');
  expect(dialog).toBeInTheDocument();
  const reason = within(dialog).getByLabelText('补录原因');
  expect(reason).toHaveValue('原补录原因');
  expect(
    within(dialog).getByText('已预填撤回申请，请核对后作为新申请提交'),
  ).toBeInTheDocument();
  fireEvent.change(reason, { target: { value: '后台刷新期间补充凭证说明' } });
  await act(async () => {
    resolveRefresh({ success: true, data: { ...order, version: '2' } });
  });
  await waitFor(() =>
    expect(
      queryClient.isFetching({ queryKey: ['orders', 'fee-options'] }),
    ).toBe(0),
  );
  expect(dialog).toBeInTheDocument();
  expect(reason).toHaveValue('后台刷新期间补充凭证说明');
  await act(async () => {});
});
