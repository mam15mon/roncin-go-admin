import type { OrderListItem } from '@/components/ui/order-list-template/types';
import { OrderBusinessType } from '@/enums.generated';

export function getDocumentsActionLabel(businessType: number) {
  return businessType === OrderBusinessType.BUSINESS_TYPE_SE
    ? '海运单证'
    : businessType === OrderBusinessType.BUSINESS_TYPE_AE
      ? '航空运单'
      : undefined;
}

export function openOrderDocuments(
  businessType: number,
  fallbackOrderKind: string,
  item: OrderListItem,
  navigate: (path: string) => void,
  openLegacyDocuments: (record: API.Order) => void,
) {
  if (
    businessType === OrderBusinessType.BUSINESS_TYPE_SE ||
    businessType === OrderBusinessType.BUSINESS_TYPE_AE
  ) {
    const path = `/orders/${item.orderKind || fallbackOrderKind}/${item.id}`;
    navigate(
      businessType === OrderBusinessType.BUSINESS_TYPE_SE
        ? `${path}/hbl`
        : path,
    );
    return;
  }
  if (item.rawRecord) {
    openLegacyDocuments(item.rawRecord);
  }
}
