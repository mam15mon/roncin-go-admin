import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceApproveOrderFeeReview } from '@/services/roncin/orderFeeService';
import { OrderFeeReviewActions } from './OrderFeeReview';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
vi.mock('@/services/roncin/orderFeeService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/orderFeeService')>()),
  orderFeeServiceApproveOrderFeeReview: vi
    .fn()
    .mockResolvedValue({ success: true, data: { orderId: 'o1' } }),
}));
const approve = vi.mocked(orderFeeServiceApproveOrderFeeReview);
const data: API.OrderFeeReviewData = {
  orderId: 'o1',
  reviewVersion: '3',
  contentVersion: '7',
  submissionId: 's1',
  previewToken: 'snapshot-1',
  canApprove: true,
  status: 'PENDING',
  fees: [
    {
      id: 'f1',
      direction: 1,
      feeName: '海运应收核对',
      totalAmount: '100',
      currency: 'CNY',
    },
    {
      id: 'f2',
      direction: 2,
      feeName: '海运应付核对',
      totalAmount: '150',
      currency: 'CNY',
    },
  ],
  summary: {
    baseCurrency: 'CNY',
    receivable: '100',
    payable: '150',
    profit: '-50',
  },
};

describe('整票费用批准', () => {
  beforeEach(() => vi.clearAllMocks());
  it('亏损审批意见必填，人工填写后提交该票快照', async () => {
    renderWithClient(
      <App>
        <OrderFeeReviewActions data={data} onChanged={vi.fn()} />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '批准整票费用' }));
    expect(screen.getByText('海运应收核对')).toBeInTheDocument();
    expect(screen.getByText('海运应付核对')).toBeInTheDocument();
    expect(
      screen.getAllByRole('columnheader', { name: '含税金额' }),
    ).toHaveLength(2);
    const submit = screen.getByRole('button', { name: /确.*认/ });
    fireEvent.click(submit);
    expect(await screen.findByText('请填写有效意见或原因')).toBeInTheDocument();
    expect(approve).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('textbox', { name: '审批意见' }), {
      target: { value: '客户首票优惠，经核实成本完整' },
    });
    fireEvent.click(submit);
    await waitFor(() => expect(approve).toHaveBeenCalledOnce());
    expect(approve.mock.calls[0][1]).toMatchObject({
      expectedReviewVersion: '3',
      expectedContentVersion: '7',
      submissionId: 's1',
      previewToken: 'snapshot-1',
      comment: '客户首票优惠，经核实成本完整',
    });
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await act(async () => {});
  });
  it('错误被请求层消费后保留意见和审批身份，重试成功才关闭', async () => {
    approve.mockResolvedValueOnce(
      undefined as unknown as API.ApproveOrderFeeReviewResponse,
    );
    const changed = vi.fn();
    const { queryClient } = renderWithClient(
      <App>
        <OrderFeeReviewActions data={data} onChanged={changed} />
      </App>,
    );
    const invalidated = vi.spyOn(queryClient, 'invalidateQueries');
    fireEvent.click(screen.getByRole('button', { name: '批准整票费用' }));
    const comment = '审'.repeat(1000);
    fireEvent.change(screen.getByRole('textbox', { name: '审批意见' }), {
      target: { value: comment },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    });
    await waitFor(() => expect(approve).toHaveBeenCalledOnce());
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: '审批意见' })).toHaveValue(
      comment,
    );
    expect(changed).not.toHaveBeenCalled();
    expect(invalidated).not.toHaveBeenCalled();
    expect(screen.queryByText('费用审核状态已更新')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(changed).toHaveBeenCalledOnce());
    expect(approve.mock.calls[1][1]).toEqual(approve.mock.calls[0][1]);
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await act(async () => {});
  });
  it('超过1000字符不能提交，1000个emoji按码点校验通过', async () => {
    renderWithClient(
      <App>
        <OrderFeeReviewActions data={data} onChanged={vi.fn()} />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '批准整票费用' }));
    fireEvent.change(screen.getByRole('textbox', { name: '审批意见' }), {
      target: { value: '审'.repeat(1001) },
    });
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    expect(
      await screen.findByText('审批意见最多填写 1000 个字符'),
    ).toBeInTheDocument();
    expect(approve).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('textbox', { name: '审批意见' }), {
      target: { value: '😀'.repeat(1000) },
    });
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(approve).toHaveBeenCalledOnce());
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await act(async () => {});
  });
  it('确认窗口打开后不静默替换为随后刷新的费用版本', async () => {
    const { rerender, queryClient } = renderWithClient(
      <App>
        <OrderFeeReviewActions
          data={{ ...data, summary: { ...data.summary, profit: '50' } }}
          onChanged={vi.fn()}
        />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '批准整票费用' }));
    rerender(
      <QueryClientProvider client={queryClient}>
        <App>
          <OrderFeeReviewActions
            data={{
              ...data,
              reviewVersion: '4',
              contentVersion: '8',
              previewToken: 'snapshot-2',
            }}
            onChanged={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(approve).toHaveBeenCalledOnce());
    expect(approve.mock.calls[0][1]).toMatchObject({
      expectedReviewVersion: '3',
      expectedContentVersion: '7',
      previewToken: 'snapshot-1',
    });
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    await act(async () => {});
  });
});
