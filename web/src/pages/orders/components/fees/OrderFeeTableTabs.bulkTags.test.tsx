import { act, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  batchAssignTags,
  batchRemoveTags,
  billedFeeC,
  getDialog,
  listFees,
  listTagOptions,
  makeProps,
  pickSelectOption,
  renderTabs,
  resetBulkActionMocks,
  triggerBulkAction,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';

describe('OrderFeeTableTabs 批量标签', () => {
  beforeEach(resetBulkActionMocks);

  it('批量标签：添加/移除两个入口按选中行提交，已建账行可参与（A2）', async () => {
    listFees.mockResolvedValue({
      data: [unbilledFeeA, billedFeeC],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-c'];
    await renderTabs(props);
    await screen.findByText('海运费');

    // 应收入口可用；应付表未勾选时入口禁用
    const bulkButtons = screen.getAllByRole('button', {
      name: /批量操作/,
    });
    expect(bulkButtons[0]).not.toBeDisabled();
    expect(bulkButtons[1]).toBeDisabled();

    await triggerBulkAction('添加标签');
    const dialog = getDialog();
    await waitFor(() =>
      expect(within(dialog).getByText('已选 2 个对象')).toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(listTagOptions).toHaveBeenCalledWith(
        expect.objectContaining({ orderId: 'order-1', page: 1, pageSize: 50 }),
      ),
    );

    await pickSelectOption(within(dialog).getByRole('combobox'), '重点单');
    await act(async () => {
      within(dialog)
        .getByRole('button', { name: /添加标签/ })
        .click();
    });

    await waitFor(() =>
      expect(batchAssignTags).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          feeIds: ['fee-a', 'fee-c'],
          tagIds: ['tag-1'],
        },
      ),
    );
    expect(await screen.findByText('已为 2 笔费用添加标签')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);

    // 「删除标签」入口以移除模式重新打开同一弹窗（重开可能重建内容节点，
    // 断言时需重查 dialog，避免持有失联的旧节点）
    await triggerBulkAction('删除标签');
    await waitFor(() => {
      const okButton = within(getDialog()).getByRole('button', {
        name: /移除标签/,
      }) as HTMLButtonElement;
      // 等上一次提交的 loading 态完全复位，避免点击被吞
      expect(okButton.className).not.toContain('ant-btn-loading');
    });
    expect(
      within(getDialog()).queryByRole('button', { name: /添加标签/ }),
    ).not.toBeInTheDocument();

    await pickSelectOption(within(getDialog()).getByRole('combobox'), '重点单');
    await act(async () => {
      within(getDialog())
        .getByRole('button', { name: /移除标签/ })
        .click();
    });

    await waitFor(() =>
      expect(batchRemoveTags).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          feeIds: ['fee-a', 'fee-c'],
          tagIds: ['tag-1'],
        },
      ),
    );
  });
});
