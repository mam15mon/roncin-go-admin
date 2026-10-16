import { describe, expect, it } from 'vitest';
import access from '@/access';
import {
  buildWorkspaceMenus,
  resolveWorkspaceMenuKey,
  toMobileWorkspaceMenus,
} from './workspaceMenus';

function menusFor(permissions: string[], isPlatform = false) {
  return buildWorkspaceMenus(
    access({
      currentUser: {
        id: 'u',
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
        roleScopes: [{ roleCode: 'tester' }],
        currentOrganization: { id: 'org', kind: isPlatform ? 1 : 2 },
      },
    }),
  );
}
describe('工作区共享导航菜单', () => {
  it('公司权限精准筛选并映射相同路径、中文名、图标；无权限无空组', () => {
    const none = menusFor([]);
    expect(toMobileWorkspaceMenus(none)).toEqual([]);
    const menus = menusFor([
      'business.partner.read',
      'business.enterprise_resource.read',
      'system.master_data.currency.read',
    ]);
    // 使用真实access函数，权限码必须匹配服务端生成清单。
    expect(menus.enterpriseResourceItems.map((item) => item.key)).toEqual([
      '/partners/customers',
      '/partners/suppliers',
      '/partners/foreign-agents',
      '/enterprise-resources/config',
    ]);
    const mobile = toMobileWorkspaceMenus(menus);
    for (const group of mobile) {
      const originals =
        group.key === 'workspace-enterprise'
          ? menus.enterpriseResourceItems
          : menus.settingsItems;
      expect(group.children).toEqual(
        originals.map((item) => ({
          key: item.key,
          path: item.key,
          name: item.label,
          icon: item.icon,
        })),
      );
    }
  });
  it('系统工作区权限与query高亮同一来源，不选另一个设置tab', () => {
    const menus = menusFor(['system.platform.access'], true);
    expect(resolveWorkspaceMenuKey('/admin', '?tab=users&extra=1', menus)).toBe(
      '/admin?tab=users',
    );
    expect(resolveWorkspaceMenuKey('/admin', '', menus)).toBeUndefined();
    expect(
      resolveWorkspaceMenuKey('/admin', '?tab=unknown', menus),
    ).toBeUndefined();
    expect(
      resolveWorkspaceMenuKey('/partners/customers/detail', '', menus),
    ).toBeUndefined();
    const onlyUsers = menusFor(['system.user.update'], true);
    expect(resolveWorkspaceMenuKey('/admin', '', onlyUsers)).toBeUndefined();
  });
});
