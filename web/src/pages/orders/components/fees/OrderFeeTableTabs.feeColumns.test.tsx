import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  hideColumns,
  makeProps,
  OrderFeeTableTabs,
  STORAGE_KEY,
  setupFeeColumnsTests,
  tableHeaderCount,
} from './OrderFeeTableTabs.feeColumns.fixtures';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

describe('OrderFeeTableTabs 列设置与只读金额列', () => {
  setupFeeColumnsTests();

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

  it('应收应付默认按计价、税额、折算、备注顺序展示，隐藏录入日期', async () => {
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
        '备注',
      ]);
    }
    expect(tableHeaderCount('录入日期')).toBe(0);
  });
});
