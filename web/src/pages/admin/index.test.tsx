import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import AdminPage from './index';

// Mock umi access & hooks
const accessState = vi.hoisted(() => ({
  value: {
    canReadOrganizations: true,
    canReadUsers: true,
    canManageDingTalkInvitations: true,
    canReadRoles: true,
    canReadMasterDataNumberRules: true,
    canReadMasterDataItems: true,
    canReadAudit: true,
    canReadTasks: true,
    canReadPermissions: true,
  },
}));

const historyMock = vi.hoisted(() => ({
  replace: vi.fn(),
}));

const locationState = vi.hoisted(() => ({
  pathname: '/admin',
  search: '',
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState.value,
}));

vi.mock('@/router/history', () => ({
  history: historyMock,
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return { ...actual, useLocation: () => locationState };
});

// Mock sub-panels to isolate test
vi.mock('./organizations', () => ({
  default: () => <div data-testid="organizations-panel">组织架构面板</div>,
}));
vi.mock('./users', () => ({
  default: () => <div data-testid="users-panel">用户管理面板</div>,
}));
vi.mock('./roles', () => ({
  default: () => <div data-testid="roles-panel">角色权限面板</div>,
}));
vi.mock('./components/NumberRulesPanel', () => ({
  default: () => <div data-testid="number-rules-panel">单据规则面板</div>,
}));
vi.mock('./components/AbnormalCasesPanel', () => ({
  default: () => <div data-testid="abnormal-cases-panel">业务异常面板</div>,
}));
vi.mock('./audit', () => ({
  default: () => <div data-testid="audit-panel">审计日志面板</div>,
}));
vi.mock('./background-tasks', () => ({
  default: () => <div data-testid="background-tasks-panel">后台任务面板</div>,
}));
vi.mock('./permissions', () => ({
  default: () => <div data-testid="permissions-panel">权限字典面板</div>,
}));

describe('AdminPage (系统管理中心多Tab模板)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.search = '';
    accessState.value = {
      canReadOrganizations: true,
      canReadUsers: true,
      canManageDingTalkInvitations: true,
      canReadRoles: true,
      canReadMasterDataNumberRules: true,
      canReadMasterDataItems: true,
      canReadAudit: true,
      canReadTasks: true,
      canReadPermissions: true,
    };
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染系统管理中心标题、副标题与8个核心业务Tab', () => {
    render(<AdminPage />);

    expect(screen.getByText('系统管理')).toBeInTheDocument();
    expect(
      screen.getByText(
        '统一管理组织架构、用户账号、角色权限、单据规则及审计日志',
      ),
    ).toBeInTheDocument();

    expect(screen.getByText('组织架构')).toBeInTheDocument();
    expect(screen.getByText('用户管理')).toBeInTheDocument();
    expect(screen.getByText('角色权限')).toBeInTheDocument();
    expect(screen.getByText('单据规则')).toBeInTheDocument();
    expect(screen.getByText('业务异常')).toBeInTheDocument();
    expect(screen.getByText('审计日志')).toBeInTheDocument();
    expect(screen.getByText('后台任务')).toBeInTheDocument();
    expect(screen.getByText('权限字典')).toBeInTheDocument();

    // 钉钉邀请与注册审批已下沉收敛至用户管理内部，不在顶级Tab展示
    expect(screen.queryByText('钉钉邀请')).not.toBeInTheDocument();
    expect(screen.queryByText('注册审批')).not.toBeInTheDocument();

    // 默认展示 organizations 面板
    expect(screen.getByTestId('organizations-panel')).toBeInTheDocument();
  });

  it('旧链接 tab=dingtalk-registrations 自动平滑重定向至用户管理下的子页签', () => {
    locationState.search = '?tab=dingtalk-registrations';
    render(<AdminPage />);

    expect(historyMock.replace).toHaveBeenCalledWith(
      '/admin?tab=users&subTab=registrations',
    );
  });

  it('当没有任何管理权限时展示暂无权限警告卡片', () => {
    accessState.value = {
      canReadOrganizations: false,
      canReadUsers: false,
      canManageDingTalkInvitations: false,
      canReadRoles: false,
      canReadMasterDataNumberRules: false,
      canReadMasterDataItems: false,
      canReadAudit: false,
      canReadTasks: false,
      canReadPermissions: false,
    };

    render(<AdminPage />);
    expect(screen.getByText('暂无可用的管理权限')).toBeInTheDocument();
  });
});
