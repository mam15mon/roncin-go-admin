import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { FEE_CONFIRMED, RECEIVABLE } from './feeConstants';
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
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    getTableColumns: undefined,
    columnSettingScope: SCOPE,
  };
}

const taxFee = {
  id: 'fee-tax-1',
  direction: RECEIVABLE,
  status: FEE_CONFIRMED,
  currency: 'USD',
  expenseDate: '2026-09-20',
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
  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
}

/** 在列设置弹窗中找到指定列的复选框输入。 */
async function columnCheckbox(title: string) {
  return waitFor(() => {
    const label = screen
      .getAllByText(title)
      .find((element) => element.closest('.ant-modal'));
    expect(label).not.toBeUndefined();
    const input = label?.closest('label')?.querySelector('input');
    expect(input).not.toBeNull();
    return input as HTMLInputElement;
  });
}

/** 打开列设置、切换若干列的勾选状态后确定。 */
async function toggleColumnsAndConfirm(titles: string[]) {
  await openColumnSettings();
  for (const title of titles) {
    const checkbox = await columnCheckbox(title);
    await act(async () => {
      fireEvent.click(checkbox);
    });
  }
  await act(async () => {
    screen.getByRole('button', { name: /确\s*定/ }).click();
  });
}

describe('OrderFeeTableTabs 列设置与只读金额列', () => {
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

  it('税额四列默认可见展示后端快照；列设置取消勾选后隐藏并持久化（A1/A3）', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );

    await screen.findByText('海运费');
    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(2));
    expect(tableHeaderCount('税金')).toBe(2);
    expect(screen.getByText('6%')).toBeInTheDocument();
    expect(screen.getByText('5.66')).toBeInTheDocument();
    expect(screen.getByText('94.34')).toBeInTheDocument();
    expect(screen.getByText('700.00')).toBeInTheDocument();

    await toggleColumnsAndConfirm(['税率(%)', '税金']);

    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(0));
    expect(tableHeaderCount('税金')).toBe(0);

    const stored = window.localStorage.getItem(STORAGE_KEY);
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored as string).hidden).toContain('taxRate');
    expect(JSON.parse(stored as string).hidden).toContain('taxAmount');
  });

  it('取消不改变表格且不写偏好；重新挂载读取已保存偏好（A1）', async () => {
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
    await act(async () => {
      screen.getByRole('button', { name: /取\s*消/ }).click();
    });

    expect(tableHeaderCount('税金')).toBe(2);
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
    unmount();

    // 预置偏好后重新挂载：税率列按偏好隐藏，税金列保持默认可见
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        order: ['status', 'taxAmount', 'feeSettingId'],
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
      JSON.stringify({ order: ['status', 'taxRate'], hidden: ['taxRate'] }),
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
    await act(async () => {
      screen.getByRole('button', { name: /确\s*定/ }).click();
    });

    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(2));
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it('零税率显示 0%；不含税口径旧行在税率列标注未税单价（A3）', async () => {
    listFees.mockResolvedValue({
      data: [{ ...taxFee, id: 'fee-zero', taxRate: '0', taxInclusive: false }],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');

    await waitFor(() => expect(screen.getByText('0%')).toBeInTheDocument());
    expect(screen.getByText('未税单价')).toBeInTheDocument();
  });

  it('折本币金额列同时标示本位币代码（A3）', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');

    await waitFor(() => expect(tableHeaderCount('折本币金额')).toBe(2));
    expect(screen.getByText('700.00')).toBeInTheDocument();
    expect(screen.getByText('CNY')).toBeInTheDocument();
  });

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
    await screen.findByText('海运费');

    const settingsButton = screen.getAllByRole('button', {
      name: /列设置/,
    })[0];
    expect(settingsButton).not.toBeDisabled();

    await act(async () => {
      screen.getByRole('button', { name: /新增应收费用/ }).click();
    });
    await waitFor(() => expect(screen.getByText('保存')).toBeInTheDocument());
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
