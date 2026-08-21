import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { resolveRouteTitle } from './routeUtils';
import {
  FIXED_TAB,
  type TagItem,
  TagsView,
  computeNextActivePath,
  shouldIgnorePath,
} from './TagsView';

const mockPush = vi.fn();
let mockPathname = '/welcome';

vi.mock('@umijs/max', () => ({
  history: {
    push: (path: string) => mockPush(path),
  },
  useLocation: () => ({
    pathname: mockPathname,
    search: '',
    hash: '',
  }),
}));

describe('routeUtils', () => {
  it('正确解析已知路由与前缀路由标题', () => {
    expect(resolveRouteTitle('/welcome')).toBe('工作台');
    expect(resolveRouteTitle('/orders')).toBe('订单管理');
    expect(resolveRouteTitle('/orders/detail')).toBe('订单管理');
    expect(resolveRouteTitle('/partners/customers')).toBe('客户');
    expect(resolveRouteTitle('/partners/suppliers')).toBe('供应商');
    expect(resolveRouteTitle('/partners/foreign-agents')).toBe('国外代理');
    expect(resolveRouteTitle('/master-data')).toBe('主数据');
    expect(resolveRouteTitle('/admin')).toBe('系统管理');
    expect(resolveRouteTitle('/')).toBe('工作台');
    expect(resolveRouteTitle('/unknown-page')).toBe('unknown-page');
  });
});

describe('TagsView logic', () => {
  it('正确过滤无需加入页签的路径', () => {
    expect(shouldIgnorePath('/user/login')).toBe(true);
    expect(shouldIgnorePath('/user')).toBe(true);
    expect(shouldIgnorePath('/login')).toBe(true);
    expect(shouldIgnorePath('/orders')).toBe(false);
    expect(shouldIgnorePath('/welcome')).toBe(false);
  });

  it('关闭非当前激活页签时不触发跳转', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      { key: '/orders', path: '/orders', title: '订单管理', closable: true },
      {
        key: '/partners/customers',
        path: '/partners/customers',
        title: '客户',
        closable: true,
      },
    ];
    // 当前在 /orders，关闭 /partners/customers
    const next = computeNextActivePath(tags, '/partners/customers', '/orders');
    expect(next).toBeNull();
  });

  it('关闭当前激活页签时激活前一个页签', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      { key: '/orders', path: '/orders', title: '订单管理', closable: true },
      {
        key: '/partners/customers',
        path: '/partners/customers',
        title: '客户',
        closable: true,
      },
    ];
    // 当前在客户页，关闭客户页签后激活订单页签
    const next = computeNextActivePath(tags, '/partners/customers', '/partners/customers');
    expect(next).toBe('/orders');
  });

  it('关闭列表中唯一可关闭页签时回退到工作台', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      { key: '/orders', path: '/orders', title: '订单管理', closable: true },
    ];
    const next = computeNextActivePath(tags, '/orders', '/orders');
    expect(next).toBe('/welcome');
  });
});

describe('TagsView Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPathname = '/welcome';
  });

  afterEach(() => {
    cleanup();
  });

  it('工作台页签固定展示且不可关闭', () => {
    render(<TagsView />);
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.queryByLabelText('关闭 工作台')).not.toBeInTheDocument();
  });

  it('初始处于业务页面时正确展示工作台和业务页签', () => {
    mockPathname = '/orders';
    render(<TagsView />);

    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText('订单管理')).toBeInTheDocument();
    expect(screen.getByLabelText('关闭 订单管理')).toBeInTheDocument();
  });

  it('点击页签时触发 history.push 跳转', () => {
    mockPathname = '/orders';
    render(<TagsView />);

    const workbenchTab = screen.getByText('工作台');
    fireEvent.click(workbenchTab);

    expect(mockPush).toHaveBeenCalledWith('/welcome');
  });

  it('点击关闭按钮时移除页签并跳转到前序页签', () => {
    mockPathname = '/orders';
    render(<TagsView />);

    const closeBtn = screen.getByLabelText('关闭 订单管理');
    fireEvent.click(closeBtn);

    expect(mockPush).toHaveBeenCalledWith('/welcome');
    expect(screen.queryByText('订单管理')).not.toBeInTheDocument();
  });
});
