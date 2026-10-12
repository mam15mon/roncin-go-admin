import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Table, type TableProps } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ResponsiveTable } from './ResponsiveTable';

type Row = {
  id: string;
  name: string;
  email: string;
  nested?: { name: string };
  'nested,name'?: string;
};
const rows: Row[] = [
  {
    id: '1',
    name: '张芳',
    email: 'test@example.com',
    nested: { name: '嵌套名' },
    'nested,name': '逗号名',
  },
];
const props: TableProps<Row> = {
  rowKey: 'id',
  dataSource: rows,
  pagination: false,
  sticky: false,
};
function header(title: string) {
  const cell = screen
    .getAllByRole('columnheader')
    .find((node) => node.textContent === title);
  if (!cell) throw new Error('没有表头');
  return cell;
}
function handle(title: string) {
  const element = header(title).querySelector<HTMLElement>(
    '[data-column-resize-handle]',
  );
  if (!element) throw new Error('没有调宽手柄');
  return element;
}
function drag(title: string, delta: number) {
  fireEvent.mouseDown(handle(title), { clientX: 100 });
  fireEvent.mouseMove(window, { clientX: 100 + delta });
  fireEvent.mouseUp(window);
}
function width(container: HTMLElement, index: number) {
  return container.querySelectorAll<HTMLTableColElement>('colgroup col')[index]
    ?.style.width;
}

