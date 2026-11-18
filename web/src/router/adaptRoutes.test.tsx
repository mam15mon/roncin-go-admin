import { describe, expect, it } from 'vitest';
import access from '@/access';
import { AuthOrganizationKind } from '@/enums.generated';
import configuredRoutes from '../../config/routes';
import { buildMenuData, buildRouterConfig } from './adaptRoutes';

// 路由适配层冒烟：锁死「routes.ts → RouterConfig + 菜单」两端完整性。
// 背景：import.meta.glob 的相对路径错级（../../pages vs ../pages）不会被
// tsc/build 捕获（lazy 不在构建期求值），只有真实加载才能暴露。
describe('adaptRoutes 路由适配', () => {
  it('全部配置页面和布局路由均能解析并加载模块', async () => {
    interface FlatRoute {
      path?: string;
      lazy?: () => Promise<unknown>;
      children?: FlatRoute[];
      errorElement?: unknown;
    }
    const flat: FlatRoute[] = [];
    const walk = (routes: FlatRoute[]) => {
      for (const route of routes) {
        flat.push(route);
        if (route.children) walk(route.children);
      }
    };
    walk(buildRouterConfig() as unknown as FlatRoute[]);

    expect(flat.every((route) => route.errorElement)).toBe(true);

    const lazyRoutes = flat.filter((route) => typeof route.lazy === 'function');
    // 从唯一配置核对所有组件声明，缺失 loader 不能因菜单增减而被固定数量掩盖。
    const countComponents = (routes: typeof configuredRoutes): number =>
      routes.reduce(
        (count, route) =>
          count +
          Number(Boolean(route.component)) +
          (route.routes ? countComponents(route.routes) : 0),
        0,
      );
    expect(lazyRoutes).toHaveLength(countComponents(configuredRoutes) + 1);
    expect(
      lazyRoutes.some((route) => route.path === '/orders/sea-import'),
    ).toBe(true);
    expect(
      lazyRoutes.some((route) => route.path === '/orders/sea-import/:id/hbl'),
    ).toBe(true);

    await Promise.all(
      lazyRoutes.map((route) =>
        route.lazy?.().then((loaded) => {
          expect((loaded as { Component?: unknown }).Component).toBeTruthy();
        }),
      ),
    );
  });

  it('菜单数据按权限过滤：未登录产出空菜单，超管产出完整顶级菜单', () => {
    const anonymous = buildMenuData(access({ currentUser: undefined }));
    expect(anonymous.routes).toHaveLength(0);

    const superAdmin = buildMenuData(
      access({
        currentUser: {
          id: 'x',
          username: 'admin',
          currentOrganization: { kind: 2 },
          permissionCapabilities: [
            'system.platform.access',
            'business.order.se.read',
          ].map((key) => ({ key, dataScope: 'all' })),
          roleScopes: [{ roleCode: 'r' }],
        } as unknown as API.CurrentUser,
      }),
    );
    const paths = (superAdmin.routes ?? []).map((item) => item.path);
    expect(paths).toContain('/welcome');
    expect(paths).toContain('/orders');
  });

  it('财务报表独立显示，费用管理只列有权访问的业务操作', () => {
    const menus = (keys: string[]) =>
      buildMenuData(
        access({
          currentUser: {
            id: 'u',
            currentOrganization: {
              id: 'org',
              kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
            },
            permissionCapabilities: keys.map((key) => ({
              key,
              dataScope: 'organization',
            })),
          },
        }),
      ).routes;

    expect(menus(['system.finance.report.read'])).toMatchObject([
      { path: '/finance/reports', name: '财务报表' },
    ]);
    expect(
      menus(['system.finance.exchange_rate.read', 'system.finance.fee_setting.read']),
    ).toEqual([]);
    expect(
      menus(['system.finance.fee.read', 'system.finance.netting.read']),
    ).toMatchObject([
      {
        path: '/finance',
        name: '费用管理',
        routes: [{ path: '/finance/fees' }, { path: '/finance/nettings' }],
      },
    ]);
  });
});
