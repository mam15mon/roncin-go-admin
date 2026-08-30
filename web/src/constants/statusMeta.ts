import { createElement } from 'react';
import { Tag, type TagProps } from 'antd';
import {
  AdminUserStatus,
  BackgroundTaskStatus,
  OrderAbnormalCaseStatus,
  OrderBusinessType,
  OrderFeeStatus,
  OrderReleasePodStatus,
  PartnerContractStatus,
} from '@/enums.generated';

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

export const adminUserStatusMeta: Record<number, StatusMeta> = {
  [AdminUserStatus.ADMIN_USER_STATUS_ACTIVE]: {
    text: '在职',
    color: 'success',
  },
  [AdminUserStatus.ADMIN_USER_STATUS_PENDING_AUTHORIZATION]: {
    text: '待授权',
    color: 'warning',
  },
  [AdminUserStatus.ADMIN_USER_STATUS_TERMINATED]: {
    text: '已离职',
    color: 'default',
  },
  [AdminUserStatus.ADMIN_USER_STATUS_REMOVED_FROM_ORGANIZATION]: {
    text: '已移出本组织',
    color: 'default',
  },
  [AdminUserStatus.ADMIN_USER_STATUS_DISABLED]: {
    text: '已停用',
    color: 'default',
  },
};

export const backgroundTaskStatusMeta: Record<number, StatusMeta> = {
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_PENDING]: {
    text: '待执行',
    color: 'default',
  },
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_RUNNING]: {
    text: '执行中',
    color: 'processing',
  },
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_SUCCEEDED]: {
    text: '执行成功',
    color: 'success',
  },
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_FAILED]: {
    text: '等待重试',
    color: 'warning',
  },
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_DEAD_LETTER]: {
    text: '已停止',
    color: 'error',
  },
};

export const partnerContractStatusMeta: Record<number, StatusMeta> = {
  [PartnerContractStatus.PARTNER_CONTRACT_STATUS_PENDING]: {
    text: '待生效',
    color: 'processing',
  },
  [PartnerContractStatus.PARTNER_CONTRACT_STATUS_ACTIVE]: {
    text: '生效中',
    color: 'success',
  },
  [PartnerContractStatus.PARTNER_CONTRACT_STATUS_EXPIRED]: {
    text: '已到期',
  },
  [PartnerContractStatus.PARTNER_CONTRACT_STATUS_TERMINATED]: {
    text: '已终止',
    color: 'error',
  },
};

export const orderAbnormalCaseStatusMeta: Record<number, StatusMeta> = {
  [OrderAbnormalCaseStatus.ORDER_ABNORMAL_CASE_STATUS_ACTIVE]: {
    text: '处理中',
    color: 'error',
  },
  [OrderAbnormalCaseStatus.ORDER_ABNORMAL_CASE_STATUS_RESOLVED]: {
    text: '已解决',
    color: 'success',
  },
};

export const orderReleasePodStatusMeta: Record<number, StatusMeta> = {
  [OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_PENDING]: {
    text: '待签收',
    color: 'default',
  },
  [OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_SIGNED]: {
    text: '已签收',
    color: 'processing',
  },
  [OrderReleasePodStatus.ORDER_RELEASE_POD_STATUS_RETURNED]: {
    text: '已回单',
    color: 'success',
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
