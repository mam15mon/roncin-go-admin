import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderFeeDirection } from '@/enums.generated';
import { orderFeeServiceGetOrderFeeReview } from '@/services/roncin/orderFeeService';
import { orderLockServiceLockOrder } from '@/services/roncin/orderLockService';
import OrderLockControl from './OrderLockControl';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceGetOrderFeeReview: vi.fn(),
}));
vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceLockOrder: vi.fn(),
  orderLockServiceRequestOrderUnlock: vi.fn(),
}));
vi.mock('./UnlockRequestHistoryDrawer', () => ({ default: () => null }));

const getReview = vi.mocked(orderFeeServiceGetOrderFeeReview);
const lockOrder = vi.mocked(orderLockServiceLockOrder);
const snapshot: API.OrderLockStateData = {
  orderId: 'review-order',
  orderNo: 'SE2026102700001',
  canLock: true,
  orderVersion: '17',
  feeContentVersion: '8',
  feeContentFingerprint: 'fee-fingerprint-8',
  feeSummary: {
    baseCurrency: 'CNY',
    receivable: '12000',
    payable: '8100',
    profit: '3900',
    receivableCount: 12,
    payableCount: 1,
  },
};

function renderControl(onRetry = vi.fn().mockResolvedValue(snapshot)) {
  return renderWithApp(
    <OrderLockControl
      orderId="review-order"
      state={snapshot}
      loading={false}
      error={null}
      onRetry={onRetry}
      onSynchronize={vi.fn().mockResolvedValue(undefined)}
    />,
  );
}

function reviewResponse(data: Partial<API.OrderFeeReviewData> = {}) {
  return {
    success: true,
    data: { orderId: 'review-order', contentVersion: '8', fees: [], ...data },
  };
}

