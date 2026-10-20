import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServicePreviewFeeQuantity } from '@/services/roncin/orderFeeService';
import FeeQuantityAssist from './FeeQuantityAssist';
import { quantitySelectionForSave } from './feeQuantitySource';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServicePreviewFeeQuantity: vi.fn(),
}));
const preview = vi.mocked(orderFeeServicePreviewFeeQuantity);

describe('空运费用显式重量来源', () => {
  beforeEach(() => {
    preview.mockReset();
  });
  it('毛重与确认计费重显式选择，携带确认 ID/版本，实际和计划来源互不回退', async () => {
    const onConfirm = vi.fn();
    preview.mockImplementation(async ({ quantityBasis, source }) => ({
      data:
        quantityBasis === 'CONFIRMED_CHARGEABLE_WEIGHT'
          ? source === 'ACTUAL'
            ? {
                available: true,
                quantity: '130',
                sourceLabel: '本票实际确认计费重',
                quantitySource: {
                  mode: 'DIRECT',
                  confirmationId: 'confirmation',
                  confirmationVersion: '2',
                },
              }
            : { available: false, reason: '尚未确认委托计费重' }
          : { available: true, quantity: '100', sourceLabel: '实际毛重' },
    }));
    renderWithApp(
      <FeeQuantityAssist
        orderId="order"
        billingUnit={{ id: 'kg', code: 'KG', name: '公斤' }}
        onConfirm={onConfirm}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: '从订单带入' }));
    await screen.findByText('100 公斤');
    fireEvent.click(screen.getByLabelText('本票当前确认计费重'));
    await screen.findByText('130 公斤');
    fireEvent.click(screen.getByLabelText('订单委托货量确认重量'));
    await screen.findByText('尚未确认委托计费重');
    expect(screen.getByRole('button', { name: '确认带入' })).toBeDisabled();
    fireEvent.click(screen.getByLabelText('实际货物明细确认重量'));
    await screen.findByText('130 公斤');
    fireEvent.click(screen.getByRole('button', { name: '确认带入' }));
    expect(onConfirm).toHaveBeenCalledWith(
      '130',
      expect.objectContaining({
        mode: 'DIRECT',
        confirmationId: 'confirmation',
        confirmationVersion: '2',
        referenceQuantity: '130',
      }),
    );
  });

  it('手改 130→135 切换人工调整并保留参考，纯手工不冒用来源', () => {
    const source = {
      mode: 'DIRECT',
      confirmationId: 'c',
      confirmationVersion: '1',
      referenceQuantity: '130',
      reason: '客户确认',
    };
    expect(quantitySelectionForSave(source, '135')).toEqual({
      mode: 'MANUAL_ADJUSTED',
      confirmationId: 'c',
      confirmationVersion: '1',
      reason: '客户确认',
    });
    expect(quantitySelectionForSave(source, '130.000')).toMatchObject({
      mode: 'DIRECT',
    });
    expect(quantitySelectionForSave(undefined, '120')).toBeUndefined();
  });
});
