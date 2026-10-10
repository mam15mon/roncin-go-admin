import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';

describe('列设置禁用生命周期', () => {
  afterEach(cleanup);

  it('已经打开后进入编辑禁用，关闭浮层且不再修改冻结', async () => {
    const onChange = vi.fn();
    const props = {
      fields: [{ key: 'name', title: '姓名' }],
      value: {
        order: ['name'],
        hidden: [],
        freeze: { first: false, last: false },
      },
      onChange,
      onReset: vi.fn(),
    };
    const { rerender } = render(<ColumnSettingsEntry {...props} />);
    fireEvent.click(screen.getByRole('button', { name: '列设置' }));
    const first = await screen.findByRole('switch', { name: '冻结首列' });
    rerender(
      <ColumnSettingsEntry {...props} disabled disabledReason="编辑中" />,
    );
    expect(screen.getByRole('button', { name: '列设置' })).toBeDisabled();
    fireEvent.click(first);
    expect(onChange).not.toHaveBeenCalled();
    const popup = first.closest('.ant-popover');
    if (!popup) throw new Error('未找到列设置浮层');
    // happy-dom 不产生 CSS transitionend，按 rc-motion 协议结束离场动画。
    await waitFor(() => expect(popup.className).toContain('leave-active'));
    await act(async () => {
      fireEvent.transitionEnd(popup);
      await new Promise((resolve) => requestAnimationFrame(resolve));
    });
    rerender(<ColumnSettingsEntry {...props} />);
    await act(async () => {});
    // 禁用期间清除打开状态，解除禁用不自动重新打开。
    expect(screen.getByRole('button', { name: '列设置' })).toBeEnabled();
    expect(popup).toHaveClass('ant-popover-hidden');
  });
});
