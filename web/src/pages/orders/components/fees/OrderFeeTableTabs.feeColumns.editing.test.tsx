import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  columnCheckbox,
  makeProps,
  OrderFeeTableTabs,
  openColumnSettings,
  setupFeeColumnsTests,
} from './OrderFeeTableTabs.feeColumns.fixtures';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

describe('OrderFeeTableTabs 列设置编辑约束', () => {
  setupFeeColumnsTests();
  it('正在编辑费用行时禁用列设置入口（A2）', async () => {
    const props = {
      ...makeProps('order-1'),
      feeWritesDisabled: false,
      billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
    };
    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await waitFor(() =>
      expect(
        document.querySelector('tr[data-row-key="fee-tax-1"]'),
      ).not.toBeNull(),
    );

    const settingsButton = screen.getAllByRole('button', {
      name: /列设置/,
    })[0];
    expect(settingsButton).not.toBeDisabled();

    await act(async () => {
      screen.getByRole('button', { name: /新增应收费用/ }).click();
    });
    await waitFor(() =>
      expect(document.querySelector('tr[data-row-key^="new_"]')).not.toBeNull(),
    );
    expect(settingsButton).toBeDisabled();
  });

  it('必显列在列设置中勾选锁定且不可取消（A2）', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');

    await openColumnSettings();
    const lockedCheckbox = await columnCheckbox('费用名称');
    expect(lockedCheckbox).toBeDisabled();
    expect(lockedCheckbox.checked).toBe(true);
  });
});
