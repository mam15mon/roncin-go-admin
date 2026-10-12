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
import {
  orderFeeServiceApproveOrderFeeReviews,
  orderFeeServicePreviewOrderFeeReviews,
} from '@/services/roncin/orderFeeService';
import { BatchFeeReviewModal } from './BatchFeeReviewModal';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
vi.mock('@/services/roncin/orderFeeService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/orderFeeService')>()),
  orderFeeServiceApproveOrderFeeReviews: vi.fn(),
  orderFeeServicePreviewOrderFeeReviews: vi.fn(),
}));

it('批量批准保留每票预览和亏损意见，只移除成功订单且不自动重审失败新版本', async () => {
  vi.mocked(orderFeeServicePreviewOrderFeeReviews).mockResolvedValue({
    data: [
      {
        orderId: 'o1',
        success: true,
        data: {
          orderId: 'o1',
          orderNo: 'ORD-1',
          status: 'PENDING',
          fees: [
            {
              id: 'f1',
              direction: 1,
              feeName: '整票海运收入',
              currency: 'CNY',
              totalAmount: '100',
              baseCurrencyAmount: '100',
            },
            {
              id: 'f2',
              direction: 2,
              feeName: '整票海运成本',
              currency: 'CNY',
              totalAmount: '150',
              baseCurrencyAmount: '150',
            },
          ],
          canApprove: true,
          reviewVersion: '3',
          contentVersion: '2',
          previewToken: 'p1',
          summary: {
            profit: '-50',
            baseCurrency: 'CNY',
            receivable: '100',
            payable: '150',
            receivableCount: 1,
            payableCount: 1,
            profitRate: '-50',
          },
        },
      },
      {
        orderId: 'o2',
        success: true,
        data: {
          orderId: 'o2',
          orderNo: 'ORD-2',
          status: 'PENDING',
          canApprove: true,
          reviewVersion: '8',
          contentVersion: '4',
          previewToken: 'p2',
          summary: { profit: '100', baseCurrency: 'CNY' },
        },
      },
    ],
  });
  vi.mocked(orderFeeServiceApproveOrderFeeReviews).mockResolvedValue({
    data: [
      { orderId: 'o1', success: true },
      { orderId: 'o2', success: false, message: '费用版本已变化，请重新查看' },
    ],
  });
  const completed = vi.fn();
  const { queryClient } = renderWithClient(
    <App>
      <BatchFeeReviewModal
        orderIds={['o1', 'o2']}
        onClose={vi.fn()}
        onCompleted={completed}
      />
    </App>,
  );
  expect(await screen.findByText('ORD-1')).toBeInTheDocument();
  await act(async () => {
    queryClient.setQueriesData<API.OrderFeeReviewResult[]>(
      { queryKey: ['order-fee-review', 'batch'] },
      (previous) =>
        previous?.map((row) => ({
          ...row,
          data: {
            ...row.data,
            reviewVersion: '99',
            previewToken: 'background-new-token',
          },
        })),
    );
  });
  fireEvent.click(screen.getByRole('button', { name: '展开 ORD-1 费用明细' }));
  expect(screen.getByText('整票海运收入')).toBeInTheDocument();
  expect(screen.getByText('整票海运成本')).toBeInTheDocument();
  expect(
    screen.getByRole('columnheader', { name: '整票利润' }),
  ).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '批准可审核订单' }));
  expect(
    await screen.findByText('请为该亏损订单填写审批意见'),
  ).toBeInTheDocument();
  expect(orderFeeServiceApproveOrderFeeReviews).not.toHaveBeenCalled();
  fireEvent.change(
    screen.getByRole('textbox', { name: '审批意见（亏损原因，必填）' }),
    { target: { value: '亏损已核实' } },
  );
  fireEvent.click(screen.getByRole('button', { name: '批准可审核订单' }));
  expect(
    await screen.findByText('费用版本已变化，请重新查看'),
  ).toBeInTheDocument();
  expect(completed).toHaveBeenCalledWith(['o1']);
  const approvedRow = screen.getByText('ORD-1').closest('tr');
  expect(approvedRow).not.toBeNull();
  expect(
    within(approvedRow as HTMLElement).queryByText('待审核'),
  ).not.toBeInTheDocument();
  expect(
    within(approvedRow as HTMLElement).getAllByText('已批准'),
  ).toHaveLength(2);
  const items = vi.mocked(orderFeeServiceApproveOrderFeeReviews).mock
    .calls[0][0].items;
  expect(items).toEqual([
    expect.objectContaining({
      orderId: 'o1',
      previewToken: 'p1',
      expectedReviewVersion: '3',
      comment: '亏损已核实',
    }),
    expect.objectContaining({
      orderId: 'o2',
      previewToken: 'p2',
      expectedReviewVersion: '8',
    }),
  ]);
  await waitFor(() =>
    expect(
      screen.getByRole('button', { name: '批准可审核订单' }),
    ).toBeDisabled(),
  );
  expect(orderFeeServicePreviewOrderFeeReviews).toHaveBeenCalledOnce();
  await act(async () => {});
});
