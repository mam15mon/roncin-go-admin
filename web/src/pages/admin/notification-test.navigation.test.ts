import { describe, expect, it } from 'vitest';
import access from '@/access';
import {
  resolveRouteTitle,
  resolveTabKey,
} from '@/components/layout/routeUtils';
import {
  buildWorkspaceMenus,
  resolveWorkspaceMenuKey,
  toMobileWorkspaceMenus,
} from '@/components/layout/workspaceMenus';
import routes from '../../../config/routes';

function permissions(keys: string[]) {
  return access({
    currentUser: {
      id: 'user',
      currentOrganization: { id: 'system', kind: 1 },
      permissionCapabilities: keys.map((key) => ({
        key,
        dataScope: 'organization',
      })),
      roleScopes: [{ roleCode: 'non-admin' }],
    },
  });
}

describe('通知测试独立 Admin 入口', () => {
  it('路由独立受预览权限保护，页签不与系统管理合并', () => {
    expect(
      routes.find((route) => route.path === '/admin/notification-test'),
    ).toMatchObject({
      name: '通知测试',
      component: './admin/notification-test',
      access: 'canPreviewNotifications',
    });
    expect(resolveRouteTitle('/admin/notification-test')).toBe('通知测试');
    expect(resolveTabKey('/admin/notification-test')).toBe(
      '/admin/notification-test',
    );
    expect(resolveTabKey('/admin')).toBe('/admin');
  });
  it('入口精准消费权限，桌面和移动可发现并高亮，不靠角色名', () => {
    const preview = permissions(['system.notification.preview']);
    expect(preview.canPreviewNotifications).toBe(true);
    expect(preview.canSendNotificationTests).toBe(false);
    const menus = buildWorkspaceMenus(preview);
    expect(menus.settingsItems).toEqual([
      expect.objectContaining({
        key: '/admin/notification-test',
        label: '通知测试',
      }),
    ]);
    expect(toMobileWorkspaceMenus(menus)[0].children).toEqual([
      expect.objectContaining({
        path: '/admin/notification-test',
        name: '通知测试',
      }),
    ]);
    expect(resolveWorkspaceMenuKey('/admin/notification-test', '', menus)).toBe(
      '/admin/notification-test',
    );
    const noPermission = buildWorkspaceMenus(
      permissions(['system.platform.access']),
    );
    expect(
      noPermission.settingsItems.some(
        (item) => item.key === '/admin/notification-test',
      ),
    ).toBe(false);
    expect(
      permissions([
        'system.notification.preview',
        'system.notification.test_send',
      ]).canSendNotificationTests,
    ).toBe(true);
  });
});
