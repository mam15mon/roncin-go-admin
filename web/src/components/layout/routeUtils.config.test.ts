import { describe, expect, it, vi } from 'vitest';

// 测试只增加集中配置：标题解析器不能要求 layout 同步维护另一张名称表。
vi.mock('@root/config/routes', () => ({
  default: [
    { path: '/new-business', name: '新增业务', hideInMenu: true },
    { path: '/new-business/:id', name: '新增业务详情' },
    { path: '/new-business/create', name: '新建业务' },
    { path: '/entry', redirect: '/middle' },
    { path: '/middle', redirect: '/new-business' },
    {
      path: '/public',
      layout: false,
      routes: [{ path: '/public/callback', name: '公开回调' }],
    },
  ],
}));

import {
  isRouteWithoutLayout,
  resolveRouteTitle,
  resolveTabKey,
} from './routeUtils';

describe('集中路由配置驱动展示', () => {
  it('新命名页面、隐藏菜单和动态详情自动获得标题，静态入口优先', () => {
    expect(resolveRouteTitle('/new-business')).toBe('新增业务');
    expect(resolveRouteTitle('/new-business/id-1')).toBe('新增业务详情');
    expect(resolveRouteTitle('/new-business/create')).toBe('新建业务');
    expect(resolveRouteTitle('/new-business/?keyword=测试#section')).toBe(
      '新增业务',
    );
  });

  it('重定向取配置的最终目标，不创建中转页签身份', () => {
    expect(resolveRouteTitle('/entry')).toBe('新增业务');
    expect(resolveTabKey('/entry')).toBe('/new-business');
  });

  it('无布局属性对子路由生效，隐藏菜单仍是业务页', () => {
    expect(isRouteWithoutLayout('/public/callback')).toBe(true);
    expect(isRouteWithoutLayout('/new-business')).toBe(false);
  });
});
