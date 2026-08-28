import { App } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import type { BaseMasterDataItem } from './types';

export interface UseMasterDataCrudOptions<
  TItem extends BaseMasterDataItem,
  TApiItem = any,
> {
  entityName: string;
  fetchList: () => Promise<{ data?: TApiItem[] }>;
  mapItem: (apiItem: TApiItem) => TItem;
  createItem: (values: any) => Promise<{ data?: TApiItem }>;
  updateItem: (
    id: string,
    values: any,
    enabled: boolean,
    currentRecord: TItem,
  ) => Promise<{ data?: TApiItem }>;
}

export function useMasterDataCrud<
  TItem extends BaseMasterDataItem,
  TApiItem = any,
>({
  entityName,
  fetchList,
  mapItem,
  createItem,
  updateItem,
}: UseMasterDataCrudOptions<TItem, TApiItem>) {
  const { message } = App.useApp();
  const [data, setData] = useState<TItem[]>([]);
  const [loading, setLoading] = useState(false);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetchList();
      setData((response.data ?? []).map(mapItem));
    } catch (error: any) {
      message.error(error?.message || `${entityName}主数据加载失败`);
    } finally {
      setLoading(false);
    }
  }, [fetchList, mapItem, entityName, message]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const saveResponse = useCallback(
    (response: { data?: TApiItem }) => {
      if (!response.data) {
        throw new Error(`${entityName}响应缺少数据`);
      }
      const saved = mapItem(response.data);
      setData((current) => {
        const exists = current.some((item) => item.id === saved.id);
        return exists
          ? current.map((item) => (item.id === saved.id ? saved : item))
          : [saved, ...current];
      });
    },
    [entityName, mapItem],
  );

  const handleCreate = useCallback(
    async (values: any) => {
      const response = await createItem(values);
      saveResponse(response);
    },
    [createItem, saveResponse],
  );

  const handleUpdate = useCallback(
    async (id: string, values: any) => {
      const record = data.find((item) => item.id === id);
      if (!record) {
        throw new Error(`待更新${entityName}不存在`);
      }
      const response = await updateItem(id, values, record.enabled, record);
      saveResponse(response);
    },
    [data, entityName, updateItem, saveResponse],
  );

  const handleToggleActive = useCallback(
    async (record: TItem) => {
      const response = await updateItem(
        record.id,
        record,
        !record.enabled,
        record,
      );
      saveResponse(response);
    },
    [updateItem, saveResponse],
  );

  return {
    data,
    setData,
    loading,
    reload,
    handleCreate,
    handleUpdate,
    handleToggleActive,
  };
}
