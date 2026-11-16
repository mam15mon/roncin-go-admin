import { act, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import {
  bulkRemove,
  listFees,
  resetBulkActionMocks,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';
import {
  renderAndAdd,
  submitDelete,
} from './OrderFeeTableTabs.deleteDrafts.fixtures';

beforeEach(resetBulkActionMocks);
it('无费用删除权限也可勾选移除本地新增行，不请求远端且不刷新财务投影', async () => {
  const { props, row } = await renderAndAdd(false);
  await submitDelete();
  await waitFor(() => expect(row).not.toBeInTheDocument());
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  );
  await act(async () => {});
  expect(bulkRemove).not.toHaveBeenCalled();
  expect(props.onFeeSaved).not.toHaveBeenCalled();
});

it('混选已保存和本地行只向远端提交真实费用，成功后一起移除', async () => {
  listFees.mockResolvedValue({ data: [unbilledFeeA] } as any);
  const { row, select } = await renderAndAdd();
  await select(
    document.querySelector('tr[data-row-key="fee-a"]') as HTMLElement,
  );
  bulkRemove.mockImplementationOnce(async () => {
    listFees.mockResolvedValue({ data: [] } as any);
    return { success: true, removedCount: 1 } as any;
  });
  await submitDelete();
  await waitFor(() => expect(row).not.toBeInTheDocument());
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  );
  await act(async () => {});
  expect(bulkRemove).toHaveBeenCalledWith(
    { orderId: 'order-1' },
    {
      orderId: 'order-1',
      targets: [{ feeId: 'fee-a', expectedVersion: '3' }],
      reason: undefined,
    },
  );
  expect(document.querySelector('tr[data-row-key="fee-a"]')).toBeNull();
});
