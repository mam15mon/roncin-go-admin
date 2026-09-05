import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { resolveRouteTitle, resolveTabKey } from './routeUtils';
import {
  FIXED_TAB,
  type TagItem,
  TagsView,
  computeNextActivePath,
  shouldIgnorePath,
} from './TagsView';

const mockPush = vi.fn();
let mockPathname = '/welcome';
let mockSearch = '';
let mockHash = '';

vi.mock('@umijs/max', () => ({
  history: {
    push: (path: string) => mockPush(path),
  },
  useLocation: () => ({
    pathname: mockPathname,
    search: mockSearch,
    hash: mockHash,
  }),
}));

describe('routeUtils', () => {
  describe('resolveRouteTitle', () => {
    it('正确解析已知路由与前缀路由标题', () => {
      expect(resolveRouteTitle('/welcome')).toBe('工作台');
      expect(resolveRouteTitle('/orders')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/detail')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/sea-export')).toBe('海运出口订单列表');
      expect(resolveRouteTitle('/orders/sea-export/new')).toBe('新增海运出口');
      expect(resolveRouteTitle('/orders/sea-export/SE2026082600004')).toBe('海运出口详情');
      expect(resolveRouteTitle('/orders/sea-export/SE2026082600004/fees')).toBe('海运出口费用录入');
      expect(resolveRouteTitle('/orders/sea-export/SE2026082600004/split')).toBe('海运出口拆票');
      expect(resolveRouteTitle('/orders/sea-import')).toBe('海运进口订单列表');
      expect(resolveRouteTitle('/orders/sea-import/new')).toBe('新增海运进口');
      expect(resolveRouteTitle('/orders/air-export')).toBe('空运出口订单列表');
      expect(resolveRouteTitle('/orders/air-export/new')).toBe('新增空运出口');
      expect(resolveRouteTitle('/orders/air-export/AE2026082600001')).toBe('空运出口详情');
      expect(resolveRouteTitle('/orders/air-export/AE2026082600001/fees')).toBe('空运出口费用录入');
      expect(resolveRouteTitle('/orders/air-import')).toBe('空运进口订单列表');
      expect(resolveRouteTitle('/orders/air-import/new')).toBe('新增空运进口');
      expect(resolveRouteTitle('/partners/customers')).toBe('客户');
      expect(resolveRouteTitle('/partners/customers/create')).toBe('新建客户');
      expect(resolveRouteTitle('/partners/customers/123')).toBe('客户详情');
      expect(resolveRouteTitle('/partners/suppliers')).toBe('供应商');
      expect(resolveRouteTitle('/partners/suppliers/create')).toBe('新建供应商');
      expect(resolveRouteTitle('/partners/suppliers/456')).toBe('供应商详情');
      expect(resolveRouteTitle('/partners/foreign-agents')).toBe('国外代理');
      expect(resolveRouteTitle('/partners/foreign-agents/create')).toBe('新建国外代理');
      expect(resolveRouteTitle('/partners/foreign-agents/789')).toBe('国外代理详情');
      expect(resolveRouteTitle('/finance/fees')).toBe('集运费用明细');
      expect(resolveRouteTitle('/finance/fees/detail/FEE123')).toBe('费用详情');
      expect(resolveRouteTitle('/master-data')).toBe('主数据');
      expect(resolveRouteTitle('/admin')).toBe('系统管理');
      expect(resolveRouteTitle('/')).toBe('工作台');
      expect(resolveRouteTitle('/unknown-page')).toBe('unknown-page');
    });
  });

  describe('resolveTabKey', () => {
    it('海运出口所有子路由归组为 /orders/sea-export', () => {
      expect(resolveTabKey('/orders/sea-export')).toBe('/orders/sea-export');
      expect(resolveTabKey('/orders/sea-export/new')).toBe('/orders/sea-export');
      expect(resolveTabKey('/orders/sea-export/SE2026082600004')).toBe('/orders/sea-export');
      expect(resolveTabKey('/orders/sea-export/SE2026082600004/fees')).toBe('/orders/sea-export');
      expect(resolveTabKey('/orders/sea-export/SE2026082600004/split')).toBe('/orders/sea-export');
    });

    it('客商三类入口各自归组且互不合并', () => {
      expect(resolveTabKey('/partners/customers')).toBe('/partners/customers');
      expect(resolveTabKey('/partners/customers/create')).toBe('/partners/customers');
      expect(resolveTabKey('/partners/customers/123')).toBe('/partners/customers');

      expect(resolveTabKey('/partners/suppliers')).toBe('/partners/suppliers');
      expect(resolveTabKey('/partners/suppliers/create')).toBe('/partners/suppliers');
      expect(resolveTabKey('/partners/suppliers/456')).toBe('/partners/suppliers');

      expect(resolveTabKey('/partners/foreign-agents')).toBe('/partners/foreign-agents');
      expect(resolveTabKey('/partners/foreign-agents/create')).toBe('/partners/foreign-agents');
      expect(resolveTabKey('/partners/foreign-agents/789')).toBe('/partners/foreign-agents');
    });

    it('集运费用明细与单票详情归组为 /finance/fees', () => {
      expect(resolveTabKey('/finance/fees')).toBe('/finance/fees');
      expect(resolveTabKey('/finance/fees/detail/FEE123')).toBe('/finance/fees');
      // 费用其他设置项不归入费用明细
      expect(resolveTabKey('/finance/fee-settings')).toBe('/finance/fee-settings');
      expect(resolveTabKey('/finance/bills')).toBe('/finance/bills');
    });

    it('工作台与根路径归组为 /welcome', () => {
      expect(resolveTabKey('/welcome')).toBe('/welcome');
      expect(resolveTabKey('/')).toBe('/welcome');
      expect(resolveTabKey('')).toBe('/welcome');
    });

    it('未命中特殊归组规则的页面回退到自身规范化路径', () => {
      expect(resolveTabKey('/settings')).toBe('/settings');
      expect(resolveTabKey('/master-data')).toBe('/master-data');
      expect(resolveTabKey('/admin')).toBe('/admin');
      expect(resolveTabKey('/unknown-page/')).toBe('/unknown-page');
    });
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
      { key: '/orders/sea-export', path: '/orders/sea-export', title: '海运出口', closable: true },
      {
        key: '/partners/customers',
        path: '/partners/customers',
        title: '客户',
        closable: true,
      },
    ];
    // 当前稳定 key 为 /orders/sea-export，关闭 /partners/customers
    const next = computeNextActivePath(tags, '/partners/customers', '/orders/sea-export');
    expect(next).toBeNull();
  });

  it('关闭当前激活页签时激活前一个页签的最新 path（包含查询参数）', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      {
        key: '/orders/sea-export',
        path: '/orders/sea-export/SE001?tab=fees',
        title: '海运出口详情',
        closable: true,
      },
      {
        key: '/partners/customers',
        path: '/partners/customers/123',
        title: '客户详情',
        closable: true,
      },
    ];
    // 当前在客户详情，关闭客户页签后激活海运出口并恢复其最新完整 path
    const next = computeNextActivePath(tags, '/partners/customers', '/partners/customers');
    expect(next).toBe('/orders/sea-export/SE001?tab=fees');
  });

  it('关闭列表中唯一可关闭页签时回退到工作台', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      { key: '/orders/sea-export', path: '/orders/sea-export', title: '海运出口', closable: true },
    ];
    const next = computeNextActivePath(tags, '/orders/sea-export', '/orders/sea-export');
    expect(next).toBe('/welcome');
  });
});

