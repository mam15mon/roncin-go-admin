import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import type { Modal } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { ModalStaticFunctions } from 'antd/es/modal/confirm';
import type { NotificationInstance } from 'antd/es/notification/interface';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { setAppFeedback } from '@/utils/appFeedback';
import { getFormDraftKey, getFormDraftScope, hasTabDraft } from './formDraft';
import { HeaderTitle } from './HeaderTitle';
import { resolveRouteTitle, resolveTabKey } from './routeUtils';
import {
  computeNextActivePath,
  FIXED_TAB,
  shouldIgnorePath,
  type TagItem,
  TagsView,
} from './TagsView';
import {
  _clearAllTabCloseGuards,
  registerTabCloseGuard,
} from './tabCloseGuard';

const mockPush = vi.fn();
let mockPathname = '/welcome';
let mockSearch = '';
let mockHash = '';

/**
 * 确认关闭行为只需要读取 confirm 参数并主动执行 onOk。
 * 经 AppFeedbackBridge 注入实例（与生产运行时同链路），不再依赖
 * antd 静态 Modal.confirm，避免 portal 在 happy-dom 清理后继续调度 React 更新。
 */
function spyOnConfirm() {
  const confirmMock = vi.fn(
    (_props: Parameters<typeof Modal.confirm>[0]) =>
      ({
        destroy: vi.fn(),
        update: vi.fn(),
      }) as ReturnType<typeof Modal.confirm>,
  );
  setAppFeedback({
    message: { error: vi.fn() } as unknown as MessageInstance,
    notification: {} as unknown as NotificationInstance,
    modal: { confirm: confirmMock } as unknown as Omit<
      ModalStaticFunctions,
      'warn'
    >,
  });
  return confirmMock;
}

vi.mock('@/router/history', () => ({
  history: {
    push: (path: string) => mockPush(path),
  },
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1' },
      },
    },
  }),
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useLocation: () => ({
      pathname: mockPathname,
      search: mockSearch,
      hash: mockHash,
    }),
  };
});

