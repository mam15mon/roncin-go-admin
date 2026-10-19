import { renderWithApp } from '@root/tests/renderWithApp';
import { cleanup, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import access from '@/access';
import { buildWorkspaceMenus } from '@/components/layout/workspaceMenus';
import { AccessGuard } from '@/router/guard';

afterEach(cleanup);

describe('通知测试入口与后端有效权限投影一致', () => {
  it.each([
    ['系统管理员有效通知权限', 1, 'administrator', true],
    ['系统普通角色无有效通知权限', 1, 'operator', false],
    ['公司管理员无有效通知权限', 2, 'administrator', false],
    ['只有系统管理员角色展示无有效权限', 1, 'administrator', false],
  ] as const)('%s', (_name, kind, roleCode, allowed) => {
    const permissions = allowed
      ? ['system.notification.preview', 'system.notification.test_send']
      : ['system.platform.access'];
    const user: API.CurrentUser = {
      id: 'notification-user',
      currentOrganization: { id: 'workspace', kind },
      roleScopes: [{ roleCode, roleName: '系统管理员' }],
      permissions,
      permissionCapabilities: permissions.map((key) => ({
        key,
        dataScope: 'organization',
      })),
    };
    const effectiveAccess = access({ currentUser: user });
    expect(effectiveAccess.canPreviewNotifications).toBe(allowed);
    expect(effectiveAccess.canSendNotificationTests).toBe(allowed);
    expect(
      buildWorkspaceMenus(effectiveAccess).settingsItems.some(
        (item) => item.key === '/admin/notification-test',
      ),
    ).toBe(allowed);
    renderWithApp(
      <AccessGuard accessKey="canPreviewNotifications">
        <div>通知测试页面</div>
      </AccessGuard>,
      { user },
    );
    if (allowed) {
      expect(screen.getByText('通知测试页面')).toBeInTheDocument();
    } else {
      expect(screen.getByText('无权访问此页面')).toBeInTheDocument();
      expect(screen.queryByText('通知测试页面')).not.toBeInTheDocument();
    }
  });
});
