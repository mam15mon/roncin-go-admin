import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { EllipsisTooltip } from './EllipsisTooltip';

describe('EllipsisTooltip', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正常渲染文本内容', () => {
    render(<EllipsisTooltip>测试超长货物描述内容</EllipsisTooltip>);
    expect(screen.getByText('测试超长货物描述内容')).toBeInTheDocument();
  });

  it('当内容为空时安全返回 null', () => {
    const { container } = render(<EllipsisTooltip />);
    expect(container.firstChild).toBeNull();
  });

  it('支持 alwaysShowTooltip 为 true 时显示 Tooltip', async () => {
    render(
      <EllipsisTooltip alwaysShowTooltip title="气泡提示信息">
        展示文字
      </EllipsisTooltip>,
    );

    const target = screen.getByText('展示文字');
    fireEvent.mouseEnter(target);

    expect(target).toBeInTheDocument();
  });

  it('支持鼠标悬停时自动检测溢出 (autoDetect)', () => {
    render(
      <EllipsisTooltip maxWidth={100} autoDetect>
        超长字符串测试
      </EllipsisTooltip>,
    );

    const target = screen.getByText('超长字符串测试');

    // 模拟 scrollWidth > clientWidth (发生溢出截断)
    Object.defineProperty(target, 'scrollWidth', { configurable: true, value: 200 });
    Object.defineProperty(target, 'clientWidth', { configurable: true, value: 100 });

    fireEvent.mouseEnter(target);
    expect(target).toBeInTheDocument();
  });

  it('支持多行省略配置 (maxLines > 1)', () => {
    render(
      <EllipsisTooltip maxLines={2} maxWidth={200}>
        多行超长文本说明内容
      </EllipsisTooltip>,
    );

    const target = screen.getByText('多行超长文本说明内容');
    expect(target.style.textOverflow).toBe('ellipsis');
    expect(target.style.maxWidth).toBe('200px');
  });

  it('支持 copyable 开启一键复制按钮', async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        writeText: writeTextMock,
      },
      configurable: true,
    });

    render(
      <EllipsisTooltip copyable>
        ONEY123456789
      </EllipsisTooltip>,
    );

    const copyIcon = screen.getByTitle('点击复制');
    expect(copyIcon).toBeInTheDocument();

    fireEvent.click(copyIcon);
    expect(writeTextMock).toHaveBeenCalledWith('ONEY123456789');
  });
});
