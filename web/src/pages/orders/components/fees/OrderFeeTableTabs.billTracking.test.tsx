import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';

const { historyPush } = vi.hoisted(() => ({ historyPush: vi.fn() }));
vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return window.location;
    },
    push: historyPush,
  },
}));

import type { FeeBillTrackingView } from './feeBillTracking';
import { RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);
const updateFee = vi.mocked(orderFeeServiceUpdateFee);

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: { totalAmount: 0, count: 0 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: [],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: false,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    getTableColumns: undefined,
  };
}

/** 勾选关联账单两列；弹窗内操作即时生效。 */
async function enableTrackingColumns() {
  await act(async () => {
    screen.getAllByRole('button', { name: /列设置/ })[0].click();
  });
  const dialog = await screen.findByRole('dialog', { name: '列设置' });
  for (const title of ['账单号', '关联账单财务进度']) {
    const input = within(dialog).getByRole('checkbox', { name: title });
    await act(async () => {
      fireEvent.click(input as HTMLInputElement);
    });
  }
  await waitFor(() =>
    expect(
      screen.queryAllByRole('columnheader', { name: '账单号' }).length,
    ).toBe(2),
  );
}

function trackingView(
  state: FeeBillTrackingView['state'],
  byFeeId: FeeBillTrackingView['byFeeId'],
  onRetry = vi.fn(),
): FeeBillTrackingView {
  return { state, byFeeId, onRetry };
}

describe('OrderFeeTableTabs 关联账单列', () => {
  beforeEach(() => {
    window.localStorage.clear();
    listFees.mockReset();
    resolveRate.mockReset();
    resolveRate.mockResolvedValue({
      exchangeRate: '1.0000',
      exchangeRateSource: 'SYSTEM',
    } as Awaited<ReturnType<typeof resolveRate>>);
    updateFee.mockReset();
  });

  it('就绪投影：展示活动账单号与整账单进度（A4/A5）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-billed',
          direction: RECEIVABLE,
          hasActiveBill: true,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '海运费',
        },
        {
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
      ],
    } as any);

    const props = {
      ...makeProps('order-1'),
      setAllReceivableItems: vi.fn(),
      setReceivableSummary: vi.fn(),
    };
    render(
      <App>
        <OrderFeeTableTabs
          {...props}
          feeBillTracking={trackingView('ready', {
            'fee-billed': {
              billNo: 'BILL-2026-001',
              financialProgress: 3,
            },
            'fee-unbilled': {},
          })}
        />
      </App>,
    );
    await screen.findByText('海运费');

    await waitFor(() =>
      expect(props.setAllReceivableItems).toHaveBeenCalledWith([
        expect.objectContaining({ id: 'fee-billed' }),
        expect.objectContaining({ id: 'fee-unbilled' }),
      ]),
    );
    expect(props.setReceivableSummary).toHaveBeenCalledWith({
      totalAmount: 0,
      count: 2,
    });

    await enableTrackingColumns();

    expect(screen.getByText('BILL-2026-001')).toBeInTheDocument();
    expect(screen.getAllByText('已开票未核销')).toHaveLength(1);
    // 未建账在账单号与财务进度两格分别表达
    expect(screen.getAllByText('未建账')).toHaveLength(2);
  });

  it('加载中与加载失败分别表达，失败提供重试入口且不显示成未建账（A7）', async () => {
    const onRetry = vi.fn();
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-1',
          direction: RECEIVABLE,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '海运费',
        },
      ],
    } as any);

    const { rerender } = render(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-1')}
          feeBillTracking={trackingView('loading', {})}
        />
      </App>,
    );
    await screen.findByText('海运费');
    await enableTrackingColumns();
    expect(screen.getAllByText('加载中…').length).toBeGreaterThan(0);
    expect(screen.queryByText('未建账')).not.toBeInTheDocument();

    rerender(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-1')}
          feeBillTracking={trackingView('error', {}, onRetry)}
        />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getAllByText('加载失败').length).toBeGreaterThan(0),
    );
    expect(screen.queryByText('未建账')).not.toBeInTheDocument();

    await act(async () => {
      screen.getAllByRole('button', { name: /重试/ })[0].click();
    });
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('无关联投影时不提供账单列与财务详情入口，不影响列设置（A6）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-1',
          direction: RECEIVABLE,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '海运费',
        },
      ],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');

    expect(
      screen.queryByRole('button', { name: /财务详情/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('columnheader', { name: '账单号' }),
    ).not.toBeInTheDocument();

    await act(async () => {
      screen.getAllByRole('button', { name: /列设置/ })[0].click();
    });
    await screen.findByPlaceholderText('搜索列名');
    expect(screen.queryByText('账单号')).not.toBeInTheDocument();
    expect(screen.queryByText('关联账单财务进度')).not.toBeInTheDocument();
  });
});
