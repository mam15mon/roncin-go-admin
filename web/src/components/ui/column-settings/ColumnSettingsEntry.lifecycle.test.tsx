import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';
import type { ColumnSettingsPanelProps } from './ColumnSettingsPanel';

const captured = vi.hoisted(() => ({
  current: null as ColumnSettingsPanelProps | null,
}));
vi.mock('./ColumnSettingsPanel', () => ({
  default: (props: ColumnSettingsPanelProps) => {
    captured.current = props;
    return <div>配置面板</div>;
  },
}));

const value = { order: ['name'], hidden: [], freeze: { left: [], right: [] } };
const next = { ...value, freeze: { left: ['name'], right: [] } };

function setup() {
  const onChange = vi.fn();
  const onReset = vi.fn();
  const view = render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <ColumnSettingsEntry
        fields={[{ key: 'name', title: '姓名' }]}
        value={value}
        onChange={onChange}
        onReset={onReset}
        advanced={<div>高级内容</div>}
      />
    </ConfigProvider>,
  );
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
  return {
    ...view,
    onChange,
    onReset,
    callbacks: captured.current as ColumnSettingsPanelProps,
  };
}

describe('列设置打开批次回调隔离', () => {
  afterEach(() => {
    cleanup();
    captured.current = null;
  });

  it('正常操作可写入，关闭后和重新打开后旧回调均失效', () => {
    const { callbacks, onChange, onReset } = setup();
    act(() => callbacks.onChange(next));
    expect(onChange).toHaveBeenCalledWith(next);
    onChange.mockClear();
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    act(() => {
      callbacks.onChange(next);
      callbacks.onReset();
      callbacks.onOpenAdvanced?.();
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(onReset).not.toHaveBeenCalled();
    expect(screen.queryByText('高级内容')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '列设置' }));
    act(() => {
      callbacks.onChange(next);
      callbacks.onReset();
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(onReset).not.toHaveBeenCalled();
    act(() => captured.current?.onChange(next));
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('进入高级设置后旧面板不得改写偏好', () => {
    const { callbacks, onChange, onReset } = setup();
    act(() => callbacks.onOpenAdvanced?.());
    expect(screen.getByText('高级内容')).toBeInTheDocument();
    act(() => {
      callbacks.onChange(next);
      callbacks.onReset();
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(onReset).not.toHaveBeenCalled();
  });

  it('卸载后残留回调失效，账号或组织切换的旧实例不得写入', () => {
    const { callbacks, onChange, onReset, unmount } = setup();
    unmount();
    act(() => {
      callbacks.onChange(next);
      callbacks.onReset();
    });
    expect(onChange).not.toHaveBeenCalled();
    expect(onReset).not.toHaveBeenCalled();
  });
});
