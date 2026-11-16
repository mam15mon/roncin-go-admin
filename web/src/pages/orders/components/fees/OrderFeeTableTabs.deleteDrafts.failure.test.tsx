import { screen, waitFor, within } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import {
  billedFeeC,
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
it('远端删除失败时保留混选的本地新增行和选择', async () => {
  listFees.mockResolvedValue({ data: [unbilledFeeA] } as any);
  const { row, select } = await renderAndAdd();
  await select(
    document.querySelector('tr[data-row-key="fee-a"]') as HTMLElement,
  );
  bulkRemove.mockRejectedValueOnce(new Error('删除失败'));
  await submitDelete();
  await waitFor(() => expect(bulkRemove).toHaveBeenCalledTimes(1));
  expect(screen.getByRole('dialog')).toBeVisible();
  expect(row).toBeInTheDocument();
  expect(within(row).getByRole('checkbox')).toBeChecked();
});

it('混选已建账费用时不删除远端费用或本地新增行', async () => {
  listFees.mockResolvedValue({ data: [billedFeeC] } as any);
  const { row, select } = await renderAndAdd();
  await select(
    document.querySelector('tr[data-row-key="fee-c"]') as HTMLElement,
  );
  await submitDelete();
  expect(screen.getByRole('dialog')).toBeVisible();
  expect(
    document.querySelector('tr[data-row-key="fee-c"]'),
  ).toBeInTheDocument();
  expect(bulkRemove).not.toHaveBeenCalled();
  expect(row).toBeInTheDocument();
});
