import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';

export interface PersistedMasterDataItem extends BaseMasterDataItem {
  source: string;
  sortOrder: number;
}

export function mapPersistedMasterDataItem(
  item: API.MasterDataItem,
): PersistedMasterDataItem {
  if (!item.id || !item.code || !item.name || item.source === undefined) {
    throw new Error('主数据响应缺少必填字段');
  }
  return {
    id: item.id,
    code: item.code,
    name: item.name,
    nameEn: item.nameEn,
    // ProtoJSON 会省略非 optional 标量的零值，按契约还原停用状态和排序。
    enabled: item.enabled ?? false,
    source: item.source,
    sortOrder: item.sortOrder ?? 0,
    updatedAt: item.updatedAt,
  };
}

export function requireMasterDataResponse(
  response: API.CreateItemResponse | API.UpdateItemResponse,
): API.MasterDataItem {
  if (!response.data) {
    throw new Error('主数据响应缺少数据');
  }
  return response.data;
}
