import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { FEE_CONFIRMED, PAYABLE, RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: { totalAmount: 100, count: 1 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: ['fee-1'],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: true,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    onOpenFeeModal: vi.fn(),
    getTableColumns: () => [{ dataIndex: 'id', title: '费用 ID' }],
  };
}

describe('OrderFeeTableTabs 业务锁策略', () => {
  beforeEach(() => {
    listFees.mockReset();
    listFees.mockResolvedValue({ items: [] } as Awaited<
      ReturnType<typeof listFees>
    >);
  });

  it('业务费用只读时禁用新增，但保留已确认费用的账单创建入口', async () => {
    const props = makeProps('order-1');

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() => expect(listFees).toHaveBeenCalled());
    expect(
      screen.getByRole('button', { name: /生成账单（1）/ }),
    ).not.toBeDisabled();
    expect(screen.getByRole('button', { name: /新增应收费用/ })).toBeDisabled();
  });

  it('订单 A 的费用响应晚于订单 B 时，不得覆盖 B 的费用与汇总', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    listFees.mockImplementation(({ orderId }) => {
      return orderId === 'order-A' ? responseA.promise : responseB.promise;
    });

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-A' }),
    );

    const propsB = {
      ...props,
      orderId: 'order-B',
      selectedReceivableFeeIds: [],
    };
    rerender(
      <App>
        <OrderFeeTableTabs {...propsB} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'fee-B',
      direction: RECEIVABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '200',
    } as API.OrderFee;
    responseB.resolve({ data: [feeB] });

    await waitFor(() =>
      expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]),
    );
    expect(props.setReceivableSummary).toHaveBeenLastCalledWith({
      totalAmount: 200,
      count: 1,
    });

    const feeA = {
      id: 'fee-A',
      direction: RECEIVABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '999',
    } as API.OrderFee;
    await act(async () => {
      responseA.resolve({ data: [feeA] });
    });

    expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]);
    expect(props.setReceivableSummary).toHaveBeenLastCalledWith({
      totalAmount: 200,
      count: 1,
    });
    expect(props.setAllReceivableItems).not.toHaveBeenCalledWith([feeA]);
  });

  it('订单 A 已展示后切换到 B 时，立即移除 A 的表格行且 B 失败也不恢复', async () => {
    const responseB = deferred<any>();
    const feeA = {
      id: 'visible-fee-A',
      direction: RECEIVABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '100',
    } as API.OrderFee;
    listFees
      .mockResolvedValueOnce({ data: [feeA] } as any)
      .mockImplementationOnce(() => responseB.promise);

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByText('visible-fee-A')).toBeInTheDocument(),
    );

    rerender(
      <App>
        <OrderFeeTableTabs {...props} orderId="order-B" />
      </App>,
    );

    expect(screen.queryByText('visible-fee-A')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    await act(async () => {
      responseB.reject(new Error('B 费用加载失败'));
    });

    expect(screen.queryByText('visible-fee-A')).not.toBeInTheDocument();
  });

  it('订单 A 的迟到异常不得破坏已经展示的订单 B 数据', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    listFees.mockImplementation(({ orderId }) =>
      orderId === 'order-A' ? responseA.promise : responseB.promise,
    );

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-A' }),
    );

    rerender(
      <App>
        <OrderFeeTableTabs {...props} orderId="order-B" />
      </App>,
    );
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'visible-fee-B',
      direction: RECEIVABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '200',
    } as API.OrderFee;
    await act(async () => {
      responseB.resolve({ data: [feeB] });
    });
    await waitFor(() =>
      expect(screen.getByText('visible-fee-B')).toBeInTheDocument(),
    );

    await act(async () => {
      responseA.reject(new Error('A 迟到失败'));
    });

    expect(screen.getByText('visible-fee-B')).toBeInTheDocument();
    expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]);
  });

  it('组件卸载后迟到响应不得回写父组件状态', async () => {
    const response = deferred<any>();
    listFees.mockReturnValue(response.promise);
    const props = makeProps('order-A');
    const { unmount } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await waitFor(() => expect(listFees).toHaveBeenCalledTimes(1));
    unmount();

    await act(async () => {
      response.resolve({
        data: [
          {
            id: 'fee-after-unmount',
            direction: RECEIVABLE,
            status: FEE_CONFIRMED,
          },
        ],
      });
    });

    expect(props.setAllReceivableItems).not.toHaveBeenCalled();
    expect(props.setReceivableSummary).not.toHaveBeenCalled();
  });
  it('订单 A 的应付响应晚于订单 B 时，不得覆盖 B 的应付费用与汇总', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    let orderACallCount = 0;
    listFees.mockImplementation(({ orderId }) => {
      if (orderId === 'order-B') return responseB.promise;
      orderACallCount += 1;
      return orderACallCount === 1
        ? Promise.resolve({ data: [] } as any)
        : responseA.promise;
    });

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() => expect(listFees).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole('tab', { name: /应付费用/ }));
    await waitFor(() => expect(listFees).toHaveBeenCalledTimes(2));

    const propsB = {
      ...props,
      orderId: 'order-B',
      selectedReceivableFeeIds: [],
    };
    rerender(
      <App>
        <OrderFeeTableTabs {...propsB} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'payable-B',
      direction: PAYABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '80',
    } as API.OrderFee;
    responseB.resolve({ data: [feeB] });

    await waitFor(() =>
      expect(props.setAllPayableItems).toHaveBeenLastCalledWith([feeB]),
    );
    expect(props.setPayableSummary).toHaveBeenLastCalledWith({
      totalAmount: 80,
      count: 1,
    });

    const feeA = {
      id: 'payable-A',
      direction: PAYABLE,
      status: FEE_CONFIRMED,
      baseCurrencyAmount: '666',
    } as API.OrderFee;
    await act(async () => {
      responseA.resolve({ data: [feeA] });
    });

    expect(props.setAllPayableItems).toHaveBeenLastCalledWith([feeB]);
    expect(props.setPayableSummary).toHaveBeenLastCalledWith({
      totalAmount: 80,
      count: 1,
    });
    expect(props.setAllPayableItems).not.toHaveBeenCalledWith([feeA]);
  });
});
