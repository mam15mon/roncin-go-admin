import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { OrderFeeDirection } from '@/enums.generated';
import { orderFeeServiceGetOrderFeeReview } from '@/services/roncin/orderFeeService';
import { orderLockServiceGetOrderLockState } from '@/services/roncin/orderLockService';
import { OrderFeeReviewDrawer, OrderFeeReviewSection } from './OrderFeeReview';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceGetOrderFeeReview: vi.fn(),
}));
vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceGetOrderLockState: vi.fn(),
}));
vi.mock('../../detail/OrderLockControl', () => ({
  default: () => <button type="button">审核订单</button>,
  OrderLockStatusTag: () => <span>未审核</span>,
}));

const getReview = vi.mocked(orderFeeServiceGetOrderFeeReview);
const getLockState = vi.mocked(orderLockServiceGetOrderLockState);

function reviewResponse(orderId: string, fees: API.OrderFee[] = []) {
  return { success: true, data: { orderId, fees } };
}

function drawer(orderId?: string) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderFeeReviewDrawer
        orderId={orderId}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />
    </ConfigProvider>
  );
}

beforeEach(() => {
  vi.resetAllMocks();
  getLockState.mockImplementation(async ({ orderId }) => ({
    success: true,
    data: { orderId, isLocked: false },
  }));
});
afterEach(cleanup);

it('打开抽屉即完整展示全部应收应付及业务明细，无需再点击审核', async () => {
  const receivables: API.OrderFee[] = Array.from(
    { length: 21 },
    (_, index) => ({
      id: `receivable-${index}`,
      direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
      feeCode: `RC${index}`,
      feeName: `应收费用${index}`,
      settlementPartyName: '完整客户名称',
      billingUnit: '票',
      quantity: '2.50000000',
      unitPrice: '400.12500000',
      totalAmount: '1000.31250000',
      currency: 'USD',
      exchangeRate: '6.71830000',
      baseCurrency: 'CNY',
      baseCurrencyAmount: '6720.39946875',
      taxRate: '6.00000000',
      netAmount: '943.69103774',
      taxAmount: '56.62146226',
      note: `完整费用备注${index}`,
    }),
  );
  getReview.mockResolvedValue(
    reviewResponse('order-1', [
      ...receivables,
      {
        id: 'payable',
        direction: OrderFeeDirection.ORDER_FEE_DIRECTION_PAYABLE,
        feeName: '应付海运费',
        totalAmount: '8100.00000000',
        currency: 'CNY',
      },
    ]),
  );
  renderWithApp(drawer('order-1'));

  const receivable = await screen.findByRole('region', { name: '应收费用' });
  const payable = screen.getByRole('region', { name: '应付费用' });
  expect(screen.getByText('订单审核')).toBeInTheDocument();
  expect(getReview).toHaveBeenCalledExactlyOnceWith({ orderId: 'order-1' });
  expect(within(receivable).getByText('应收费用（21笔）')).toBeInTheDocument();
  for (let index = 0; index < 21; index++) {
    expect(
      within(receivable).getByText(`应收费用${index}`),
    ).toBeInTheDocument();
    expect(
      within(receivable).getByText(`完整费用备注${index}`),
    ).toBeInTheDocument();
  }
  expect(within(payable).getByText('应付海运费')).toBeInTheDocument();
  const firstRow = within(
    within(receivable).getByText('RC0').closest('tr') as HTMLElement,
  );
  for (const text of [
    '完整客户名称',
    '票',
    '2.5',
    '400.125',
    '1,000.31 USD',
    '6.7183',
    '6,720.40 CNY',
    '6%',
    '943.69',
    '56.62',
  ]) {
    expect(firstRow.getByText(text)).toBeInTheDocument();
  }
  expect(within(receivable).queryByRole('navigation')).not.toBeInTheDocument();
});

it('明细加载有反馈，失败可重试，空费用分别展示应收应付空态', async () => {
  const pending =
    Promise.withResolvers<
      Awaited<ReturnType<typeof orderFeeServiceGetOrderFeeReview>>
    >();
  getReview.mockReturnValueOnce(pending.promise);
  getReview.mockResolvedValueOnce(reviewResponse('order-1'));
  renderWithApp(drawer('order-1'));
  expect(await screen.findByText('正在加载全部费用明细')).toBeInTheDocument();
  await act(async () => pending.reject(new Error('模拟加载失败')));
  expect(
    await screen.findByText('费用明细加载失败，请重试'),
  ).toBeInTheDocument();
  expect(
    screen.queryByRole('region', { name: '应收费用' }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '重试费用明细' }));
  expect(await screen.findByText('暂无应收费用')).toBeInTheDocument();
  expect(screen.getByText('暂无应付费用')).toBeInTheDocument();
  expect(getReview).toHaveBeenCalledTimes(2);
});

it('仅打开时请求明细，切换订单不会渲染上一票迟到的费用', async () => {
  const first =
    Promise.withResolvers<
      Awaited<ReturnType<typeof orderFeeServiceGetOrderFeeReview>>
    >();
  getReview.mockImplementation(({ orderId }) =>
    orderId === 'order-first'
      ? first.promise
      : Promise.resolve(
          reviewResponse(orderId, [
            {
              id: 'second-fee',
              direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
              feeName: '第二票费用',
            },
          ]),
        ),
  );
  const { rerender } = renderWithApp(drawer());
  expect(getReview).not.toHaveBeenCalled();
  rerender(drawer('order-first'));
  expect(await screen.findByText('正在加载全部费用明细')).toBeInTheDocument();
  rerender(drawer('order-second'));
  expect(await screen.findByText('第二票费用')).toBeInTheDocument();
  await act(async () =>
    first.resolve(
      reviewResponse('order-first', [
        {
          id: 'first-fee',
          direction: OrderFeeDirection.ORDER_FEE_DIRECTION_RECEIVABLE,
          feeName: '第一票迟到费用',
        },
      ]),
    ),
  );
  expect(screen.queryByText('第一票迟到费用')).not.toBeInTheDocument();
  expect(screen.getByText('第二票费用')).toBeInTheDocument();
  rerender(drawer());
  expect(getReview).toHaveBeenCalledTimes(2);
});

it('费用工作台的审核区仍只显示摘要，不重复请求或渲染只读费用表', async () => {
  getLockState.mockResolvedValue({
    success: true,
    data: {
      orderId: 'order-1',
      feeSummary: { baseCurrency: 'CNY', receivable: '1200' },
    },
  });
  renderWithApp(
    <OrderFeeReviewSection orderId="order-1" onChanged={vi.fn()} />,
  );
  expect(await screen.findByText('1,200.00')).toBeInTheDocument();
  expect(getReview).not.toHaveBeenCalled();
  expect(
    screen.queryByRole('region', { name: '应收费用' }),
  ).not.toBeInTheDocument();
});
