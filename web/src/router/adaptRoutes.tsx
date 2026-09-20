import {
  ContactsOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  GlobalOutlined,
  OrderedListOutlined,
  SettingOutlined,
  ShopOutlined,
  TransactionOutlined,
} from '@ant-design/icons';
import type { MenuDataItem } from '@ant-design/pro-components';
import type { ComponentType, ReactNode } from 'react';
import type { RouteObject } from 'react-router';
import { Navigate, Outlet } from 'react-router';
import umiRoutes from '../../config/routes';
import { AccessGuard } from './guard';
import type { AccessState, UmiRoute } from './routeTypes';

type PageLoader = () => Promise<{ default: ComponentType }>;

// pro-components v3 的 MenuDataItem.routes 类型为 undefined，嵌套菜单需用
// Route 形状（Omit<MenuDataItem,'routes'> & { routes?: ... }），此处本地等价定义。
type ProRoute = Omit<MenuDataItem, 'routes'> & { routes?: ProRoute[] };

const pageModules = import.meta.glob([
  '../pages/**/*.tsx',
  '!../pages/**/*.test.tsx',
]) as Record<string, PageLoader>;

// 组件路径两段式解析：先 `pages/<path>.tsx`，未命中回退 `pages/<path>/index.tsx`
// （routes.ts 中单文件与目录 index 两种形态并存）。
function resolvePageLoader(component: string): PageLoader | undefined {
  const base = component.replace(/^\.\//, '');
  return (
    pageModules[`../pages/${base}.tsx`] ??
    pageModules[`../pages/${base}/index.tsx`]
  );
}

// routes.ts 的 icon 字符串 → antd 图标组件（接管 Umi layout 插件的编译期映射）。
const ICON_MAP: Record<string, ReactNode> = {
  contacts: <ContactsOutlined />,
  dashboard: <DashboardOutlined />,
  database: <DatabaseOutlined />,
  global: <GlobalOutlined />,
  orderedList: <OrderedListOutlined />,
  setting: <SettingOutlined />,
  shop: <ShopOutlined />,
  transaction: <TransactionOutlined />,
};

// 通配符 `./*` 规范化为 `*`；其余均为绝对路径，原样保留。
function normalizePath(path: string): string {
  return path.replace(/^\.\//, '');
}

function toRoute(route: UmiRoute): RouteObject {
  const children = route.routes?.map(toRoute);
  const result: RouteObject = {};
  if (route.path !== undefined) {
    result.path = normalizePath(route.path);
  }
  const loader = route.component
    ? resolvePageLoader(route.component)
    : undefined;
  if (loader) {
    const accessKey = route.access;
    result.lazy = async () => {
      const { default: Component } = await loader();
      return accessKey
        ? {
            Component: () => (
              <AccessGuard accessKey={accessKey}>
                <Component />
              </AccessGuard>
            ),
          }
        : { Component };
    };
  } else if (route.redirect) {
    result.element = <Navigate to={route.redirect} replace />;
  } else if (children) {
    // 无 component 的嵌套组显式兜底 Outlet（RR 隐式行为本就如此，显式更清晰）
    result.element = <Outlet />;
  }
  if (children) {
    result.children = children;
  }
  return result;
}

export function buildRouterConfig(): RouteObject[] {
  const bare: RouteObject[] = [];
  const shelled: RouteObject[] = [];
  for (const route of umiRoutes) {
    (route.layout === false ? bare : shelled).push(toRoute(route));
  }
  return [
    ...bare,
    {
      // 布局壳懒加载，同时切断 router → AppLayout → history → router 的模块环
      lazy: async () => {
        const { AppLayout } = await import('../app/AppLayout');
        return { Component: AppLayout };
      },
      children: shelled,
    },
  ];
}

// 菜单数据：同一份 routes.ts 产出 ProLayout route 树，按 access 过滤 +
// 剔除 layout:false / hideInMenu 项（对齐 Umi layout 插件菜单行为）。
export function buildMenuData(accessState: AccessState): ProRoute {
  return {
    path: '/',
    routes: buildMenuItems(umiRoutes, accessState),
  };
}

function buildMenuItems(
  routes: UmiRoute[],
  accessState: AccessState,
): ProRoute[] {
  const items: ProRoute[] = [];
  for (const route of routes) {
    if (route.layout === false || route.hideInMenu) continue;
    if (!route.name || !route.path) continue;
    if (route.access && !accessState[route.access]) continue;
    const icon = route.icon ? ICON_MAP[route.icon] : undefined;
    const children = route.routes
      ? buildMenuItems(route.routes, accessState)
      : undefined;
    const item: ProRoute = {
      path: route.path,
      name: route.name,
      ...(icon ? { icon } : {}),
      ...(children ? { routes: children } : {}),
    };
    items.push(item);
  }
  return items;
}
