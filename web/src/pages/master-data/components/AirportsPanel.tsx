import { SendOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import {
  masterDataServiceCreateAirport,
  masterDataServiceListAirports,
  masterDataServiceUpdateAirport,
} from '@/services/roncin/masterDataService';
import type { PersistedMasterDataItem } from './masterDataMapper';

export interface AirportItem extends PersistedMasterDataItem {
  icaoCode: string;
  cityName: string;
  cityNameEn?: string;
  countryCode: string;
  countryName?: string;
}

const mapAirport = (item: API.Airport): AirportItem => {
  if (!item.id || !item.iataCode || !item.nameEn || !item.countryCode || item.enabled === undefined || item.source === undefined || item.sortOrder === undefined) {
    throw new Error('机场响应缺少必填字段');
  }
  return { id: item.id, code: item.iataCode, icaoCode: item.icaoCode ?? '', name: item.nameZh ?? '', nameEn: item.nameEn, cityName: item.cityNameZh ?? '', cityNameEn: item.cityNameEn, countryCode: item.countryCode, enabled: item.enabled, source: item.source, sortOrder: item.sortOrder, updatedAt: item.updatedAt };
};

export default function AirportsPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<AirportItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListAirports({ page: 1, pageSize: 100 });
      setData((response.data ?? []).map(mapAirport));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '机场主数据加载失败'));
  }, [fetchServerData, message]);

  const saveResponse = (response: API.AirportReply) => {
    if (!response.data) throw new Error('机场响应缺少数据');
    const saved = mapAirport(response.data);
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateAirport({
      iataCode: values.code.toUpperCase().trim(),
      nameZh: values.name?.trim() || undefined,
      nameEn: values.nameEn.trim(),
      icaoCode: values.icaoCode?.toUpperCase().trim() || undefined,
      cityNameZh: values.cityName?.trim() || undefined,
      cityNameEn: values.cityNameEn?.trim() || undefined,
      countryCode: values.countryCode.toUpperCase().trim(),
      sortOrder: 100,
    });
    saveResponse(response);
  };

  const updateItem = async (record: AirportItem, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateAirport(
      { id: record.id },
      {
        id: record.id,
        nameZh: values.name?.trim() || undefined,
        nameEn: values.nameEn.trim(),
        icaoCode: values.icaoCode?.toUpperCase().trim() || undefined,
        cityNameZh: values.cityName?.trim() || undefined,
        cityNameEn: values.cityNameEn?.trim() || undefined,
        countryCode: values.countryCode.toUpperCase().trim(),
        sortOrder: record.sortOrder,
        enabled,
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新机场不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: AirportItem) => {
    await updateItem(record, record, !record.enabled);
  };

  return (
    <MasterDataTemplate<AirportItem>
      title="空运机场管理"
      subtitle="维护全球主要货运/客运机场 IATA 三字码、ICAO 四字码与地理枢纽属性"
      icon={<SendOutlined />}
      codeLabel="IATA 三字码"
      items={data}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索三字码(如 PVG) / 四字码(如 ZSPD) / 机场中英文名..."
      extraStats={[
        { label: '国内枢纽', value: data.filter((a) => ['CN', 'HK', 'TW'].includes(a.countryCode)).length, color: '#1677ff' },
        { label: '国际枢纽', value: data.filter((a) => !['CN', 'HK', 'TW'].includes(a.countryCode)).length, color: '#722ed1' },
      ]}
      filterOptions={[
        {
          key: 'countryCode',
          label: '国家筛选',
          placeholder: '所属国家',
          options: [
            { label: '全部国家', value: 'all' },
            { label: '🇨🇳 中国 (CN)', value: 'CN' },
            { label: '🇭🇰 中国香港 (HK)', value: 'HK' },
            { label: '🇺🇸 美国 (US)', value: 'US' },
            { label: '🇩🇪 德国 (DE)', value: 'DE' },
            { label: '🇬🇧 英国 (GB)', value: 'GB' },
            { label: '🇯🇵 日本 (JP)', value: 'JP' },
            { label: '🇸🇬 新加坡 (SG)', value: 'SG' },
          ],
          width: 140,
        },
      ]}
      extraColumns={[
        {
          title: 'ICAO 四字码',
          dataIndex: 'icaoCode',
          key: 'icaoCode',
          width: 120,
          render: (icao: string) =>
            icao ? (
              <Tag style={{ fontFamily: 'monospace', fontWeight: 600, color: '#722ed1', backgroundColor: '#f9f0ff', borderColor: '#d3adf7', margin: 0 }}>
                {icao}
              </Tag>
            ) : (
              <span style={{ color: '#bfbfbf' }}>-</span>
            ),
        },
        {
          title: '所在城市',
          dataIndex: 'cityName',
          key: 'cityName',
          width: 100,
          render: (city: string) => <span style={{ fontWeight: 500 }}>{city || '-'}</span>,
        },
        {
          title: '所属国家',
          dataIndex: 'countryCode',
          key: 'countryCode',
          width: 100,
          render: (cc: string) => (
            <Tag color="cyan" style={{ fontFamily: 'monospace', fontWeight: 600 }}>
              {cc}
            </Tag>
          ),
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: 'IATA 三字码',
          placeholder: '例如：PVG、LAX (3位字母代码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入IATA三字码' },
            { pattern: /^[A-Za-z]{3}$/, message: '请输入标准的3位字母IATA代码' },
          ],
        },
        {
          name: 'icaoCode',
          label: 'ICAO 四字码',
          placeholder: '例如：ZSPD、KLAX (4位民航代码)',
          rules: [{ pattern: /^[A-Za-z]{4}$/, message: '请输入标准的4位字母ICAO代码' }],
        },
        {
          name: 'name',
          label: '中文机场名',
          placeholder: '选填，例如：上海浦东国际机场',
          required: false,
        },
        {
          name: 'nameEn',
          label: '英文机场名',
          placeholder: '例如：Shanghai Pudong International Airport',
          required: true,
        },
        {
          name: 'cityName',
          label: '所在城市中文名',
          placeholder: '选填，例如：上海、洛杉矶',
          required: false,
        },
        {
          name: 'cityNameEn',
          label: '所在城市英文名',
          placeholder: '例如：Shanghai、Los Angeles',
        },
        {
          name: 'countryCode',
          label: '所属国家代码',
          placeholder: '例如：CN、US (2位代码)',
          required: true,
          initialValue: 'CN',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
