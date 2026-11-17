import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { QuickCreateSelectFooter } from './QuickCreateSelectFooter';

describe('下拉快捷创建入口', () => {
  it('保留候选内容，阻止新增按钮按下丢焦并允许键盘点击', () => {
    const create = vi.fn();
    render(
      <QuickCreateSelectFooter label="新增档案" onCreate={create}>
        <div>已有候选</div>
      </QuickCreateSelectFooter>,
    );
    const button = screen.getByRole('button', { name: '新增档案' });
    expect(screen.getByText('已有候选')).toBeVisible();
    expect(fireEvent.mouseDown(button)).toBe(false);
    fireEvent.click(button);
    expect(create).toHaveBeenCalledTimes(1);
  });
});
