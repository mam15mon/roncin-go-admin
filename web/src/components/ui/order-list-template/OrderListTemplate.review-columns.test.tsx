import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { OrderListTemplate } from './OrderListTemplate';

afterEach(cleanup);
beforeEach(() => localStorage.clear());

it('追加的锁定状态列参与普通列显隐、调序与冻结，操作列仍为辅助列', async () => {
  const { container } = render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        columnSettingsKey="orders:review-columns-test"
        queryOrders={vi.fn().mockResolvedValue({ data: [], total: 0 })}
        trailingColumns={[
          { key: 'feeReview', title: '锁定状态', render: () => '未审核' },
        ]}
      />
    </ConfigProvider>,
  );
  const headers = within(
    container.querySelector('thead') as HTMLTableSectionElement,
  );
  expect(headers.getByText('锁定状态')).toBeInTheDocument();
  fireEvent.click(screen.getByLabelText('列设置'));
  const checkbox = await screen.findByLabelText('锁定状态', {
    selector: 'input[type=checkbox]',
  });
  await waitFor(() => expect(checkbox.closest('label')).toBeVisible());
  expect(checkbox).toBeChecked();
  expect(screen.getByLabelText('锁定状态目标序号')).toBeEnabled();
  expect(screen.getByLabelText('锁定状态冻结位置')).toBeEnabled();
  expect(
    screen.queryByLabelText('操作', { selector: 'input[type=checkbox]' }),
  ).not.toBeInTheDocument();
  expect(screen.getByText('辅助列 · 顺序固定')).toBeVisible();
  fireEvent.click(checkbox);
  await waitFor(() =>
    expect(headers.queryByText('锁定状态')).not.toBeInTheDocument(),
  );
  fireEvent.click(checkbox);
  await waitFor(() =>
    expect(headers.getByText('锁定状态')).toBeInTheDocument(),
  );
});
