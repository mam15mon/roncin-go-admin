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
  pickSelectOption,
  renderTabs,
  resetBulkActionMocks,
  saveFees,
} from './OrderFeeTableTabs.bulkActions.fixtures';

function taxedFee(id: string, extra: Partial<API.OrderFee> = {}) {
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
    unitPrice: '106',
    taxRate: '6.00',
    taxInclusive: true,
    note: '',
    ...extra,
  });
}

const rowOf = (key: string) =>
  document.querySelector(`tr[data-row-key="${key}"]`) as HTMLElement;
const taxInput = (key: string) =>
  within(rowOf(key)).queryByPlaceholderText('默认取目录税率');
const clickSave = async () => {
  await act(async () => {
    fireEvent.click(screen.getByText('保存应收费用'));
  });
};

describe('OrderFeeTableTabs 税率列', () => {
  beforeEach(() => {
    resetBulkActionMocks();
    saveFees.mockResolvedValue({ success: true, data: [] } as never);
  });

  it('具备税率更改权限：未建账行可编辑税率，已建账行只读；改税率才携带 taxRate', async () => {
    listFees.mockResolvedValue({
      data: [
        taxedFee('r1'),
        taxedFee('billed', { canEdit: false, hasActiveBill: true }),
      ],
    } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canEditFeeTaxRate: true });
    await waitFor(() => expect(taxInput('r1')).not.toBeNull());
    expect(taxInput('r1')).toHaveValue('6.00');
    expect(taxInput('billed')).toBeNull();
    expect(within(rowOf('billed')).getByText('6%')).toBeInTheDocument();

    // 只改备注：税率输入未变，不得携带 taxRate。
    fireEvent.change(within(rowOf('r1')).getByPlaceholderText('备注（可选）'), {
      target: { value: '只改备注' },
    });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows[0]).not.toHaveProperty('taxRate');

    // 尾零不同视为同值（6 与 6.00）。
    fireEvent.change(taxInput('r1') as HTMLElement, { target: { value: '6' } });
    fireEvent.change(within(rowOf('r1')).getByPlaceholderText('备注（可选）'), {
      target: { value: '再改备注' },
    });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
    expect(saveFees.mock.calls[1][1].rows[0]).not.toHaveProperty('taxRate');

    // 真正修改税率。
    fireEvent.change(taxInput('r1') as HTMLElement, { target: { value: '9' } });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(3));
    expect(saveFees.mock.calls[2][1].rows[0]).toMatchObject({
      clientRowId: 'r1',
      taxRate: '9',
    });
  });

  it('无税率更改权限：税率列只读展示，提交永不携带 taxRate', async () => {
    listFees.mockResolvedValue({ data: [taxedFee('r1')] } as never);
    const { props } = makeProps('order-1');
    await renderTabs({ ...props, canEditFeeTaxRate: false });
    await waitFor(() =>
      expect(
        within(rowOf('r1')).getByPlaceholderText('备注（可选）'),
      ).toBeInTheDocument(),
    );
    expect(taxInput('r1')).toBeNull();
    expect(within(rowOf('r1')).getByText('6%')).toBeInTheDocument();
    fireEvent.change(within(rowOf('r1')).getByPlaceholderText('备注（可选）'), {
      target: { value: '改备注' },
    });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows[0]).not.toHaveProperty('taxRate');
  });

  it('新增行：税率与目录税率相同时不携带 taxRate', async () => {
    listFees.mockResolvedValue({ data: [] } as never);
    const { props } = makeProps('order-1');
    await renderTabs({
      ...props,
      canEditFeeTaxRate: true,
      feeSettings: [
        { id: 'setting-of', feeCode: 'OF', nameZh: '海运费', taxRate: '6.00' },
      ],
      order: { id: 'order-1', customerId: 'party-old' } as API.Order,
      customerName: '原结算单位',
    });
    await act(async () => {
      fireEvent.click(screen.getByText('新增应收费用'));
    });
    await waitFor(() =>
      expect(document.querySelector('tr[data-row-key^="new_"]')).not.toBeNull(),
    );
    const newRow = document.querySelector(
      'tr[data-row-key^="new_"]',
    ) as HTMLElement;
    await pickSelectOption(
      within(newRow).getAllByRole('combobox')[0],
      '海运费 (OF)',
    );
    fireEvent.change(within(newRow).getByPlaceholderText('0.00'), {
      target: { value: '50' },
    });
    const tax = within(newRow).getByPlaceholderText('默认取目录税率');

    fireEvent.change(tax, { target: { value: '6' } });
    await clickSave();
    await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
    expect(saveFees.mock.calls[0][1].rows[0]).not.toHaveProperty('taxRate');
  });
});
