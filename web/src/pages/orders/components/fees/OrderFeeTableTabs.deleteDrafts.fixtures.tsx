import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { expect } from 'vitest';
import {
  makeProps,
  renderTabs,
} from './OrderFeeTableTabs.bulkActions.fixtures';
export async function renderAndAdd(canDeleteFees = true) {
  const { props } = makeProps('order-1');
  props.canDeleteFees = canDeleteFees;
  const { rerenderTabs } = await renderTabs(props, {
    wrapper: ({ children }) => (
      <ConfigProvider theme={{ token: { motion: false } }}>
        {children}
      </ConfigProvider>
    ),
  });
  await waitFor(() => expect(props.setAllReceivableItems).toHaveBeenCalled());
  const loadedRows = props.setAllReceivableItems.mock.calls.at(
    -1,
  )?.[0] as API.OrderFee[];
  for (const row of loadedRows)
    await waitFor(() =>
      expect(
        document.querySelector(`tr[data-row-key="${row.id}"]`),
      ).not.toBeNull(),
    );
  await act(async () =>
    fireEvent.click(
      screen.getByText('新增应收费用').closest('button') as HTMLButtonElement,
    ),
  );
  const row = await waitFor(() => {
    const element = document.querySelector('tr[data-row-key^="new_"]');
    expect(element).not.toBeNull();
    return element as HTMLElement;
  });
  const select = async (target: HTMLElement) => {
    await act(async () =>
      fireEvent.click(within(target).getByRole('checkbox')),
    );
    props.selectedReceivableFeeIds =
      props.setSelectedReceivableFeeIds.mock.calls.at(-1)?.[0];
    await act(async () => rerenderTabs());
  };
  await select(row);
  return { props, row, select };
}

export async function submitDelete() {
  await act(async () =>
    fireEvent.click(
      screen.getAllByText('删除')[0].closest('button') as HTMLButtonElement,
    ),
  );
  const dialog = await screen.findByRole('dialog');
  const submit = within(dialog)
    .getByText('删 除')
    .closest('button') as HTMLButtonElement;
  await waitFor(() => expect(submit).toBeVisible());
  await act(async () => fireEvent.click(submit));
}
