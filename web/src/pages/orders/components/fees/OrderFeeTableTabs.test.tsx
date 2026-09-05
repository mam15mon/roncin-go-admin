import type { ActionType } from '@ant-design/pro-components';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

describe('OrderFeeTableTabs 业务锁策略', () => {
  beforeEach(() => {
    listFees.mockReset();
    listFees.mockResolvedValue({ items: [] } as Awaited<
      ReturnType<typeof listFees>
    >);
  });

  it('业务费用只读时禁用新增，但保留已确认费用的账单创建入口', async () => {
    const onOpenBillWorkbench = vi.fn();

    render(
      <App>
        <OrderFeeTableTabs
          orderId="order-1"
          receivableActionRef={
            { current: undefined } as React.RefObject<ActionType | undefined>
          }
          payableActionRef={
            { current: undefined } as React.RefObject<ActionType | undefined>
          }
          receivableSummary={{ totalAmount: 100, count: 1 }}
          payableSummary={{ totalAmount: 0, count: 0 }}
          selectedReceivableFeeIds={['fee-1']}
          setSelectedReceivableFeeIds={vi.fn()}
          selectedPayableFeeIds={[]}
          setSelectedPayableFeeIds={vi.fn()}
          setAllReceivableItems={vi.fn()}
          setAllPayableItems={vi.fn()}
          setReceivableSummary={vi.fn()}
          setPayableSummary={vi.fn()}
          canCreateFinanceBills
          feeWritesDisabled
          onOpenBillWorkbench={onOpenBillWorkbench}
          onOpenFeeModal={vi.fn()}
          getTableColumns={() => []}
        />
      </App>,
    );

    await waitFor(() => expect(listFees).toHaveBeenCalled());
    expect(
      screen.getByRole('button', { name: /生成账单（1）/ }),
    ).not.toBeDisabled();
    expect(screen.getByRole('button', { name: /新增应收费用/ })).toBeDisabled();
  });
});
