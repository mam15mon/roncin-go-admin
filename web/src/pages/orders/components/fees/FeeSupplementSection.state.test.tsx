import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { useQuery } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { useOrderFeeReview } from '@/features/orders/fee-review';
import {
  orderFeeServiceCreateOrderFeeSupplement,
  orderFeeServiceRejectOrderFeeSupplement,
  orderFeeServiceWithdrawOrderFeeSupplement,
} from '@/services/roncin/orderFeeService';
import type { FeeSupplementFormValues } from './FeeSupplementModal';
import FeeSupplementSection from './FeeSupplementSection';

const state = vi.hoisted(() => ({ pending: false, created: false }));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
const request = () => ({
  id: 's1',
  orderId: 'o1',
  version: '1',
  status: state.pending ? 'PENDING' : 'WITHDRAWN',
  canWithdraw: state.pending,
  canApprove: state.pending,
  approverAvailable: true,
});
vi.mock('@/services/roncin/orderFeeService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/orderFeeService')>()),
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
  orderFeeServiceCreateOrderFeeSupplement: vi
    .fn()
    .mockImplementation(async () => {
      state.pending = true;
      state.created = true;
      return { success: true, data: request() };
    }),
  orderFeeServiceWithdrawOrderFeeSupplement: vi
    .fn()
    .mockImplementation(async () => {
      state.pending = false;
      return { success: true, data: request() };
    }),
  orderFeeServiceApproveOrderFeeSupplement: vi
    .fn()
    .mockImplementation(async () => {
      state.pending = false;
      return { success: true, data: request() };
    }),
  orderFeeServiceRejectOrderFeeSupplement: vi
    .fn()
    .mockImplementation(async () => {
      state.pending = false;
      return { success: true, data: request() };
    }),
  orderFeeServicePreviewOrderFeeSupplementApproval: vi.fn().mockResolvedValue({
    data: {
      previewToken: 'supplement-token',
      baseCurrency: 'CNY',
      currentProfit: '100',
      projectedProfit: '50',
      currentProfitRate: '50',
      projectedProfitRate: '25',
    },
  }),
  orderFeeServiceListOrderFeeSupplementRequests: vi
    .fn()
    .mockImplementation(async () => ({
      success: true,
      data: {
        items: state.created ? [request()] : [],
        total: state.created ? 1 : 0,
      },
    })),
}));
vi.mock('./FeeSupplementModal', () => ({
  default: ({
    open,
    onSubmit,
    initialRequest,
  }: {
    open: boolean;
    onSubmit: (values: FeeSupplementFormValues) => Promise<boolean>;
    initialRequest?: API.OrderFeeSupplementRequestData;
  }) =>
    open ? (
      <div data-testid="draft" data-prefill-id={initialRequest?.id}>
        <button
          type="button"
          onClick={() =>
            onSubmit({
              feeSettingId: 'f1',
              settlementPartyId: 'p1',
              billingUnitId: 'b1',
              quantity: '1',
              unitPrice: '50',
              currency: 'CNY',
              expenseDate: '2026-10-12',
              reason: '补录',
            })
          }
        >
          测试提交
        </button>
      </div>
    ) : null,
}));
function Harness() {
  const review = useOrderFeeReview('o1');
  const order = useQuery({
    queryKey: ['order-detail', 'o1'],
    queryFn: async () => ({
      closureStatus: 'CLOSED',
      hasPendingSupplement: state.pending,
    }),
  });
  return (
    <>
      <output data-testid="order-pending">
        {String(order.data?.hasPendingSupplement)}
      </output>
      <output data-testid="closure">{order.data?.closureStatus}</output>
      <output data-testid="bill">{String(review.data?.canBill)}</output>
      <output data-testid="pending">
        {String(review.data?.hasPendingSupplement)}
      </output>
      <FeeSupplementSection
        orderId="o1"
        canCreate
        feeReviewApproved={review.data?.status === 'APPROVED'}
        hasPendingSupplement={Boolean(review.data?.hasPendingSupplement)}
        feeSettings={[]}
        settlementParties={[]}
        currencies={[]}
        billingUnits={[]}
        onFeeTablesReload={vi.fn()}
      />
    </>
  );
}
beforeEach(() => {
  vi.clearAllMocks();
  state.pending = false;
  state.created = false;
});

it('创建立即阻止建账，待审保留撤回重提，撤回成功预填并恢复建账状态', async () => {
  renderWithClient(
    <App>
      <Harness />
    </App>,
  );
  await waitFor(() =>
    expect(screen.getByTestId('bill')).toHaveTextContent('true'),
  );
  fireEvent.click(screen.getByRole('button', { name: '补录费用' }));
  fireEvent.click(screen.getByRole('button', { name: '测试提交' }));
  await waitFor(() =>
    expect(screen.getByTestId('pending')).toHaveTextContent('true'),
  );
  expect(screen.getByTestId('bill')).toHaveTextContent('false');
  expect(screen.getByTestId('order-pending')).toHaveTextContent('true');
  expect(screen.getByTestId('closure')).toHaveTextContent('CLOSED');
  expect(
    screen.queryByRole('button', { name: '补录费用' }),
  ).not.toBeInTheDocument();
  fireEvent.click(await screen.findByText('撤回并重提'));
  fireEvent.click(screen.getByRole('button', { name: '撤回并重提' }));
  await waitFor(() =>
    expect(screen.getByTestId('pending')).toHaveTextContent('false'),
  );
  expect(screen.getByTestId('bill')).toHaveTextContent('true');
  expect(screen.getByTestId('order-pending')).toHaveTextContent('false');
  expect(screen.getByTestId('draft')).toHaveAttribute('data-prefill-id', 's1');
  expect(orderFeeServiceCreateOrderFeeSupplement).toHaveBeenCalledOnce();
  await act(async () => {});
});

