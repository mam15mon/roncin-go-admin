import { CompassOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';
import {
  mapPersistedMasterDataItem,
  type PersistedMasterDataItem,
  requireMasterDataResponse,
} from './masterDataMapper';

export interface CityData extends PersistedMasterDataItem {
  level: number;
  parentCode?: string;
  parentName?: string;
}

const mapCity = (item: API.MasterDataItem): CityData => ({
  ...mapPersistedMasterDataItem(item),
  level: item.attributes?.regionLevel ?? 0,
  parentCode: item.parentCode,
});

export default function CitiesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<CityData[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListItems({ kind: 3, page: 1, pageSize: 100 });
      setData((response.data ?? []).map(mapCity));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '行政区划主数据加载失败'));
  }, [fetchServerData, message]);

  const namesByCode = useMemo(
    () => new Map(data.map((item) => [item.code, item.name])),
    [data],
  );
  const displayedData = useMemo(
    () => data.map((item) => ({ ...item, parentName: item.parentCode ? namesByCode.get(item.parentCode) : undefined })),
    [data, namesByCode],
  );

  const saveResponse = (response: API.MasterDataItemReply) => {
    const saved = mapCity(requireMasterDataResponse(response));
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateItem({
      kind: 3,
      code: values.code.trim(),
      name: values.name.trim(),
      nameEn: values.nameEn?.trim(),
      parentCode: values.parentCode?.trim(),
      source: 'manual',
      sortOrder: 100,
      attributes: { regionLevel: values.level },
    });
    saveResponse(response);
  };

  const updateItem = async (record: CityData, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateItem(
      { id: record.id },
      {
        id: record.id,
        kind: 3,
        name: values.name.trim(),
        nameEn: values.nameEn?.trim(),
        parentCode: values.parentCode?.trim(),
        source: record.source,
        sortOrder: record.sortOrder,
        enabled,
        attributes: { regionLevel: values.level },
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新行政区划不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: CityData) => {
    await updateItem(record, record, !record.enabled);
  };

  return (
    <MasterDataTemplate<CityData>
      title="城市与行政区划管理"
      subtitle="维护中国民政部 6 位标准行政区划代码、省市区县层级与货代中英文映射"
      icon={<CompassOutlined />}
      codeLabel="行政代码"
      items={displayedData}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索行政区划代码(如 310000) / 城市中英文名称..."
      extraStats={[
        { label: '省级行政区', value: data.filter((c) => c.level === 1 || c.code.endsWith('0000')).length, color: '#1677ff' },
        { label: '地级市与区县', value: data.filter((c) => c.level > 1 && !c.code.endsWith('0000')).length, color: '#722ed1' },
      ]}
      extraColumns={[
        {
          title: '区划层级',
          dataIndex: 'level',
          key: 'level',
          width: 110,
          render: (level: number) => {
            if (level === 1) return <Tag color="blue">省级 / 直辖</Tag>;
            if (level === 2) return <Tag color="cyan">地级市</Tag>;
            return <Tag color="default">区县</Tag>;
          },
        },
        {
          title: '所属上级代码',
          dataIndex: 'parentCode',
          key: 'parentCode',
          width: 140,
          render: (pCode: string) =>
            pCode ? (
              <Tag style={{ fontFamily: 'monospace', margin: 0 }}>{pCode}</Tag>
            ) : (
              <span style={{ color: '#bfbfbf' }}>-</span>
            ),
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: '行政区划代码',
          placeholder: '例如：330100 (6位标准行政代码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入行政区划代码' },
            { pattern: /^[0-9A-Za-z]+$/, message: '仅支持字母与数字' },
          ],
        },
        {
          name: 'name',
          label: '城市/区划名称',
          placeholder: '例如：杭州市、浦东新区',
          required: true,
        },
        {
          name: 'nameEn',
          label: '英文名称',
          placeholder: '例如：Hangzhou',
        },
        {
          name: 'level',
          label: '区划层级',
          type: 'select',
          required: true,
          initialValue: 1,
          options: [
            { label: '省级 / 直辖', value: 1 },
            { label: '地级市', value: 2 },
            { label: '区县', value: 3 },
          ],
        },
        {
          name: 'parentCode',
          label: '所属上级代码',
          placeholder: '例如：330000 (留空为省直辖)',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