describe('普通表格统一列宽', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('调宽以真实表头宽度起步，松手更新总横向宽度且不触发排序/业务事件', async () => {
    const onChange = vi.fn();
    const { container } = render(
      <ResponsiveTable
        {...props}
        onChange={onChange}
        columns={[
          {
            key: 'name',
            title: '姓名',
            dataIndex: 'name',
            width: 120,
            sorter: true,
          },
          { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
        ]}
      />,
    );
    expect(
      container.querySelector<HTMLTableElement>('table')?.style.tableLayout,
    ).toBe('auto');
    vi.spyOn(header('姓名'), 'getBoundingClientRect').mockReturnValue({
      width: 200,
      right: 300,
    } as DOMRect);
    fireEvent.mouseDown(handle('姓名'), { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 180 });
    expect(width(container, 0)).toBe('120px');
    fireEvent.mouseUp(window);
    expect(width(container, 0)).toBe('280px');
    expect(
      container.querySelector<HTMLTableElement>('table')?.style.tableLayout,
    ).toBe('fixed');
    expect(
      container.querySelector<HTMLTableElement>('table')?.style.width,
    ).toBe('540px');
    fireEvent.click(handle('姓名'));
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.click(header('姓名'));
    expect(onChange).toHaveBeenCalledTimes(1);
    await act(async () => {});
  });

  it('宽度按显式key跟随显隐/重排/冻结，不跟随数组位置或dataIndex别名', async () => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(1200);
    const name = {
      key: 'primary-name',
      title: '姓名',
      dataIndex: 'name',
      width: 180,
    };
    const email = {
      key: 'email',
      title: '邮箱',
      dataIndex: 'email',
      width: 260,
    };
    const { container, rerender } = render(
      <ResponsiveTable {...props} columns={[name, email]} />,
    );
    drag('姓名', 60);
    expect(width(container, 0)).toBe('240px');
    rerender(<ResponsiveTable {...props} columns={[email]} />);
    expect(width(container, 0)).toBe('260px');
    rerender(
      <ResponsiveTable
        {...props}
        columns={[email, { ...name, dataIndex: 'email', fixed: 'right' }]}
      />,
    );
    expect(width(container, 1)).toBe('240px');
    await waitFor(() =>
      expect(header('姓名')).toHaveClass('ant-table-cell-fix-end'),
    );
    expect(header('姓名').style.position).toBe('');
    drag('姓名', 40);
    expect(width(container, 1)).toBe('280px');
    await act(async () => {});
  });

  it('双击展开/还原，拖动和展开均遵守50–600px边界', async () => {
    const { container } = render(
      <ResponsiveTable
        {...props}
        columns={[
          { key: 'name', title: '姓名', dataIndex: 'name', width: 400 },
          { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
        ]}
      />,
    );
    fireEvent.doubleClick(handle('姓名'));
    expect(width(container, 0)).toBe('600px');
    fireEvent.doubleClick(handle('姓名'));
    expect(width(container, 0)).toBe('400px');
    drag('姓名', -1000);
    expect(width(container, 0)).toBe('50px');
    drag('姓名', 1000);
    expect(width(container, 0)).toBe('600px');
    await act(async () => {});
  });

  it('分组递归到叶子列，选择/展开/序号结构列不出现手柄', async () => {
    const columns: ColumnsType<Row> = [
      Table.SELECTION_COLUMN,
      Table.EXPAND_COLUMN,
      { key: 'index', title: '序号', width: 50, render: () => 1 },
      {
        key: 'group',
        title: '人员资料',
        children: [
          { title: '姓名', dataIndex: 'name', width: 180 },
          { title: '邮箱', dataIndex: 'email', width: 260 },
        ],
      },
    ];
    const { container } = render(
      <ResponsiveTable
        {...props}
        columns={columns}
        rowSelection={{}}
        expandable={{ expandedRowRender: () => '展开详情' }}
      />,
    );
    expect(
      container.querySelectorAll('[data-column-resize-handle]'),
    ).toHaveLength(2);
    expect(
      header('人员资料').querySelector('[data-column-resize-handle]'),
    ).toBeNull();
    expect(
      header('序号').querySelector('[data-column-resize-handle]'),
    ).toBeNull();
    drag('姓名', 20);
    expect(width(container, 3)).toBe('200px');
    fireEvent.click(screen.getByRole('button', { name: 'Expand row' }));
    expect(screen.getByText('展开详情')).toBeInTheDocument();
    await act(async () => {});
  });

  it('保留自定义header/body组件、原表头事件与样式，不泄漏调宽props', async () => {
    const onClick = vi.fn();
    const headerCell = vi.fn(
      (cellProps: React.ThHTMLAttributes<HTMLTableCellElement>) => (
        <th {...cellProps} data-custom-header />
      ),
    );
    const bodyCell = (
      cellProps: React.TdHTMLAttributes<HTMLTableCellElement>,
    ) => <td {...cellProps} data-custom-body />;
    const { container } = render(
      <ResponsiveTable
        {...props}
        components={{ header: { cell: headerCell }, body: { cell: bodyCell } }}
        columns={[
          {
            key: 'name',
            title: '姓名',
            dataIndex: 'name',
            width: 180,
            onHeaderCell: () => ({
              onClick,
              style: { color: 'red', position: 'sticky' },
              className: 'original-header',
            }),
          },
        ]}
      />,
    );
    expect(header('姓名')).toHaveClass('original-header');
    expect(header('姓名').style.position).toBe('sticky');
    expect(header('姓名').style.color).toBe('red');
    fireEvent.click(header('姓名'));
    expect(onClick).toHaveBeenCalledTimes(1);
    drag('姓名', 30);
    fireEvent.click(handle('姓名'));
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(width(container, 0)).toBe('210px');
    expect(container.querySelector('[data-custom-body]')).not.toBeNull();
    expect(
      headerCell.mock.calls.every(
        ([cellProps]) => !('columnResize' in cellProps),
      ),
    ).toBe(true);
    await act(async () => {});
  });

  it('百分比/无宽度列正常调宽，复合dataIndex不与逗号字段共用状态', async () => {
    const { container } = render(
      <ResponsiveTable
        {...props}
        columns={[
          { title: '嵌套姓名', dataIndex: ['nested', 'name'], width: '30%' },
          { title: '逗号姓名', dataIndex: 'nested,name' },
        ]}
      />,
    );
    vi.spyOn(header('嵌套姓名'), 'getBoundingClientRect').mockReturnValue({
      width: 250,
    } as DOMRect);
    vi.spyOn(header('逗号姓名'), 'getBoundingClientRect').mockReturnValue({
      width: 300,
    } as DOMRect);
    drag('嵌套姓名', 20);
    expect(width(container, 0)).toBe('270px');
    expect(width(container, 1)).toBe('300px');
    drag('逗号姓名', 30);
    expect(width(container, 1)).toBe('330px');
    fireEvent.doubleClick(handle('逗号姓名'));
    expect(width(container, 1)).toBe('480px');
    fireEvent.doubleClick(handle('逗号姓名'));
    expect(width(container, 1)).toBe('300px');
    await act(async () => {});
  });

  it('业务显式指定auto布局时调宽保留其布局契约', async () => {
    const { container } = render(
      <ResponsiveTable
        {...props}
        tableLayout="auto"
        columns={[
          { key: 'name', title: '姓名', dataIndex: 'name', width: 180 },
          { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
        ]}
      />,
    );
    drag('姓名', 20);
    expect(width(container, 0)).toBe('200px');
    expect(
      container.querySelector<HTMLTableElement>('table')?.style.tableLayout,
    ).toBe('auto');
    await act(async () => {});
  });

  it('首次操作同步实测业务/序号/选择/展开列宽，排除嵌套表并只在松手生效', async () => {
    const { container } = render(
      <ResponsiveTable
        {...props}
        rowSelection={{}}
        expandable={{
          expandedRowKeys: ['1'],
          expandedRowRender: () => (
            <ResponsiveTable
              {...props}
              columns={[
                {
                  key: 'email',
                  title: '内邮箱',
                  dataIndex: 'email',
                  width: 900,
                },
              ]}
            />
          ),
        }}
        columns={[
          { key: 'index', title: '序号', width: 50, render: () => 1 },
          { key: 'name', title: '姓名', dataIndex: 'name', width: 180 },
          { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
        ]}
      />,
    );
    for (const [title, actualWidth] of [
      ['序号', 60],
      ['姓名', 220],
      ['邮箱', 360],
      ['内邮箱', 900],
    ] as const) {
      vi.spyOn(header(title), 'getBoundingClientRect').mockReturnValue({
        width: actualWidth,
      } as DOMRect);
    }
    for (const [selector, actualWidth] of [
      ['th.ant-table-selection-column', 42],
      ['th.ant-table-row-expand-icon-cell', 54],
    ] as const) {
      const cell = container.querySelector<HTMLElement>(selector);
      if (!cell) throw new Error('没有结构表头');
      vi.spyOn(cell, 'getBoundingClientRect').mockReturnValue({
        width: actualWidth,
      } as DOMRect);
    }
    fireEvent.mouseDown(handle('姓名'), { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 120 });
    expect(width(container, 3)).toBe('180px');
    expect(width(container, 4)).toBe('260px');
    fireEvent.mouseUp(window);
    expect(width(container, 0)).toBe('54px');
    expect(width(container, 1)).toBe('42px');
    expect(width(container, 2)).toBe('60px');
    expect(width(container, 3)).toBe('240px');
    expect(width(container, 4)).toBe('360px');
    expect(
      container.querySelector<HTMLTableElement>('table')?.style.width,
    ).toBe('756px');
    expect(
      header('序号').querySelector('[data-column-resize-handle]'),
    ).toBeNull();
    await act(async () => {});
  });
});
