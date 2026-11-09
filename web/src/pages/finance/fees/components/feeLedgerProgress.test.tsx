import { ProTable } from '@ant-design/pro-components';
import { render, screen, within } from '@testing-library/react';
import { expect, it } from 'vitest';
import { FeeLedgerFinancialProgress } from '@/enums.generated';
import { getBaseFeeLedgerColumns } from './feeLedgerColumns';

it('只有部分或全额对冲的费用按 API 付款状态展示，金额仍独立显示', async () => {
  const rows: API.FeeLedgerItem[] = [
    {
      id: 'partial-netting',
      feeName: '部分对冲费用',
      totalAmount: '300.00000000',
      verifiedAmount: '0.00000000',
      nettedAmount: '200.00000000',
      unverifiedAmount: '100.00000000',
      financialProgress:
        FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED,
    },
    {
      id: 'full-netting',
      feeName: '全额对冲费用',
      totalAmount: '200.00000000',
      verifiedAmount: '0.00000000',
      nettedAmount: '200.00000000',
      unverifiedAmount: '0.00000000',
      financialProgress:
        FeeLedgerFinancialProgress.FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED,
    },
  ];
  const keys = [
    'feeName',
    'financialProgress',
    'verifiedAmount',
    'nettedAmount',
    'unverifiedAmount',
  ];
  render(
    <ProTable<API.FeeLedgerItem>
      rowKey="id"
      columns={getBaseFeeLedgerColumns().filter((column) =>
        keys.includes(String(column.dataIndex)),
      )}
      dataSource={rows}
      search={false}
      options={false}
      pagination={false}
    />,
  );
  const partial = (await screen.findByText('部分对冲费用')).closest('tr');
  const full = screen.getByText('全额对冲费用').closest('tr');
  if (!partial || !full) throw new Error('缺少费用数据行');
  for (const row of [partial, full]) {
    expect(
      within(row).getByText('未开票未付款', { exact: true }),
    ).toBeVisible();
    expect(within(row).getByText('200.00', { exact: true })).toBeVisible();
  }
  expect(within(partial).getByText('100.00', { exact: true })).toBeVisible();
  expect(within(full).getAllByText('0.00', { exact: true })).toHaveLength(2);
  expect(
    screen.queryByText('部分核销未开票', { exact: true }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByText('已核销未开票', { exact: true }),
  ).not.toBeInTheDocument();
});
