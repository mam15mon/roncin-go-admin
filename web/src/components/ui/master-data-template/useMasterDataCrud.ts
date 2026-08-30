import { App } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { unwrapPage } from '@/utils/api';
import type { BaseMasterDataItem, MasterDataListQuery } from './types';

export interface UseMasterDataCrudOptions<
  TItem extends BaseMasterDataItem,
  TApiItem = any,
> {
  entityName: string;
  fetchList: (query: MasterDataListQuery) => Promise<{ data?: TApiItem[]; total?: number }>;
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
  const [total, setTotal] = useState(0);
  const [activeTotal, setActiveTotal] = useState(0);
  const [disabledTotal, setDisabledTotal] = useState(0);
  const [query, setQuery] = useState<MasterDataListQuery>({ page: 1, pageSize: 10 });

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetchList(query);
      const page = unwrapPage(response);
      setData(page.data.map(mapItem));
      setTotal(page.total);
    } catch (error: any) {
      message.error(error?.message || `${entityName}主数据加载失败`);
    } finally {
      setLoading(false);
    }
  }, [fetchList, mapItem, entityName, message, query]);

  const reloadStats = useCallback(async () => {
    try {
      const [activeResponse, disabledResponse] = await Promise.all([
        fetchList({ page: 1, pageSize: 1, enabled: true }),
        fetchList({ page: 1, pageSize: 1, enabled: false }),
      ]);
      setActiveTotal(activeResponse.total ?? 0);
      setDisabledTotal(disabledResponse.total ?? 0);
    } catch (error: any) {
      message.error(error?.message || `${entityName}统计加载失败`);
    }
  }, [entityName, fetchList, message]);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    void reloadStats();
  }, [reloadStats]);

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
      await Promise.all([reload(), reloadStats()]);
    },
    [createItem, reload, reloadStats, saveResponse],
  );

  const handleUpdate = useCallback(
    async (id: string, values: any) => {
      const record = data.find((item) => item.id === id);
      if (!record) {
        throw new Error(`待更新${entityName}不存在`);
      }
      const response = await updateItem(id, values, record.enabled, record);
      saveResponse(response);
      await Promise.all([reload(), reloadStats()]);
    },
    [data, entityName, reload, reloadStats, updateItem, saveResponse],
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
      await Promise.all([reload(), reloadStats()]);
    },
    [reload, reloadStats, updateItem, saveResponse],
  );

  return {
    data,
    setData,
    loading,
    total,
    activeTotal,
    disabledTotal,
    query,
    setQuery,
    reload,
    handleCreate,
    handleUpdate,
    handleToggleActive,
  };
}
