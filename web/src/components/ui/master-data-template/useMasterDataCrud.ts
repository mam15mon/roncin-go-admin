import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import { unwrapPage } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import type { BaseMasterDataItem, MasterDataListQuery } from './types';

/**
 * TFormValues 默认保留 any 的权衡：主数据面板的表单存在「表单选填但服务端
 * 契约标记必填」的存量缺口（如机场 cityNameZh、港口 nameZh），收紧为
 * Record<string, unknown> 会在这些提交点暴露契约不一致，需先对齐契约才能收紧；
 * 需要强类型的调用方可显式传入 TFormValues。
 */
export interface UseMasterDataCrudOptions<
  TItem extends BaseMasterDataItem,
  TApiItem = unknown,
  // biome-ignore lint/suspicious/noExplicitAny: 表单选填与服务端必填契约存在存量缺口，见上方注释
  TFormValues = any,
> {
  entityName: string;
  fetchList: (
    query: MasterDataListQuery,
  ) => Promise<{ data?: TApiItem[]; total?: number }>;
  mapItem: (apiItem: TApiItem) => TItem;
  createItem: (values: TFormValues) => Promise<{ data?: TApiItem }>;
  updateItem: (
    id: string,
    values: TFormValues,
    enabled: boolean,
    currentRecord: TItem,
  ) => Promise<{ data?: TApiItem }>;
}

/** 列表 query 缓存载荷：items 已映射为领域对象，total 为服务端分页总数。 */
interface MasterDataPage<TItem> {
  items: TItem[];
  total: number;
}

/** 启停统计 query 缓存载荷。 */
interface MasterDataStats {
  activeTotal: number;
  disabledTotal: number;
}

/**
 * 更新类 mutation 变量：与外部 updateItem 回调签名一一对应。values 取并集
 * 是原行为的精确转写——编辑时传表单值（TFormValues），启停切换时传整条
 * 记录（TItem，见原 handleToggleActive 的 updateItemRef.current 调用）。
 */
interface UpdateMutationVariables<TItem, TFormValues> {
  id: string;
  values: TFormValues | TItem;
  enabled: boolean;
  record: TItem;
}

/**
 * 主数据通用 CRUD 数据层（React Query 唯一模式，规范见
 * .trellis/spec/web/frontend/state-management.md）。
 *
 * 原「useState + useEffect + ref 稳定外部回调」实现的迁移说明：
 * - 用户 09-19 修复过的「内联回调引用变化引发 useEffect 无限重取」在
 *   queryKey 模式下天然消失：外部回调不参与缓存键，换参（query 状态）是
 *   唯一驱动重查的输入；禁止把函数引用放进 queryKey。
 * - 列表与启停统计各一条 query；写操作 useMutation 成功后按域前缀
 *   invalidateQueries，等价替代原 saveResponse 合并 + reload + reloadStats。
 */
export function useMasterDataCrud<
  TItem extends BaseMasterDataItem,
  TApiItem = unknown,
  // biome-ignore lint/suspicious/noExplicitAny: 表单选填与服务端必填契约存在存量缺口，见上方注释
  TFormValues = any,
