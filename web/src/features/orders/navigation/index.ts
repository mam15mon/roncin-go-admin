import { normalizeBusinessType } from '@/constants/statusMeta';
import { OrderBusinessType } from '@/enums.generated';

/** 已开放订单的业务身份与路由关系；页面注册、工作台和通知共用。 */
export const ORDER_BUSINESS_ROUTES = {
  [OrderBusinessType.BUSINESS_TYPE_SE]: 'sea-export',
  [OrderBusinessType.BUSINESS_TYPE_SI]: 'sea-import',
  [OrderBusinessType.BUSINESS_TYPE_AE]: 'air-export',
  [OrderBusinessType.BUSINESS_TYPE_AI]: 'air-import',
} as const;

export type OrderRouteKind =
  (typeof ORDER_BUSINESS_ROUTES)[keyof typeof ORDER_BUSINESS_ROUTES];

export function orderListPath(businessType: unknown): string | undefined {
  const type = normalizeBusinessType(businessType);
  const kind = Object.hasOwn(ORDER_BUSINESS_ROUTES, type)
    ? ORDER_BUSINESS_ROUTES[type as keyof typeof ORDER_BUSINESS_ROUTES]
    : undefined;
  return kind ? `/orders/${kind}` : undefined;
}

export function orderDetailPath(
  businessType: unknown,
  orderId?: string,
  fees = false,
): string | undefined {
  const list = orderListPath(businessType);
  if (!list || !orderId) return undefined;
  return `${list}/${encodeURIComponent(orderId)}${fees ? '/fees' : ''}`;
}
