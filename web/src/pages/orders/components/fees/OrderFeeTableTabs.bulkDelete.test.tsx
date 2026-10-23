import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  bulkRemove,
  getDialog,
  listFees,
  makeProps,
  renderTabs,
  resetBulkActionMocks,
  triggerBulkAction,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';

describe('OrderFeeTableTabs 批量删除', () => {
  beforeEach(resetBulkActionMocks);

  it('只有删除权限也能批量删除，确认笔数及删除原因随请求传递', async () => {
    bulkRemove.mockResolvedValueOnce({ success: true, removedCount: 1 } as any);
    listFees.mockResolvedValue({
      data: [{ ...unbilledFeeA, canEdit: false, canDelete: true }],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await triggerBulkAction('批量删除');
    await waitFor(() =>
      expect(
        screen.getByText('批量删除费用（已选 1 笔应收费用）'),
      ).toBeInTheDocument(),
    );
    expect(screen.getByText('将删除 1 笔应收费用')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/删除原因/), {
      target: { value: '客户取消订单' },
    });
    await act(async () => {
      within(getDialog())
        .getByRole('button', { name: /删\s*除/ })
        .click();
    });

    await waitFor(() =>
      expect(bulkRemove).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [{ feeId: 'fee-a', expectedVersion: '3' }],
          reason: '客户取消订单',
        },
      ),
    );
    expect(await screen.findByText('已删除 1 笔费用')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);
  });
});
