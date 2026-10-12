import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ResponsiveProTable } from './ResponsiveTable';

describe('Pro表格公共调宽状态', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });
  it('调宽保留分页、搜索输入与选中行，不触发请求/排序，并据新宽度临时解除固定', async () => {
    let containerWidth = 1000;
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(
      () => containerWidth,
    );
    const request = vi.fn(async () => ({
      data: [{ id: '1', name: '张芳', email: 'test@example.com' }],
      total: 80,
      success: true,
    }));
    const { container } = render(
      <App>
        <ResponsiveProTable
          rowKey="id"
          request={request}
          columns={[
            { key: 'index', title: '序号', valueType: 'index', width: 50 },
            {
              key: 'name',
              title: '姓名',
              dataIndex: 'name',
              width: 180,
              fixed: 'left',
              sorter: true,
            },
            { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
            {
              key: 'option',
              title: '操作',
              valueType: 'option',
              width: 160,
              fixed: 'right',
            },
          ]}
          rowSelection={{}}
          options={{ setting: false }}
          sticky={false}
          pagination={{ defaultPageSize: 20 }}
        />
      </App>,
    );
    await screen.findByText('张芳');
    fireEvent.click(screen.getByTitle('2'));
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    const input = screen.getByLabelText('姓名');
    fireEvent.change(input, { target: { value: '未提交检索' } });
    const checkbox = screen
      .getAllByRole('checkbox')
      .find((node) => node.closest('tbody'));
    if (!checkbox) throw new Error('没有选择框');
    fireEvent.click(checkbox);
    const header = () => {
      const cell = screen
        .getAllByRole('columnheader')
        .find((node) => node.textContent === '姓名');
      if (!cell) throw new Error('没有姓名表头');
      return cell;
    };
    await waitFor(() =>
      expect(header()).toHaveClass('ant-table-cell-fix-start'),
    );
    const handle = () => {
      const element = header().querySelector<HTMLElement>(
        '[data-column-resize-handle]',
      );
      if (!element) throw new Error('没有调宽手柄');
      return element;
    };
    fireEvent.mouseDown(handle(), { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 520 });
    expect(
      document.querySelector('.roncin-table-resizer-guide'),
    ).not.toBeNull();
    fireEvent.mouseUp(window);
    fireEvent.click(handle());
    await waitFor(() =>
      expect(header()).not.toHaveClass('ant-table-cell-fix-start'),
    );
    expect(
      container.querySelectorAll<HTMLTableColElement>('colgroup col')[2].style
        .width,
    ).toBe('600px');
    expect(screen.getByLabelText('姓名')).toHaveValue('未提交检索');
    expect(screen.getByTitle('2')).toHaveClass('ant-pagination-item-active');
    expect(checkbox).toBeChecked();
    expect(request).toHaveBeenCalledTimes(2);
    expect(
      container.querySelectorAll('[data-column-resize-handle]'),
    ).toHaveLength(3);
    containerWidth = 1600;
    fireEvent.resize(window);
    await waitFor(() =>
      expect(header()).toHaveClass('ant-table-cell-fix-start'),
    );
    expect(header().style.position).toBe('');
    expect(request).toHaveBeenCalledTimes(2);
    await act(async () => {});
  });
});
