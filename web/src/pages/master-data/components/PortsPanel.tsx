import { CompassOutlined } from '@ant-design/icons';
import { App, Space, Tag } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
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

export interface PortItem extends PersistedMasterDataItem {
  countryCode: string;
  countryName?: string;
  modes: string[];
  isBorder?: boolean;
}

const MODE_LABELS: Record<string, { label: string; color: string }> = {
  PORT: { label: '海港', color: 'blue' },
  RAIL: { label: '铁路枢纽', color: 'volcano' },
  ROAD: { label: '公路集散', color: 'green' },
  AIRPORT: { label: '空运联运', color: 'purple' },
};

const mapPort = (item: API.MasterDataItem): PortItem => ({
  ...mapPersistedMasterDataItem(item),
  countryCode: item.attributes?.countryCode ?? '',
  modes: item.attributes?.modes ?? [],
  isBorder: item.attributes?.isBorder,
});

export default function PortsPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<PortItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListItems({ kind: 4, page: 1, pageSize: 100 });
      setData((response.data ?? []).map(mapPort));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '港口主数据加载失败'));
  }, [fetchServerData, message]);

  const saveResponse = (response: API.MasterDataItemReply) => {
    const saved = mapPort(requireMasterDataResponse(response));
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateItem({
      kind: 4,
      code: values.code.toUpperCase().trim(),
      name: values.name.trim(),
      nameEn: values.nameEn.trim().toUpperCase(),
      source: 'manual',
      sortOrder: 100,
      attributes: {
        countryCode: values.countryCode.toUpperCase().trim(),
        modes: values.modes ?? [],
      },
    });
    saveResponse(response);
  };

  const updateItem = async (record: PortItem, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateItem(
      { id: record.id },
      {
        id: record.id,
        kind: 4,
        name: values.name.trim(),
        nameEn: values.nameEn.trim().toUpperCase(),
        source: record.source,
        sortOrder: record.sortOrder,
        enabled,
        attributes: {
          countryCode: values.countryCode.toUpperCase().trim(),
          modes: values.modes ?? [],
          isBorder: record.isBorder,
        },
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新港口不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: PortItem) => {
    await updateItem(record, record, !record.enabled);
  };

  return (
    <MasterDataTemplate<PortItem>
      title="海运港口管理"
      subtitle="维护全球海运港口 UN/LOCODE 五字码及运输枢纽属性"
      icon={<CompassOutlined />}
      codeLabel="UN/LOCODE 五字码"
      items={data}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索五字码(如 CNSHG) / 港口中英文名..."
      extraStats={[
        { label: '海港枢纽', value: data.filter((p) => p.modes.includes('PORT')).length, color: '#1677ff' },
        { label: '多式联运港', value: data.filter((p) => p.modes.length > 1).length, color: '#722ed1' },
      ]}
      filterOptions={[
        {
          key: 'countryCode',
          label: '国家筛选',
          placeholder: '所属国家',
          options: [
            { label: '全部国家', value: 'all' },
            { label: '🇨🇳 中国 (CN)', value: 'CN' },
            { label: '🇺🇸 美国 (US)', value: 'US' },
            { label: '🇳🇱 荷兰 (NL)', value: 'NL' },
            { label: '🇩🇪 德国 (DE)', value: 'DE' },
            { label: '🇸🇬 新加坡 (SG)', value: 'SG' },
            { label: '🇯🇵 日本 (JP)', value: 'JP' },
            { label: '🇦🇪 阿联酋 (AE)', value: 'AE' },
          ],
          width: 140,
        },
      ]}
      extraColumns={[
        {
          title: '所属国家',
          dataIndex: 'countryCode',
          key: 'countryCode',
          width: 110,
          render: (cc: string) => (
            <Tag color="cyan" style={{ fontFamily: 'monospace', fontWeight: 600 }}>
              {cc}
            </Tag>
          ),
        },
        {
          title: '枢纽运输类型',
          dataIndex: 'modes',
          key: 'modes',
          width: 200,
          render: (modes: string[] = []) => (
            <Space size={4} wrap>
              {modes.map((m) => {
                const conf = MODE_LABELS[m] || { label: m, color: 'default' };
                return (
                  <Tag key={m} color={conf.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
                    {conf.label}
                  </Tag>
                );
              })}
            </Space>
          ),
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: 'UN/LOCODE 五字码',
          placeholder: '例如：CNSHG、USLAX (5位标准码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入五字码' },
            { pattern: /^[A-Za-z]{5}$/, message: '请输入标准的5位字母UN/LOCODE' },
          ],
        },
        {
          name: 'name',
          label: '中文港口名',
          placeholder: '例如：上海港、洛杉矶港',
          required: true,
        },
        {
          name: 'nameEn',
          label: '英文港口名',
          placeholder: '例如：SHANGHAI, LOS ANGELES',
          required: true,
        },
        {
          name: 'countryCode',
          label: '所属国家代码',
          placeholder: '例如：CN、US、NL (2位ISO代码)',
          required: true,
          initialValue: 'CN',
        },
        {
          name: 'modes',
          label: '枢纽类型',
          type: 'checkboxGroup',
          initialValue: ['PORT'],
          options: [
            { label: '海港 (PORT)', value: 'PORT' },
            { label: '铁路枢纽 (RAIL)', value: 'RAIL' },
            { label: '公路集散 (ROAD)', value: 'ROAD' },
            { label: '空运联运 (AIRPORT)', value: 'AIRPORT' },
          ],
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