describe('routeUtils', () => {
  describe('resolveRouteTitle', () => {
    it('正确解析已知路由与前缀路由标题', () => {
      expect(resolveRouteTitle('/welcome')).toBe('工作台');
      expect(resolveRouteTitle('/orders')).toBe('海运出口订单列表');
      expect(resolveRouteTitle('/partners')).toBe('客户');
      expect(resolveRouteTitle('/orders/detail')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/sea-export')).toBe('海运出口订单列表');
      expect(resolveRouteTitle('/orders/sea-export/new')).toBe('SE 新建订单');
      // 订单动态路由统一使用中性占位标题，具体订单页加载后回填真实标题
      expect(resolveRouteTitle('/orders/sea-export/SE2026082600004')).toBe(
        '订单详情',
      );
      expect(resolveRouteTitle('/orders/sea-export/SE2026082600004/fees')).toBe(
        '订单费用录入',
      );
      expect(
        resolveRouteTitle('/orders/sea-export/SE2026082600004/split'),
      ).toBe('订单拆票');
      // 未注册订单类型不再伪装成已知业务，回落到订单管理中性标题
      expect(resolveRouteTitle('/orders/sea-import')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/sea-import/new')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/air-export')).toBe('空运出口订单列表');
      expect(resolveRouteTitle('/orders/air-export/new')).toBe('AE 新建订单');
      expect(resolveRouteTitle('/orders/air-export/AE2026082600001')).toBe(
        '订单详情',
      );
      expect(resolveRouteTitle('/orders/air-export/AE2026082600001/fees')).toBe(
        '订单费用录入',
      );
      expect(resolveRouteTitle('/orders/air-import')).toBe('订单管理');
      expect(resolveRouteTitle('/orders/air-import/new')).toBe('订单管理');
      expect(resolveRouteTitle('/partners/customers')).toBe('客户');
      expect(resolveRouteTitle('/partners/customers/create')).toBe('新建客户');
      expect(resolveRouteTitle('/partners/customers/123')).toBe('客户详情');
      expect(resolveRouteTitle('/partners/suppliers')).toBe('供应商');
      expect(resolveRouteTitle('/partners/suppliers/create')).toBe(
        '新建供应商',
      );
      expect(resolveRouteTitle('/partners/suppliers/456')).toBe('供应商详情');
      expect(resolveRouteTitle('/partners/foreign-agents')).toBe('国外代理');
      expect(resolveRouteTitle('/partners/foreign-agents/create')).toBe(
        '新建国外代理',
      );
      expect(resolveRouteTitle('/partners/foreign-agents/789')).toBe(
        '国外代理详情',
      );
      expect(resolveRouteTitle('/finance/fees')).toBe('集运费用明细');
      expect(resolveRouteTitle('/finance/fees/detail/FEE123')).toBe('费用详情');
      expect(resolveRouteTitle('/master-data')).toBe('主数据');
      expect(resolveRouteTitle('/admin')).toBe('系统管理');
      expect(resolveRouteTitle('/')).toBe('工作台');
      expect(resolveRouteTitle('/unknown-page')).toBe('unknown-page');
    });
  });

  describe('resolveTabKey', () => {
    it('纯重定向入口正确解析为最终业务菜单页签', () => {
      expect(resolveTabKey('/orders')).toBe('/orders/sea-export');
      expect(resolveTabKey('/partners')).toBe('/partners/customers');
      expect(resolveTabKey('/settings')).toBe('/finance/fee-settings');
    });

    it('已知前缀下的未知子路径不进行合并，保持自身规范化路径', () => {
      expect(resolveTabKey('/orders/sea-export/SE2026082600004/unknown')).toBe(
        '/orders/sea-export/SE2026082600004/unknown',
      );
      expect(resolveTabKey('/orders/sea-export/foo/bar')).toBe(
        '/orders/sea-export/foo/bar',
      );
      expect(resolveTabKey('/partners/customers/123/logs')).toBe(
        '/partners/customers/123/logs',
      );
      expect(resolveTabKey('/partners/suppliers/456/audit')).toBe(
        '/partners/suppliers/456/audit',
      );
      expect(resolveTabKey('/finance/fees/export')).toBe(
        '/finance/fees/export',
      );
    });
    it('海运出口列表、新建和各订单页面使用独立页签', () => {
      for (const path of [
        '/orders/sea-export',
        '/orders/sea-export/new',
        '/orders/sea-export/SE001',
        '/orders/sea-export/SE001/fees',
        '/orders/sea-export/SE001/split',
        '/orders/sea-export/SE001/hbl',
      ]) {
        expect(resolveTabKey(path)).toBe(path);
      }
    });

    it('客商三类入口各自归组且互不合并', () => {
      expect(resolveTabKey('/partners/customers')).toBe('/partners/customers');
      expect(resolveTabKey('/partners/customers/create')).toBe(
        '/partners/customers',
      );
      expect(resolveTabKey('/partners/customers/123')).toBe(
        '/partners/customers',
      );

      expect(resolveTabKey('/partners/suppliers')).toBe('/partners/suppliers');
      expect(resolveTabKey('/partners/suppliers/create')).toBe(
        '/partners/suppliers',
      );
      expect(resolveTabKey('/partners/suppliers/456')).toBe(
        '/partners/suppliers',
      );

      expect(resolveTabKey('/partners/foreign-agents')).toBe(
        '/partners/foreign-agents',
      );
      expect(resolveTabKey('/partners/foreign-agents/create')).toBe(
        '/partners/foreign-agents',
      );
      expect(resolveTabKey('/partners/foreign-agents/789')).toBe(
        '/partners/foreign-agents',
      );
    });

    it('集运费用明细与单票详情归组为 /finance/fees', () => {
      expect(resolveTabKey('/finance/fees')).toBe('/finance/fees');
      expect(resolveTabKey('/finance/fees/detail/FEE123')).toBe(
        '/finance/fees',
      );
      // 费用其他设置项不归入费用明细
      expect(resolveTabKey('/finance/fee-settings')).toBe(
        '/finance/fee-settings',
      );
      expect(resolveTabKey('/finance/bills')).toBe('/finance/bills');
    });

    it('工作台与根路径归组为 /welcome', () => {
      expect(resolveTabKey('/welcome')).toBe('/welcome');
      expect(resolveTabKey('/')).toBe('/welcome');
      expect(resolveTabKey('')).toBe('/welcome');
    });

    it('未命中特殊归组规则的页面回退到自身规范化路径', () => {
      expect(resolveTabKey('/settings')).toBe('/finance/fee-settings');
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
    expect(shouldIgnorePath('/user/login/dingtalk/callback')).toBe(true);
    expect(shouldIgnorePath('/enterprise-resources/config')).toBe(false);
    expect(shouldIgnorePath('/orders')).toBe(false);
    expect(shouldIgnorePath('/welcome')).toBe(false);
  });

  it('关闭非当前激活页签时不触发跳转', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      {
        key: '/orders/sea-export',
        path: '/orders/sea-export',
        title: '海运出口',
        closable: true,
      },
      {
        key: '/partners/customers',
        path: '/partners/customers',
        title: '客户',
        closable: true,
      },
    ];
    // 当前稳定 key 为 /orders/sea-export，关闭 /partners/customers
    const next = computeNextActivePath(
      tags,
      '/partners/customers',
      '/orders/sea-export',
    );
    expect(next).toBeNull();
  });

  it('关闭当前激活页签时激活前一个页签的最新 path（包含查询参数）', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      {
        key: '/orders/sea-export',
        path: '/orders/sea-export/SE001?tab=fees',
        title: '订单详情',
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
    const next = computeNextActivePath(
      tags,
      '/partners/customers',
      '/partners/customers',
    );
    expect(next).toBe('/orders/sea-export/SE001?tab=fees');
  });

  it('关闭列表中唯一可关闭页签时回退到工作台', () => {
    const tags: TagItem[] = [
      FIXED_TAB,
      {
        key: '/orders/sea-export',
        path: '/orders/sea-export',
        title: '海运出口',
        closable: true,
      },
    ];
    const next = computeNextActivePath(
      tags,
      '/orders/sea-export',
      '/orders/sea-export',
    );
    expect(next).toBe('/welcome');
  });
});

