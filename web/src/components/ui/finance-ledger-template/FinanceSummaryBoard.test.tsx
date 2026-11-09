import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { FinanceSummaryBoard } from './FinanceSummaryBoard';
import type { FinanceLedgerSummaryItem } from './types';

describe('财务台账底部汇总', () => {
  it('选中和全部汇总的多币种利润只保留一个正负号，应收仍保留币种分隔符', () => {
    const rows: FinanceLedgerSummaryItem[] = [
      { direction: 1, currency: 'CNY', totalAmount: '3106.4252' },
      { direction: 2, currency: 'CNY', totalAmount: '1739.1046' },
      { direction: 1, currency: 'USD', totalAmount: '111' },
      { direction: 2, currency: 'EUR', totalAmount: '25.5' },
    ];
    render(<FinanceSummaryBoard selectedRows={rows} allRows={rows} />);

    const profits = screen.getAllByText('利润:');
    expect(profits).toHaveLength(2);
    for (const label of profits) {
      expect(label.parentElement).toHaveTextContent(
        '利润: +1,367.32 CNY+111.00 USD-25.50 EUR',
      );
      expect(label.parentElement).not.toHaveTextContent('++');
      expect(label.parentElement).not.toHaveTextContent('+-');
    }
    for (const label of screen.getAllByText('应收(含税):')) {
      expect(label.parentElement).toHaveTextContent(
        '应收(含税): 3,106.43 CNY+111.00 USD',
      );
    }
  });

  it('利润相抵为零时不显示正负号', () => {
    render(
      <FinanceSummaryBoard
        allRows={[
          { direction: 1, currency: 'USD', totalAmount: '111' },
          { direction: 2, currency: 'USD', totalAmount: '111' },
        ]}
      />,
    );
    expect(screen.getByText('利润:').parentElement).toHaveTextContent(
      '利润: 0.00折合 0.00 CNY',
    );
  });
});