describe('TagsView Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPathname = '/welcome';
    mockSearch = '';
    mockHash = '';
  });

  afterEach(() => {
    cleanup();
  });

  it('工作台页签固定展示且不可关闭', () => {
    render(<TagsView />);
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.queryByLabelText('关闭 工作台')).not.toBeInTheDocument();
  });

  it('直接打开海运出口新建订单时，只建立工作台和海运出口 1 个业务页签', () => {
    mockPathname = '/orders/sea-export/new';
    render(<TagsView />);

    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText('新增海运出口')).toBeInTheDocument();
    // 只有 2 个 tab
    const tabs = screen.getAllByRole('tab');
    expect(tabs.length).toBe(2);
  });

  it('同一菜单内连续跳转（列表 → 新建 → 详情 → 费用 → 列表）不增加重复页签', () => {
    mockPathname = '/orders/sea-export';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('海运出口订单列表')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(2);

    // 跳转至新建订单
    mockPathname = '/orders/sea-export/new';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(screen.getByText('新增海运出口')).toBeInTheDocument();
    expect(screen.queryByText('海运出口订单列表')).not.toBeInTheDocument();

    // 跳转至订单详情
    mockPathname = '/orders/sea-export/SE2026082600004';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(screen.getByText('海运出口详情')).toBeInTheDocument();

    // 跳转至费用录入
    mockPathname = '/orders/sea-export/SE2026082600004/fees';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(screen.getByText('海运出口费用录入')).toBeInTheDocument();

    // 返回列表
    mockPathname = '/orders/sea-export';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(screen.getByText('海运出口订单列表')).toBeInTheDocument();
  });

  it('连续打开不同订单详情时，页签保持单个并指向最后打开的订单', () => {
    mockPathname = '/orders/sea-export/SE001';
    const { rerender } = render(<TagsView />);

    expect(screen.getAllByRole('tab').length).toBe(2);

    mockPathname = '/orders/sea-export/SE002';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);

    // 切到工作台后点击海运出口页签，应恢复到最后打开的 SE002
    mockPathname = '/welcome';
    rerender(<TagsView />);

    const orderTab = screen.getByText('海运出口详情');
    fireEvent.click(orderTab);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/SE002');
  });

  it('客户、供应商、国外代理各自复用且互不合并', () => {
    mockPathname = '/partners/customers/create';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/suppliers/create';
    rerender(<TagsView />);

    mockPathname = '/partners/foreign-agents/create';
    rerender(<TagsView />);

    // 应该有：工作台、客户、供应商、国外代理共 4 个页签
    const tabs = screen.getAllByRole('tab');
    expect(tabs.length).toBe(4);
    expect(screen.getByText('新建客户')).toBeInTheDocument();
    expect(screen.getByText('新建供应商')).toBeInTheDocument();
    expect(screen.getByText('新建国外代理')).toBeInTheDocument();
  });

  it('集运费用详情复用费用页签，跳转至业务订单时为订单新建独立页签', () => {
    mockPathname = '/finance/fees/detail/FEE001';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('费用详情')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(2);

    // 从费用详情跳转查看业务订单
    mockPathname = '/orders/sea-export/SE001';
    rerender(<TagsView />);

    // 保留费用页签，并建立海运出口页签，共 3 个页签
    expect(screen.getAllByRole('tab').length).toBe(3);
    expect(screen.getByText('费用详情')).toBeInTheDocument();
    expect(screen.getByText('海运出口详情')).toBeInTheDocument();

    // 点击费用页签应能回到费用详情
    fireEvent.click(screen.getByText('费用详情'));
    expect(mockPush).toHaveBeenCalledWith('/finance/fees/detail/FEE001');
  });

  it('保留查询参数并可在切换后恢复', () => {
    mockPathname = '/settings';
    mockSearch = '?tab=security';
    mockHash = '#logs';
    const { rerender } = render(<TagsView />);

    // 切到工作台
    mockPathname = '/welcome';
    mockSearch = '';
    mockHash = '';
    rerender(<TagsView />);

    // 点击参数设置页签，应带回完整 search 与 hash
    const settingsTab = screen.getByText('参数设置');
    fireEvent.click(settingsTab);
    expect(mockPush).toHaveBeenCalledWith('/settings?tab=security#logs');
  });

  it('响应 roncin:update-tab-title 动态更新标题，并拦截迟到事件', () => {
    mockPathname = '/orders/sea-export/SE001';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('海运出口详情')).toBeInTheDocument();

    // 异步加载完成，发送动态标题事件
    act(() => {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: '/orders/sea-export/SE001',
            title: 'SE001 (海运出口)',
          },
        }),
      );
    });

    expect(screen.getByText('SE001 (海运出口)')).toBeInTheDocument();

    // 用户跳转到新建页面
    mockPathname = '/orders/sea-export/new';
    rerender(<TagsView />);
    expect(screen.getByText('新增海运出口')).toBeInTheDocument();

    // 之前 SE001 迟到的标题事件到达，应被门禁拦截，不能覆盖新建页面的标题
    act(() => {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: '/orders/sea-export/SE001',
            title: '迟到的SE001标题',
          },
        }),
      );
    });

    expect(screen.getByText('新增海运出口')).toBeInTheDocument();
    expect(screen.queryByText('迟到的SE001标题')).not.toBeInTheDocument();
  });

  it('点击关闭当前激活页签时移除并激活前序页签', () => {
    mockPathname = '/orders/sea-export/SE001';
    render(<TagsView />);

    const closeBtn = screen.getByLabelText('关闭 海运出口详情');
    fireEvent.click(closeBtn);

    expect(mockPush).toHaveBeenCalledWith('/welcome');
    expect(screen.queryByText('海运出口详情')).not.toBeInTheDocument();
  });
});
