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
import { FEE_UNBILLED, RECEIVABLE } from './feeConstants';
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
  status: FEE_UNBILLED,
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

  it('税额相关列默认可见展示后端快照；列设置取消勾选后隐藏并持久化（A1/A3）', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );

    await screen.findByText('海运费');
    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(2));
    expect(tableHeaderCount('税金')).toBe(2);
    expect(tableHeaderCount('不含税单价')).toBe(2);
    expect(screen.getByText('6%')).toBeInTheDocument();
    expect(screen.getByText('5.66')).toBeInTheDocument();
    expect(screen.getByText('94.34')).toBeInTheDocument();
    expect(screen.getByText('700.00')).toBeInTheDocument();

    await toggleColumnsAndConfirm(['税率(%)', '税金', '不含税单价']);

    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(0));
    expect(tableHeaderCount('税金')).toBe(0);
    expect(tableHeaderCount('不含税单价')).toBe(0);

    const stored = window.localStorage.getItem(STORAGE_KEY);
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored as string).hidden).toContain('taxRate');
    expect(JSON.parse(stored as string).hidden).toContain('taxAmount');
    expect(JSON.parse(stored as string).hidden).toContain('netUnitPrice');
  });

  it('不含税单价列默认顺序紧跟单价之后（A1）', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );

    await screen.findByText('海运费');
    await waitFor(() => expect(tableHeaderCount('不含税单价')).toBe(2));
    const unitPriceHeader = screen.getAllByRole('columnheader', {
      name: '单价',
    })[0];
    const headerTitles = Array.from(
      unitPriceHeader.closest('tr')?.querySelectorAll('th') ?? [],
    ).map((th) => th.textContent?.trim());
    expect(headerTitles[headerTitles.indexOf('单价') + 1]).toBe('不含税单价');
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

  it('零税率显示 0%；不含税口径旧行由不含税单价列直接展示单价（A2/A4）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          ...taxFee,
          id: 'fee-zero',
          unitPrice: '100',
          taxRate: '0',
          taxInclusive: false,
        },
      ],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('海运费');

    await waitFor(() => expect(screen.getByText('0%')).toBeInTheDocument());
    // 历史不含税行单价即不含税口径，直接按两位小数展示
    expect(screen.getByText('100.00')).toBeInTheDocument();
    // 税率列的「未税单价」Tag 已由独立列替代
    expect(screen.queryByText('未税单价')).not.toBeInTheDocument();
  });

  it('已保存行不含税单价：含税按税率反算、不含税直取单价、缺税率显示 -（A2）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          ...taxFee,
          id: 'fee-gross',
          feeName: '含税行',
          feeCode: 'FA',
          unitPrice: '106',
          quantity: '1',
          settlementPartyName: '测试客户',
          billingUnit: '票',
          exchangeRate: '7.0',
          note: '备注',
        },
        {
          ...taxFee,
          id: 'fee-net',
          feeName: '不含税行',
          feeCode: 'FB',
          unitPrice: '100',
          quantity: '1',
          // protojson 省略零值：真实响应中 tax_inclusive=false 以字段缺失（undefined）到达
          taxInclusive: undefined,
          settlementPartyName: '测试客户',
          billingUnit: '票',
          exchangeRate: '7.0',
          note: '备注',
        },
        {
          ...taxFee,
          id: 'fee-norate',
          feeName: '缺税率行',
          feeCode: 'FC',
          unitPrice: '50',
          quantity: '1',
          taxRate: undefined,
          settlementPartyName: '测试客户',
          billingUnit: '票',
          exchangeRate: '7.0',
          note: '备注',
        },
      ],
    } as any);

    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );
    await screen.findByText('含税行');

    // 含税单价 106 ÷ (1 + 6%) = 100.00；不含税行直取单价 100.00
    await waitFor(() => expect(screen.getAllByText('100.00').length).toBe(2));
    const noRateRow = screen
      .getByText('缺税率行')
      .closest('tr') as HTMLTableRowElement;
    // 缺税率行：税率与不含税单价两列均显示 -
    expect(within(noRateRow).getAllByText('-').length).toBe(2);
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
