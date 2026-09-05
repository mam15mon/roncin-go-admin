import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PageHeaderShell } from './PageHeaderShell';

const mockPush = vi.fn();

vi.mock('@umijs/max', () => ({
  Link: ({ to, children, onClick, ...rest }: any) => (
    <a
      href={to}
      onClick={(e) => {
        e.preventDefault();
        onClick?.(e);
        mockPush(to);
      }}
      {...rest}
    >
      {children}
    </a>
  ),
}));

describe('PageHeaderShell', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('支持在面包屑中使用 href 渲染真实链接', () => {
    render(
      <PageHeaderShell
        title="当前功能"
        breadcrumbs={[
          { label: '订单管理', href: '/orders' },
          { label: '海运出口', href: '/orders/sea-export' },
        ]}
      />,
    );

    const link1 = screen.getByText('订单管理');
    expect(link1.tagName).toBe('A');
    expect(link1).toHaveAttribute('href', '/orders');

    const link2 = screen.getByText('海运出口');
    expect(link2.tagName).toBe('A');
    expect(link2).toHaveAttribute('href', '/orders/sea-export');

    fireEvent.click(link1);
    expect(mockPush).toHaveBeenCalledWith('/orders');
  });

  it('兼容仅使用 onClick 的纯按钮面包屑', () => {
    const handleCustomClick = vi.fn();
    render(
      <PageHeaderShell
        title="客户详情"
        breadcrumbs={[
          { label: '客户管理', onClick: handleCustomClick },
        ]}
      />,
    );

    const btn = screen.getByRole('button', { name: '客户管理' });
    fireEvent.click(btn);
    expect(handleCustomClick).toHaveBeenCalledTimes(1);
  });

  it('无 href/onClick 时渲染不可点击的文本面包屑', () => {
    render(
      <PageHeaderShell
        title="只读层级"
        breadcrumbs={[{ label: '静态分类' }]}
      />,
    );

    const staticItem = screen.getByText('静态分类');
    expect(staticItem.tagName).not.toBe('A');
    expect(staticItem.tagName).not.toBe('BUTTON');
  });

  it('正确渲染主要返回按钮与点击回调', () => {
    const onBack = vi.fn();
    render(
      <PageHeaderShell
        title="订单详情"
        onBack={onBack}
        backText="返回列表"
      />,
    );

    const backBtn = screen.getByRole('button', { name: /返回列表/ });
    fireEvent.click(backBtn);
    expect(onBack).toHaveBeenCalledTimes(1);
  });
});
