/**
 * 路由名称映射与解析工具
 */

export const ROUTE_TITLE_MAP: Record<string, string> = {
  '/welcome': '工作台',
  '/partners/customers': '客户',
  '/partners/customers/create': '新建客户',
  '/partners/suppliers': '供应商',
  '/partners/suppliers/create': '新建供应商',
  '/partners/foreign-agents': '国外代理',
  '/partners/foreign-agents/create': '新建国外代理',
  '/orders/sea-export': '海运出口',
  '/orders/sea-export/new': '新建海运出口订单',
  '/orders/sea-import': '海运进口',
  '/orders/sea-import/new': '新建海运进口订单',
  '/orders/air-export': '空运出口',
  '/orders/air-export/new': '新建空运出口订单',
  '/orders/air-import': '空运进口',
  '/orders/air-import/new': '新建空运进口订单',
  '/orders': '订单管理',
  '/master-data': '主数据',
  '/finance/fee-settings': '费用设置',
  '/finance/exchange-rates': '汇率设置',
  '/admin': '系统管理',
};

const DYNAMIC_ROUTE_PATTERNS: Array<{
  pattern: RegExp;
  title: string;
}> = [
  { pattern: /^\/partners\/customers\/(?!create)[^/]+$/, title: '客户详情' },
  { pattern: /^\/partners\/suppliers\/(?!create)[^/]+$/, title: '供应商详情' },
  {
    pattern: /^\/partners\/foreign-agents\/(?!create)[^/]+$/,
    title: '国外代理详情',
  },
];

/**
 * 根据 pathname 解析展示的模块/页面标题
 */
export function resolveRouteTitle(pathname: string): string {
  if (!pathname || pathname === '/') return '工作台';
  if (ROUTE_TITLE_MAP[pathname]) return ROUTE_TITLE_MAP[pathname];

  for (const { pattern, title } of DYNAMIC_ROUTE_PATTERNS) {
    if (pattern.test(pathname)) {
      return title;
    }
  }

  // 按路径长度倒序进行前缀匹配，确保更具体的子路径优先命中
  const sortedPrefixes = Object.entries(ROUTE_TITLE_MAP).sort(
    (a, b) => b[0].length - a[0].length,
  );
  for (const [routePath, title] of sortedPrefixes) {
    if (pathname === routePath || pathname.startsWith(`${routePath}/`)) {
      return title;
    }
  }

  const segments = pathname.split('/').filter(Boolean);
  return segments[segments.length - 1] || '工作台';
}
