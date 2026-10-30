import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { OrderListTemplate } from './OrderListTemplate';

const TABLE_KEY = 'orders:default-columns-test';
const STORAGE_KEY = `roncin:column-settings:v1:${TABLE_KEY}:anonymous:default`;
const SECONDARY_TITLES = [
  '序号',
  '业务类型',
  '创建人',
  '订单锁定时间',
  '配舱备注',
  '订舱备注',
  '操作备注',
  '提成',
];

beforeEach(() => localStorage.clear());
afterEach(cleanup);

function renderOrders() {
  return render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        columnSettingsKey={TABLE_KEY}
        queryOrders={vi.fn().mockResolvedValue({
          data: [
            { id: 'order-1', orderNo: 'SE-001' },
            { id: 'order-2', orderNo: 'SE-002' },
          ],
          total: 2,
        })}
        extraColumns={[
          {
            title: '提成',
            dataIndex: 'commissionSummary',
            render: () => '待发 ¥100.00',
          },
        ]}
        trailingColumns={[
          { title: '订单审核', key: 'feeReview', render: () => '未审核' },
        ]}
      />
    </ConfigProvider>,
  );
}

it('辅助列默认隐藏，序号与提成可重新开启并恢复精简默认布局', async () => {
  const { container } = renderOrders();
  await screen.findByText('SE-001');
  const headers = within(
    container.querySelector('thead') as HTMLTableSectionElement,
  );
  for (const title of SECONDARY_TITLES) {
    expect(headers.queryByText(title)).not.toBeInTheDocument();
  }
  for (const title of [
    '订单编号',
    '委托单位',
    '船名航次',
    '箱型箱量',
    '操作人员',
    '操作进度',
    '订单审核',
    '异常',
    '操作',
  ]) {
    expect(headers.getByText(title)).toBeInTheDocument();
  }

  fireEvent.click(screen.getByLabelText('列设置'));
  const sequenceCheckbox = await screen.findByLabelText('序号', {
    selector: 'input[type=checkbox]',
  });
  for (const title of SECONDARY_TITLES) {
    expect(
      screen.getByLabelText(title, { selector: 'input[type=checkbox]' }),
    ).not.toBeChecked();
  }
  fireEvent.click(sequenceCheckbox);
  fireEvent.click(
    screen.getByLabelText('提成', { selector: 'input[type=checkbox]' }),
  );
  await waitFor(() => {
    expect(headers.getByText('序号')).toBeInTheDocument();
    expect(headers.getByText('提成')).toBeInTheDocument();
  });
  const firstRow = within(
    screen.getByText('SE-001').closest('tr') as HTMLElement,
  );
  const secondRow = within(
    screen.getByText('SE-002').closest('tr') as HTMLElement,
  );
  expect(firstRow.getByText('1')).toBeInTheDocument();
  expect(secondRow.getByText('2')).toBeInTheDocument();
  expect(firstRow.getByText('待发 ¥100.00')).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
  await waitFor(() => {
    for (const title of SECONDARY_TITLES) {
      expect(headers.queryByText(title)).not.toBeInTheDocument();
    }
  });
  expect(headers.getByText('订单审核')).toBeInTheDocument();
});

it('保留已保存的辅助列显隐、顺序和冻结偏好，不改写浏览器存储', async () => {
  const saved = JSON.stringify({
    order: ['commissionSummary', 'businessType', 'orderNo', 'operatorName'],
    hidden: ['operatorName'],
    freeze: { left: ['commissionSummary'], right: ['option'] },
  });
  localStorage.setItem(STORAGE_KEY, saved);
  const { container } = renderOrders();
  await screen.findByText('SE-001');
  const headerElement = container.querySelector(
    'thead',
  ) as HTMLTableSectionElement;
  const headers = within(headerElement);
  expect(headers.getByText('提成')).toBeInTheDocument();
  expect(headers.getByText('业务类型')).toBeInTheDocument();
  expect(headers.queryByText('操作人员')).not.toBeInTheDocument();
  expect(headers.queryByText('序号')).not.toBeInTheDocument();
  const titles = Array.from(headerElement.querySelectorAll('th')).map(
    (cell) => cell.textContent,
  );
  expect(titles.indexOf('提成')).toBeLessThan(titles.indexOf('业务类型'));
  expect(titles.indexOf('业务类型')).toBeLessThan(titles.indexOf('订单编号'));
  expect(localStorage.getItem(STORAGE_KEY)).toBe(saved);
});
