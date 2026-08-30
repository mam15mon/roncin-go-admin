import { CompassOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import { masterDataServiceListAdministrativeRegions } from '@/services/roncin/masterDataService';
import { unwrapList } from '@/utils/api';

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
    enabled: item.enabled ?? true,
    updatedAt: item.updatedAt,
  };
};

export default function CitiesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<RegionItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListAdministrativeRegions({});
      setData(unwrapList(response).map(mapRegion));
    } catch (error: any) {
      message.error(error.message || '行政区划数据加载失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void fetchServerData();
  }, [fetchServerData]);

  // 根据上级代码关联上级名称
  const namesByCode = useMemo(
    () => new Map(data.map((item) => [item.code, item.name])),
    [data],
  );

  const displayedData = useMemo(
    () =>
      data.map((item) => ({
        ...item,
        parentName: item.parentCode ? namesByCode.get(item.parentCode) : undefined,
      })),
    [data, namesByCode],
  );

  return (
    <MasterDataTemplate<RegionItem>
      title="城市与行政区划管理"
      subtitle="中国民政部 12 位标准行政区划代码、省市县区层级与货代基础地理数据"
      icon={<CompassOutlined />}
      codeLabel="12位行政区划代码"
      items={displayedData}
      loading={loading}
      onRefresh={fetchServerData}
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
                <Tag style={{ fontFamily: 'monospace', margin: 0, marginRight: 6 }}>
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
