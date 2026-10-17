import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type { ActionType } from '@ant-design/pro-components';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { csvAmount } from '@/utils/csv';
import { FinanceLedgerTemplate } from './FinanceLedgerTemplate';

const rows = [
  { id: '1', name: '第一条', currency: 'USD', totalAmount: '-3.50000000' },
  { id: '2', name: '第二条', currency: 'EUR', totalAmount: '1234.50000000' },
];

type Page = { data: typeof rows; total: number; success?: boolean };

function setup(data = rows, load?: () => Promise<Page>) {
  const request = vi.fn<() => Promise<Page>>(
    load ?? (async () => ({ data, total: 100 })),
  );
  const actionRef = { current: undefined as ActionType | undefined };
  const create = vi
    .spyOn(URL, 'createObjectURL')
    .mockReturnValue('blob:ledger');
  const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  const click = vi
    .spyOn(HTMLAnchorElement.prototype, 'click')
    .mockImplementation(() => {});
  const view = render(
    <App>
      <FinanceLedgerTemplate
        pageTitle="导出台账"
        actionRef={actionRef}
        exportFileName="业务台账.csv"
        columns={[{ title: '名称', dataIndex: 'name', search: false }]}
        exportColumns={[
          { header: '名称', value: (row) => row.name },
          { header: '币种', value: (row) => row.currency },
          {
            header: '金额',
            kind: 'number',
            value: (row) => csvAmount(row.totalAmount),
          },
        ]}
        request={request}
        showSummaryBoard={false}
        search={false}
      />
    </App>,
  );
  return { ...view, request, create, revoke, click, actionRef };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe('台账实际导出交互', () => {
  it('未选择时只下载当前页，固定业务列不受可见表格列影响并回收资源', async () => {
    const { request, create, revoke, click } = setup();
    await screen.findByText('第一条');
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /导出当前页/ })),
    );
    const blob = create.mock.calls[0][0] as Blob;
    expect(await blob.text()).toBe(
      '\uFEFF名称,币种,金额\r\n第一条,USD,-3.50\r\n第二条,EUR,1234.50',
    );
    expect(click).toHaveBeenCalledOnce();
    expect(revoke).toHaveBeenCalledWith('blob:ledger');
    expect(document.querySelector('a[download]')).toBeNull();
    expect(
      await screen.findByText('已导出当前页 2 条数据'),
    ).toBeInTheDocument();
    expect(request).toHaveBeenCalledOnce();
  });

  it('选择行后明确提示范围且仅下载选中行', async () => {
    const { create, request } = setup();
    await screen.findByText('第二条');
    fireEvent.click(screen.getAllByRole('checkbox')[2]);
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /导出选中 \(1\)/ })),
    );
    expect(await (create.mock.calls[0][0] as Blob).text()).toBe(
      '\uFEFF名称,币种,金额\r\n第二条,EUR,1234.50',
    );
    expect(await screen.findByText('已导出选中 1 条数据')).toBeInTheDocument();
    expect(request).toHaveBeenCalledOnce();
  });

  it('空页提示且不创建下载资源', async () => {
    const { request, create, click } = setup([]);
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /导出当前页/ })),
    );
    expect(await screen.findByText('当前无数据可导出')).toBeInTheDocument();
    expect(create).not.toHaveBeenCalled();
    expect(click).not.toHaveBeenCalled();
  });

  it('新查询先返回、旧查询迟到后，导出保持官方表格已采纳的当前结果', async () => {
    let resolveOld!: (page: Page) => void;
    const oldPage = new Promise<Page>((resolve) => {
      resolveOld = resolve;
    });
    let attempts = 0;
    const { actionRef, request, create } = setup(rows, async () => {
      attempts++;
      return attempts === 1
        ? oldPage
        : { data: [{ ...rows[1], name: '当前B' }], total: 1 };
    });
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    await act(async () => {
      await actionRef.current?.reload();
    });
    expect(await screen.findByText('当前B')).toBeVisible();
    await act(async () => {
      resolveOld({ data: rows, total: 100 });
    });
    expect(screen.queryByText('第一条')).not.toBeInTheDocument();
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /导出当前页/ })),
    );
    expect(await (create.mock.calls[0][0] as Blob).text()).toBe(
      '\uFEFF名称,币种,金额\r\n当前B,EUR,1234.50',
    );
  });

  it('失败查询返回的部分数据不会替换仍在表格展示的导出快照', async () => {
    const { actionRef, request, create } = setup();
    await screen.findByText('第一条');
    request.mockResolvedValueOnce({
      data: [{ ...rows[0], name: '不应采纳的失败数据' }],
      total: 1,
      success: false,
    });
    await act(async () => {
      await actionRef.current?.reload();
    });
    expect(screen.queryByText('不应采纳的失败数据')).not.toBeInTheDocument();
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /导出当前页/ })),
    );
    expect(await (create.mock.calls[0][0] as Blob).text()).toBe(
      '\uFEFF名称,币种,金额\r\n第一条,USD,-3.50\r\n第二条,EUR,1234.50',
    );
  });
});
