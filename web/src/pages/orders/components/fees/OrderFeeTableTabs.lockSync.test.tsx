import type { ActionType } from '@ant-design/pro-components';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

const emptySummary = {
  count: 0,
  gross: '0',
  net: '0',
  unconvertedCount: 0,
  currencies: [],
};

const refs = {
  receivable: { current: undefined } as React.RefObject<ActionType | undefined>,
  payable: { current: undefined } as React.RefObject<ActionType | undefined>,
};

function tabs(lock: { feeWritesDisabled: boolean; feeRowsLocked?: boolean }) {
  return (
    <App>
      <OrderFeeTableTabs
        orderId="order-1"
        receivableActionRef={refs.receivable}
        payableActionRef={refs.payable}
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
        canCreateFinanceBills={false}
        canCreateFees
        canUpdateFees
        canDeleteFees
        onOpenBillWorkbench={vi.fn()}
        {...lock}
      />
    </App>
  );
}

const rowInputs = (container: HTMLElement) =>
  container.querySelectorAll(
    'tr[data-row-key="recv-1"] input:not([type="checkbox"])',
  );

describe('OrderFeeTableTabs 审核状态后台同步', () => {
  beforeEach(() => {
    listFees.mockReset();
    listFees.mockResolvedValue({
      data: [
        {
          id: 'recv-1',
          direction: RECEIVABLE,
          canEdit: true,
          version: '1',
          feeCode: 'THC',
          feeName: '码头操作费',
          currency: 'CNY',
          quantity: '1',
          unitPrice: '1500',
          totalAmount: '1500',
          baseCurrencyAmount: '1500',
        },
      ],
    } as Awaited<ReturnType<typeof listFees>>);
  });

  it('保存后审核状态重读期间行内输入保持展开，仅提交入口失败关闭', async () => {
    const { container, rerender } = render(
      tabs({ feeWritesDisabled: false, feeRowsLocked: false }),
    );
    await waitFor(() => expect(rowInputs(container).length).toBeGreaterThan(0));
    const before = rowInputs(container).length;

    // 后台重读：写策略失败关闭，但行展示沿用最近确认的未锁定状态。
    rerender(tabs({ feeWritesDisabled: true, feeRowsLocked: false }));
    expect(rowInputs(container)).toHaveLength(before);
    expect(screen.getByRole('button', { name: /保存应收费用/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /新增应收费用/ })).toBeEnabled();

    rerender(tabs({ feeWritesDisabled: false, feeRowsLocked: false }));
    expect(rowInputs(container)).toHaveLength(before);
  });

  it('确认不可写时收起行内输入', async () => {
    const { container, rerender } = render(
      tabs({ feeWritesDisabled: false, feeRowsLocked: false }),
    );
    await waitFor(() => expect(rowInputs(container).length).toBeGreaterThan(0));

    rerender(tabs({ feeWritesDisabled: true, feeRowsLocked: true }));
    await waitFor(() => expect(rowInputs(container)).toHaveLength(0));
    expect(screen.getByRole('button', { name: /新增应收费用/ })).toBeDisabled();
  });
});
