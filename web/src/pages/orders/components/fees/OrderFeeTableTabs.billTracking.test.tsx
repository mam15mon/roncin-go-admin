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
  history: { push: historyPush },
}));

import type { FeeBillTrackingView } from './feeBillTracking';
import {
  FEE_BILLED,
  FEE_CANCELLED,
  FEE_UNBILLED,
  PAYABLE,
  RECEIVABLE,
} from './feeConstants';
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

/** 勾选关联账单两列后确定。 */
async function enableTrackingColumns() {
  await act(async () => {
    screen.getAllByRole('button', { name: /列设置/ })[0].click();
  });
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  for (const title of ['账单号', '关联账单财务进度']) {
    const label = await waitFor(() => {
      const found = screen
        .getAllByText(title)
        .find((element) => element.closest('.ant-modal'));
      expect(found).not.toBeUndefined();
      return found as HTMLElement;
    });
    const input = label.closest('label')?.querySelector('input');
    expect(input).not.toBeNull();
    await act(async () => {
      fireEvent.click(input as HTMLInputElement);
    });
  }
  await act(async () => {
    screen.getByRole('button', { name: /确\s*定/ }).click();
  });
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

  it('就绪投影：展示活动账单号与整账单进度；已作废历史行不进入录入表（A4/A5）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-billed',
          direction: RECEIVABLE,
          status: FEE_BILLED,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '海运费',
        },
        {
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
        {
          id: 'fee-cancelled',
          direction: RECEIVABLE,
          status: FEE_CANCELLED,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '报关费',
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
            'fee-cancelled': {},
          })}
        />
      </App>,
    );
    await screen.findByText('海运费');

    // 已作废（历史软删除）行不进入录入表、父级集合与笔数统计
    expect(screen.queryByText('报关费')).not.toBeInTheDocument();
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
    // 未建账在状态、账单号与财务进度三格分别表达
    expect(screen.getAllByText('未建账')).toHaveLength(3);
  });

  it('加载中与加载失败分别表达，失败提供重试入口且不显示成未建账（A7）', async () => {
    const onRetry = vi.fn();
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-1',
          direction: RECEIVABLE,
          status: FEE_BILLED,
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
          status: FEE_UNBILLED,
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
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
    expect(screen.queryByText('账单号')).not.toBeInTheDocument();
    expect(screen.queryByText('关联账单财务进度')).not.toBeInTheDocument();
  });

  it('行内保存成功后通知页面刷新关联投影（A4）', async () => {
    const today = '2026-09-21';
    const onFeeSaved = vi.fn();
    updateFee.mockResolvedValue({} as any);
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-edit-1',
          direction: PAYABLE,
          status: FEE_BILLED,
          currency: 'CNY',
          quantity: '1',
          unitPrice: '100',
          expenseDate: today,
          version: '2',
          feeSettingId: 'setting-of',
          settlementPartyId: 'agent-1',
          billingUnitId: 'unit-piao',
        } as API.OrderFee,
      ],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-1')}
          feeWritesDisabled={false}
          feeBillTracking={trackingView('ready', {
            'fee-edit-1': { billNo: 'BILL-1', financialProgress: 7 },
          })}
          onFeeSaved={onFeeSaved}
        />
      </App>,
    );

    await screen.findByText(today);
    await act(async () => {
      screen.getAllByRole('button', { name: /编\s*辑/ })[0].click();
    });
    const saveButton = await waitFor(() => {
      const button = screen.getByText('保存');
      expect(button).toBeInTheDocument();
      return button;
    });
    await act(async () => {
      fireEvent.click(saveButton);
    });

    await waitFor(() => expect(onFeeSaved).toHaveBeenCalledTimes(1));
    expect(updateFee).toHaveBeenCalledTimes(1);
  });

  it('删除入口仅对未建账费用展示并回调页面删除流程', async () => {
    const onCancelFee = vi.fn();
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
        {
          id: 'fee-billed',
          direction: RECEIVABLE,
          status: FEE_BILLED,
          expenseDate: '2026-09-20',
          version: '3',
          feeName: '海运费',
        },
      ],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-1')}
          feeWritesDisabled={false}
          onCancelFee={onCancelFee}
        />
      </App>,
    );
    await screen.findByText('拖车费');

    const deleteButtons = screen.getAllByRole('button', { name: /删\s*除/ });
    expect(deleteButtons).toHaveLength(1);
    expect(
      screen.queryByRole('button', { name: /作\s*废/ }),
    ).not.toBeInTheDocument();
    await act(async () => {
      deleteButtons[0].click();
    });
    expect(onCancelFee).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'fee-unbilled' }),
    );
  });

  it('未建账与已建账费用均可勾选，操作列不再提供确认/撤回入口（A1/R2）', async () => {
    const setSelectedReceivableFeeIds = vi.fn();
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
          expenseDate: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
        {
          id: 'fee-billed',
          direction: RECEIVABLE,
          status: FEE_BILLED,
          expenseDate: '2026-09-20',
          version: '3',
          feeName: '海运费',
        },
      ],
    } as any);

    const props = {
      ...makeProps('order-1'),
      feeWritesDisabled: false,
      setSelectedReceivableFeeIds,
    };
    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('拖车费');

    const rowCheckbox = (rowText: string) => {
      const row = screen.getByText(rowText).closest(
        'tr',
      ) as HTMLTableRowElement;
      return within(row).getByRole('checkbox') as HTMLInputElement;
    };
    // 已保存的未建账与已建账费用都可直接勾选，无需先行确认
    expect(rowCheckbox('拖车费')).not.toBeDisabled();
    expect(rowCheckbox('海运费')).not.toBeDisabled();

    await act(async () => {
      fireEvent.click(rowCheckbox('拖车费'));
    });
    // rowSelection.onChange 首参为选中行 key 集合
    expect(setSelectedReceivableFeeIds.mock.calls[0]?.[0]).toEqual([
      'fee-unbilled',
    ]);

    // 确认/撤回按钮及其 Popconfirm 文案已随费用确认流程退役
    expect(screen.queryByRole('button', { name: /^确\s*认$/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /撤\s*回/ })).toBeNull();
    expect(
      screen.queryByText('确认后该费用才能进入账单，确定继续？'),
    ).not.toBeInTheDocument();
  });

  it('有投影时提供订单财务详情入口（A3 入口）', async () => {
    historyPush.mockClear();
    listFees.mockResolvedValue({ data: [] } as any);
    render(
      <App>
        <OrderFeeTableTabs
          {...makeProps('order-1')}
          feeBillTracking={trackingView('ready', {})}
        />
      </App>,
    );

    const entry = await screen.findAllByRole('button', { name: /财务详情/ });
    expect(entry.length).toBe(2);
    act(() => {
      entry[0].click();
    });
    expect(historyPush).toHaveBeenCalledWith('/finance/fees/detail/order-1');
  });
});
