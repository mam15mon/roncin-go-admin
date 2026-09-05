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
    pattern: /^\/orders\/([^/]+)\/([^/]+)\/split$/,
    title: (m) => `${KIND_NAMES[m[1]] || '订单'}拆票`,
  },
  {
    pattern: /^\/orders\/([^/]+)\/(?!new)[^/]+$/,
    title: (m) => `${KIND_NAMES[m[1]] || '订单'}详情`,
  },
  {
    pattern: /^\/finance\/fees\/detail\/[^/]+$/,
    title: '费用详情',
  },
];

/**
 * 纯重定向入口向最终业务菜单页签的映射表，避免进入中转重定向时产生冗余中间页签
 */
export const REDIRECT_ROUTES: Record<string, string> = {
  '/orders': '/orders/sea-export',
  '/partners': '/partners/customers',
};

/**
 * 集中配置内部子页面向稳定菜单页签键的归组规则表。
 * 严格限定在实际存在的路径结构，未知扩展路径不得被过度合并。
 */
export const TAB_KEY_RULES: Array<{ pattern: RegExp; key: string }> = [
  // 1. 海运出口订单：列表、新建、详情、费用录入、拆票
  // 仅允许：/orders/sea-export, /orders/sea-export/new, /orders/sea-export/:id, /orders/sea-export/:id/fees, /orders/sea-export/:id/split
  {
    pattern: /^\/orders\/sea-export(?:\/(?:new|[^/]+(?:\/(?:fees|split))?))?$/,
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
  const normalized =
    pathname.length > 1 && pathname.endsWith('/')
      ? pathname.slice(0, -1)
      : pathname;

  const targetPath = REDIRECT_ROUTES[normalized] || normalized;

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
  if (!pathname || pathname === '/') return '工作台';

  const normalized =
    pathname.length > 1 && pathname.endsWith('/')
      ? pathname.slice(0, -1)
      : pathname;

  const targetPath = REDIRECT_ROUTES[normalized] || normalized;

  if (ROUTE_TITLE_MAP[targetPath]) return ROUTE_TITLE_MAP[targetPath];

  for (const { pattern, title } of DYNAMIC_ROUTE_PATTERNS) {
    const match = targetPath.match(pattern);
    if (match) {
      return typeof title === 'function' ? title(match) : title;
    }
  }

  // 按路径长度倒序进行前缀匹配，确保更具体的子路径优先命中
  const sortedPrefixes = Object.entries(ROUTE_TITLE_MAP).sort(
    (a, b) => b[0].length - a[0].length,
  );
  for (const [routePath, title] of sortedPrefixes) {
    if (targetPath === routePath || targetPath.startsWith(`${routePath}/`)) {
      return title;
    }
  }

  const segments = targetPath.split('/').filter(Boolean);
  return segments[segments.length - 1] || '工作台';
}
