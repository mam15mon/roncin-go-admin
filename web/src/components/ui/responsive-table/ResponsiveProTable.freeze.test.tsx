import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Table } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ResponsiveProTable } from './ResponsiveTable';

type Row = { id: string; name: string };

describe('Pro表格动态冻结展示', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('同实例应用最新固定位置，不重取业务数据且保留序号/官方哨兵锚点', async () => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(1200);
    const request = vi.fn(async () => ({
      data: [{ id: '1', name: '张芳' }],
      total: 1,
      success: true,
    }));
    const props = {
      rowKey: 'id',
      search: false as const,
      options: { setting: false },
      pagination: false as const,
      request,
      rowSelection: {},
      expandable: { expandedRowRender: () => '展开资料' },
      sticky: false as const,
    };
    const columns = [
      { title: '序号', key: 'index', valueType: 'index' as const, width: 50 },
      { title: '姓名', key: 'name', dataIndex: 'name', width: 180 },
      {
        title: '操作',
        key: 'option',
        valueType: 'option' as const,
        width: 160,
      },
    ];
    const renderTable = (first: boolean, last: boolean) => (
      <App>
        <ResponsiveProTable<Row>
          {...props}
          columns={[
            Table.SELECTION_COLUMN,
            Table.EXPAND_COLUMN,
            ...columns.map((column) => ({
              ...column,
              fixed:
                column.key === 'name' && first
                  ? 'left'
                  : column.key === 'option' && last
                    ? 'right'
                    : false,
            })),
          ]}
        />
      </App>
    );
    const { rerender } = render(renderTable(false, false));
    await screen.findByText('张芳');
    const headers = () => screen.getAllByRole('columnheader');
    const titles = () => headers().map((node) => node.textContent);
    const initialTitles = titles();
    expect(initialTitles.slice(-3)).toEqual(['序号', '姓名', '操作']);
    rerender(renderTable(true, true));
    await waitFor(() =>
      expect(headers().find((node) => node.textContent === '姓名')).toHaveClass(
        'ant-table-cell-fix-start',
      ),
    );
    expect(titles()).toEqual(initialTitles);
    expect(headers().find((node) => node.textContent === '操作')).toHaveClass(
      'ant-table-cell-fix-end',
    );
    expect(request).toHaveBeenCalledTimes(1);
    rerender(renderTable(false, false));
    await waitFor(() =>
      expect(
        headers().find((node) => node.textContent === '姓名'),
      ).not.toHaveClass('ant-table-cell-fix-start'),
    );
    expect(
      headers().find((node) => node.textContent === '操作'),
    ).not.toHaveClass('ant-table-cell-fix-end');
    expect(titles()).toEqual(initialTitles);
    expect(screen.getByText('张芳')).toBeInTheDocument();
    expect(request).toHaveBeenCalledTimes(1);
    await act(async () => {});
  });

  it('窄宽切换释放与恢复fixed，同时保持输入顺序', async () => {
    let width = 1200;
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(
      () => width,
    );
    render(
      <App>
        <ResponsiveProTable<Row>
          rowKey="id"
          columns={[
            { key: 'index', title: '序号', valueType: 'index', width: 50 },
            {
              key: 'name',
              title: '姓名',
              dataIndex: 'name',
              width: 180,
              fixed: 'left',
            },
            { key: 'option', title: '操作', width: 160, fixed: 'right' },
          ]}
          dataSource={[{ id: '1', name: '张芳' }]}
          search={false}
          options={{ setting: false }}
          pagination={false}
          sticky={false}
        />
      </App>,
    );
    const headers = () => screen.getAllByRole('columnheader');
    const name = () => headers().find((node) => node.textContent === '姓名');
    await waitFor(() => expect(name()).toHaveClass('ant-table-cell-fix-start'));
    expect(headers().map((node) => node.textContent)).toEqual([
      '序号',
      '姓名',
      '操作',
    ]);
    width = 390;
    fireEvent.resize(window);
    await waitFor(() =>
      expect(name()).not.toHaveClass('ant-table-cell-fix-start'),
    );
    expect(headers().map((node) => node.textContent)).toEqual([
      '序号',
      '姓名',
      '操作',
    ]);
    width = 1200;
    fireEvent.resize(window);
    await waitFor(() => expect(name()).toHaveClass('ant-table-cell-fix-start'));
    expect(headers().map((node) => node.textContent)).toEqual([
      '序号',
      '姓名',
      '操作',
    ]);
    await act(async () => {});
  });
});
