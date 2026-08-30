import { createElement } from 'react';
import { Tag, type TagProps } from 'antd';
import { OrderBusinessType, OrderFeeStatus } from '@/enums.generated';

export type StatusMeta = {
  text: string;
  color?: TagProps['color'];
};

export const businessTypeMeta: Record<number, StatusMeta> = {
  [OrderBusinessType.BUSINESS_TYPE_SE]: { text: '海运出口' },
  [OrderBusinessType.BUSINESS_TYPE_SI]: { text: '海运进口' },
  [OrderBusinessType.BUSINESS_TYPE_AE]: { text: '空运出口' },
  [OrderBusinessType.BUSINESS_TYPE_AI]: { text: '空运进口' },
  [OrderBusinessType.BUSINESS_TYPE_LAND]: { text: '陆运' },
  [OrderBusinessType.BUSINESS_TYPE_RAIL]: { text: '铁路' },
};

export const orderFeeStatusMeta: Record<number, StatusMeta> = {
  [OrderFeeStatus.ORDER_FEE_STATUS_DRAFT]: { text: '草稿', color: 'gold' },
  [OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED]: {
    text: '已确认',
    color: 'green',
  },
  [OrderFeeStatus.ORDER_FEE_STATUS_BILLED]: {
    text: '已进账单',
    color: 'blue',
  },
  [OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED]: {
    text: '已作废',
    color: 'default',
  },
};

const businessTypeCodes: Record<string, number> = {
  SE: OrderBusinessType.BUSINESS_TYPE_SE,
  SI: OrderBusinessType.BUSINESS_TYPE_SI,
  AE: OrderBusinessType.BUSINESS_TYPE_AE,
  AI: OrderBusinessType.BUSINESS_TYPE_AI,
  LAND: OrderBusinessType.BUSINESS_TYPE_LAND,
  RAIL: OrderBusinessType.BUSINESS_TYPE_RAIL,
  BUSINESS_TYPE_SE: OrderBusinessType.BUSINESS_TYPE_SE,
  BUSINESS_TYPE_SI: OrderBusinessType.BUSINESS_TYPE_SI,
  BUSINESS_TYPE_AE: OrderBusinessType.BUSINESS_TYPE_AE,
  BUSINESS_TYPE_AI: OrderBusinessType.BUSINESS_TYPE_AI,
  BUSINESS_TYPE_LAND: OrderBusinessType.BUSINESS_TYPE_LAND,
  BUSINESS_TYPE_RAIL: OrderBusinessType.BUSINESS_TYPE_RAIL,
};

const orderFeeStatusCodes: Record<string, number> = {
  DRAFT: OrderFeeStatus.ORDER_FEE_STATUS_DRAFT,
  CONFIRMED: OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED,
  BILLED: OrderFeeStatus.ORDER_FEE_STATUS_BILLED,
  CANCELLED: OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED,
  ORDER_FEE_STATUS_DRAFT: OrderFeeStatus.ORDER_FEE_STATUS_DRAFT,
  ORDER_FEE_STATUS_CONFIRMED: OrderFeeStatus.ORDER_FEE_STATUS_CONFIRMED,
  ORDER_FEE_STATUS_BILLED: OrderFeeStatus.ORDER_FEE_STATUS_BILLED,
  ORDER_FEE_STATUS_CANCELLED: OrderFeeStatus.ORDER_FEE_STATUS_CANCELLED,
};

function normalizeCode(value: unknown, codes: Record<string, number>): number {
  if (typeof value === 'number') return value;
  const text = String(value ?? '').trim().toUpperCase();
  if (text === '') return 0;
  if (/^\d+$/.test(text)) return Number(text);
  return codes[text] ?? 0;
}

export function normalizeBusinessType(value: unknown): number {
  return normalizeCode(value, businessTypeCodes);
}

export function normalizeOrderFeeStatus(value: unknown): number {
  return normalizeCode(value, orderFeeStatusCodes);
}

export function makeValueEnum(
  meta: Record<string | number, StatusMeta>,
): Record<string, { text: string }> {
  return Object.fromEntries(
    Object.entries(meta).map(([key, item]) => [key, { text: item.text }]),
  );
}

export function statusText(
  meta: Record<string | number, StatusMeta>,
  value: string | number,
  fallback = '-',
): string {
  return meta[value]?.text ?? fallback;
}

export function statusTag(
  meta: Record<string | number, StatusMeta>,
  value: string | number,
  fallback = '-',
) {
  const item = meta[value];
  return createElement(
    Tag,
    { color: item?.color, style: { margin: 0 } },
    item?.text ?? fallback,
  );
}
