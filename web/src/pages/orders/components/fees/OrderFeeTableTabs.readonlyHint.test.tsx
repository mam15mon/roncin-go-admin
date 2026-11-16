import type { ActionType } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

const baseFee = {
  version: '1',
  feeCode: 'THC',
  feeName: '码头操作费',
  currency: 'CNY',
  quantity: '1',
  unitPrice: '1',
  totalAmount: '1',
  baseCurrencyAmount: '1',
};

const emptySummary = {
  count: 0,
  gross: '0',
  net: '0',
  unconvertedCount: 0,
  currencies: [],
};

function renderTabs(canUpdateFees = true) {
  const ref = () =>
    ({ current: undefined }) as React.RefObject<ActionType | undefined>;
  return render(
    <App>
      <OrderFeeTableTabs
        orderId="order-1"
        receivableActionRef={ref()}
        payableActionRef={ref()}
        receivableSummary={emptySummary}
        payableSummary={emptySummary}
        selectedReceivableFeeIds={[]}
        setSelectedReceivableFeeIds={vi.fn()}
        selectedPayableFeeIds={[]}
        setSelectedPayableFeeIds={vi.fn()}
        setAllReceivableItems={vi.fn()}
        setAllPayableItems={vi.fn()}
        setReceivableSummary={vi.fn()}
        setPayableSummary={vi.fn()}
        canCreateFinanceBills
        canCreateFees
        canUpdateFees={canUpdateFees}
        canDeleteFees
        feeWritesDisabled={false}
        onOpenBillWorkbench={vi.fn()}
      />
    </App>,
  );
}

describe('OrderFeeTableTabs 只读行悬停提示', () => {
  beforeEach(() => {
    listFees.mockReset();
  });

  it('已建账的应付行显示锁标并悬停展示服务端原因，可编辑行不显示', async () => {
    listFees.mockResolvedValue({
      data: [
        { ...baseFee, id: 'recv-1', direction: RECEIVABLE, canEdit: true },
        {
          ...baseFee,
          id: 'pay-1',
          direction: PAYABLE,
          canEdit: false,
          hasActiveBill: true,
          blockedReasons: ['费用已建立账单，请先逐步回退至账单未建立'],
        },
      ],
    } as Awaited<ReturnType<typeof listFees>>);

    renderTabs();

    const lock = await screen.findByTestId('fee-readonly-pay-1');
    expect(screen.queryByTestId('fee-readonly-recv-1')).not.toBeInTheDocument();

    fireEvent.mouseEnter(lock);
    expect(
      await screen.findByText('费用已建立账单，请先逐步回退至账单未建立'),
    ).toBeInTheDocument();
  });

  it('无编辑权限且服务端未给原因时提示没有费用编辑权限', async () => {
    listFees.mockResolvedValue({
      data: [{ ...baseFee, id: 'pay-2', direction: PAYABLE, canEdit: false }],
    } as Awaited<ReturnType<typeof listFees>>);

    renderTabs(false);

    const lock = await screen.findByTestId('fee-readonly-pay-2');
    fireEvent.mouseEnter(lock);
    await waitFor(() =>
      expect(screen.getByText('没有费用编辑权限')).toBeInTheDocument(),
    );
  });
});
