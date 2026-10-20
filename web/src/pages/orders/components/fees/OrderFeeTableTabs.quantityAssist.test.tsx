import type { ActionType } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServicePreviewFeeQuantity,
} from '@/services/roncin/orderFeeService';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServicePreviewFeeQuantity: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

function table() {
  return (
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
      selectedReceivableFeeIds={[]}
      selectedPayableFeeIds={[]}
      setSelectedReceivableFeeIds={vi.fn()}
      setSelectedPayableFeeIds={vi.fn()}
      setAllReceivableItems={vi.fn()}
      setAllPayableItems={vi.fn()}
      setReceivableSummary={vi.fn()}
      setPayableSummary={vi.fn()}
      canCreateFinanceBills={false}
      feeWritesDisabled={false}
      onOpenBillWorkbench={vi.fn()}
      billingUnits={[{ id: 'unit', name: '立方米', code: 'CBM' }]}
      columnSettingScope={{ userId: 'user', organizationId: 'org' }}
    />
  );
}

describe('行内费用数量带入', () => {
  beforeEach(() => {
    vi.mocked(orderFeeServiceListFees).mockResolvedValue({
      data: [
        {
          id: 'fee-1',
          direction: 1,
          quantity: '1',
          unitPrice: '100',
          totalAmount: '100',
          currency: 'CNY',
          billingUnitId: 'unit',
          billingUnit: '立方米',
          feeName: '拼箱费用',
          version: '1',
        },
      ],
    });
    vi.mocked(orderFeeServicePreviewFeeQuantity).mockResolvedValue({
      data: {
        available: true,
        quantity: '8',
        sourceLabel: '本订单实际货物明细',
        formula: '体积合计',
      },
    });
  });

  it('真实可编辑数量字段确认后写入数量并即时更新总金额，取消编辑后销毁入口', async () => {
    renderWithApp(table());
    await screen.findByText('拼箱费用');
    fireEvent.click(screen.getByRole('button', { name: /编\s*辑/ }));
    const fromOrder = await screen.findByRole('button', { name: '从订单带入' });
    fireEvent.click(fromOrder);
    await screen.findByText('8 立方米');
    fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
    const row = screen.getByText('保存').closest('tr');
    expect(row).not.toBeNull();
    expect(
      within(row as HTMLElement).getByDisplayValue('8'),
    ).toBeInTheDocument();
    expect(
      await within(row as HTMLElement).findByText('800.00 CNY'),
    ).toBeInTheDocument();
    await act(async () => {
      fireEvent.click(within(row as HTMLElement).getByText('取消'));
    });
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: '从订单带入' }),
      ).not.toBeInTheDocument(),
    );
  });
  it('已建账费用即使进入现有编辑态也不能使用数量带入', async () => {
    vi.mocked(orderFeeServiceListFees).mockResolvedValue({
      data: [
        {
          id: 'fee-billed',
          direction: 1,
          quantity: '1',
          unitPrice: '100',
          currency: 'CNY',
          billingUnitId: 'unit',
          feeName: '已建账费用',
          hasActiveBill: true,
        },
      ],
    });
    renderWithApp(table());
    await screen.findByText('已建账费用');
    fireEvent.click(screen.getByRole('button', { name: /编\s*辑/ }));
    await screen.findByText('保存');
    expect(
      screen.queryByRole('button', { name: '从订单带入' }),
    ).not.toBeInTheDocument();
  });
});
