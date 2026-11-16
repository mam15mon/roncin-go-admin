import { CompassOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Alert, Button, Tag } from 'antd';
import React, { useMemo } from 'react';
import { useAccess } from '@/app/access';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import { masterDataServiceListAdministrativeRegions } from '@/services/roncin/masterDataService';
import { ensureListResponse, unwrapList } from '@/utils/api';

export interface RegionItem extends BaseMasterDataItem {
  level: number;
  parentCode?: string;
  parentName?: string;
  regionType?: string;
  sourceVersion?: string;
}

const mapRegion = (item: API.AdministrativeRegion): RegionItem => {
  const code = item.code ?? '';
  const name = item.name ?? '';
  const level = Number(item.level) || 1;
  return {
    id: item.id || code,
    code,
    name,
    level,
    parentCode: item.parentCode || undefined,
    regionType: item.regionType || undefined,
    sourceVersion: item.sourceVersion || undefined,
    enabled: Boolean(item.enabled),
    updatedAt: item.updatedAt,
  };
};

export default function CitiesPanel() {
  const access = useAccess();
  const query = useQuery({
    queryKey: ['master-data', 'administrative-regions', {}],
    queryFn: async ({ signal }) =>
      unwrapList(
        ensureListResponse(
          await masterDataServiceListAdministrativeRegions(
            {},
            { signal, skipErrorHandler: true },
          ),
          '行政区划数据加载失败',
        ),
      ).map(mapRegion),
    meta: { silent: true },
  });
  const data = query.data ?? [];
  const fetchServerData = async () => {
    await query.refetch();
  };

  // 根据上级代码关联上级名称
  const namesByCode = useMemo(
    () => new Map(data.map((item) => [item.code, item.name])),
    [data],
  );

  const displayedData = useMemo(
    () =>
      data.map((item) => ({
        ...item,
        parentName: item.parentCode
          ? namesByCode.get(item.parentCode)
          : undefined,
      })),
    [data, namesByCode],
  );

  if (query.isError && !query.data) {
    return (
      <Alert
        type="error"
        showIcon
        title="行政区划数据加载失败"
        description={query.error.message}
        action={<Button onClick={() => void query.refetch()}>重试</Button>}
      />
    );
  }

  return (
    <MasterDataTemplate<RegionItem>
      title="城市与行政区划管理"
      subtitle="中国民政部 12 位标准行政区划代码、省市县区层级与货代基础地理数据"
      icon={<CompassOutlined />}
      codeLabel="12位行政区划代码"
      items={displayedData}
      loading={query.isFetching}
      onRefresh={fetchServerData}
      notice={
        query.isError
          ? '行政区划刷新失败，请重试。当前显示上次读取的内容。'
          : access.isSystemWorkspace
            ? undefined
            : '由系统管理员统一维护与共享'
      }
      searchPlaceholder="搜索12位区划代码(如 310115000000) / 城市区划名称..."
      extraStats={[
        {
          label: '省级行政区',
          value: data.filter((c) => c.level === 1).length,
          color: '#1677ff',
        },
        {
          label: '地级市',
          value: data.filter((c) => c.level === 2).length,
          color: '#13c2c2',
        },
        {
          label: '区县级',
          value: data.filter((c) => c.level === 3).length,
          color: '#722ed1',
        },
      ]}
      filterOptions={[
        {
          key: 'level',
          label: '区划层级',
          placeholder: '全部层级',
          options: [
            { label: '全部层级', value: 'all' },
            { label: '省级 / 直辖 (1级)', value: 1 },
            { label: '地级市 (2级)', value: 2 },
            { label: '区县级 (3级)', value: 3 },
          ],
          width: 150,
        },
      ]}
      extraColumns={[
        {
          title: '区划层级',
          dataIndex: 'level',
          key: 'level',
          width: 120,
          render: (level: number) => {
            if (level === 1) return <Tag color="blue">省级 / 直辖</Tag>;
            if (level === 2) return <Tag color="cyan">地级市</Tag>;
            return <Tag color="default">区县级</Tag>;
          },
        },
        {
          title: '所属上级区划',
          dataIndex: 'parentCode',
          key: 'parentCode',
          width: 220,
          render: (pCode: string, record: RegionItem) =>
            pCode ? (
              <span>
                <Tag
                  style={{ fontFamily: 'monospace', margin: 0, marginRight: 6 }}
                >
                  {pCode}
                </Tag>
                {record.parentName && (
                  <span style={{ fontSize: 12, color: '#595959' }}>
                    {record.parentName}
                  </span>
                )}
              </span>
            ) : (
              <span style={{ color: '#bfbfbf' }}>-</span>
            ),
        },
      ]}
      formFields={[]}
    />
  );
}
