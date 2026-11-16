import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  bulkRemove,
  listFees,
  makeProps,
  renderTabs,
  resetBulkActionMocks,
  unbilledFeeA,
} from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(resetBulkActionMocks);

it('移除本地新增行不重读远端，保留同侧其他费用的输入与原版本', async () => {
  listFees.mockResolvedValue({
    data: [{ ...unbilledFeeA, note: '原备注' }],
  } as any);
  const { props: baseProps } = makeProps('order-1');
  const props = { ...baseProps, onDraftChange: vi.fn() };
  const { rerenderTabs } = await renderTabs(props, {
    wrapper: ({ children }) => (
      <ConfigProvider theme={{ token: { motion: false } }}>
        {children}
      </ConfigProvider>
    ),
  });
  const savedRow = await waitFor(() => {
    const row = document.querySelector('tr[data-row-key="fee-a"]');
    expect(row).not.toBeNull();
    return row as HTMLElement;
  });
  const noteInput =
    await within(savedRow).findByPlaceholderText('备注（可选）');
  fireEvent.change(noteInput, { target: { value: '本地尚未保存的备注' } });
  const addButton = screen.getByText('新增应收费用').closest('button');
  expect(addButton).toBeVisible();
  await act(async () => fireEvent.click(addButton as HTMLButtonElement));
  const newRow = await waitFor(() => {
    const row = document.querySelector('tr[data-row-key^="new_"]');
    expect(row).not.toBeNull();
    return row as HTMLElement;
  });
  await act(async () => fireEvent.click(within(newRow).getByRole('checkbox')));
  props.selectedReceivableFeeIds =
    props.setSelectedReceivableFeeIds.mock.calls.at(-1)?.[0] ?? [];
  await act(async () => rerenderTabs());
  const readsBeforeRemoval = listFees.mock.calls.length;
  // 若移除新行错误地刷新远端，此版本会覆盖另一行尚未保存的输入。
  listFees.mockResolvedValue({
    data: [{ ...unbilledFeeA, version: '4', note: '他人更新后的备注' }],
  } as any);
  const deleteButton = screen.getAllByRole('button', { name: /^删\s*除$/ })[0];
  expect(deleteButton).toBeEnabled();
  await act(async () => fireEvent.click(deleteButton));
  const dialog = await screen.findByRole('dialog');
  const confirmButton = within(dialog).getByRole('button', {
    name: /^删\s*除$/,
  });
  await waitFor(() => expect(confirmButton).toBeVisible());
  await act(async () => fireEvent.click(confirmButton));
  await waitFor(() => expect(newRow).not.toBeInTheDocument());
  expect(listFees).toHaveBeenCalledTimes(readsBeforeRemoval);
  expect(bulkRemove).not.toHaveBeenCalled();
  expect(screen.getByDisplayValue('本地尚未保存的备注')).toBeVisible();
  expect(
    screen.queryByDisplayValue('他人更新后的备注'),
  ).not.toBeInTheDocument();
  expect(props.onDraftChange).toHaveBeenLastCalledWith(true);
});
