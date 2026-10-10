import { act, screen, waitFor, within } from '@testing-library/react';
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

describe('OrderFeeTableTabs 批量修改版本冲突', () => {
  beforeEach(resetBulkActionMocks);

  it('版本冲突整批失败：展示后端原因，保留选中且不刷新（A3）', async () => {
    bulkUpdate.mockRejectedValueOnce(
      new Error('费用 fee-a：已被更新，请刷新后重试'),
    );
    listFees.mockResolvedValue({ data: [unbilledFeeA, unbilledFeeB] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-b'];
    await renderTabs(props);
    await screen.findByText('海运费');
    const listCallsBefore = listFees.mock.calls.length;

    await triggerBulkAction('批量改结算单位');
    await waitFor(() =>
      expect(
        screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
      ).toBeInTheDocument(),
    );
    await pickSelectOption(
      within(getDialog()).getByRole('combobox'),
      '新结算单位',
    );
    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    expect(
      await screen.findByText(/费用 fee-a：已被更新，请刷新后重试/),
    ).toBeVisible();
    // 失败：选择保留供核对，不触发刷新链路
    expect(props.setSelectedReceivableFeeIds).not.toHaveBeenCalledWith([]);
    expect(listFees.mock.calls.length).toBe(listCallsBefore);
    expect(props.onFeeSaved).not.toHaveBeenCalled();
    // 弹窗保持打开，用户可取消或调整后重试
    expect(
      screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
    ).toBeInTheDocument();
  });
});