it.each(['withdraw', 'approve', 'reject'] as const)(
  '%s 成功后刷新父层审核与建账能力',
  async (action) => {
    state.pending = true;
    state.created = true;
    renderWithClient(
      <App>
        <Harness />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByTestId('bill')).toHaveTextContent('false'),
    );
    if (action === 'withdraw') {
      fireEvent.click(await screen.findByText('撤回申请'));
      fireEvent.click(screen.getByRole('button', { name: /撤\s*回/ }));
    } else {
      fireEvent.click(await screen.findByText('审核'));
      const approve = await screen.findByRole('button', {
        name: '批准并确认提成影响',
      });
      await waitFor(() => expect(approve).not.toBeDisabled());
      if (action === 'approve') fireEvent.click(approve);
      else {
        fireEvent.click(screen.getByRole('button', { name: /退\s*回/ }));
        fireEvent.change(
          await screen.findByPlaceholderText('请输入驳回原因（必填）'),
          { target: { value: '依据不足' } },
        );
        fireEvent.click(screen.getByRole('button', { name: /^确\s*认$/ }));
      }
    }
    await waitFor(() =>
      expect(screen.getByTestId('pending')).toHaveTextContent('false'),
    );
    expect(screen.getByTestId('bill')).toHaveTextContent('true');
    expect(screen.getByTestId('order-pending')).toHaveTextContent('false');
    expect(screen.getByTestId('closure')).toHaveTextContent('CLOSED');
    expect(
      screen.getByRole('button', { name: '补录费用' }),
    ).toBeInTheDocument();
    await act(async () => {});
  },
);

it('撤回重提空响应保留确认弹窗，不预填或刷新为成功', async () => {
  state.pending = true;
  state.created = true;
  vi.mocked(orderFeeServiceWithdrawOrderFeeSupplement).mockResolvedValueOnce(
    undefined as unknown as API.WithdrawOrderFeeSupplementResponse,
  );
  const { queryClient } = renderWithClient(
    <App>
      <Harness />
    </App>,
  );
  const invalidated = vi.spyOn(queryClient, 'invalidateQueries');
  fireEvent.click(await screen.findByText('撤回并重提'));
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: '撤回并重提' }));
  });
  await waitFor(() =>
    expect(orderFeeServiceWithdrawOrderFeeSupplement).toHaveBeenCalledOnce(),
  );
  expect(screen.getByRole('dialog')).toBeInTheDocument();
  expect(screen.queryByTestId('draft')).not.toBeInTheDocument();
  expect(screen.getByTestId('pending')).toHaveTextContent('true');
  expect(invalidated).not.toHaveBeenCalled();
});

it('驳回空响应保留已填写原因，重试成功才关闭确认框和恢复建账', async () => {
  state.pending = true;
  state.created = true;
  const reject = vi.mocked(orderFeeServiceRejectOrderFeeSupplement);
  reject.mockResolvedValueOnce(
    undefined as unknown as API.RejectOrderFeeSupplementResponse,
  );
  renderWithClient(
    <App>
      <Harness />
    </App>,
  );
  fireEvent.click(await screen.findByText('审核'));
  await screen.findByRole('button', { name: '批准并确认提成影响' });
  fireEvent.click(screen.getByRole('button', { name: /退\s*回/ }));
  const input = await screen.findByPlaceholderText('请输入驳回原因（必填）');
  fireEvent.change(input, { target: { value: '请补齐凭证' } });
  const confirm = screen.getByRole('button', { name: /^确\s*认$/ });
  fireEvent.click(confirm);
  await waitFor(() => expect(reject).toHaveBeenCalledOnce());
  expect(input).toHaveValue('请补齐凭证');
  expect(screen.getByTestId('pending')).toHaveTextContent('true');
  expect(screen.queryByText('补录申请已驳回')).not.toBeInTheDocument();
  await waitFor(() =>
    expect(screen.getByRole('button', { name: /确\s*认$/ })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  fireEvent.click(confirm);
  await waitFor(() =>
    expect(screen.getByTestId('pending')).toHaveTextContent('false'),
  );
  expect(reject.mock.calls[1][1]).toEqual(reject.mock.calls[0][1]);
  await act(async () => {});
});

it('提交空响应保留草稿和幂等身份，不把审核缓存标成待审', async () => {
  const create = vi.mocked(orderFeeServiceCreateOrderFeeSupplement);
  create.mockResolvedValueOnce(
    undefined as unknown as API.CreateOrderFeeSupplementResponse,
  );
  const { queryClient } = renderWithClient(
    <App>
      <Harness />
    </App>,
  );
  await waitFor(() =>
    expect(screen.getByTestId('bill')).toHaveTextContent('true'),
  );
  const invalidated = vi.spyOn(queryClient, 'invalidateQueries');
  fireEvent.click(screen.getByRole('button', { name: '补录费用' }));
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: '测试提交' }));
  });
  await waitFor(() => expect(create).toHaveBeenCalledOnce());
  expect(screen.getByTestId('draft')).toBeInTheDocument();
  expect(screen.getByTestId('pending')).toHaveTextContent('false');
  expect(invalidated).not.toHaveBeenCalled();
  expect(
    screen.queryByText('补录申请已提交，等待审批'),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '测试提交' }));
  await waitFor(() =>
    expect(screen.getByTestId('pending')).toHaveTextContent('true'),
  );
  expect(create.mock.calls[1][1]).toEqual(create.mock.calls[0][1]);
  await act(async () => {});
});
