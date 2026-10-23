import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  bulkUpdate,
  getDialog,
  listFees,
  makeProps,
  pickSelectOption,
  renderTabs,
  resetBulkActionMocks,
  triggerBulkAction,
  unbilledFeeA,
  unbilledFeeB,
} from './OrderFeeTableTabs.bulkActions.fixtures';

describe('OrderFeeTableTabs 批量修改单位与日期', () => {
  beforeEach(resetBulkActionMocks);

  it('批量改结算单位：提交版本快照 targets 与目标单位，成功后清空选择并刷新（A2/A3）', async () => {
    bulkUpdate.mockResolvedValueOnce({ success: true, updatedCount: 2 } as any);
    listFees.mockResolvedValue({
      data: [unbilledFeeA, unbilledFeeB],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-b', 'fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');
    // 成功后两表 reload 会重新发起费用查询
    const listCallsBefore = listFees.mock.calls.length;

    // 应收入口可用；应付表未勾选时入口禁用
    const bulkButtons = screen.getAllByRole('button', {
      name: /批量操作/,
    });
    expect(bulkButtons[0]).not.toBeDisabled();
    expect(bulkButtons[1]).toBeDisabled();

    await act(async () => {
      fireEvent.click(bulkButtons[0]);
    });
    const partyMenuItem = await screen.findByText('批量改结算单位');
    await act(async () => {
      fireEvent.click(partyMenuItem);
    });
    await waitFor(() =>
      expect(
        screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
      ).toBeInTheDocument(),
    );

    const combobox = within(getDialog()).getByRole('combobox');
    await pickSelectOption(combobox, '新结算单位');

    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    await waitFor(() =>
      expect(bulkUpdate).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [
            { feeId: 'fee-a', expectedVersion: '3' },
            { feeId: 'fee-b', expectedVersion: '1' },
          ],
          settlementPartyId: 'party-new',
        },
      ),
    );
    expect(await screen.findByText('已修改 2 笔费用的结算单位')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);
    await waitFor(() =>
      // 应收/应付两表 reload 重新拉取费用列表
      expect(listFees.mock.calls.length).toBeGreaterThan(listCallsBefore),
    );
    expect(props.onFeeSaved).toHaveBeenCalled();
  });

  it('批量改费用日期：仅选择日期并提交 YYYY-MM-DD', async () => {
    listFees.mockResolvedValue({ data: [unbilledFeeA] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await triggerBulkAction('批量改费用日期');
    await waitFor(() =>
      expect(
        screen.getByText('批量修改费用日期（已选 1 笔应收费用）'),
      ).toBeInTheDocument(),
    );

    const pickerInput = screen.getByPlaceholderText('请选择费用发生日期');
    await act(async () => {
      fireEvent.click(pickerInput);
      fireEvent.change(pickerInput, { target: { value: '2026-09-23' } });
      fireEvent.keyDown(pickerInput, { key: 'Enter' });
    });
    expect(document.querySelector('.ant-picker-time-panel')).toBeNull();

    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    await waitFor(() =>
      expect(bulkUpdate).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [{ feeId: 'fee-a', expectedVersion: '3' }],
          expenseDate: '2026-09-23',
        },
      ),
    );
  });
});
