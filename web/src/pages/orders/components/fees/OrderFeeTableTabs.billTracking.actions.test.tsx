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
import { PAYABLE, RECEIVABLE } from './feeConstants';
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
    canCreateFees: true,
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
    updateFee.mockReset();
  });

  it('行内保存成功后通知页面刷新关联投影（A4）', async () => {
    const today = '2026-09-21';
    const onFeeSaved = vi.fn();
    updateFee.mockResolvedValue({
      data: { canEdit: true, canBill: true, id: 'fee-edit-1' },
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

    await screen.findByText(`${today} 00:00`);
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
    await screen.findByText('拖车费');

    const rowCheckbox = (rowText: string) => {
      const row = screen
        .getByText(rowText)
        .closest('tr') as HTMLTableRowElement;
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
