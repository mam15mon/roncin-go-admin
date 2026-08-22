import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';

export interface PersistedMasterDataItem extends BaseMasterDataItem {
  source: string;
  sortOrder: number;
}

export function mapPersistedMasterDataItem(
  item: API.MasterDataItem,
): PersistedMasterDataItem {
  if (
    !item.id ||
    !item.code ||
    !item.name ||
    item.enabled === undefined ||
    item.source === undefined ||
    item.sortOrder === undefined
  ) {
    throw new Error('主数据响应缺少必填字段');
  }
  return {
    id: item.id,
    code: item.code,
    name: item.name,
    nameEn: item.nameEn,
    enabled: item.enabled,
    source: item.source,
    sortOrder: item.sortOrder,
    updatedAt: item.updatedAt,
  };
}

export function requireMasterDataResponse(
  response: API.MasterDataItemReply,
): API.MasterDataItem {
  if (!response.data) {
    throw new Error('主数据响应缺少数据');
  }
  return response.data;
}
