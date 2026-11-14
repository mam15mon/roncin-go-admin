import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it } from 'vitest';
import ProfitChart from './ProfitChart';

const amount = (
  baseCurrency: string,
  gross: string,
  net: string,
): API.ProfitReportAmount => ({
  baseCurrency,
  receivableAmount: '100',
  payableAmount: '0',
  profitAmount: gross,
  netReceivableAmount: '100',
  netPayableAmount: '0',
  netProfitAmount: net,
});
describe('完整月序列利润图', () => {
  it('绘制完整月份和负利润，并附同源可读数表，币种分开选择', () => {
    renderWithApp(
      <ProfitChart
        response={{
          totals: [amount('CNY', '60', '50'), amount('USD', '7', '6')],
          months: [
            {
              month: '2026-01',
              amounts: [amount('CNY', '100', '90'), amount('USD', '10', '9')],
            },
            {
              month: '2026-02',
              amounts: [amount('CNY', '0', '0'), amount('USD', '0', '0')],
            },
            {
              month: '2026-03',
              amounts: [amount('CNY', '-40', '-40'), amount('USD', '-3', '-3')],
            },
          ],
        }}
      />,
    );
    expect(
      screen.getByRole('img', { name: /完整筛选范围.*CNY/ }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('2026-02').length).toBeGreaterThan(0);
    expect(screen.getAllByText('-40.00')).toHaveLength(2);
    const negativeBars = [
      ...screen
        .getByRole('img', { name: /完整筛选范围.*CNY/ })
        .querySelectorAll('rect'),
    ].filter(
      (rect) =>
        rect.getAttribute('y') === '150' &&
        Number(rect.getAttribute('height')) > 0,
    );
    expect(negativeBars).toHaveLength(2);
    fireEvent.mouseDown(screen.getByRole('combobox', { name: '图表本币' }));
    fireEvent.click(
      screen.getByText('USD', { selector: '.ant-select-item-option-content' }),
    );
    expect(
      screen.getByRole('img', { name: /完整筛选范围.*USD/ }),
    ).toBeInTheDocument();
    expect(screen.getByText('未税利润（USD）')).toBeInTheDocument();
    expect(screen.queryByText('未税利润（CNY）')).not.toBeInTheDocument();
  });
  it('空查询显示无数据，不伪造零利润图', () => {
    renderWithApp(<ProfitChart response={{}} />);
    expect(screen.getByText('当前筛选没有可分析的费用')).toBeInTheDocument();
    expect(
      screen.queryByRole('img', { name: /完整筛选范围/ }),
    ).not.toBeInTheDocument();
  });
});
