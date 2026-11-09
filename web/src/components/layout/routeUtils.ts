import routes from '@root/config/routes';
import { matchPath } from 'react-router';
import type { UmiRoute } from '@/router/routeTypes';

/** 页面标题、重定向与布局属性统一读取真实路由配置，隐藏菜单不影响标题。 */
function flattenRoutes(items: UmiRoute[], parentLayout = true): UmiRoute[] {
  return items.flatMap((route) => {
    const layout = parentLayout && route.layout !== false;
    return [{ ...route, layout }, ...flattenRoutes(route.routes ?? [], layout)];
  });
}

const configuredRoutes = flattenRoutes(routes);
const titleRoutes = configuredRoutes
  .filter(
    (route) => route.path && route.layout && (route.pageTitle || route.name),
  )
  .sort(
    (a, b) => routeSpecificity(b.path ?? '') - routeSpecificity(a.path ?? ''),
  );

function routeSpecificity(path: string): number {
  return path
    .split('/')
    .reduce((score, segment) => score + (segment.startsWith(':') ? 1 : 10), 0);
}

function normalizePath(pathname: string): string {
  const path = pathname.split(/[?#]/, 1)[0] || '/';
  return path.length > 1 && path.endsWith('/') ? path.slice(0, -1) : path;
}

function resolveRedirectPath(pathname: string): string {
  let path = normalizePath(pathname);
  const visited = new Set<string>();
  while (!visited.has(path)) {
    visited.add(path);
    const redirect = configuredRoutes.find(
      (route) => route.path === path && route.redirect,
    )?.redirect;
    if (!redirect) return path;
    path = normalizePath(redirect);
  }
  throw new Error('路由重定向存在循环');
}

/** 无布局页面及其子路由不创建业务页签。 */
export function isRouteWithoutLayout(pathname: string): boolean {
  const path = normalizePath(pathname);
  return configuredRoutes.some(
    (route) =>
      route.layout === false &&
      route.path &&
      matchPath({ path: route.path, end: false }, path),
  );
}

/**
 * 集中配置内部子页面向稳定菜单页签键的归组规则表。
 * 严格限定在实际存在的路径结构，未知扩展路径不得被过度合并。
 */
export const TAB_KEY_RULES: Array<{ pattern: RegExp; key: string }> = [
  // 1. 海运出口订单：列表、新建、详情、费用录入、拆票、HBL 制作
  {
    pattern:
      /^\/orders\/sea-export(?:\/(?:new|[^/]+(?:\/(?:fees|split|hbl))?))?$/,
    key: '/orders/sea-export',
  },
  // 2. 客商管理：客户列表、新建、详情
  // 仅允许：/partners/customers, /partners/customers/create, /partners/customers/:id
  {
    pattern: /^\/partners\/customers(?:\/(?:create|[^/]+))?$/,
    key: '/partners/customers',
  },
  // 3. 客商管理：供应商列表、新建、详情
  // 仅允许：/partners/suppliers, /partners/suppliers/create, /partners/suppliers/:id
  {
    pattern: /^\/partners\/suppliers(?:\/(?:create|[^/]+))?$/,
    key: '/partners/suppliers',
  },
  // 4. 客商管理：国外代理列表、新建、详情
  // 仅允许：/partners/foreign-agents(?:\/(?:create|[^/]+))?$/,
  {
    pattern: /^\/partners\/foreign-agents(?:\/(?:create|[^/]+))?$/,
    key: '/partners/foreign-agents',
  },
  // 5. 费用管理：集运费用明细与单票费用详情
  // 仅允许：/finance/fees, /finance/fees/detail/:orderId
  {
    pattern: /^\/finance\/fees(?:\/detail\/[^/]+)?$/,
    key: '/finance/fees',
  },
];

/**
 * 将任意 URL pathname 解析为对应的稳定页签 key
 */
export function resolveTabKey(pathname: string): string {
  if (!pathname || pathname === '/' || pathname === '/welcome') {
    return '/welcome';
  }

  // 规范化：去除末尾的斜杠
  const targetPath = resolveRedirectPath(pathname);

  for (const { pattern, key } of TAB_KEY_RULES) {
    if (pattern.test(targetPath)) {
      return key;
    }
  }

  return targetPath;
}

/**
 * 根据 pathname 解析展示的模块/页面标题
 */
export function resolveRouteTitle(pathname: string): string {
  const targetPath = resolveRedirectPath(pathname);
  for (const route of titleRoutes) {
    const match = matchPath({ path: route.path ?? '', end: true }, targetPath);
    if (!match) continue;
    return typeof route.pageTitle === 'function'
      ? route.pageTitle(match.params)
      : (route.pageTitle ?? route.name ?? '工作台');
  }

  // 仅使用静态父入口兜底，避免未知路径被不相干的动态详情吞掉。
  for (const route of titleRoutes) {
    if (
      route.path?.includes(':') ||
      !route.path ||
      !targetPath.startsWith(`${route.path}/`)
    )
      continue;
    return typeof route.pageTitle === 'string'
      ? route.pageTitle
      : (route.name ?? '工作台');
  }
  return targetPath.split('/').filter(Boolean).at(-1) || '工作台';
}
