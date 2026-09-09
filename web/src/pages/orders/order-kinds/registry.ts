import { seaExportDefinition } from './sea-export/definition';
import type { OrderKind, OrderKindDefinition } from './types';

/**
 * 订单类型唯一注册表：页面、列表查询与数据 Hook 只从这里取得类型定义。
 * 未注册类型一律 fail-closed，不存在默认 Sea/Air 兜底。
 */
export const ORDER_KIND_REGISTRY = {
  'sea-export': seaExportDefinition,
} as const satisfies Record<OrderKind, OrderKindDefinition>;

function lookupRegistryKey(key: string): OrderKindDefinition | undefined {
  return Object.hasOwn(ORDER_KIND_REGISTRY, key)
    ? ORDER_KIND_REGISTRY[key as OrderKind]
    : undefined;
}

/**
 * 解析订单类型：接受直接 kind（如 `sea-export`）或订单业务路径
 * （如 `/orders/sea-export/new`）；空值、未知或未注册类型返回 `undefined`。
 */
export function getOrderKindDefinition(
  kindOrPath?: string,
): OrderKindDefinition | undefined {
  if (!kindOrPath) return undefined;
  const direct = lookupRegistryKey(kindOrPath);
  if (direct) return direct;
  const match = kindOrPath.match(/\/orders\/([^/]+)/);
  if (!match) return undefined;
  return lookupRegistryKey(match[1]);
}

/**
 * 按业务枚举反查注册定义：只遍历已注册类型，未注册的
 * SI/AE/AI/LAND/RAIL 返回 `undefined`，调用方不得为其生成业务链接。
 */
export function getOrderKindDefinitionByBusinessType(
  businessType: number,
): OrderKindDefinition | undefined {
  for (const definition of Object.values(ORDER_KIND_REGISTRY)) {
    if (definition.businessType === businessType) {
      return definition;
    }
  }
  return undefined;
}
