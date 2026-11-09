import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';

export interface PersistedMasterDataItem extends BaseMasterDataItem {
  source: string;
  sortOrder: number;
}

type PersistedMasterDataFields = Pick<
  PersistedMasterDataItem,
  'id' | 'source' | 'enabled' | 'sortOrder' | 'updatedAt'
>;

/** 只适配各主数据实体共有且已核对 Proto presence 的持久化字段。 */
export function mapPersistedMasterDataFields(
  item: Pick<API.MasterDataItem, keyof PersistedMasterDataFields>,
  entityName = '主数据',
): PersistedMasterDataFields {
  if (!item.id || item.source === undefined) {
    throw new Error(`${entityName}响应缺少必填字段`);
  }
  return {
    id: item.id,
    source: item.source,
    // ProtoJSON 会省略非 optional 标量的零值；业务必填与 optional 字段不在这里补值。
    enabled: item.enabled ?? false,
    sortOrder: item.sortOrder ?? 0,
    updatedAt: item.updatedAt,
  };
}

export function mapPersistedMasterDataItem(
  item: API.MasterDataItem,
): PersistedMasterDataItem {
  if (!item.code || !item.name) {
    throw new Error('主数据响应缺少必填字段');
  }
  return {
    ...mapPersistedMasterDataFields(item),
    code: item.code,
    name: item.name,
    nameEn: item.nameEn,
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
