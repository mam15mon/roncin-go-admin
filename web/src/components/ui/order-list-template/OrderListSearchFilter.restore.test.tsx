import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { OrderListSearchFilter } from './OrderListSearchFilter';

afterEach(cleanup);
it('恢复五类日期、候选外标签与名称，收展保留输入，重置不回旧初值', async () => {
  const onSearch = vi.fn();
  const onReset = vi.fn();
  const range: [string, string] = ['2026-10-01', '2026-10-12'];
  render(
    <App>
      <OrderListSearchFilter
        initialValues={{
          numberKeyword: '旧单号',
          customerId: 'customer-a',
          tagIds: ['tag-a'],
          createdAtRange: range,
          etaRange: range,
          etdRange: range,
          lockedAtRange: range,
          statusTimeRange: range,
        }}
        initialLabels={{ customerId: '客户甲', 'tagIds:tag-a': '高优先级' }}
        options={{ loadPartners: async () => [] }}
        onSearch={onSearch}
        onReset={onReset}
      />
    </App>,
  );
  expect(screen.getByText('客户甲')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: /展开更多/ }));
  expect(screen.getByText('高优先级')).toBeInTheDocument();
  expect(screen.getAllByDisplayValue('2026-10-01')).toHaveLength(5);
  fireEvent.change(screen.getByPlaceholderText('输入单号'), {
    target: { value: '未提交单号' },
  });
  fireEvent.click(screen.getByRole('button', { name: /收起筛选/ }));
  fireEvent.click(screen.getByRole('button', { name: /展开更多/ }));
  expect(screen.getByPlaceholderText('输入单号')).toHaveValue('未提交单号');
  expect(onSearch).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: /查询/ }));
  await waitFor(() =>
    expect(onSearch).toHaveBeenCalledWith(
      expect.objectContaining({
        numberKeyword: '未提交单号',
        createdAtRange: range,
        etaRange: range,
        etdRange: range,
        lockedAtRange: range,
        statusTimeRange: range,
        tagIds: ['tag-a'],
      }),
      expect.objectContaining({ 'tagIds:tag-a': '高优先级' }),
    ),
  );
  fireEvent.click(screen.getByRole('button', { name: /重置/ }));
  expect(onReset).toHaveBeenCalledOnce();
  expect(screen.getByPlaceholderText('输入单号')).toHaveValue('');
  fireEvent.click(screen.getByRole('button', { name: /查询/ }));
  await waitFor(() =>
    expect(onSearch).toHaveBeenLastCalledWith(
      expect.objectContaining({
        customerId: undefined,
        tagIds: undefined,
        numberKeyword: undefined,
        etdRange: undefined,
      }),
      {},
    ),
  );
});
it('恢复客户甲后改选客户乙，后续候选缺失仍保留乙名，提交名称与ID一致', async () => {
  const onSearch = vi.fn();
  const view = render(
    <App>
      <OrderListSearchFilter
        initialValues={{ customerId: 'customer-a' }}
        initialLabels={{ customerId: '客户甲' }}
        options={{ partners: [{ value: 'customer-b', label: '客户乙' }] }}
        onSearch={onSearch}
        onReset={vi.fn()}
      />
    </App>,
  );
  const input = screen.getByRole('combobox', { name: '委托单位' });
  fireEvent.mouseDown(input);
  fireEvent.click(await screen.findByText('客户乙'));
  view.rerender(
    <App>
      <OrderListSearchFilter
        initialValues={{ customerId: 'customer-a' }}
        initialLabels={{ customerId: '客户甲' }}
        options={{ partners: [] }}
        onSearch={onSearch}
        onReset={vi.fn()}
      />
    </App>,
  );
  expect(
    input
      .closest('.ant-select')
      ?.querySelector('.ant-select-content-has-value'),
  ).toHaveTextContent('客户乙');
  fireEvent.click(screen.getByRole('button', { name: /查询/ }));
  await waitFor(() =>
    expect(onSearch).toHaveBeenCalledWith(
      expect.objectContaining({ customerId: 'customer-b' }),
      expect.objectContaining({ customerId: '客户乙' }),
    ),
  );
});
