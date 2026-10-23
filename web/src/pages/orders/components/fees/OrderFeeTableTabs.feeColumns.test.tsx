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
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    getTableColumns: undefined,
    columnSettingScope: SCOPE,
  };
}

const taxFee = {
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

/** 打开列设置、取消若干列的勾选；弹窗内操作即时生效并持久化。 */
async function hideColumns(titles: string[]) {
  await openColumnSettings();
  for (const title of titles) {
    const checkbox = await columnCheckbox(title);
    await act(async () => {
      fireEvent.click(checkbox);
    });
  }
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
    expect(screen.getByText('2026-09-20')).toBeInTheDocument();
    expect(screen.queryByText('2026-09-20 14:30')).not.toBeInTheDocument();
    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(2));
    expect(tableHeaderCount('税金')).toBe(2);
    expect(tableHeaderCount('不含税单价')).toBe(2);
    expect(screen.getByText('6%')).toBeInTheDocument();
    expect(screen.getByText('5.66')).toBeInTheDocument();
    expect(screen.getByText('94.34')).toBeInTheDocument();
    expect(screen.getByText('700.00')).toBeInTheDocument();

    await hideColumns(['税率(%)', '税金', '不含税单价']);

    await waitFor(() => expect(tableHeaderCount('税率(%)')).toBe(0));
    expect(tableHeaderCount('税金')).toBe(0);
    expect(tableHeaderCount('不含税单价')).toBe(0);

    const stored = window.localStorage.getItem(STORAGE_KEY);
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored as string).hidden).toContain('taxRate');
    expect(JSON.parse(stored as string).hidden).toContain('taxAmount');
    expect(JSON.parse(stored as string).hidden).toContain('netUnitPrice');
  });

  it('应收应付默认按计价、税额、折算、日期备注顺序展示', async () => {
    render(
      <App>
        <OrderFeeTableTabs {...makeProps('order-1')} />
      </App>,
    );

    await screen.findByText('海运费');
    await waitFor(() => expect(tableHeaderCount('不含税单价')).toBe(2));
    for (const unitPriceHeader of screen.getAllByRole('columnheader', {
      name: '单价',
    })) {
      const headerTitles = Array.from(
        unitPriceHeader.closest('tr')?.querySelectorAll('th') ?? [],
      )
        .map((th) => th.textContent?.trim())
        .filter(Boolean);
      expect(headerTitles).toEqual([
        '费用代码',
        '费用名称',
        '结算单位',
        '币种',
        '计费单位',
        '数量',
        '单价',
        '不含税单价',
        '总金额',
        '税率(%)',
        '税金',
        '不含税总额',
        '汇率',
        '折本币金额',
        '发生日期',
        '备注',
        '操作',
      ]);
    }
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
