import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { RECEIVABLE } from './feeConstants';
import {
  listFees,
  makeFee,
  makeProps,
  renderTabs,
  resetBulkActionMocks,
  resolveRate,
  saveFees,
} from './OrderFeeTableTabs.bulkActions.fixtures';

function manualFee(id: string): API.OrderFee {
  return makeFee({
    id,
    direction: RECEIVABLE,
    version: '2',
    canEdit: true,
    feeSettingId: 'setting-of',
    settlementPartyId: 'party-old',
    billingUnitId: 'unit-piao',
    currency: 'USD',
    exchangeRate: '7',
    exchangeRateSource: 'MANUAL',
    baseCurrency: 'CNY',
    baseCurrencyAmount: '700',
    totalAmount: '100',
    taxRate: '0',
  });
}

function row(id: string) {
  const element = document.querySelector(`tr[data-row-key="${id}"]`);
  if (!(element instanceof HTMLElement)) throw new Error('费用行未渲染');
  return within(element);
}

beforeEach(() => {
  resetBulkActionMocks();
  resolveRate.mockResolvedValue({ exchangeRate: '6.6' } as never);
});

describe('费用行系统汇率 Query', () => {
  it('两行同方向同币种共用一次解析，同时保留各行独立的改回意图', async () => {
    listFees.mockResolvedValue({
      data: [manualFee('first'), manualFee('second')],
    } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: true });
    await waitFor(() =>
      expect(
        row('second').getByRole('button', { name: '改回系统汇率' }),
      ).toBeVisible(),
    );
    await act(async () => {
      fireEvent.click(
        row('first').getByRole('button', { name: '改回系统汇率' }),
      );
      fireEvent.click(
        row('second').getByRole('button', { name: '改回系统汇率' }),
      );
    });
    await waitFor(() => {
      expect(row('first').getByLabelText('手工汇率')).toHaveAttribute(
        'placeholder',
        '6.6 系统',
      );
      expect(row('second').getByLabelText('手工汇率')).toHaveAttribute(
        'placeholder',
        '6.6 系统',
      );
    });
    expect(resolveRate).toHaveBeenCalledOnce();
    expect(resolveRate).toHaveBeenCalledWith(
      { orderId: 'order-1', direction: RECEIVABLE, currency: 'USD' },
      expect.objectContaining({ skipErrorHandler: true }),
    );
  });

  it('成功保存并采纳新版本后清除旧参考汇率与改回意图', async () => {
    listFees.mockResolvedValue({ data: [manualFee('fee')] } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: true });
    await waitFor(() =>
      expect(
        row('fee').getByRole('button', { name: '改回系统汇率' }),
      ).toBeVisible(),
    );
    await act(async () =>
      fireEvent.click(row('fee').getByRole('button', { name: '改回系统汇率' })),
    );
    await waitFor(() =>
      expect(row('fee').getByLabelText('手工汇率')).toHaveAttribute(
        'placeholder',
        '6.6 系统',
      ),
    );
    saveFees.mockImplementation(async () => {
      listFees.mockResolvedValue({
        data: [
          {
            ...manualFee('fee'),
            version: '3',
            exchangeRate: '6.5',
            exchangeRateSource: 'SYSTEM',
            baseCurrencyAmount: '650',
          },
        ],
      } as never);
      return { success: true, data: [], feeContentVersion: '3' };
    });
    await act(async () => fireEvent.click(screen.getByText('保存应收费用')));
    await waitFor(() => {
      expect(saveFees).toHaveBeenCalledOnce();
      expect(row('fee').getByLabelText('手工汇率')).toHaveAttribute(
        'placeholder',
        '6.5 系统',
      );
      expect(
        row('fee').queryByRole('button', { name: '撤销改回系统汇率' }),
      ).not.toBeInTheDocument();
    });
    expect(row('fee').queryByText('660.00')).not.toBeInTheDocument();
    expect(row('fee').getByText('650.00')).toBeInTheDocument();
  });
});
