import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { FinanceSummaryBoard } from './FinanceSummaryBoard';
import { aggregateLedgerAmounts } from './summary';

const config = {
  groups: [
    { direction: 'RECEIVABLE', label: '应收' },
    { direction: 'PAYABLE', label: '应付' },
  ],
  profit: {
    label: '费用利润',
    receivableDirection: 'RECEIVABLE',
    payableDirection: 'PAYABLE',
  },
};

describe('财务台账单行条件汇总', () => {
  it('按原币与本币分别分组，正负利润仅有一个符号，decimal求和不丢失小数', () => {
    const summary = aggregateLedgerAmounts([
      {
        direction: 'RECEIVABLE',
        currency: 'USD',
        amount: '3106.4252',
        baseCurrency: 'CNY',
        baseAmount: '100',
      },
      {
        direction: 'PAYABLE',
        currency: 'USD',
        amount: '1739.1046',
        baseCurrency: 'CNY',
        baseAmount: '10',
      },
      {
        direction: 'RECEIVABLE',
        currency: 'EUR',
        amount: '0.1',
        baseCurrency: 'USD',
        baseAmount: '0.1',
      },
      {
        direction: 'RECEIVABLE',
        currency: 'EUR',
        amount: '0.2',
        baseCurrency: 'USD',
        baseAmount: '0.2',
      },
      {
        direction: 'PAYABLE',
        currency: 'GBP',
        amount: '111',
        baseCurrency: 'USD',
        baseAmount: '100',
      },
    ]);
    render(<FinanceSummaryBoard config={config} summary={summary} selected />);
    const board = screen.getByRole('status', { name: '金额合计' });
    expect(board).toHaveTextContent('已选 5 条');
    expect(board).not.toHaveTextContent('筛选结果');
    const profit = screen.getByText('费用利润:').parentElement;
    expect(profit).toHaveTextContent('+1,367.32 USD');
    expect(profit).toHaveTextContent('+0.30 EUR');
    expect(profit).toHaveTextContent('-111.00 GBP');
    expect(profit).toHaveTextContent('+90.00 CNY');
    expect(profit).toHaveTextContent('-99.70 USD');
    expect(profit).not.toHaveTextContent('++');
    expect(profit).not.toHaveTextContent('+-');
  });

  it('无选择使用完整筛选汇总，零折本币与未折算分别显示，不伪造CNY', () => {
    render(
      <FinanceSummaryBoard
        config={{ groups: [{ direction: '', label: '抵销金额' }] }}
        summary={{
          count: 99,
          amounts: [
            {
              direction: '',
              currency: 'EUR',
              amount: '125.5',
              unconvertedCount: 2,
            },
            {
              direction: '',
              currency: 'USD',
              amount: '1',
              baseCurrency: 'USD',
              baseAmount: '0',
              unconvertedCount: 0,
            },
          ],
        }}
      />,
    );
    const board = screen.getByRole('status', { name: '金额合计' });
    expect(board).toHaveTextContent('筛选结果共 99 条');
    expect(board).toHaveTextContent('125.50 EUR');
    expect(board).toHaveTextContent('折合 0.00 USD');
    expect(board).toHaveTextContent('2 条未折算');
    expect(board).not.toHaveTextContent('利润');
    expect(board).not.toHaveTextContent('CNY');
  });

  it('尚无成功汇总时不伪装为成功零值，空结果显示真实零', () => {
    const { rerender } = render(<FinanceSummaryBoard config={config} />);
    expect(screen.getByText('汇总尚未加载')).toBeVisible();
    expect(screen.queryByText('筛选结果共 0 条')).not.toBeInTheDocument();
    rerender(
      <FinanceSummaryBoard
        config={config}
        summary={{ count: 0, amounts: [] }}
      />,
    );
    expect(screen.getByText('筛选结果共 0 条')).toBeVisible();
  });
});
