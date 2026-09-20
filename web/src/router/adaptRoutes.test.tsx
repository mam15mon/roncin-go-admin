import access from '@/access';
import { describe, expect, it } from 'vitest';
import { buildMenuData, buildRouterConfig } from './adaptRoutes';

// 路由适配层冒烟：锁死「routes.ts → RouterConfig + 菜单」两端完整性。
// 背景：import.meta.glob 的相对路径错级（../../pages vs ../pages）不会被
// tsc/build 捕获（lazy 不在构建期求值），只有真实加载才能暴露。
describe('adaptRoutes 路由适配', () => {
  it('全部 35 个带 component 的路由均能解析并加载页面模块', async () => {
    const flat: { lazy?: () => Promise<unknown>; children?: unknown[] }[] = [];
    const walk = (routes: typeof flat) => {
      for (const route of routes) {
        flat.push(route);
        if (Array.isArray(route.children)) walk(route.children);
      }
    };
    walk(buildRouterConfig() as unknown as typeof flat);

    const lazyRoutes = flat.filter((route) => typeof route.lazy === 'function');
    // 35 个页面组件 + 1 个 AppLayout 布局壳。
    expect(lazyRoutes.length).toBe(36);

    await Promise.all(
      lazyRoutes.map((route) =>
        route.lazy!().then((loaded) => {
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
          permissions: ['system.platform.access', 'business.order.se.read'],
          roleScopes: [{ roleCode: 'r', dataScope: 'all' }],
        } as unknown as API.CurrentUser,
      }),
    );
    const paths = (superAdmin.routes ?? []).map((item) => item.path);
    expect(paths).toContain('/welcome');
    expect(paths).toContain('/orders');
  });
});
