import { cleanup, render, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { OrderListTemplate } from './OrderListTemplate';
import type { OrderListItem } from './types';

beforeEach(() => localStorage.clear());
afterEach(cleanup);

const rows: OrderListItem[] = [
  {
    id: '1',
    orderNo: 'SE-001',
    masterBlNo: 'MBL-SHARED',
    billRelation: { kind: 'house', houseNo: 'HBL-001', memberCount: 2 },
  },
  {
    id: '2',
    orderNo: 'SE-002',
    masterBlNo: 'MBL-SHARED',
    billRelation: { kind: 'house', houseNo: 'HBL-002', memberCount: 2 },
  },
  {
    id: '3',
    orderNo: 'SE-003',
    masterBlNo: 'MBL-DIRECT',
    billRelation: { kind: 'direct', memberCount: 1 },
  },
  {
    id: '4',
    orderNo: 'SE-004',
    billRelation: { kind: 'house', memberCount: 0 },
  },
  {
    id: '5',
    orderNo: 'SE-005',
    billRelation: { kind: 'unknown', memberCount: 0 },
  },
];

it('逐票显示共用主单、各自分单与直单，不把缺省或未知数据猜成有效关系', async () => {
  const queryOrders = vi.fn().mockResolvedValue({ data: rows, total: 5 });
  const { container } = render(
    <OrderListTemplate showBillRelations queryOrders={queryOrders} />,
  );
  await screen.findByText('HBL-001');
  const row = (orderNo: string) =>
    within(screen.getByText(orderNo).closest('tr') as HTMLTableRowElement);
  expect(row('SE-001').getByText('MBL-SHARED')).toBeInTheDocument();
  expect(row('SE-002').getByText('MBL-SHARED')).toBeInTheDocument();
  expect(row('SE-002').getByText('HBL-002')).toBeInTheDocument();
  expect(
    row('SE-001').getByRole('button', { name: '共 2 票' }),
  ).toBeInTheDocument();
  expect(row('SE-003').getByText('直单 · 无分单')).toBeInTheDocument();
  expect(row('SE-004').getAllByText('未填写')).toHaveLength(2);
  expect(row('SE-004').queryByText('共 1 票')).not.toBeInTheDocument();
  expect(row('SE-005').getByText('单证结构待确认')).toBeInTheDocument();
  expect(row('SE-005').queryByText('直单 · 无分单')).not.toBeInTheDocument();
  const headers = Array.from(container.querySelectorAll('thead th')).map(
    (el) => el.textContent,
  );
  expect(headers.indexOf('主单 / 分单')).toBeLessThan(
    headers.indexOf('委托单位'),
  );
});

it('未启用关系展示的列表保留原主单号列', async () => {
  render(
    <OrderListTemplate
      queryOrders={vi.fn().mockResolvedValue({ data: [rows[0]], total: 1 })}
    />,
  );
  await screen.findByText('MBL-SHARED');
  expect(
    screen.getByRole('columnheader', { name: '主单号' }),
  ).toBeInTheDocument();
  expect(screen.queryByText('主单 / 分单')).not.toBeInTheDocument();
  expect(screen.queryByText('HBL-001')).not.toBeInTheDocument();
});
