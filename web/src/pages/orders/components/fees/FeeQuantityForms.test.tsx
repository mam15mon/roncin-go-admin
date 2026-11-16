import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import {
  orderFeeServicePreviewFeeQuantity,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import FeeSupplementModal from './FeeSupplementModal';

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOverrideFeeExchangeRate: false }),
}));
vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServicePreviewFeeQuantity: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const options = {
  feeSettings: [
    {
      id: 'fs',
      feeCode: 'LCL',
      nameZh: '拼箱运费',
      defaultBillingUnitId: 'unit',
      defaultCurrency: 'CNY',
    },
  ],
  settlementParties: [{ id: 'sp', name: '结算客户' }],
  currencies: [{ code: 'CNY', name: '人民币' }],
  billingUnits: [{ id: 'unit', name: '计费吨', code: 'RT' }],
};

async function confirmQuantity() {
  fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
  await screen.findByText('8 计费吨');
  fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
}

describe('费用数量带入表单入口', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
    vi.mocked(orderFeeServicePreviewFeeQuantity).mockResolvedValue({
      data: {
        available: true,
        quantity: '8',
        sourceLabel: '本订单实际货物明细',
        formula: 'max(8, 2500 / 1000)',
      },
    });
    vi.mocked(orderFeeServiceResolveFeeExchangeRate).mockResolvedValue({
      exchangeRate: '1',
      exchangeRateSource: 'SYSTEM',
    });
  });

  it('统一费用补录确认即时重算金额', async () => {
    renderWithApp(
      <FeeSupplementModal
        {...options}
        orderId="order-1"
        open
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        initialRequest={{
          id: 'request',
          billingUnitId: 'unit',
          quantity: '1',
          unitPrice: '100',
          currency: 'CNY',
        }}
      />,
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();
    await confirmQuantity();
    const switchOrganization = vi.fn();
    expect(confirmIfAnyTabDirty(switchOrganization, vi.fn())).toBe(false);
    expect(switchOrganization).not.toHaveBeenCalled();
    expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
    expect(screen.getByLabelText('数量')).toHaveValue('8');
    expect(await screen.findByText(/800\.00/)).toBeInTheDocument();
  });
});
