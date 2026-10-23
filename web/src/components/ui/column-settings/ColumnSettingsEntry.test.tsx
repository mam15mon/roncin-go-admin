import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';

describe('列设置禁用生命周期', () => {
  afterEach(cleanup);
  it('已打开后禁用即关闭，解除禁用不自动打开且残留回调不得写入', async () => {
    const onChange = vi.fn();
    const props = {
      fields: [{ key: 'name', title: '姓名' }],
      value: { order: ['name'], hidden: [], freeze: { left: [], right: [] } },
      onChange,
      onReset: vi.fn(),
    };
    const { rerender } = render(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <ColumnSettingsEntry {...props} />
      </ConfigProvider>,
    );
    fireEvent.click(screen.getByRole('button', { name: '列设置' }));
    const dialog = await screen.findByRole('dialog', { name: '列设置' });
    const reset = screen.getByRole('button', { name: '恢复默认' });
    rerender(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <ColumnSettingsEntry {...props} disabled disabledReason="编辑中" />
      </ConfigProvider>,
    );
    fireEvent.click(reset);
    expect(props.onReset).not.toHaveBeenCalled();
    await act(async () => {
      fireEvent.animationEnd(dialog);
      fireEvent.transitionEnd(dialog);
    });
    rerender(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <ColumnSettingsEntry {...props} />
      </ConfigProvider>,
    );
    expect(screen.getByRole('button', { name: '列设置' })).toBeEnabled();
    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: '列设置' }),
      ).not.toBeInTheDocument(),
    );
  });
});
