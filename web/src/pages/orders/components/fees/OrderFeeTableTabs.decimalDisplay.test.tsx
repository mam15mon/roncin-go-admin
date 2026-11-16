import type { ActionType } from '@ant-design/pro-components';
import { render, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const emptySummary = {
  count: 0,
  gross: '0',
  net: '0',
  unconvertedCount: 0,
  currencies: [],
};

it('可编辑行的数量与单价去掉 decimal 尾零展示，且不被视为未保存修改', async () => {
  vi.mocked(orderFeeServiceListFees).mockResolvedValue({
    data: [
      {
        id: 'recv-1',
        direction: RECEIVABLE,
        version: '1',
        canEdit: true,
        feeCode: 'THC',
        currency: 'CNY',
        quantity: '1.0000',
        unitPrice: '1500.5000',
        totalAmount: '1500.5',
        baseCurrencyAmount: '1500.5',
      },
    ],
  } as Awaited<ReturnType<typeof orderFeeServiceListFees>>);
  const onDraftChange = vi.fn();
  const ref = () =>
    ({ current: undefined }) as React.RefObject<ActionType | undefined>;

  const { container } = render(
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
        canUpdateFees
        canDeleteFees
        feeWritesDisabled={false}
        onOpenBillWorkbench={vi.fn()}
        onDraftChange={onDraftChange}
      />
    </App>,
  );

  await waitFor(() => {
    const values = Array.from(
      container.querySelectorAll<HTMLInputElement>(
        '[data-row-key="recv-1"] input:not([type=checkbox])',
      ),
    ).map((input) => input.value);
    expect(values).toContain('1');
    expect(values).toContain('1500.5');
    expect(values).not.toContain('1.0000');
    expect(values).not.toContain('1500.5000');
  });
  expect(onDraftChange).not.toHaveBeenCalledWith(true);
});
