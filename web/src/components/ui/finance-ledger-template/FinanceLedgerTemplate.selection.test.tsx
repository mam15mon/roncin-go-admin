import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { FinanceLedgerTemplate } from './FinanceLedgerTemplate';
import type { FinanceLedgerSummaryConfig } from './types';

type Row = { id: string; name: string; amount: string };
const rows: Row[] = Array.from({ length: 5 }, (_, i) => ({
  id: String(i + 1),
  name: `明细${i + 1}`,
  amount: String(i + 1),
}));
const config: FinanceLedgerSummaryConfig<Row> = {
  groups: [{ direction: '', label: '金额' }],
  projectRow: (row) => ({
    direction: '',
    currency: 'USD',
    amount: row.amount,
    baseCurrency: 'CNY',
    baseAmount: row.amount,
  }),
};
function setup(nativeSearch = false) {
  const actionRef = { current: undefined as ActionType | undefined };
  const onSelectionChange = vi.fn();
  const request = vi.fn(
    async ({
      current = 1,
      pageSize = 2,
    }: {
      current?: number;
      pageSize?: number;
    }) => ({
      data: rows.slice((current - 1) * pageSize, current * pageSize),
      total: 5,
      matchedSummary: {
        count: 5,
        amounts: [
          {
            direction: '',
            currency: 'USD',
            amount: '15',
            baseCurrency: 'CNY',
            baseAmount: '15',
          },
        ],
      },
    }),
  );
  const viewFor = (scope: string) => (
    <App>
      <FinanceLedgerTemplate<Row>
        actionRef={actionRef}
        selectionScope={scope}
        summaryConfig={config}
        columns={[{ title: '名称', dataIndex: 'name' }]}
        pagination={{ defaultPageSize: 2 }}
        exportColumns={[]}
        request={request}
        rowSelection={{ onChange: onSelectionChange }}
        search={nativeSearch ? {} : false}
      />
    </App>
  );
  const view = render(viewFor('A'));
  return { ...view, request, actionRef, onSelectionChange, viewFor };
}
const board = () => screen.getByRole('status', { name: '金额合计' });
function select(name: string) {
  const row = screen.getByText(name).closest('tr');
  if (!row) throw new Error('缺少测试明细行');
  fireEvent.click(within(row).getByRole('checkbox'));
}

describe('财务台账跨页勾选', () => {
  it('原生搜索草稿保留选择，提交条件时立即清空而不等待请求结果', async () => {
    const { request } = setup(true);
    await screen.findByText('明细1');
    select('明细1');
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: '新条件' },
    });
    expect(board()).toHaveTextContent('已选 1 条');
    request.mockRejectedValueOnce(new Error('新条件查询失败'));
    fireEvent.click(screen.getByRole('button', { name: /查.*询/ }));
    await waitFor(() => expect(board()).not.toHaveTextContent('已选'));
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    expect(board()).not.toHaveTextContent('已选');
  });

  it('手动跨页2+2+1等于5，当前页全选不选择其他页，外部onChange不覆盖公共选择', async () => {
    const { onSelectionChange } = setup();
    await screen.findByText('明细1');
    expect(board()).toHaveTextContent('筛选结果共 5 条');
    expect(board()).toHaveTextContent('15.00 USD');
    fireEvent.click(screen.getAllByRole('checkbox')[0]);
    expect(board()).toHaveTextContent('已选 2 条');
    expect(board()).toHaveTextContent('3.00 USD');
    fireEvent.click(screen.getByTitle('2'));
    await screen.findByText('明细3');
    expect(board()).toHaveTextContent('已选 2 条');
    select('明细3');
    select('明细4');
    expect(board()).toHaveTextContent('已选 4 条');
    fireEvent.click(screen.getByTitle('3'));
    await screen.findByText('明细5');
    select('明细5');
    expect(board()).toHaveTextContent('已选 5 条');
    expect(board()).toHaveTextContent('15.00 USD');
    expect(onSelectionChange).toHaveBeenCalled();
    select('明细5');
    fireEvent.click(screen.getByTitle('2'));
    await screen.findByText('明细3');
    select('明细3');
    select('明细4');
    fireEvent.click(screen.getByTitle('1'));
    await screen.findByText('明细1');
    select('明细1');
    select('明细2');
    expect(board()).toHaveTextContent('筛选结果共 5 条');
  });

  it('切换筛选或公司立即清空选择，即使新查询失败也不恢复旧勾选', async () => {
    const { rerender, viewFor, actionRef, request } = setup();
    await screen.findByText('明细1');
    select('明细1');
    expect(board()).toHaveTextContent('已选 1 条');
    request.mockRejectedValueOnce(new Error('筛选失败'));
    rerender(viewFor('公司B/新筛选'));
    expect(board()).not.toHaveTextContent('已选');
    await act(async () => {
      await actionRef.current?.reload();
    });
    expect(board()).not.toHaveTextContent('已选');
    expect(board()).toHaveTextContent('汇总尚未加载');
    expect(
      screen
        .getAllByRole('checkbox')
        .slice(1)
        .every((input) => !(input as HTMLInputElement).checked),
    ).toBe(true);
  });

  it('同条件刷新保留勾选，当前页已选金额使用新采纳行，跨页快照仍保留', async () => {
    const { request, actionRef } = setup();
    await screen.findByText('明细1');
    select('明细1');
    request.mockResolvedValueOnce({
      data: [{ ...rows[0], amount: '20' }, rows[1]],
      total: 5,
      matchedSummary: {
        count: 5,
        amounts: [
          {
            direction: '',
            currency: 'USD',
            amount: '34',
            baseCurrency: 'CNY',
            baseAmount: '34',
          },
        ],
      },
    });
    await act(async () => {
      await actionRef.current?.reload();
    });
    await waitFor(() => expect(board()).toHaveTextContent('20.00 USD'));
    expect(board()).toHaveTextContent('已选 1 条');
  });
});
