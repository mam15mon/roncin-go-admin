import { GlobalOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import React from 'react';
import { useAccess } from '@/app/access';
import { MasterDataTemplate } from '@/components/ui';
import { MasterDataKind } from '@/enums.generated';
import {
  masterDataServiceCreateItem,
  masterDataServiceListAllItems,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';
import { ensureListResponse, isWriteResponse, unwrapList } from '@/utils/api';
import {
  mapPersistedMasterDataItem,
  type PersistedMasterDataItem,
  requireMasterDataResponse,
} from './masterDataMapper';

type ShippingAgentFormValues = {
  code: string;
  name: string;
  nameEn?: string;
  sortOrder: number;
};

const kind = MasterDataKind.MASTER_DATA_KIND_SHIPPING_AGENT;
const queryKey = ['master-data', 'shipping-agents'];

export default function ShippingAgentsPanel() {
  const access = useAccess();
  const queryClient = useQueryClient();
  const canCreate = access.isSystemWorkspace && access.canCreateMasterDataItems;
  const canUpdate = access.isSystemWorkspace && access.canUpdateMasterDataItems;
  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await masterDataServiceListAllItems({ kind });
      if (response?.success === false) throw new Error('船代主数据加载失败');
      return unwrapList(
        ensureListResponse(response, '船代主数据加载失败，请稍后重试'),
      ).map(mapPersistedMasterDataItem);
    },
    meta: { errorMessage: '船代主数据加载失败' },
  });
  const createMutation = useMutation({
    mutationFn: (body: API.CreateItemRequest) =>
      masterDataServiceCreateItem(body),
    meta: { silent: true },
  });
  const updateMutation = useMutation({
    mutationFn: (body: API.UpdateItemRequest) =>
      masterDataServiceUpdateItem({ id: body.id }, body),
    meta: { silent: true },
  });
  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey });
  };
  const update = async (
    record: PersistedMasterDataItem,
    values: ShippingAgentFormValues,
    enabled: boolean,
  ) => {
    const response = await updateMutation.mutateAsync({
      id: record.id,
      kind,
      name: values.name.trim(),
      nameEn: values.nameEn?.trim() ?? '',
      source: record.source,
      sortOrder: values.sortOrder,
      enabled,
    });
    if (!isWriteResponse(response, true)) return false;
    return requireMasterDataResponse(response);
  };

  return (
    <MasterDataTemplate<PersistedMasterDataItem, ShippingAgentFormValues>
      title="船代管理"
      subtitle="统一维护海运订单使用的船代名称、编码、排序与启用状态"
      icon={<GlobalOutlined />}
      codeLabel="船代编码"
      items={query.data ?? []}
      loading={query.isFetching}
      onRefresh={refresh}
      searchPlaceholder="搜索船代编码 / 中文名称 / 英文名称..."
      extraColumns={[
        { title: '排序', dataIndex: 'sortOrder', key: 'sortOrder', width: 80 },
      ]}
      formFields={[
        {
          name: 'code',
          label: '船代编码',
          required: true,
          disabledOnEdit: true,
          placeholder: '请输入唯一船代编码',
        },
        { name: 'name', label: '中文名称', required: true },
        { name: 'nameEn', label: '英文名称' },
        {
          name: 'sortOrder',
          label: '排序',
          type: 'number',
          required: true,
          initialValue: 100,
        },
      ]}
      onCreate={
        canCreate
          ? async (values) => {
              const response = await createMutation.mutateAsync({
                kind,
                code: values.code.trim(),
                name: values.name.trim(),
                nameEn: values.nameEn?.trim() ?? '',
                source: 'manual',
                sortOrder: values.sortOrder,
              });
              if (!isWriteResponse(response, true)) return false;
              return requireMasterDataResponse(response);
            }
          : undefined
      }
      onUpdate={
        canUpdate
          ? async (id, values) => {
              const record = query.data?.find((item) => item.id === id);
              if (!record) throw new Error('船代记录不存在，请刷新后重试');
              return update(record, values, record.enabled);
            }
          : undefined
      }
      onToggleActive={
        canUpdate
          ? async (record) => {
              const response = await update(record, record, !record.enabled);
              if (response === false) return false;
              await refresh();
              return true;
            }
          : undefined
      }
      notice={
        access.isSystemWorkspace ? undefined : '由系统管理员统一维护与共享'
      }
    />
  );
}