>({
  entityName,
  fetchList,
  mapItem,
  createItem,
  updateItem,
}: UseMasterDataCrudOptions<TItem, TApiItem>) {
  const queryClient = useQueryClient();
  const [query, setQuery] = useState<MasterDataListQuery>({
    page: 1,
    pageSize: 10,
  });

  // 列表 query：翻页/搜索期间保留旧内容（keepPreviousData），错误文案按
  // 动态文案惯例在 queryFn 内包装 Error（服务端明细透出，兜底
  // 「XX主数据加载失败」），由全局 onError 展示，与原 reload catch 的
  // message.error 一致。
  const listQuery = useQuery({
    queryKey: ['master-data', entityName, query],
    queryFn: async (): Promise<MasterDataPage<TItem>> => {
      try {
        const page = unwrapPage(await fetchList(query));
        return {
          items: page.data.map((item) => mapItem(item)),
          total: page.total,
        };
      } catch (error) {
        throw new Error(getErrorMessage(error, `${entityName}主数据加载失败`));
      }
    },
    placeholderData: keepPreviousData,
  });

  // 启停统计 query：保持原 reloadStats 的 Promise.all 形态（enabled
  // true/false 两次并行）；原实现失败会提示「XX统计加载失败」（非静默），
  // 同样以包装 Error 交给全局 onError。
  const statsQuery = useQuery({
    queryKey: ['master-data', entityName, 'stats'],
    queryFn: async (): Promise<MasterDataStats> => {
      try {
        const [activeResponse, disabledResponse] = await Promise.all([
          fetchList({ page: 1, pageSize: 1, enabled: true }),
          fetchList({ page: 1, pageSize: 1, enabled: false }),
        ]);
        return {
          activeTotal: activeResponse.total ?? 0,
          disabledTotal: disabledResponse.total ?? 0,
        };
      } catch (error) {
        throw new Error(getErrorMessage(error, `${entityName}统计加载失败`));
      }
    },
  });

  // 写操作成功后按域前缀失效：列表与统计一起重查（原 reload + reloadStats）。
  // invalidateQueries 的重查失败不会拒绝本 promise（错误进 query 状态并由
  // 全局 onError 提示），与原 reload 内部 catch 吞掉刷新失败保持一致。
  const invalidateMasterData = useCallback(
    () =>
      queryClient.invalidateQueries({ queryKey: ['master-data', entityName] }),
    [queryClient, entityName],
  );

  // 原 saveResponse 的契约校验保留在 mutationFn 内：响应缺数据 / mapItem
  // 抛错仍在写链路中以失败语义透出（调用方模板统一 catch 后提示）；本地
  // 合并被失效重查覆盖，无需再写。
  // mutation meta.silent：原 hook 从不直接弹错（错误交调用方模板 catch 展示
  // 「操作失败」/「状态切换失败」），silent 避免全局 onError 二次弹窗。
  const createMutation = useMutation({
    mutationFn: async (values: TFormValues) => {
      const response = await createItem(values);
      if (!response.data) {
        throw new Error(`${entityName}响应缺少数据`);
      }
      mapItem(response.data);
      await invalidateMasterData();
      return response.data;
    },
    meta: { silent: true },
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      id,
      values,
      enabled,
      record,
    }: UpdateMutationVariables<TItem, TFormValues>) => {
      const response = await updateItem(id, values, enabled, record);
      if (!response.data) {
        throw new Error(`${entityName}响应缺少数据`);
      }
      mapItem(response.data);
      await invalidateMasterData();
      return response.data;
    },
    meta: { silent: true },
  });

  const data = listQuery.data?.items ?? [];

  // reload：包装为 Promise<void>，保持与原 reload 一致的外部签名
  // （模板 onRefresh?: () => Promise<void> | void，避免 refetch 结果类型外泄）。
  const reload = useCallback(async () => {
    await listQuery.refetch();
  }, [listQuery]);

  // setData：setQueryData 薄封装，保留 setState 的值 / 函数式更新签名。
  const setData = useCallback(
    (next: TItem[] | ((current: TItem[]) => TItem[])) => {
      queryClient.setQueryData<MasterDataPage<TItem>>(
        ['master-data', entityName, query],
        (previous) => {
          const items =
            typeof next === 'function' ? next(previous?.items ?? []) : next;
          return { items, total: previous?.total ?? 0 };
        },
      );
    },
    [queryClient, entityName, query],
  );

  const handleCreate = useCallback(
    async (values: TFormValues) => {
      await createMutation.mutateAsync(values);
    },
    [createMutation],
  );

  const handleUpdate = useCallback(
    async (id: string, values: TFormValues) => {
      const record = data.find((item) => item.id === id);
      if (!record) {
        throw new Error(`待更新${entityName}不存在`);
      }
      await updateMutation.mutateAsync({
        id,
        values,
        enabled: record.enabled,
        record,
      });
    },
    [data, entityName, updateMutation],
  );

  const handleToggleActive = useCallback(
    async (target: TItem) => {
      await updateMutation.mutateAsync({
        id: target.id,
        values: target,
        enabled: !target.enabled,
        record: target,
      });
    },
    [updateMutation],
  );

  return {
    data,
    setData,
    // isFetching 覆盖首次加载与重查，与原 loading（reload 前后置位）一致。
    loading: listQuery.isFetching,
    total: listQuery.data?.total ?? 0,
    activeTotal: statsQuery.data?.activeTotal ?? 0,
    disabledTotal: statsQuery.data?.disabledTotal ?? 0,
    query,
    setQuery,
    reload,
    handleCreate,
    handleUpdate,
    handleToggleActive,
  };
}
