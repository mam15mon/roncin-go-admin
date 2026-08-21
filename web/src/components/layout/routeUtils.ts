/**
 * 路由名称映射与解析工具
 */

export const ROUTE_TITLE_MAP: Record<string, string> = {
  '/welcome': '工作台',
  '/partners/customers': '客户',
  '/partners/suppliers': '供应商',
  '/partners/foreign-agents': '国外代理',
  '/orders': '订单管理',
  '/master-data': '主数据',
  '/admin': '系统管理',
};

/**
 * 根据 pathname 解析展示的模块/页面标题
 */
export function resolveRouteTitle(pathname: string): string {
  if (!pathname || pathname === '/') return '工作台';
  if (ROUTE_TITLE_MAP[pathname]) return ROUTE_TITLE_MAP[pathname];

  for (const [routePath, title] of Object.entries(ROUTE_TITLE_MAP)) {
    if (pathname === routePath || pathname.startsWith(`${routePath}/`)) {
      return title;
    }
  }

  const segments = pathname.split('/').filter(Boolean);
  return segments[segments.length - 1] || '工作台';
}
