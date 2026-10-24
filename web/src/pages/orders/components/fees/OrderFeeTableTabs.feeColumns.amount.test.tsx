import type { ActionType } from '@ant-design/pro-components';
import { render, screen, waitFor, within } from '@testing-library/react';
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
  createdAt: '2026-09-20 14:30',
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

describe('OrderFeeTableTabs 只读金额列', () => {
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

  it('零税率显示 0%；不含税口径旧行由不含税单价列直接展示单价（A2/A4）', async () => {
    listFees.mockResolvedValue({
      data: [
        {
          canEdit: true,
          canBill: true,
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
          canEdit: true,
          canBill: true,
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
          canEdit: true,
          canBill: true,
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
          canEdit: true,
          canBill: true,
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
});
