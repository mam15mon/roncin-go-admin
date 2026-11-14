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
  orderFeeServiceSaveOrderFees,
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
import { PAYABLE, RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
  orderFeeServiceSaveOrderFees: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);
const saveFees = vi.mocked(orderFeeServiceSaveOrderFees);

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
    payableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
    selectedReceivableFeeIds: [],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: false,
    canCreateFees: true,
    canUpdateFees: true,
    canDeleteFees: true,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    getTableColumns: undefined,
  };
}

function trackingView(
  state: FeeBillTrackingView['state'],
  byFeeId: FeeBillTrackingView['byFeeId'],
  onRetry = vi.fn(),
): FeeBillTrackingView {
  return { state, byFeeId, onRetry };
}

describe('OrderFeeTableTabs 账单投影相关操作', () => {
  beforeEach(() => {
    window.localStorage.clear();
    listFees.mockReset();
    resolveRate.mockReset();
    resolveRate.mockResolvedValue({
      exchangeRate: '1.0000',
      exchangeRateSource: 'SYSTEM',
    } as Awaited<ReturnType<typeof resolveRate>>);
    saveFees.mockReset();
  });

  it('整表保存成功后携带费用内容版本通知页面刷新关联投影（A4）', async () => {
    const today = '2026-09-21';
    const onFeeSaved = vi.fn();
    saveFees.mockResolvedValue({
      success: true,
      data: [],
      feeContentVersion: '9',
    } as any);
    listFees.mockResolvedValue({
      data: [
        {
          canEdit: true,
          canBill: true,
          id: 'fee-edit-1',
          direction: PAYABLE,
          hasActiveBill: false,
          currency: 'CNY',
          quantity: '1',
          unitPrice: '100',
          createdAt: today,
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

    const price = (await screen.findAllByDisplayValue('100'))[0];
    fireEvent.change(price, { target: { value: '120' } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /保存应付费用/ }));
    });

    await waitFor(() => expect(onFeeSaved).toHaveBeenCalledWith('9'));
    expect(saveFees).toHaveBeenCalledTimes(1);
  });

  it('只有删除权限的未建账费用仍可删除，已建账费用不开放操作', async () => {
    const onCancelFee = vi.fn();
    listFees.mockResolvedValue({
      data: [
        {
          canEdit: false,
          canDelete: true,
          canBill: true,
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          createdAt: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
        {
          canEdit: false,
          canDelete: false,
          canBill: false,
          id: 'fee-billed',
          direction: RECEIVABLE,
          hasActiveBill: true,
          createdAt: '2026-09-20',
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

    // 行内「删除」操作锚定整名匹配，避开工具栏批量维护新增的「批量删除」入口
    const deleteButtons = screen.getAllByRole('button', {
      name: /^删\s*除$/,
    });
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
          canEdit: true,
          canBill: true,
          id: 'fee-unbilled',
          direction: RECEIVABLE,
          createdAt: '2026-09-20',
          version: '2',
          feeName: '拖车费',
        },
        {
          canEdit: false,
          canBill: false,
          id: 'fee-billed',
          direction: RECEIVABLE,
          hasActiveBill: true,
          createdAt: '2026-09-20',
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
    await screen.findByText('海运费');

    // 未建账行处于常开编辑态（名称在下拉框内），按行 key 定位。
    const rowCheckbox = (rowKey: string) => {
      const row = document.querySelector(
        `tr[data-row-key="${rowKey}"]`,
      ) as HTMLTableRowElement;
      return within(row).getByRole('checkbox') as HTMLInputElement;
    };
    // 已保存的未建账与已建账费用都可直接勾选，无需先行确认
    expect(rowCheckbox('fee-unbilled')).not.toBeDisabled();
    expect(rowCheckbox('fee-billed')).not.toBeDisabled();

    await act(async () => {
      fireEvent.click(rowCheckbox('fee-unbilled'));
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
