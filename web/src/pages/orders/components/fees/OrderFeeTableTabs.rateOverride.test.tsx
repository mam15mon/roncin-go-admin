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

function usdFee(id: string, extra: Partial<API.OrderFee> = {}) {
  return makeFee({
    id,
    direction: RECEIVABLE,
    version: '2',
    canEdit: true,
    canDelete: true,
    feeSettingId: 'setting-of',
    settlementPartyId: 'party-old',
    billingUnitId: 'unit-piao',
    billingUnit: '票',
    quantity: '1',
    unitPrice: '100',
    totalAmount: '100.00000000',
    currency: 'USD',
    exchangeRate: '6.66230000',
    exchangeRateSource: 'SYSTEM',
    baseCurrency: 'CNY',
    baseCurrencyAmount: '666.23000000',
    taxRate: '6.00',
    taxInclusive: true,
    note: '',
    ...extra,
  });
}

const rowOf = (key: string) =>
  document.querySelector(`tr[data-row-key="${key}"]`) as HTMLElement;
const rateInput = (key: string) =>
  within(rowOf(key)).queryByLabelText('手工汇率');
const clickSave = async () => {
  await act(async () => {
    fireEvent.click(screen.getByText('保存应收费用'));
  });
};

describe('OrderFeeTableTabs 行内手工汇率', () => {
  beforeEach(() => {
    resetBulkActionMocks();
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
  });

  it('具备汇率覆盖权限：外币可编辑行留空不携带，填写后预览折本币并提交手工汇率；已建账行只读', async () => {
    listFees.mockResolvedValue({
      data: [
        usdFee('r1'),
        usdFee('billed', { canEdit: false, hasActiveBill: true }),
      ],
    } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: true });
    await waitFor(() => expect(rateInput('r1')).not.toBeNull());
    const input = rateInput('r1') as HTMLInputElement;
    expect(input).toHaveValue('');
    expect(input).toHaveAttribute('placeholder', '6.6623 系统');
    expect(input).toBeEnabled();
    expect(rateInput('billed')).toBeNull();
    expect(within(rowOf('billed')).getByText('6.6623')).toBeInTheDocument();

    // 只改备注：汇率留空，不得携带 exchangeRateOverride。
    fireEvent.change(within(rowOf('r1')).getByPlaceholderText('备注（可选）'), {
      target: { value: '只改备注' },
    });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows[0]).not.toHaveProperty(
      'exchangeRateOverride',
    );

    // 格式错误就地拦截，不发请求。
    fireEvent.change(input, { target: { value: '7.12345' } });
    await clickSave();
    await waitFor(() =>
      expect(input.closest('.ant-form-item-has-error')).not.toBeNull(),
    );
    expect(saveFees).toHaveBeenCalledOnce();

    // 有效手工汇率：折本币即时按手工汇率预览，保存携带规范化文本。
    fireEvent.change(input, { target: { value: '7.1000' } });
    expect(await within(rowOf('r1')).findByText('710.00')).toBeInTheDocument();
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
    expect(saveFees.mock.calls[1][1].rows[0]).toMatchObject({
      clientRowId: 'r1',
      exchangeRateOverride: '7.1',
    });
  });

  it('已存手工汇率可改回系统汇率：预览系统汇率并提交 useSystemExchangeRate，填写手工汇率即撤销', async () => {
    resolveRate.mockResolvedValue({ exchangeRate: '6.6623' } as never);
    listFees.mockResolvedValue({
      data: [
        usdFee('manual', {
          exchangeRate: '7.00000000',
          exchangeRateSource: 'MANUAL',
          baseCurrencyAmount: '700.00000000',
        }),
        usdFee('system'),
      ],
    } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: true });
    await waitFor(() => expect(rateInput('manual')).not.toBeNull());
    expect(rateInput('manual')).toHaveAttribute('placeholder', '7 手工');
    // 系统来源的行没有改回入口。
    expect(
      within(rowOf('system')).queryByRole('button', { name: '改回系统汇率' }),
    ).toBeNull();

    await act(async () => {
      fireEvent.click(
        within(rowOf('manual')).getByRole('button', { name: '改回系统汇率' }),
      );
    });
    await waitFor(() =>
      expect(rateInput('manual')).toHaveAttribute('placeholder', '6.6623 系统'),
    );
    expect(within(rowOf('manual')).getByText('666.23')).toBeInTheDocument();
    expect(
      within(rowOf('manual')).getByRole('button', { name: '撤销改回系统汇率' }),
    ).toBeInTheDocument();
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    const row = saveFees.mock.calls[0][1].rows[0];
    expect(row).toMatchObject({
      clientRowId: 'manual',
      useSystemExchangeRate: true,
    });
    expect(row).not.toHaveProperty('exchangeRateOverride');

    // 仍处于改回状态时填写手工汇率：自动放弃改回系统汇率，只提交手工汇率。
    expect(
      within(rowOf('manual')).getByRole('button', { name: '撤销改回系统汇率' }),
    ).toBeInTheDocument();
    fireEvent.change(rateInput('manual') as HTMLElement, {
      target: { value: '7.2' },
    });
    await waitFor(() =>
      expect(
        within(rowOf('manual')).getByRole('button', { name: '改回系统汇率' }),
      ).toBeInTheDocument(),
    );
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
    const overridden = saveFees.mock.calls[1][1].rows[0];
    expect(overridden).toMatchObject({ exchangeRateOverride: '7.2' });
    expect(overridden).not.toHaveProperty('useSystemExchangeRate');
  });

  it('本币费用的汇率输入禁用并固定显示 1', async () => {
    listFees.mockResolvedValue({
      data: [
        usdFee('cny', {
          currency: 'CNY',
          exchangeRate: '1.00000000',
          baseCurrencyAmount: '100.00000000',
        }),
      ],
    } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: true });
    await waitFor(() => expect(rateInput('cny')).not.toBeNull());
    expect(rateInput('cny')).toBeDisabled();
    expect(rateInput('cny')).toHaveAttribute('placeholder', '1');
  });

  it('无汇率覆盖权限：汇率列只读展示，提交永不携带手工汇率', async () => {
    listFees.mockResolvedValue({ data: [usdFee('r1')] } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canOverrideFeeExchangeRate: false });
    await waitFor(() =>
      expect(
        within(rowOf('r1')).getByPlaceholderText('备注（可选）'),
      ).toBeInTheDocument(),
    );
    expect(rateInput('r1')).toBeNull();
    expect(within(rowOf('r1')).getByText('6.6623')).toBeInTheDocument();
    fireEvent.change(within(rowOf('r1')).getByPlaceholderText('备注（可选）'), {
      target: { value: '改备注' },
    });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows[0]).not.toHaveProperty(
      'exchangeRateOverride',
    );
  });
});