describe('订单审核费用明细', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('应收应付全部逐笔展示，完整保留结算、计费、折算、税额与备注', async () => {
    const receivables: API.OrderFee[] = Array.from({ length: 12 }, (_, i) => ({
      id: `receivable-${i}`,
      direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
      feeCode: `RC${i}`,
      feeName: `应收费用项目${i}`,
      feeNameEn: `Receivable item ${i}`,
      settlementPartyName: `完整客户名称${i}`,
      billingUnit: '票',
      quantity: '2.50000000',
      unitPrice: '400.12500000',
      totalAmount: '1000.31250000',
      currency: 'USD',
      exchangeRate: '6.71830000',
      baseCurrency: 'CNY',
      baseCurrencyAmount: '6720.39946875',
      taxRate: '6.00000000',
      taxInclusive: true,
      netAmount: '943.69103774',
      taxAmount: '56.62146226',
      note: `第${i}笔费用完整备注，不截断任何文字`,
      hasActiveBill: i === 0,
      economicEffect: 'INCREASE',
    }));
    getReview.mockResolvedValue(
      reviewResponse({
        fees: [
          ...receivables,
          {
            id: 'payable',
            direction: OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE,
            feeCode: 'OCEAN',
            feeName: '应付海运费',
            settlementPartyName: '完整供应商名称',
            currency: 'CNY',
            totalAmount: '8100.00000000',
            note: '供应商应付明细',
          },
        ],
      }),
    );
    renderControl();
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));

    const receivable = await screen.findByRole('region', { name: '应收费用' });
    const payable = screen.getByRole('region', { name: '应付费用' });
    expect(
      within(receivable).getByText('应收费用（12笔）'),
    ).toBeInTheDocument();
    for (let i = 0; i < 12; i++) {
      expect(
        within(receivable).getByText(`应收费用项目${i}`),
      ).toBeInTheDocument();
      expect(
        within(receivable).getByText(`第${i}笔费用完整备注，不截断任何文字`),
      ).toBeInTheDocument();
    }
    const firstRow = within(receivable).getByText('RC0').closest('tr');
    expect(firstRow).not.toBeNull();
    for (const text of [
      'Receivable item 0',
      '完整客户名称0',
      '票',
      '2.5',
      '400.125',
      '含税',
      '1,000.31 USD',
      '6.7183',
      '6,720.40 CNY',
      '6%',
      '943.69',
      '56.62',
      '已建账',
    ]) {
      expect(
        within(firstRow as HTMLTableRowElement).getByText(text),
      ).toBeInTheDocument();
    }
    expect(within(payable).getByText('应付海运费')).toBeInTheDocument();
    expect(within(payable).getByText('8,100.00 CNY')).toBeInTheDocument();
    expect(within(payable).getByText('完整供应商名称')).toBeInTheDocument();
    expect(within(payable).getByText('供应商应付明细')).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: '审核意见' })).toBeNull();
    expect(getReview).toHaveBeenCalledExactlyOnceWith({
      orderId: 'review-order',
    });
  });

  it('明细未完成或失败时禁止确认，成功重试后才允许提交', async () => {
    let rejectReview: (reason: Error) => void = () => {};
    getReview.mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          rejectReview = reject;
        }),
    );
    renderControl();
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    expect(await screen.findByText('正在加载全部费用明细')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /确.*认/ })).toBeDisabled();
    rejectReview(new Error('网络失败'));
    expect(
      await screen.findByText('费用明细加载失败，请重试后审核'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /确.*认/ })).toBeDisabled();
    expect(lockOrder).not.toHaveBeenCalled();

    getReview.mockResolvedValue(reviewResponse());
    fireEvent.click(screen.getByRole('button', { name: '重试费用明细' }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确.*认/ })).toBeEnabled(),
    );
    expect(screen.getByText('暂无应收费用')).toBeInTheDocument();
    expect(screen.getByText('暂无应付费用')).toBeInTheDocument();
    expect(getReview).toHaveBeenCalledTimes(2);
  });

  it('费用版本与确认快照不同时阻断提交，提供关闭刷新并重新打开的路径', async () => {
    getReview.mockResolvedValue(reviewResponse({ contentVersion: '9' }));
    const onRetry = vi.fn().mockResolvedValue(snapshot);
    renderControl(onRetry);
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    expect(
      await screen.findByText('订单或费用已变化，请刷新后重新审核'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /确.*认/ })).toBeDisabled();
    expect(screen.queryByRole('region', { name: '应收费用' })).toBeNull();
    expect(lockOrder).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '刷新审核信息' }));
    await waitFor(() => expect(onRetry).toHaveBeenCalledOnce());
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

    getReview.mockResolvedValue(reviewResponse());
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确.*认/ })).toBeEnabled(),
    );
    expect(getReview).toHaveBeenCalledTimes(2);
  });

  it('缓存明细刷新中及弹窗打开后订单版本变化时均禁止提交', async () => {
    let resolveReview: (value: ReturnType<typeof reviewResponse>) => void =
      () => {};
    getReview.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveReview = resolve;
        }),
    );
    const { queryClient, rerender } = renderControl();
    queryClient.setQueryData(
      ['order-fee-review', undefined, 'review-order'],
      reviewResponse().data,
    );
    fireEvent.click(screen.getByRole('button', { name: /审核并锁定/ }));
    expect(await screen.findByText('正在加载全部费用明细')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认审核' })).toBeDisabled();
    expect(screen.queryByRole('region', { name: '应收费用' })).toBeNull();
    resolveReview(reviewResponse());
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '确认审核' })).toBeEnabled(),
    );

    rerender(
      <OrderLockControl
        orderId="review-order"
        state={{ ...snapshot, orderVersion: '18' }}
        loading={false}
        error={null}
        onRetry={vi.fn()}
        onSynchronize={vi.fn()}
      />,
    );
    expect(
      await screen.findByText('订单或费用已变化，请刷新后重新审核'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认审核' })).toBeDisabled();
    expect(lockOrder).not.toHaveBeenCalled();
  });
});
