import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { history } from '@umijs/max';
import { authServiceLogout } from '@/services/roncin/authService';
import { clearOrderMasterDataCache } from '@/utils/order-options-cache';
import { AvatarDropdown } from './AvatarDropdown';

let mockCurrentUser: any = null;
const mockSetInitialState = vi.fn();

vi.mock('@umijs/max', () => ({
  history: {
    replace: vi.fn(),
  },
  useModel: (model: string) => {
    if (model === '@@initialState') {
      return {
        initialState: { currentUser: mockCurrentUser },
        setInitialState: mockSetInitialState,
      };
    }
    return {};
  },
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceLogout: vi.fn().mockResolvedValue({}),
}));

vi.mock('@/utils/order-options-cache', () => ({
  clearOrderMasterDataCache: vi.fn(),
}));

const mockLogout = vi.mocked(authServiceLogout);
const mockClearCache = vi.mocked(clearOrderMasterDataCache);

describe('AvatarDropdown Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = null;
  });

  afterEach(() => {
    cleanup();
  });

  it('未登录时展示 Spin 加载态', () => {
    const { container } = render(<AvatarDropdown />);
    expect(container.querySelector('.ant-spin')).toBeInTheDocument();
  });

  it('已登录时展示用户头像首字母与带 roncin-avatar-name 类的用户名', () => {
    mockCurrentUser = {
      username: 'testadmin',
      displayName: '测试管理员',
      currentOrganization: { id: 'org-1', name: '总部' },
      roleScopes: [{ roleCode: 'admin' }],
    };

    const { container } = render(<AvatarDropdown />);
    const nameEl = container.querySelector('.roncin-avatar-name');
    expect(nameEl).toBeInTheDocument();
    expect(nameEl).toHaveTextContent('测试管理员');
    expect(screen.getByText('测')).toBeInTheDocument();
  });

  it('退出登录成功时调用 clearOrderMasterDataCache() 全量清理订单会话缓存并跳转登录页', async () => {
    mockLogout.mockResolvedValueOnce({});
    mockCurrentUser = {
      username: 'testadmin',
      displayName: '测试管理员',
      currentOrganization: { id: 'org-1', name: '总部' },
      roleScopes: [{ roleCode: 'admin' }],
    };

    const { container } = render(<AvatarDropdown />);
    const trigger = container.querySelector('.roncin-avatar-trigger');
    expect(trigger).toBeInTheDocument();
    if (!trigger) throw new Error('trigger not found');

    fireEvent.mouseEnter(trigger);

    await waitFor(() => {
      expect(screen.getByText('退出登录')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('退出登录'));

    await waitFor(() => {
      expect(mockLogout).toHaveBeenCalledTimes(1);
      expect(mockClearCache).toHaveBeenCalledWith();
      expect(history.replace).toHaveBeenCalledWith('/user/login');
    });
  });

  it('退出登录接口失败时，保持当前会话缓存与登录态不变', async () => {
    mockLogout.mockRejectedValueOnce(new Error('登出接口服务异常'));
    mockCurrentUser = {
      username: 'testadmin',
      displayName: '测试管理员',
      currentOrganization: { id: 'org-1', name: '总部' },
      roleScopes: [{ roleCode: 'admin' }],
    };

    const { container } = render(<AvatarDropdown />);
    const trigger = container.querySelector('.roncin-avatar-trigger');
    expect(trigger).toBeInTheDocument();
    if (!trigger) throw new Error('trigger not found');
    fireEvent.mouseEnter(trigger);

    await waitFor(() => {
      expect(screen.getByText('退出登录')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('退出登录'));

    await waitFor(() => {
      expect(mockLogout).toHaveBeenCalledTimes(1);
    });

    // 失败时不清理缓存，不跳转登录页，不重置 currentUser
    expect(mockClearCache).not.toHaveBeenCalled();
    expect(history.replace).not.toHaveBeenCalled();
  });
});
