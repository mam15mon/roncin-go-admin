import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React, { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ColumnSettingsPanel from './ColumnSettingsPanel';
import type { ColumnSettingsField, ColumnSettingsValue } from './types';

const FIELDS: ColumnSettingsField[] = [
  { key: 'code', title: '业务单号' },
  { key: 'customer', title: '客户名称' },
  { key: 'note', title: '备注' },
  { key: 'option', title: '操作', lockVisible: true, fixed: 'right' },
];

const VALUE: ColumnSettingsValue = {
  order: ['code', 'customer', 'note', 'option'],
  hidden: [],
};

function renderPanel(
  props: {
    fields?: ColumnSettingsField[];
    initial?: ColumnSettingsValue;
    onOpenAdvanced?: () => void;
  } = {},
) {
  const onChange = vi.fn();
  const onReset = vi.fn();
  function Harness() {
    const [value, setValue] = useState<ColumnSettingsValue>(
      props.initial ?? VALUE,
    );
    return (
      <ColumnSettingsPanel
        fields={props.fields ?? FIELDS}
        value={value}
        onChange={(next) => {
          onChange(next);
          setValue(next);
        }}
        onReset={onReset}
        onOpenAdvanced={props.onOpenAdvanced}
      />
    );
  }
  render(<Harness />);
  return { onChange, onReset };
}

describe('ColumnSettingsPanel', () => {
  afterEach(() => {
    cleanup();
  });

  it('渲染计数、必显标记与禁用态', () => {
    renderPanel();
    expect(screen.getByText('已显示 4 / 4')).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: '业务单号' })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: '操作' })).toBeDisabled();
    expect(screen.getByText('必显')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '下移操作' })).toBeDisabled();
  });

  it('取消勾选立即回调新的隐藏集合并更新计数', () => {
    const { onChange } = renderPanel();
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    expect(onChange).toHaveBeenCalledWith({
      order: ['code', 'customer', 'note', 'option'],
      hidden: ['customer'],
    });
    expect(screen.getByText('已显示 3 / 4')).toBeInTheDocument();
  });

  it('必显列不可隐藏；唯一可见列取消勾选被阻止', () => {
    renderPanel();
    fireEvent.click(screen.getByRole('checkbox', { name: '业务单号' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
    expect(screen.getByText('已显示 1 / 4')).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: '操作' })).toBeDisabled();
  });

  it('无必显列时至少保留一列可见', () => {
    const plainFields: ColumnSettingsField[] = [
      { key: 'a', title: '列甲' },
      { key: 'b', title: '列乙' },
      { key: 'c', title: '列丙' },
    ];
    renderPanel({
      fields: plainFields,
      initial: { order: ['a', 'b', 'c'], hidden: [] },
    });
    fireEvent.click(screen.getByRole('checkbox', { name: '列甲' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '列乙' }));
    expect(screen.getByRole('checkbox', { name: '列丙' })).toBeDisabled();
    expect(screen.getByText('已显示 1 / 3')).toBeInTheDocument();
  });

  it('上下移立即回调完整顺序且首尾边界禁用', () => {
    const { onChange } = renderPanel();
    expect(screen.getByRole('button', { name: '上移业务单号' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '下移业务单号' }));
    expect(onChange).toHaveBeenCalledWith({
      order: ['customer', 'code', 'note', 'option'],
      hidden: [],
    });
  });

  it('上下移不得跨固定区域排序', () => {
    const { onChange } = renderPanel();
    // 操作列固定在右侧：下移已到边界禁用；上移目标跨入动态列同样被禁止。
    expect(screen.getByRole('button', { name: '下移操作' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '上移操作' }));
    expect(onChange).not.toHaveBeenCalled();
  });

  it('恢复默认回调 onReset 而非 onChange', () => {
    const { onReset, onChange } = renderPanel();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    expect(onReset).toHaveBeenCalledTimes(1);
    expect(onChange).not.toHaveBeenCalled();
  });

  it('搜索过滤列表且搜索中禁用排序；空结果给出提示', () => {
    renderPanel();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '客户' },
    });
    expect(screen.getByText('客户名称')).toBeInTheDocument();
    expect(screen.queryByText('业务单号')).not.toBeInTheDocument();
    expect(
      screen.getByText('搜索中无法调整顺序，清空搜索后可拖拽或使用箭头排序'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '上移客户名称' })).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '不存在' },
    });
    expect(screen.getByText('未找到匹配的列')).toBeInTheDocument();
  });

  it('提供高级内容入口时出现「更多设置」并回调打开', () => {
    const onOpenAdvanced = vi.fn();
    renderPanel({ onOpenAdvanced });
    fireEvent.click(screen.getByRole('button', { name: '更多设置' }));
    expect(onOpenAdvanced).toHaveBeenCalledTimes(1);
  });

  it('未提供高级内容入口时不出现「更多设置」', () => {
    renderPanel();
    expect(
      screen.queryByRole('button', { name: '更多设置' }),
    ).not.toBeInTheDocument();
  });
});
