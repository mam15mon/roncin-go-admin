import type { ActionType } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type React from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
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
      canCreateFees
      canDeleteFees
      canCreateFinanceBills={false}
      feeWritesDisabled={false}
      onOpenBillWorkbench={vi.fn()}
      billingUnits={[{ id: 'unit', name: '立方米', code: 'CBM' }]}
      columnSettingScope={{ userId: 'user', organizationId: 'org' }}
    />
  );
}

beforeEach(() => {
  vi.mocked(orderFeeServiceListFees).mockResolvedValue({ data: [] });
  vi.mocked(orderFeeServiceResolveFeeExchangeRate).mockResolvedValue({
    exchangeRate: '1',
    exchangeRateSource: 'SYSTEM',
  });
});

it.each(['应收', '应付'])(
  '新增%s费用后直接输入费用名称，数量和单价可全选覆盖，辅助入口不占 Tab',
  async (direction) => {
    renderWithApp(table());
    await waitFor(() => expect(orderFeeServiceListFees).toHaveBeenCalled());
    fireEvent.click(
      screen.getByRole('button', { name: new RegExp(`新增${direction}费用`) }),
    );
    const [feeName] = await screen.findAllByRole('combobox');
    await waitFor(() => expect(feeName).toHaveFocus());

    const quantity = screen.getByPlaceholderText('1') as HTMLInputElement;
    const price = screen.getByPlaceholderText('0.00') as HTMLInputElement;
    for (const [input, value] of [
      [quantity, '12.5'],
      [price, '128.50'],
    ] as const) {
      fireEvent.change(input, { target: { value } });
      act(() => input.focus());
      expect(input.selectionStart).toBe(0);
      expect(input.selectionEnd).toBe(value.length);
    }
    expect(screen.getByRole('button', { name: '从订单带入' })).toHaveAttribute(
      'tabindex',
      '-1',
    );
    const row = quantity.closest('tr');
    if (!row) throw new Error('费用编辑行不存在');
    const fields = Array.from(
      row.querySelectorAll<HTMLInputElement>(
        'input:not([type="hidden"]):not([type="checkbox"])',
      ),
    );
    for (const field of fields) {
      vi.spyOn(field, 'getClientRects').mockReturnValue({
        length: 1,
      } as DOMRectList);
    }
    // 按当前列顺序跳转，不落到下拉框清空或数量带入按钮。
    const editableFields = fields.filter(
      (field) => !field.disabled && field.tabIndex >= 0,
    );
    for (let index = 0; index < editableFields.length - 1; index += 1) {
      const current = editableFields[index];
      const next = editableFields[index + 1];
      act(() => current.focus());
      fireEvent.keyDown(current, { key: 'Tab' });
      expect(next).toHaveFocus();
      fireEvent.keyDown(next, { key: 'Tab', shiftKey: true });
      expect(current).toHaveFocus();
    }
    await act(async () => {});
  },
);
