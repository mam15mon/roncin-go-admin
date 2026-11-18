import type { ActionType } from '@ant-design/pro-components';
import { act, fireEvent, screen, within } from '@testing-library/react';
import type React from 'react';
import { beforeEach, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';

const listFees = vi.mocked(orderFeeServiceListFees);
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);

const SCOPE = { userId: 'user-9', organizationId: 'org-9' };
export const STORAGE_KEY = 'roncin:order-fee-columns:v1:user-9:org-9';

export function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
    payableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
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
    canUpdateFees: true,
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
  createdAt: '2026-09-20T14:30:00+08:00',
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
export function tableHeaderCount(title: string) {
  return screen.queryAllByRole('columnheader', { name: title }).length;
}

export async function openColumnSettings() {
  await act(async () => {
    screen.getAllByRole('button', { name: /列设置/ })[0].click();
  });
  await screen.findByPlaceholderText('搜索列名');
}

/** 在列设置弹窗中找到指定列的复选框输入。 */
export async function columnCheckbox(title: string) {
  const dialog = await screen.findByRole('dialog', { name: '列设置' });
  return within(dialog).getByRole('checkbox', {
    name: title,
  }) as HTMLInputElement;
}

/** 打开列设置、取消若干列的勾选；弹窗内操作即时生效并持久化。 */
export async function hideColumns(titles: string[]) {
  await openColumnSettings();
  for (const title of titles) {
    const checkbox = await columnCheckbox(title);
    await act(async () => {
      fireEvent.click(checkbox);
    });
  }
}

export function setupFeeColumnsTests() {
  beforeEach(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
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
}

export { default as OrderFeeTableTabs } from './OrderFeeTableTabs';
