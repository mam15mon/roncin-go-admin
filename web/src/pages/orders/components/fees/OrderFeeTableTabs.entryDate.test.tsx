import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import { formatDate } from '@/utils/format';
import {
  listFees,
  makeProps,
  renderTabs,
  resetBulkActionMocks,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(() => {
  window.localStorage.clear();
  resetBulkActionMocks();
});

it('录入日期默认隐藏，可勾选显示，编辑或新增费用时均不可填写', async () => {
  const fee = { ...unbilledFeeA, createdAt: '2026-09-20T14:30:00+08:00' };
  listFees.mockResolvedValue({ data: [fee] });
  await renderTabs(makeProps('order-1').props);
  await screen.findByText('海运费');
  expect(screen.queryByRole('columnheader', { name: '录入日期' })).toBeNull();

  fireEvent.click(screen.getAllByRole('button', { name: '列设置' })[0]);
  const dialog = await screen.findByRole('dialog', { name: '列设置' });
  const checkbox = within(dialog).getByRole('checkbox', { name: '录入日期' });
  expect(checkbox).not.toBeDisabled();
  expect(checkbox).not.toBeChecked();
  fireEvent.click(checkbox);
  await waitFor(() =>
    expect(
      screen.getAllByRole('columnheader', { name: '录入日期' }),
    ).toHaveLength(2),
  );

  fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
  await screen.findByText('保存');
  const dateCell = screen
    .getByText(formatDate(fee.createdAt, 'minute'))
    .closest('td');
  expect(dateCell).not.toBeNull();
  expect(dateCell?.querySelector('input')).toBeNull();

  fireEvent.click(screen.getByText('取消'));
  await waitFor(() =>
    expect(screen.queryByText('保存')).not.toBeInTheDocument(),
  );
  fireEvent.click(screen.getByRole('button', { name: /新增应收费用/ }));
  await screen.findByText('保存');
  const row = document.querySelector('tr[data-row-key^="new_"]');
  expect(row).not.toBeNull();
  const newDateCell = row?.querySelectorAll('td')[dateCell?.cellIndex ?? -1];
  expect(newDateCell).toHaveTextContent('-');
  expect(newDateCell?.querySelector('input')).toBeNull();
});
