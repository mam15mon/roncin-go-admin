import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React, { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ColumnSettingsPanel from './ColumnSettingsPanel';
import type { ColumnSettingsField, ColumnSettingsValue } from './types';

const fields: ColumnSettingsField[] = [
  { key: 'code', title: '业务单号' },
  { key: 'customer', title: '客户名称' },
  { key: 'note', title: '备注', lockVisible: true },
];
function setup(
  initial: ColumnSettingsValue = {
    order: ['code', 'customer', 'note'],
    hidden: [],
    freeze: { left: [], right: [] },
  },
  customFields = fields,
) {
  const onChange = vi.fn();
  const onReset = vi.fn();
  const onOpenAdvanced = vi.fn();
  function Harness() {
    const [value, setValue] = useState(initial);
    return (
      <ColumnSettingsPanel
        fields={customFields}
        structuralFields={[{ key: 'option', title: '操作', fixed: 'right' }]}
        value={value}
        onChange={(next) => {
          setValue(next);
          onChange(next);
        }}
        onReset={onReset}
        onOpenAdvanced={onOpenAdvanced}
      />
    );
  }
  render(<Harness />);
  return { onChange, onReset, onOpenAdvanced };
}
function row(key: string) {
  const element = document.querySelector(`[data-column-key="${key}"]`);
  if (!element) throw new Error('未找到字段');
  return element;
}
function drag(key: string, target: string) {
  fireEvent.dragStart(
    screen.getByRole('img', {
      name: `拖动${fields.find((field) => field.key === key)?.title}`,
    }),
  );
  fireEvent.drop(row(target), { clientX: 0 });
}
describe('全量铺排列设置', () => {
  afterEach(cleanup);
  it('连续编号、必显约束和全选保持一致', () => {
    const { onChange } = setup();
    expect(row('note')).toHaveAttribute('data-column-position', '3');
    expect(screen.getByRole('checkbox', { name: '备注' })).toBeDisabled();
    fireEvent.click(screen.getByRole('checkbox', { name: /列展示/ }));
    expect(onChange).toHaveBeenLastCalledWith({
      order: ['code', 'customer', 'note'],
      hidden: ['code', 'customer'],
      freeze: { left: [], right: [] },
    });
    fireEvent.click(screen.getByRole('checkbox', { name: /列展示/ }));
    expect(screen.getByRole('checkbox', { name: '客户名称' })).toBeChecked();
  });
  it('至少保留一列可见，结构操作列不参与显隐', () => {
    setup(
      undefined,
      fields.map((field) => ({ ...field, lockVisible: false })),
    );
    fireEvent.click(screen.getByRole('checkbox', { name: /列展示/ }));
    expect(screen.getByRole('checkbox', { name: '业务单号' })).toBeDisabled();
    expect(screen.queryByRole('checkbox', { name: '操作' })).toBeNull();
    expect(
      screen.getByRole('combobox', { name: '操作冻结位置' }),
    ).toBeInTheDocument();
  });
  it('跨行拖动和精确序号移位均更新完整顺序', () => {
    const { onChange } = setup();
    drag('note', 'code');
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ order: ['note', 'code', 'customer'] }),
    );
    const input = screen.getByRole('spinbutton', { name: '客户名称目标序号' });
    fireEvent.change(input, { target: { value: '1' } });
    fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ order: ['customer', 'note', 'code'] }),
    );
    expect(row('customer')).toHaveAttribute('data-column-position', '1');
  });
  it('可拖入空冻结区并跨区排序，取消不写入', () => {
    const { onChange } = setup();
    const handle = screen.getByRole('img', { name: '拖动客户名称' });
    fireEvent.dragStart(handle);
    fireEvent.dragEnd(handle);
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.dragStart(handle);
    fireEvent.drop(
      document.querySelector('[data-freeze-zone="left"]') as HTMLElement,
    );
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        order: ['customer', 'code', 'note'],
        freeze: { left: ['customer'], right: [] },
      }),
    );
    drag('note', 'customer');
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        order: ['note', 'customer', 'code'],
        freeze: { left: ['customer', 'note'], right: [] },
      }),
    );
  });
  it('搜索保留全局编号并禁止重排，显隐不改变冻结', () => {
    const { onChange } = setup({
      order: ['code', 'customer', 'note'],
      hidden: [],
      freeze: { left: ['code', 'customer'], right: [] },
    });
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        freeze: { left: ['code', 'customer'], right: [] },
        hidden: ['customer'],
      }),
    );
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '客户' },
    });
    expect(row('customer')).toHaveAttribute('data-column-position', '2');
    expect(
      screen.getByRole('spinbutton', { name: '客户名称目标序号' }),
    ).toBeDisabled();
    expect(screen.queryByRole('checkbox', { name: '业务单号' })).toBeNull();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '不存在' },
    });
    expect(screen.getByText('未找到匹配的列')).toBeInTheDocument();
  });
  it('恢复默认及高级设置仍由调用方管理', () => {
    const { onReset, onOpenAdvanced } = setup();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    fireEvent.click(screen.getByRole('button', { name: '更多设置' }));
    expect(onReset).toHaveBeenCalledOnce();
    expect(onOpenAdvanced).toHaveBeenCalledOnce();
  });
});
