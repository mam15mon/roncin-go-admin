import { renderWithApp } from '@root/tests/renderWithApp';
import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { type FeeSideSummary, summarizeFees } from './feeSummary';
import OrderFeeHeader from './OrderFeeHeader';

const emptyFeeSummary = summarizeFees([]);

describe('空运费用页业务摘要', () => {
  it('展示真实航空主单、航班、机场与航司，不展示海运标签或引用 UUID', () => {
    renderWithApp(
      <OrderFeeHeader
        order={{
          id: 'order',
          orderNo: 'AE-001',
          businessType: OrderBusinessType.BUSINESS_TYPE_AE,
          customerId: 'customer-uuid',
          customerName: '验收客户',
          originLocationId: 'origin-uuid',
          destinationLocationId: 'destination-uuid',
          airExport: {
            master: {
              airlineName: '中国东方航空',
              originAirportName: '上海浦东 (PVG)',
              destinationAirportName: '洛杉矶 (LAX)',
              content: {
                masterNo: '78112345675',
                flightNo: 'MU583',
                etd: '2026-10-21T06:45:00Z',
              },
            },
          },
        }}
        kind="air-export"
        orderId="order"
        configTitle="空运出口"
        lockWritePolicy={{ disabled: false }}
        onRetryLockState={vi.fn()}
        receivableSummary={emptyFeeSummary}
        payableSummary={emptyFeeSummary}
      />,
    );
    expect(screen.getByText('主运单号 (MAWB)')).toBeVisible();
    expect(screen.getByText('78112345675')).toBeVisible();
    expect(screen.getByText('MU583')).toBeVisible();
    expect(screen.getByText('上海浦东 (PVG)')).toBeVisible();
    expect(screen.getByText('洛杉矶 (LAX)')).toBeVisible();
    expect(screen.getByText('中国东方航空')).toBeVisible();
    expect(screen.queryByText('主单号 (MBL)')).not.toBeInTheDocument();
    expect(screen.queryByText('origin-uuid')).not.toBeInTheDocument();
    expect(screen.queryByText('destination-uuid')).not.toBeInTheDocument();
  });
});

const baseOrder: API.Order = { id: 'order', orderNo: 'SE-001' };

function renderHeader(
  receivableSummary?: FeeSideSummary,
  payableSummary?: FeeSideSummary,
) {
  return renderWithApp(
    <OrderFeeHeader
      order={baseOrder}
      kind="sea-export"
      orderId="order"
      configTitle="海运出口"
      lockWritePolicy={{ disabled: false }}
      onRetryLockState={vi.fn()}
      receivableSummary={receivableSummary}
      payableSummary={payableSummary}
    />,
  );
}

const fee = (over: Partial<API.OrderFee>): API.OrderFee => ({
  currency: 'USD',
  totalAmount: '0',
  netAmount: '0',
  ...over,
});

describe('费用页含税与不含税双口径看板', () => {
  it('六个金额同时可见：应收 106/100，应付 53/50，毛利 53/50', () => {
    renderHeader(
      summarizeFees([
        fee({
          baseCurrencyAmount: '106.00000000',
          baseCurrencyNetAmount: '100.00000000',
        }),
      ]),
      summarizeFees([
        fee({
          baseCurrencyAmount: '53',
          baseCurrencyNetAmount: '50',
        }),
      ]),
    );
    const recv = within(screen.getByTestId('fee-summary-receivable'));
    expect(recv.getByText('含税')).toBeVisible();
    expect(recv.getByText('不含税')).toBeVisible();
    expect(recv.getByText('¥106.00')).toBeVisible();
    expect(recv.getByText('¥100.00')).toBeVisible();
    const pay = within(screen.getByTestId('fee-summary-payable'));
    expect(pay.getByText('¥53.00')).toBeVisible();
    expect(pay.getByText('¥50.00')).toBeVisible();
    const profit = within(screen.getByTestId('fee-summary-profit'));
    expect(profit.getByText('含税')).toBeVisible();
    expect(profit.getByText('不含税')).toBeVisible();
    expect(profit.getByText('¥53.00')).toBeVisible();
    expect(profit.getByText('¥50.00')).toBeVisible();
  });

  it('零税额两组相同且都显示；空数据显示 0.00', () => {
    renderHeader(
      summarizeFees([
        fee({ baseCurrencyAmount: '100', baseCurrencyNetAmount: '100' }),
      ]),
      emptyFeeSummary,
    );
    const recv = within(screen.getByTestId('fee-summary-receivable'));
    expect(recv.getAllByText('¥100.00')).toHaveLength(2);
    const pay = within(screen.getByTestId('fee-summary-payable'));
    expect(pay.getAllByText('¥0.00')).toHaveLength(2);
  });

  it('未取得数据或全部未折算显示 -，部分未折算计数且不按 0 处理', () => {
    renderHeader(undefined, summarizeFees([fee({ baseCurrencyAmount: '10' })]));
    expect(
      within(screen.getByTestId('fee-summary-receivable')).getAllByText('-'),
    ).toHaveLength(2);
    const pay = within(screen.getByTestId('fee-summary-payable'));
    expect(pay.getAllByText('-')).toHaveLength(2);
    expect(pay.getByText('1 笔未折算，未计入本币合计')).toBeVisible();
  });

  it('同一份行数据汇总，多币种原币分组展示且不混加', () => {
    const summary = summarizeFees([
      fee({
        currency: 'USD',
        totalAmount: '106',
        netAmount: '100',
        baseCurrencyAmount: '742',
        baseCurrencyNetAmount: '700',
      }),
      fee({
        currency: 'EUR',
        totalAmount: '53',
        netAmount: '50',
        baseCurrencyAmount: '400',
        baseCurrencyNetAmount: '377.36',
      }),
    ]);
    expect(summary.gross).toBe('1142');
    expect(summary.net).toBe('1077.36');
    renderHeader(summary, emptyFeeSummary);
    const recv = screen.getByTestId('fee-summary-receivable');
    expect(recv).toHaveTextContent('USD：含税 106.00 / 不含税 100.00');
    expect(recv).toHaveTextContent('EUR：含税 53.00 / 不含税 50.00');
  });
});
