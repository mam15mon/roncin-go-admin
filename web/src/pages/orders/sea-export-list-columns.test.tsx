import { render, screen, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { expect, it, vi } from 'vitest';
import { OrderListTemplate } from '@/components/ui/order-list-template/OrderListTemplate';
import { seaExportListColumns } from './sea-export-list-columns';

it('海运可选列按展示口径处理金额、日期和空值，不渲染原始尾零或时间字符串', async () => {
  render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        columnSettingsKey="orders:sea-display-test"
        customColumns={[
          { title: '订单编号', dataIndex: 'orderNo' },
          ...seaExportListColumns,
        ]}
        queryOrders={vi.fn().mockResolvedValue({
          data: [
            {
              id: 'filled',
              orderNo: 'SE-FILLED',
              etd: '2026-11-11T03:00:00',
              orderDate: '2026-11-11',
              cargoValue: '15362.50000000',
              cargoCurrency: 'USD',
            },
            { id: 'zero', orderNo: 'SE-ZERO', cargoValue: '0' },
            { id: 'missing', orderNo: 'SE-MISSING' },
          ],
          total: 3,
        })}
      />
    </ConfigProvider>,
  );
  const filled = within(
    (await screen.findByText('SE-FILLED')).closest('tr') as HTMLElement,
  );
  expect(filled.getByText('15,362.50')).toBeInTheDocument();
  expect(filled.getByText('USD')).toBeInTheDocument();
  expect(filled.getByText('2026-11-11 03:00')).toBeInTheDocument();
  expect(filled.getByText('2026-11-11')).toBeInTheDocument();
  expect(filled.queryByText('15362.50000000')).not.toBeInTheDocument();
  expect(filled.queryByText('2026-11-11T03:00:00')).not.toBeInTheDocument();
  expect(
    within(screen.getByText('SE-ZERO').closest('tr') as HTMLElement).getByText(
      '0.00',
    ),
  ).toBeInTheDocument();
  const missing = within(
    screen.getByText('SE-MISSING').closest('tr') as HTMLElement,
  );
  expect(missing.queryByText('0.00')).not.toBeInTheDocument();
  expect(missing.getAllByText('-')).toHaveLength(seaExportListColumns.length);
});
