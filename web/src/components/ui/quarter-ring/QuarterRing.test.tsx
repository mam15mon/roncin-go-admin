import { render, screen } from '@testing-library/react';
import { Spin } from 'antd';
import { describe, expect, it } from 'vitest';
import { QuarterRing } from './QuarterRing';

describe('QuarterRing', () => {
  it('正常渲染带有 status 角色与 loading aria-label 的元素', () => {
    render(<QuarterRing />);
    const spinner = screen.getByRole('status');
    expect(spinner).toBeInTheDocument();
    expect(spinner).toHaveAttribute('aria-label', 'loading');
    expect(screen.getByText('Loading')).toHaveClass('sr-only');
  });

  it('正确应用自定义尺寸与线宽', () => {
    render(<QuarterRing size={28} strokeWidth={3} />);
    const spinner = screen.getByRole('status');
    expect(spinner).toHaveStyle({
      width: '28px',
      height: '28px',
      borderWidth: '3px',
    });
  });

  it('支持字符串尺寸与耗时配置', () => {
    render(<QuarterRing size="2rem" duration="1.5s" strokeWidth="4px" />);
    const spinner = screen.getByRole('status');
    expect(spinner.style.width).toBe('2rem');
    expect(spinner.style.height).toBe('2rem');
    expect(spinner.style.borderWidth).toBe('4px');
    expect(spinner.style.animation).toBe(
      'loading-ui-quarter-ring-rotation 1.5s linear infinite',
    );
  });

  it('支持与 Ant Design Spin 配合使用', () => {
    render(
      <Spin indicator={<QuarterRing data-testid="custom-indicator" />} spinning>
        <div>业务内容</div>
      </Spin>,
    );
    expect(screen.getByTestId('custom-indicator')).toBeInTheDocument();
    expect(screen.getByText('业务内容')).toBeInTheDocument();
  });
});
