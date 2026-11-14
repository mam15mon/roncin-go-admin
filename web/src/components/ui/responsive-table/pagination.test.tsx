import { ProTable } from '@ant-design/pro-components';
import { paginationSizeChanger } from '@root/config/pagination';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { Pagination, Select, Table } from 'antd';
import { describe, expect, it, vi } from 'vitest';

describe('分页页容量选择器', () => {
  it('普通表格继承全局配置，选择页容量前后都没有文本输入框', async () => {
    const onChange = vi.fn();
    const { container } = renderWithApp(
      <Table
        rowKey="id"
        columns={[{ title: '名称', dataIndex: 'name' }]}
        dataSource={[{ id: '1', name: '测试数据' }]}
        pagination={{ total: 100, onChange }}
      />,
    );
    const pagination = container.querySelector('.ant-pagination');
    if (!pagination) throw new Error('未找到分页控件');
    expect(pagination.querySelector('input, textarea')).toBeNull();
    const trigger = within(pagination as HTMLElement).getByRole('combobox');
    expect(trigger.tagName).toBe('BUTTON');
    expect(trigger).toHaveAttribute('type', 'button');
    fireEvent.mouseDown(trigger);
    fireEvent.click(await screen.findByText('20 条/页'));
    await waitFor(() => expect(onChange).toHaveBeenCalledWith(1, 20));
    expect(pagination.querySelector('input, textarea')).toBeNull();
  });

  it('ProTable 显式配置后仍可用键盘选择页容量', async () => {
    const onChange = vi.fn();
    const { container } = renderWithApp(
      <ProTable
        rowKey="id"
        search={false}
        options={false}
        columns={[{ title: '名称', dataIndex: 'name' }]}
        request={async () => ({
          data: [{ id: '1', name: '测试数据' }],
          total: 100,
          success: true,
        })}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: paginationSizeChanger,
          onChange,
        }}
      />,
    );
    await screen.findByText('测试数据');
    const pagination = container.querySelector('.ant-pagination');
    if (!pagination) throw new Error('未找到分页控件');
    const trigger = within(pagination as HTMLElement).getByRole('combobox');
    act(() => trigger.focus());
    expect(trigger).toHaveFocus();
    fireEvent.keyDown(trigger, { key: 'ArrowDown', keyCode: 40 });
    await screen.findByRole('option', { name: '20 条/页' });
    fireEvent.keyDown(trigger, { key: 'ArrowDown', keyCode: 40 });
    fireEvent.keyDown(trigger, { key: 'Enter', keyCode: 13 });
    await waitFor(() => expect(onChange).toHaveBeenCalledWith(1, 20));
    expect(pagination.querySelector('input, textarea')).toBeNull();
  });

  it('禁用和隐藏页容量控件继续生效，普通搜索下拉仍可输入', () => {
    const { container } = renderWithApp(
      <>
        <Pagination total={100} disabled />
        <Pagination total={100} showSizeChanger={false} />
        <Select aria-label="业务搜索" showSearch options={[]} />
      </>,
    );
    const paginations = container.querySelectorAll('.ant-pagination');
    const disabledTrigger = within(paginations[0] as HTMLElement).getByRole(
      'combobox',
    );
    expect(disabledTrigger).toBeDisabled();
    expect(
      within(paginations[1] as HTMLElement).queryByRole('combobox'),
    ).toBeNull();
    expect(screen.getByRole('combobox', { name: '业务搜索' }).tagName).toBe(
      'INPUT',
    );
    expect(
      screen.getByRole('combobox', { name: '业务搜索' }),
    ).not.toHaveAttribute('readonly');
  });
});
