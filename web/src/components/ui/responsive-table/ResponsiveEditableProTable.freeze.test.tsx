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
import { ResponsiveEditableProTable } from './ResponsiveTable';

describe('可编辑Pro表格动态冻结', () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('改变固定位置后编辑输入节点、焦点和未提交值保留', async () => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(1200);
    const value = [{ id: '1', name: '张芳', email: 'test@example.com' }];
    const props = {
      rowKey: 'id',
      value,
      onChange: vi.fn(),
      recordCreatorProps: false as const,
      editable: {
        type: 'multiple' as const,
        editableKeys: ['1'],
        actionRender: () => [],
      },
      options: { setting: false },
      pagination: false as const,
      sticky: false as const,
    };
    const renderTable = (first: boolean) => (
      <App>
        <ResponsiveEditableProTable
          {...props}
          columns={[
            { key: 'index', title: '序号', valueType: 'index', width: 50 },
            {
              key: 'name',
              title: '姓名',
              dataIndex: 'name',
              width: 180,
              fixed: first ? 'left' : false,
            },
            { key: 'email', title: '邮箱', dataIndex: 'email', width: 260 },
          ]}
        />
      </App>
    );
    const { rerender } = render(renderTable(false));
    const input = await screen.findByDisplayValue('张芳');
    fireEvent.change(input, { target: { value: '张芳未提交' } });
    input.focus();
    expect(input).toHaveFocus();
    rerender(renderTable(true));
    await waitFor(() =>
      expect(
        screen
          .getAllByRole('columnheader')
          .find((node) => node.textContent === '姓名'),
      ).toHaveClass('ant-table-cell-fix-start'),
    );
    expect(screen.getByDisplayValue('张芳未提交')).toBe(input);
    expect(input).toHaveFocus();
    expect(
      screen.getAllByRole('columnheader').map((node) => node.textContent),
    ).toEqual(['序号', '姓名', '邮箱']);

    const nameHeader = screen
      .getAllByRole('columnheader')
      .find((node) => node.textContent === '姓名');
    if (!nameHeader) throw new Error('没有姓名表头');
    const resizeHandle = nameHeader.querySelector<HTMLElement>(
      '[data-column-resize-handle]',
    );
    if (!resizeHandle) throw new Error('没有调宽手柄');
    fireEvent.mouseDown(resizeHandle, { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 180 });
    fireEvent.mouseUp(window);
    expect(nameHeader.style.position).toBe('');
    expect(screen.getByDisplayValue('张芳未提交')).toBe(input);
    expect(input).toHaveFocus();
    fireEvent.doubleClick(resizeHandle);
    expect(screen.getByDisplayValue('张芳未提交')).toBe(input);
    expect(input).toHaveFocus();
    await act(async () => {});
  });
});
