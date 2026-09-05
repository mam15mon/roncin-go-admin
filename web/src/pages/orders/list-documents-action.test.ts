import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import type { OrderListItem } from '@/components/ui/order-list-template/types';
import {
  getDocumentsActionLabel,
  openOrderDocuments,
} from './list-documents-action';

describe('订单列表单证入口', () => {
  const item = {
    id: 'order-1',
    orderNo: 'SE-001',
    orderKind: 'sea-export',
    rawRecord: { id: 'order-1' } as API.Order,
  } satisfies OrderListItem;

  it('海运出口进入真实订单详情且不打开旧分单抽屉', () => {
    const navigate = vi.fn();
    const openLegacyDocuments = vi.fn();

    expect(getDocumentsActionLabel(OrderBusinessType.BUSINESS_TYPE_SE)).toBe(
      '海运单证',
    );
    openOrderDocuments(
      OrderBusinessType.BUSINESS_TYPE_SE,
      'sea-export',
      item,
      navigate,
      openLegacyDocuments,
    );

    expect(navigate).toHaveBeenCalledWith('/orders/sea-export/order-1');
    expect(openLegacyDocuments).not.toHaveBeenCalled();
  });

  it('非海运出口保持旧分单抽屉行为', () => {
    const navigate = vi.fn();
    const openLegacyDocuments = vi.fn();

    expect(getDocumentsActionLabel(OrderBusinessType.BUSINESS_TYPE_AE)).toBeUndefined();
    openOrderDocuments(
      OrderBusinessType.BUSINESS_TYPE_AE,
      'air-export',
      { ...item, orderKind: 'air-export' },
      navigate,
      openLegacyDocuments,
    );

    expect(openLegacyDocuments).toHaveBeenCalledWith(item.rawRecord);
    expect(navigate).not.toHaveBeenCalled();
  });
});
