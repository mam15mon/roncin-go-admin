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
  '/orders/sea-export': '海运出口订单列表',
  '/orders/sea-export/new': '新增海运出口',
  '/orders/sea-import': '海运进口订单列表',
  '/orders/sea-import/new': '新增海运进口',
  '/orders/air-export': '空运出口订单列表',
  '/orders/air-export/new': '新增空运出口',
  '/orders/air-import': '空运进口订单列表',
  '/orders/air-import/new': '新增空运进口',
  '/orders': '订单管理',
  '/master-data': '主数据',
  '/settings': '参数设置',
  '/finance/fee-settings': '费用设置',
  '/finance/exchange-rates': '汇率设置',
  '/finance/fees': '集运费用明细',
  '/finance/bills': '账单管理',
  '/finance/invoices': '开票记录',
  '/finance/cashflows': '收付管理',
  '/finance/verifications': '核销管理',
  '/finance/commissions': '提成管理',
  '/admin': '系统管理',
};

const KIND_NAMES: Record<string, string> = {
  'sea-export': '海运出口',
  'sea-import': '海运进口',
  'air-export': '空运出口',
  'air-import': '空运进口',
  rail: '铁路运输',
  truck: '内陆拖车',
  customs: '报关业务',
};

export const DYNAMIC_ROUTE_PATTERNS: Array<{
  pattern: RegExp;
  title: string | ((matches: RegExpMatchArray) => string);
}> = [
  { pattern: /^\/partners\/customers\/(?!create)[^/]+$/, title: '客户详情' },
  { pattern: /^\/partners\/suppliers\/(?!create)[^/]+$/, title: '供应商详情' },
  {
    pattern: /^\/partners\/foreign-agents\/(?!create)[^/]+$/,
    title: '国外代理详情',
  },
  {
    pattern: /^\/orders\/([^/]+)\/([^/]+)\/fees$/,
    title: (m) => `${KIND_NAMES[m[1]] || '订单'}费用录入`,
  },
  {
    pattern: /^\/orders\/([^/]+)\/(?!new)[^/]+$/,
    title: (m) => `${KIND_NAMES[m[1]] || '订单'}详情`,
  },
];

/**
 * 根据 pathname 解析展示的模块/页面标题
 */
export function resolveRouteTitle(pathname: string): string {
  if (!pathname || pathname === '/') return '工作台';
  if (ROUTE_TITLE_MAP[pathname]) return ROUTE_TITLE_MAP[pathname];

  for (const { pattern, title } of DYNAMIC_ROUTE_PATTERNS) {
    const match = pathname.match(pattern);
    if (match) {
      return typeof title === 'function' ? title(match) : title;
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