describe('TagsView Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    _clearAllTabCloseGuards();
    sessionStorage.clear();
    mockPathname = '/welcome';
    mockSearch = '';
    mockHash = '';
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    sessionStorage.clear();
    cleanup();
  });

  it('工作台页签固定展示且不可关闭', () => {
    render(<TagsView />);
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.queryByLabelText('关闭 工作台')).not.toBeInTheDocument();
  });

  it.each([
    ['/finance/nettings', '对冲管理'],
    ['/enterprise-resources/config', '配置管理'],
    ['/orders/air-export/masters/MAWB-1', '航空共享主单'],
    ['/commission-adjustments/ADJ-1/my-supplement-source', '我的冲减来源'],
  ])('直接进入 %s 时页签与顶栏均显示 %s', (path, title) => {
    mockPathname = path;
    render(
      <>
        <HeaderTitle />
        <TagsView />
      </>,
    );
    expect(screen.getAllByText(title)).toHaveLength(2);
    expect(
      screen.queryByText(path.split('/').at(-1) ?? ''),
    ).not.toBeInTheDocument();
    expect(resolveRouteTitle(`${path}/`)).toBe(title);
  });

  it.each([
    ['/orders/sea-export/new', 'SE 新建订单'],
    ['/orders/air-export/new', 'AE 新建订单'],
  ])('直接打开 %s 时展示 %s，且只建立一个业务页签', (path, title) => {
    mockPathname = path;
    render(<TagsView />);

    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText(title)).toBeInTheDocument();
    // 只有 2 个 tab
    const tabs = screen.getAllByRole('tab');
    expect(tabs.length).toBe(2);
  });

  it('列表、新建和详情各自保留页签，返回列表不新增页签', () => {
    mockPathname = '/orders/sea-export';
    const { rerender } = render(<TagsView />);
    mockPathname = '/orders/sea-export/new';
    rerender(<TagsView />);
    mockPathname = '/orders/sea-export/SE001';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab')).toHaveLength(4);
    expect(screen.getByText('海运出口订单列表')).toBeInTheDocument();
    expect(screen.getByText('SE 新建订单')).toBeInTheDocument();
    expect(screen.getByText('订单详情')).toBeInTheDocument();
    mockPathname = '/orders/sea-export';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab')).toHaveLength(4);
    fireEvent.click(screen.getByText('SE 新建订单'));
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/new');
  });

  it('不同订单详情各自保留页签和路径', () => {
    mockPathname = '/orders/sea-export/SE001';
    const { rerender } = render(<TagsView />);
    mockPathname = '/orders/sea-export/SE002';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab')).toHaveLength(3);
    fireEvent.click(screen.getAllByText('订单详情')[0]);
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/SE001');
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
    expect(screen.getByText('订单详情')).toBeInTheDocument();

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

    // 点击配置声明的费用设置页签，应带回完整 search 与 hash
    const settingsTab = screen.getByText('费用设置');
    fireEvent.click(settingsTab);
    expect(mockPush).toHaveBeenCalledWith('/settings?tab=security#logs');
  });

  it('动态标题更新只影响对应订单页签', () => {
    mockPathname = '/orders/sea-export/SE001';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('订单详情')).toBeInTheDocument();

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
    expect(screen.getByText('SE 新建订单')).toBeInTheDocument();

    // SE001 的标题事件更新原页签，不影响新建页面的标题
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

    expect(screen.getByText('SE 新建订单')).toBeInTheDocument();
    expect(screen.getByText('迟到的SE001标题')).toBeInTheDocument();
  });

  it('点击关闭当前激活页签时移除并激活前序页签', () => {
    mockPathname = '/orders/sea-export/SE001';
    render(<TagsView />);

    const closeBtn = screen.getByLabelText('关闭 订单详情');
    fireEvent.click(closeBtn);

    expect(mockPush).toHaveBeenCalledWith('/welcome');
    expect(screen.queryByText('订单详情')).not.toBeInTheDocument();
  });

  it('关闭按钮Enter和Space不冒泡切换页签，原生激活仍走dirty确认', () => {
    mockPathname = '/partners/customers';
    render(<TagsView />);
    registerTabCloseGuard('/partners/customers', { isDirty: () => true });
    const confirm = spyOnConfirm();
    const close = screen.getByLabelText('关闭 客户');
    fireEvent.keyDown(close, { key: 'Enter' });
    fireEvent.keyDown(close, { key: ' ' });
    expect(mockPush).not.toHaveBeenCalled();
    fireEvent.click(close);
    expect(confirm).toHaveBeenCalledOnce();
    expect(mockPush).not.toHaveBeenCalled();
    expect(screen.getByText('客户')).toBeInTheDocument();
  });

  it('直接访问重定向入口 /orders 时只产生一个业务页签', () => {
    mockPathname = '/orders';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText('海运出口订单列表')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(2);

    // 随后重定向至 /orders/sea-export，不产生重复页签
    mockPathname = '/orders/sea-export';
    rerender(<TagsView />);
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(screen.getByText('海运出口订单列表')).toBeInTheDocument();
  });

  it('不同订单详情之间快速切换时，防止旧订单迟到响应导致标题串号', () => {
    mockPathname = '/orders/sea-export/SE001';
    const { rerender } = render(<TagsView />);

    expect(screen.getByText('订单详情')).toBeInTheDocument();

    // 快速切换至订单 SE002
    mockPathname = '/orders/sea-export/SE002';
    rerender(<TagsView />);

    // SE001 的标题事件只更新其自身页签，不覆盖 SE002 的标题
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
    expect(screen.getByText('订单详情')).toBeInTheDocument();

    // SE002 的标题更新事件正常到达，页签标题更新为 SE002
    act(() => {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: '/orders/sea-export/SE002',
            title: 'SE002 (海运出口)',
          },
        }),
      );
    });

    expect(screen.getByText('SE002 (海运出口)')).toBeInTheDocument();
  });

  it('通过 Enter 和 Space 键盘按键可以切换激活页签', () => {
    mockPathname = '/orders/sea-export';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/customers';
    rerender(<TagsView />);

    const tabs = screen.getAllByRole('tab');
    // tabs: [0: 工作台 (/welcome), 1: 海运出口 (/orders/sea-export), 2: 客户 (/partners/customers)]
    const seaExportTab = tabs[1];

    // 在非当前页签上按 Enter
    fireEvent.keyDown(seaExportTab, { key: 'Enter' });
    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export');

    mockPush.mockClear();

    // 在工作台页签上按 Space
    const welcomeTab = tabs[0];
    fireEvent.keyDown(welcomeTab, { key: ' ' });
    expect(mockPush).toHaveBeenCalledWith('/welcome');
  });

  function getTabByText(text: string): HTMLElement {
    const el = screen.getByText(text).closest('[role="tab"]');
    if (!el || !(el instanceof HTMLElement)) {
      throw new Error(`Tab element with text "${text}" not found`);
    }
    return el;
  }

  /**
   * 右键菜单弹层的 CSSMotion 步进队列由 rAF 逐帧推进，而 happy-dom 不会派发
   * transitionend，离开动画永远不会自然结束，剩余步进会在用例结束后落地并触发
   * act 警告。这里等待弹层进入离开动画步进后，在 act 内派发 transitionend
   * 结束动画（与 rc-motion 官方测试一致），再排空一帧让队列到达终态。
   */
  async function settleDropdownPopup() {
    const popup = document.querySelector('.ant-dropdown') as HTMLElement | null;
    if (!popup) {
      return;
    }
    await waitFor(() => {
      expect(popup).toHaveClass('ant-slide-up-leave-active');
    });
    act(() => {
      fireEvent.transitionEnd(popup);
    });
    await act(async () => {
      await new Promise((resolve) => {
        requestAnimationFrame(() => resolve(null));
      });
    });
    expect(popup).toHaveClass('ant-dropdown-hidden');
  }

  it('右键菜单支持重新加载当前与非当前页签', async () => {
    const originalLocation = window.location;
    const mockReload = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...originalLocation, reload: mockReload },
    });

    mockPathname = '/orders/sea-export';
    const { rerender } = render(<TagsView />);

    // 1. 在当前激活页签（海运出口）右键点击“重新加载”，应调用 window.location.reload
    const seaExportTab = getTabByText('海运出口订单列表');
    act(() => {
      fireEvent.contextMenu(seaExportTab);
    });
    const reloadItem = screen.getByText('重新加载');
    act(() => {
      fireEvent.click(reloadItem);
    });
    expect(mockReload).toHaveBeenCalledTimes(1);

    // 2. 切到工作台，然后在非激活页签（海运出口）右键点击“重新加载”，应调用 history.push('/orders/sea-export')
    mockPathname = '/welcome';
    rerender(<TagsView />);

    const seaExportTabInactive = getTabByText('海运出口订单列表');
    act(() => {
      fireEvent.contextMenu(seaExportTabInactive);
    });
    const reloadInactiveItem = screen.getAllByText('重新加载')[0];
    act(() => {
      fireEvent.click(reloadInactiveItem);
    });
    await settleDropdownPopup();

    expect(mockPush).toHaveBeenCalledWith('/orders/sea-export');

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    });
  });

  it('右键菜单：支持关闭当前标签页', () => {
    mockPathname = '/partners/customers';
    render(<TagsView />);

    expect(screen.getByText('客户')).toBeInTheDocument();

    const customerTab = getTabByText('客户');
    act(() => {
      fireEvent.contextMenu(customerTab);
    });
    const closeCurrentItem = screen.getByText('关闭标签页');
    act(() => {
      fireEvent.click(closeCurrentItem);
    });

    expect(screen.queryByText('客户')).not.toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(1);
    expect(mockPush).toHaveBeenCalledWith('/welcome');
  });

  it('右键菜单：支持关闭其他标签页', async () => {
    mockPathname = '/partners/customers';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/suppliers';
    rerender(<TagsView />);

    expect(screen.getAllByRole('tab').length).toBe(3);

    const customerTab = getTabByText('客户');
    act(() => {
      fireEvent.contextMenu(customerTab);
    });
    const closeOtherItem = screen.getByText('关闭其他标签页');
    act(() => {
      fireEvent.click(closeOtherItem);
    });
    await settleDropdownPopup();

    expect(screen.queryByText('供应商')).not.toBeInTheDocument();
    expect(screen.getByText('客户')).toBeInTheDocument();
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(2);
    expect(mockPush).toHaveBeenCalledWith('/partners/customers');
  });

  it('右键菜单：支持关闭右侧标签页', async () => {
    mockPathname = '/partners/customers';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/suppliers';
    rerender(<TagsView />);

    mockPathname = '/partners/foreign-agents';
    rerender(<TagsView />);

    expect(screen.getAllByRole('tab').length).toBe(4);

    // 当前在国外代理 (index 3)，在供应商 (index 2) 上右键关闭右侧标签页
    const supplierTab = getTabByText('供应商');
    act(() => {
      fireEvent.contextMenu(supplierTab);
    });
    const closeRightItem = screen.getByText('关闭右侧标签页');
    act(() => {
      fireEvent.click(closeRightItem);
    });
    await settleDropdownPopup();

    expect(screen.queryByText('国外代理')).not.toBeInTheDocument();
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText('客户')).toBeInTheDocument();
    expect(screen.getByText('供应商')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(3);
    expect(mockPush).toHaveBeenCalledWith('/partners/suppliers');
  });

  it('右键菜单：支持关闭左侧标签页', async () => {
    mockPathname = '/partners/customers';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/suppliers';
    rerender(<TagsView />);

    mockPathname = '/partners/foreign-agents';
    rerender(<TagsView />);

    // 当前激活在客户 (index 1)
    mockPathname = '/partners/customers';
    rerender(<TagsView />);

    expect(screen.getAllByRole('tab').length).toBe(4);

    // 在供应商 (index 2) 上右键关闭左侧标签页（关闭客户）
    const supplierTab = getTabByText('供应商');
    act(() => {
      fireEvent.contextMenu(supplierTab);
    });
    const closeLeftItem = screen.getByText('关闭左侧标签页');
    act(() => {
      fireEvent.click(closeLeftItem);
    });
    await settleDropdownPopup();

    expect(screen.queryByText('客户')).not.toBeInTheDocument();
    expect(screen.getByText('工作台')).toBeInTheDocument();
    expect(screen.getByText('供应商')).toBeInTheDocument();
    expect(screen.getByText('国外代理')).toBeInTheDocument();
    expect(screen.getAllByRole('tab').length).toBe(3);
    expect(mockPush).toHaveBeenCalledWith('/partners/suppliers');
  });

  it('关闭已编辑未保存的订单页签时，弹出确认提示并在取消时保留页签', () => {
    mockPathname = '/orders/sea-export/new';
    render(<TagsView />);

    // 注册海运出口新建订单处于 dirty 状态
    registerTabCloseGuard('/orders/sea-export/new', {
      isDirty: () => true,
    });

    const modalSpy = spyOnConfirm();

    const closeBtn = screen.getByLabelText('关闭 SE 新建订单');
    act(() => {
      fireEvent.click(closeBtn);
    });

    expect(modalSpy).toHaveBeenCalledTimes(1);
    const modalArgs = modalSpy.mock.calls[0][0];
    expect(modalArgs.title).toBe('提示');
    expect(modalArgs.content).toBe('有未保存的修改，是否放弃并关闭？');
    expect(modalArgs.okText).toBe('放弃并关闭');
    expect(modalArgs.cancelText).toBe('继续编辑');

    // 未执行 onOk，页签依然存在，未跳转
    expect(screen.getByText('SE 新建订单')).toBeInTheDocument();
    expect(mockPush).not.toHaveBeenCalled();

    modalSpy.mockRestore();
  });

  it('关闭已编辑未保存的订单页签时，用户确认离开后正常关闭页签并跳转', () => {
    mockPathname = '/orders/sea-export/SE001';
    render(<TagsView />);

    registerTabCloseGuard('/orders/sea-export/SE001', {
      isDirty: () => true,
    });

    const modalSpy = spyOnConfirm();

    const closeBtn = screen.getByLabelText('关闭 订单详情');
    act(() => {
      fireEvent.click(closeBtn);
    });

    expect(modalSpy).toHaveBeenCalledTimes(1);
    const modalArgs = modalSpy.mock.calls[0][0];

    // 模拟用户点击「放弃并关闭」
    act(() => {
      modalArgs.onOk?.();
    });

    expect(mockPush).toHaveBeenCalledWith('/welcome');
    expect(screen.queryByText('订单详情')).not.toBeInTheDocument();

    modalSpy.mockRestore();
  });

  it('右键菜单关闭其他标签页包含已编辑页签时，弹出确认提示', async () => {
    mockPathname = '/orders/sea-export/new';
    const { rerender } = render(<TagsView />);

    mockPathname = '/partners/customers';
    rerender(<TagsView />);

    // 此时标签为：工作台、新建订单、客户；新建订单处于 dirty 状态
    registerTabCloseGuard('/orders/sea-export/new', {
      isDirty: () => true,
    });

    const modalSpy = spyOnConfirm();

    // 在客户标签上右键关闭其他标签页（包括海运出口）
    const customerTab = getTabByText('客户');
    act(() => {
      fireEvent.contextMenu(customerTab);
    });
    const closeOtherItem = screen.getByText('关闭其他标签页');
    act(() => {
      fireEvent.click(closeOtherItem);
    });

    expect(modalSpy).toHaveBeenCalledTimes(1);
    expect(modalSpy.mock.calls[0][0].content).toBe(
      '有未保存的修改，是否放弃并关闭？',
    );

    // 取消时，新建订单页签依然存在
    expect(screen.getByText('SE 新建订单')).toBeInTheDocument();

    // 确认后，其他标签页被关闭
    act(() => {
      modalSpy.mock.calls[0][0].onOk?.();
    });
    // 右键菜单弹层的关闭动画需收敛，避免用例结束后更新迟到
    await settleDropdownPopup();

    expect(screen.queryByText('SE 新建订单')).not.toBeInTheDocument();
    expect(screen.getByText('客户')).toBeInTheDocument();

    modalSpy.mockRestore();
  });

  it('确认关闭已编辑页签后，自动清除该页签在 sessionStorage 中的草稿', () => {
    mockPathname = '/orders/sea-export/new';
    render(<TagsView />);

    const draftScope = getFormDraftScope('user-1', 'org-1');
    const draftKey = getFormDraftKey(
      '/orders/sea-export/new',
      '/orders/sea-export/new',
      draftScope,
    );
    sessionStorage.setItem(draftKey, JSON.stringify({ field: 'value' }));
    expect(hasTabDraft('/orders/sea-export/new', draftScope)).toBe(true);

    const modalSpy = spyOnConfirm();

    const closeBtn = screen.getByLabelText(/^关闭 /);
    act(() => {
      fireEvent.click(closeBtn);
    });

    expect(modalSpy).toHaveBeenCalledTimes(1);

    // 确认离开
    act(() => {
      modalSpy.mock.calls[0][0].onOk?.();
    });

    expect(hasTabDraft('/orders/sea-export/new', draftScope)).toBe(false);
    expect(sessionStorage.getItem(draftKey)).toBeNull();

    modalSpy.mockRestore();
  });
});
