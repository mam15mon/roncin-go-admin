import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { RouteErrorPage } from './RouteErrorPage';

describe('RouteErrorPage', () => {
  it('页面模块加载失败时提供完整页面重载入口', () => {
    render(<RouteErrorPage />);

    expect(screen.getByText('页面暂时无法打开')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '重新加载' })).toHaveAttribute(
      'href',
      window.location.href,
    );
  });
});
