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
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
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
      receivableSummary={{
        count: 1,
        gross: '100',
        net: '100',
        unconvertedCount: 0,
        currencies: [],
      }}
      payableSummary={{
        count: 0,
        gross: '0',
        net: '0',
        unconvertedCount: 0,
        currencies: [],
      }}
      selectedReceivableFeeIds={[]}
      selectedPayableFeeIds={[]}
      setSelectedReceivableFeeIds={vi.fn()}
      setSelectedPayableFeeIds={vi.fn()}
      setAllReceivableItems={vi.fn()}
      setAllPayableItems={vi.fn()}
      setReceivableSummary={vi.fn()}
      setPayableSummary={vi.fn()}
      canCreateFees
      canUpdateFees
      canDeleteFees
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
    window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
    vi.mocked(orderFeeServiceListFees).mockResolvedValue({
      data: [
        {
          canEdit: true,
          canBill: true,
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

  it.each([1, 2])(
    '方向%s确认后同步标脏并重算金额，组织切换可取消',
    async (direction) => {
      vi.mocked(orderFeeServiceListFees).mockResolvedValue({
        data: [
          {
            canEdit: true,
            canBill: true,
            id: 'fee-1',
            direction,
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
      renderWithApp(table());
      // 已保存行常开编辑，直接出现数量带入入口。
      const fromOrder = await screen.findByRole('button', {
        name: '从订单带入',
      });
      fireEvent.click(fromOrder);
      await screen.findByText('8 立方米');
      expect(getAnyDirtyTabKey()).toBeUndefined();
      fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
      const switchOrganization = vi.fn();
      const confirmation = vi.fn();
      expect(confirmIfAnyTabDirty(switchOrganization, confirmation)).toBe(
        false,
      );
      expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
      confirmation.mock.calls[0][0].onCancel?.();
      expect(switchOrganization).not.toHaveBeenCalled();
      const row = document.querySelector('tr[data-row-key="fee-1"]');
      expect(row).not.toBeNull();
      expect(
        within(row as HTMLElement).getByDisplayValue('8'),
      ).toBeInTheDocument();
      expect(
        await within(row as HTMLElement).findByText('800.00 CNY'),
      ).toBeInTheDocument();
    },
  );
  it('已建账费用不开放编辑与数量带入', async () => {
    vi.mocked(orderFeeServiceListFees).mockResolvedValue({
      data: [
        {
          canEdit: false,
          canBill: false,
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
    expect(
      screen.queryByRole('button', { name: /编\s*辑/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '从订单带入' }),
    ).not.toBeInTheDocument();
  });
});
