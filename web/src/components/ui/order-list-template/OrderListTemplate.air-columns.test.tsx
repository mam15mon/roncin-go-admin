import { cleanup, render, screen, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { OrderListTemplate } from './OrderListTemplate';

beforeEach(() => localStorage.clear());
afterEach(cleanup);

it('航空默认展开件重尺、真实主分单标签和进口服务进度', async () => {
  const queryOrders = vi.fn().mockResolvedValue({
    data: [
      {
        id: 'air-1',
        orderNo: 'AI001',
        masterBlNo: '78112345675',
        billRelation: { kind: 'house', houseNo: 'H001', memberCount: 2 },
        totalPackages: 0,
        packageUnit: 'CTN',
        grossWeightKg: 125,
        volumeCbm: 1.2,
        statusName: '服务进行中',
      },
    ],
    total: 1,
  });
  const { container } = render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        showBillRelations
        options={{ transportMode: 'air' }}
        queryOrders={queryOrders}
        progressColumnTitle="进口服务进度"
        defaultVisibleColumnKeys={[
          'totalPackages',
          'grossWeightKg',
          'volumeCbm',
        ]}
      />
    </ConfigProvider>,
  );
  await screen.findByText('AI001');
  const row = within(
    screen.getByText('AI001').closest('tr') as HTMLTableRowElement,
  );
  expect(row.getByText('主单 MAWB')).toBeInTheDocument();
  expect(row.getByText('分单 HAWB')).toBeInTheDocument();
  expect(row.getByText('H001')).toBeInTheDocument();
  const headers = within(
    container.querySelector('thead') as HTMLTableSectionElement,
  );
  for (const name of [
    '委托总件数',
    '委托总毛重(KGS)',
    '委托总体积(CBM)',
    '进口服务进度',
  ]) {
    expect(headers.getByText(name)).toBeInTheDocument();
  }
  expect(headers.queryByText('箱型箱量')).not.toBeInTheDocument();
});
