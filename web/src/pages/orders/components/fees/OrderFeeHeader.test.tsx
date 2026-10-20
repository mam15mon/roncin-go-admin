import { renderWithApp } from '@root/tests/renderWithApp';
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import OrderFeeHeader from './OrderFeeHeader';

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
        financeLocked={false}
        financeLockCommissionNos={[]}
        lockWritePolicy={{ disabled: false }}
        onRetryLockState={vi.fn()}
        receivableSummary={{ totalAmount: 0, count: 0 }}
        payableSummary={{ totalAmount: 0, count: 0 }}
        profitCny={0}
        profitRate="0"
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
