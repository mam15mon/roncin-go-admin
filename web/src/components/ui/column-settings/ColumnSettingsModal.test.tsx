import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ColumnSettingsModal from './ColumnSettingsModal';
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

const DEFAULT_VALUE: ColumnSettingsValue = {
  order: ['code', 'customer', 'note', 'option'],
  hidden: ['note'],
};

function renderModal(props: Partial<Parameters<typeof ColumnSettingsModal>[0]> = {}) {
  const onSave = vi.fn();
  const onCancel = vi.fn();
  render(
    <App>
      <ColumnSettingsModal
        open
        fields={FIELDS}
        value={VALUE}
        defaultValue={DEFAULT_VALUE}
        onSave={onSave}
        onCancel={onCancel}
        {...props}
      />
    </App>,
  );
  return { onSave, onCancel };
}

describe('ColumnSettingsModal', () => {
  afterEach(() => {
    cleanup();
  });

  it('打开时以当前配置为草稿基线并显示计数', () => {
    renderModal();
    expect(screen.getByText('已显示 4 / 4')).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: '业务单号' })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: '操作' })).toBeDisabled();
    expect(screen.getByText('必显')).toBeInTheDocument();
  });

  it('取消显隐后保存返回新的顺序与隐藏集合', () => {
    const { onSave } = renderModal();
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    expect(screen.getByText('已显示 3 / 4')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^保\s*存$/ }));
    expect(onSave).toHaveBeenCalledWith({
      order: ['code', 'customer', 'note', 'option'],
      hidden: ['customer'],
    });
  });

  it('取消按钮不提交草稿', () => {
    const { onSave, onCancel } = renderModal();
    fireEvent.click(screen.getByRole('checkbox', { name: '业务单号' }));
    fireEvent.click(screen.getByRole('button', { name: /^取\s*消$/ }));
    expect(onSave).not.toHaveBeenCalled();
    expect(onCancel).toHaveBeenCalled();
  });

  it('必显列不可隐藏；唯一可见列取消勾选被阻止', () => {
    renderModal();
    expect(screen.getByRole('checkbox', { name: '操作' })).toBeDisabled();
    fireEvent.click(screen.getByRole('checkbox', { name: '业务单号' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
    expect(screen.getByText('已显示 1 / 4')).toBeInTheDocument();
  });

  it('无必显列时至少保留一列可见', () => {
    const plainFields: ColumnSettingsField[] = [
      { key: 'a', title: '列甲' },
      { key: 'b', title: '列乙' },
      { key: 'c', title: '列丙' },
    ];
    const plainValue: ColumnSettingsValue = {
      order: ['a', 'b', 'c'],
      hidden: [],
    };
    renderModal({ fields: plainFields, value: plainValue });
    fireEvent.click(screen.getByRole('checkbox', { name: '列甲' }));
    fireEvent.click(screen.getByRole('checkbox', { name: '列乙' }));
    expect(screen.getByRole('checkbox', { name: '列丙' })).toBeDisabled();
    expect(screen.getByText('已显示 1 / 3')).toBeInTheDocument();
  });

  it('恢复默认只重置草稿，保存后才提交默认配置', () => {
    const { onSave } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    expect(screen.getByRole('checkbox', { name: '备注' })).not.toBeChecked();
    expect(screen.getByText('已显示 3 / 4')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^保\s*存$/ }));
    expect(onSave).toHaveBeenCalledWith(DEFAULT_VALUE);
  });

  it('搜索过滤列表且搜索中禁用排序；空结果给出提示', () => {
    renderModal();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '客户' },
    });
    expect(screen.getByText('客户名称')).toBeInTheDocument();
    expect(screen.queryByText('业务单号')).not.toBeInTheDocument();
    expect(
      screen.getByText('搜索中无法调整顺序，清空搜索后可拖拽或使用箭头排序'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '上移客户名称' }),
    ).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '不存在' },
    });
    expect(screen.getByText('未找到匹配的列')).toBeInTheDocument();
  });

  it('上移下移在完整顺序内交换且首尾边界禁用', () => {
    const { onSave } = renderModal();
    expect(
      screen.getByRole('button', { name: '上移业务单号' }),
    ).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '下移业务单号' }));
    fireEvent.click(screen.getByRole('button', { name: /^保\s*存$/ }));
    expect(onSave).toHaveBeenCalledWith({
      order: ['customer', 'code', 'note', 'option'],
      hidden: [],
    });
  });

  it('上下移不得跨固定区域排序', () => {
    const { onSave } = renderModal();
    // 操作列固定在右侧：下移已到边界禁用；上移目标跨入动态列同样被禁止。
    expect(screen.getByRole('button', { name: '下移操作' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '上移操作' }));
    fireEvent.click(screen.getByRole('button', { name: /^保\s*存$/ }));
    expect(onSave).toHaveBeenCalledWith({
      order: ['code', 'customer', 'note', 'option'],
      hidden: [],
    });
  });

  it('提供高级内容时出现列配置与高级设置页签', () => {
    renderModal({ advanced: <div>高级设置内容</div>, width: 960 });
    fireEvent.click(screen.getByRole('tab', { name: '高级设置' }));
    expect(screen.getByText('高级设置内容')).toBeInTheDocument();
  });
});
