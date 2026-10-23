import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);

const SCOPE = { userId: 'user-9', organizationId: 'org-9' };
const STORAGE_KEY = 'roncin:order-fee-columns:v1:user-9:org-9';

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: { totalAmount: 0, count: 0 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: [],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: false,
    canCreateFees: true,
    canDeleteFees: true,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    getTableColumns: undefined,
    columnSettingScope: SCOPE,
  };
}

const taxFee = {
  canEdit: true,
  canBill: true,
  id: 'fee-tax-1',
  direction: RECEIVABLE,
  currency: 'USD',
  expenseDate: '2026-09-20 14:30',
  version: '3',
  feeName: '海运费',
  taxRate: '6',
  taxAmount: '5.66',
  netAmount: '94.34',
  baseCurrency: 'CNY',
  baseCurrencyAmount: '700.00',
  taxInclusive: true,
} as API.OrderFee;

/** 表格列头数量（应收/应付两表各一份，共 2；隐藏列为 0；不含弹窗内同名文本）。 */
function tableHeaderCount(title: string) {
  return screen.queryAllByRole('columnheader', { name: title }).length;
}

async function openColumnSettings() {
  await act(async () => {
    screen.getAllByRole('button', { name: /列设置/ })[0].click();
  });
  await screen.findByPlaceholderText('搜索列名');
}

/** 在列设置弹窗中找到指定列的复选框输入。 */
async function columnCheckbox(title: string) {
  const dialog = await screen.findByRole('dialog', { name: '列设置' });
  return within(dialog).getByRole('checkbox', {
    name: title,
  }) as HTMLInputElement;
}

describe('OrderFeeTableTabs 列偏好重读与恢复', () => {
  beforeEach(() => {
    window.localStorage.clear();
    listFees.mockReset();
    listFees.mockResolvedValue({
      data: [taxFee],
    } as Awaited<ReturnType<typeof listFees>>);
    resolveRate.mockReset();
    resolveRate.mockResolvedValue({
      exchangeRate: '1.0000',
      exchangeRateSource: 'SYSTEM',
    } as Awaited<ReturnType<typeof resolveRate>>);
  });

  it('取消勾选即时隐藏列并写入偏好；重新挂载读取已保存偏好（A1）', async () => {
    const props = makeProps('order-1');
    const { unmount } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');

    await openColumnSettings();
    const checkbox = await columnCheckbox('税金');
    await act(async () => {
      fireEvent.click(checkbox);
    });

    expect(tableHeaderCount('税金')).toBe(0);
    expect(
      JSON.parse(window.localStorage.getItem(STORAGE_KEY) as string).hidden,
    ).toContain('taxAmount');
    unmount();

    // 预置偏好后重新挂载：税率列按偏好隐藏，税金列保持默认可见
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        order: ['taxAmount', 'feeSettingId'],
        hidden: ['taxRate'],
      }),
    );
    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await screen.findByText('海运费');
    expect(tableHeaderCount('税金')).toBe(2);
    expect(tableHeaderCount('税率(%)')).toBe(0);
    await waitFor(() => expect(screen.getByText('5.66')).toBeInTheDocument());
  });

  it('恢复默认清除场景偏好并还原默认可见列（A1）', async () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ order: ['taxRate'], hidden: ['taxRate'] }),
    );
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');
    expect(tableHeaderCount('税率(%)')).toBe(0);

    await openColumnSettings();
    await act(async () => {
      screen.getByRole('button', { name: /恢复默认/ }).click();
    });

    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(2));
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
  });
});
