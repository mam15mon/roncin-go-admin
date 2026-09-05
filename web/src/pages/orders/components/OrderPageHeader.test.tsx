import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderPageHeader } from './OrderPageHeader';

const mockPush = vi.fn();

vi.mock('@umijs/max', () => ({
  history: {
    push: (path: string) => mockPush(path),
  },
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

describe('OrderPageHeader', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('create 新建订单模式：正确渲染层级、标题与返回列表按钮', () => {
    render(
      <OrderPageHeader
        page="create"
        orderKind="sea-export"
        subTitle="填写业务委托与配舱信息"
      />,
    );

    // 面包屑上级链接
    const rootLink = screen.getByText('订单管理');
    expect(rootLink).toHaveAttribute('href', '/orders');

    const menuLink = screen.getByText('海运出口');
    expect(menuLink).toHaveAttribute('href', '/orders/sea-export');

    // 当前页标题与副标题（当前页仅作为 title 渲染，面包屑中不再包含）
    expect(screen.getByText('新建订单')).toBeInTheDocument();
    expect(screen.getByText('填写业务委托与配舱信息')).toBeInTheDocument();

    // 主要返回按钮返回列表
    const backBtn = screen.getByRole('button', { name: /返回列表/ });
    fireEvent.click(backBtn);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export');
  });

  it('detail 订单详情模式：正确渲染真实订单号与返回列表按钮', () => {
    render(
      <OrderPageHeader
        page="detail"
        orderKind="sea-export"
        orderId="ord-123"
        orderNo="SE2026082600004"
      />,
    );

    expect(screen.getByText('订单管理')).toHaveAttribute('href', '/orders');
    expect(screen.getByText('海运出口')).toHaveAttribute(
      'href',
      '/orders/sea-export',
    );
    expect(screen.getByText('SE2026082600004')).toBeInTheDocument();

    const backBtn = screen.getByRole('button', { name: /返回列表/ });
    fireEvent.click(backBtn);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export');
  });

  it('fees 费用录入模式：正确包含订单号上级链接与返回详情按钮', () => {
    render(
      <OrderPageHeader
        page="fees"
        orderKind="sea-export"
        orderId="ord-123"
        orderNo="SE2026082600004"
      />,
    );

    expect(screen.getByText('订单管理')).toHaveAttribute('href', '/orders');
    expect(screen.getByText('海运出口')).toHaveAttribute(
      'href',
      '/orders/sea-export',
    );

    // 订单号为上级链接
    const orderLink = screen.getByText('SE2026082600004');
    expect(orderLink).toHaveAttribute('href', '/orders/sea-export/ord-123');

    // 当前页标题为费用录入
    expect(screen.getByText('费用录入')).toBeInTheDocument();

    // 主要返回按钮返回订单详情
    const backBtn = screen.getByRole('button', { name: /返回订单详情/ });
    fireEvent.click(backBtn);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/ord-123');
  });

  it('split 拆票工作台模式：正确包含订单号上级链接与返回详情按钮', () => {
    render(
      <OrderPageHeader
        page="split"
        orderKind="sea-export"
        orderId="ord-123"
        orderNo="SE2026082600004"
      />,
    );

    expect(screen.getByText('订单管理')).toHaveAttribute('href', '/orders');
    expect(screen.getByText('海运出口')).toHaveAttribute(
      'href',
      '/orders/sea-export',
    );

    const orderLink = screen.getByText('SE2026082600004');
    expect(orderLink).toHaveAttribute('href', '/orders/sea-export/ord-123');

    expect(screen.getByText('拆票')).toBeInTheDocument();

    const backBtn = screen.getByRole('button', { name: /返回订单详情/ });
    fireEvent.click(backBtn);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/ord-123');
  });

  it('当 orderNo 尚未加载时，降级使用 orderId 展示并保留正确导航链接', () => {
    render(
      <OrderPageHeader
        page="fees"
        orderKind="sea-export"
        orderId="ord-pending-1"
        orderNo={undefined}
      />,
    );

    // 面包屑中使用 orderId 作为过渡标识
    const orderLink = screen.getByText('ord-pending-1');
    expect(orderLink).toHaveAttribute(
      'href',
      '/orders/sea-export/ord-pending-1',
    );
  });

  it('支持在页头中渲染 tags 与 extra 动作插槽', () => {
    render(
      <OrderPageHeader
        page="detail"
        orderKind="sea-export"
        orderId="ord-123"
        orderNo="SE2026082600004"
        tags={<span data-testid="test-tag">已锁定</span>}
        extra={
          <button type="button" data-testid="test-action">
            操作
          </button>
        }
      />,
    );

    expect(screen.getByTestId('test-tag')).toBeInTheDocument();
    expect(screen.getByTestId('test-action')).toBeInTheDocument();
  });
});
