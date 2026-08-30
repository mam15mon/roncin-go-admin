import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { HeaderMenus } from './HeaderMenus';

const mockPush = vi.fn();
let mockPathname = '/welcome';
let mockAccess: Record<string, boolean> = {
  canReadMasterData: true,
  canAccessPlatform: true,
  canReadPartners: true,
};

vi.mock('@umijs/max', () => ({
  history: {
    push: (path: string) => mockPush(path),
  },
  useLocation: () => ({
    pathname: mockPathname,
    search: '',
    hash: '',
  }),
  useAccess: () => mockAccess,
}));

describe('HeaderMenus Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPathname = '/welcome';
    mockAccess = {
      canReadMasterData: true,
      canAccessPlatform: true,
      canReadPartners: true,
    };
  });

  afterEach(() => {
    cleanup();
  });

  it('具有全部权限时展示设置中心与企业资源', () => {
    render(<HeaderMenus />);
    expect(screen.getByText('设置中心')).toBeInTheDocument();
    expect(screen.getByText('企业资源')).toBeInTheDocument();
  });

  it('没有任何模块权限时组件返回 null 不展示', () => {
    mockAccess = {
      canReadMasterData: false,
      canAccessPlatform: false,
      canReadPartners: false,
    };
    const { container } = render(<HeaderMenus />);
    expect(container.firstChild).toBeNull();
  });

  it('仅有主数据权限时仅展示设置中心', () => {
    mockAccess = {
      canReadMasterData: true,
      canAccessPlatform: false,
      canReadPartners: false,
    };
    render(<HeaderMenus />);
    expect(screen.getByText('设置中心')).toBeInTheDocument();
    expect(screen.queryByText('企业资源')).not.toBeInTheDocument();
  });

  it('仅有客商权限时仅展示企业资源', () => {
    mockAccess = {
      canReadMasterData: false,
      canAccessPlatform: false,
      canReadPartners: true,
    };
    render(<HeaderMenus />);
    expect(screen.queryByText('设置中心')).not.toBeInTheDocument();
    expect(screen.getByText('企业资源')).toBeInTheDocument();
  });

  it('仅有企业资源配置权限时仅展示企业资源', () => {
    mockAccess = {
      canReadMasterData: false,
      canAccessPlatform: false,
      canReadPartners: false,
      canReadEnterpriseResources: true,
    };
    render(<HeaderMenus />);
    expect(screen.queryByText('设置中心')).not.toBeInTheDocument();
    expect(screen.getByText('企业资源')).toBeInTheDocument();
  });
});
