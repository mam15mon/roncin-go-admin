import { RocketOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import {
  masterDataServiceCreateAirline,
  masterDataServiceListAirlines,
  masterDataServiceUpdateAirline,
} from '@/services/roncin/masterDataService';
import type { PersistedMasterDataItem } from './masterDataMapper';

export interface AirlineItem extends PersistedMasterDataItem {
  awbPrefix: string;
  icaoCode?: string;
  countryCode: string;
  countryName?: string;
  isCargoOnly?: boolean;
}

const mapAirline = (item: API.Airline): AirlineItem => {
  if (!item.id || !item.iataCode || !item.awbPrefix || !item.nameZh || !item.nameEn || !item.countryCode || item.enabled === undefined || item.source === undefined || item.sortOrder === undefined) {
    throw new Error('航司响应缺少必填字段');
  }
  return { id: item.id, code: item.iataCode, icaoCode: item.icaoCode, awbPrefix: item.awbPrefix, name: item.nameZh, nameEn: item.nameEn, countryCode: item.countryCode, isCargoOnly: item.cargoOnly, enabled: item.enabled, source: item.source, sortOrder: item.sortOrder, updatedAt: item.updatedAt };
};

export default function AirlinesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<AirlineItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListAirlines({ page: 1, pageSize: 100 });
      setData((response.data ?? []).map(mapAirline));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '航司主数据加载失败'));
  }, [fetchServerData, message]);

  const saveResponse = (response: API.CreateAirlineResponse | API.UpdateAirlineResponse) => {
    if (!response.data) throw new Error('航司响应缺少数据');
    const saved = mapAirline(response.data);
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateAirline({
      iataCode: values.code.toUpperCase().trim(),
      icaoCode: values.icaoCode?.toUpperCase().trim() || undefined,
      nameZh: values.name.trim(),
      nameEn: values.nameEn.trim(),
      awbPrefix: values.awbPrefix.trim(),
      countryCode: values.countryCode.toUpperCase().trim(),
      cargoOnly: values.isCargoOnly === true,
      source: 'manual',
      sortOrder: 100,
    });
    saveResponse(response);
  };

  const updateItem = async (record: AirlineItem, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateAirline(
      { id: record.id },
      {
        id: record.id,
        icaoCode: values.icaoCode?.toUpperCase().trim() || undefined,
        nameZh: values.name.trim(),
        nameEn: values.nameEn.trim(),
        awbPrefix: values.awbPrefix.trim(),
        countryCode: values.countryCode.toUpperCase().trim(),
        cargoOnly: values.isCargoOnly === true,
        source: record.source,
        sortOrder: record.sortOrder,
        enabled,
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新航司不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: AirlineItem) => {
    await updateItem(record, record, !record.enabled);
  };

  return (
    <MasterDataTemplate<AirlineItem>
      title="航空公司管理"
      subtitle="维护空运承运人 IATA 二字码、3位运单结算代码前缀(AWB Prefix)及全货机属性"
      icon={<RocketOutlined />}
      codeLabel="IATA 二字码"
      items={data}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索二字码(如 CA) / 运单前缀(如 999) / 航司中英文名..."
      extraStats={[
        { label: '中国国内航司', value: data.filter((a) => ['CN', 'HK', 'TW'].includes(a.countryCode)).length, color: '#1677ff' },
        { label: '全货机航司', value: data.filter((a) => a.isCargoOnly).length, color: '#fa8c16' },
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
            { label: '🇦🇪 阿联酋 (AE)', value: 'AE' },
            { label: '🇸🇬 新加坡 (SG)', value: 'SG' },
            { label: '🇯🇵 日本 (JP)', value: 'JP' },
          ],
          width: 140,
        },
      ]}
      extraColumns={[
        {
          title: 'ICAO 三字码',
          dataIndex: 'icaoCode',
          key: 'icaoCode',
          width: 110,
          render: (code: string) => <Tag color="purple">{code || '-'}</Tag>,
        },
        {
          title: '运单前缀 (AWB Prefix)',
          dataIndex: 'awbPrefix',
          key: 'awbPrefix',
          width: 160,
          render: (prefix: string) => (
            <Tag
              style={{
                fontFamily: 'monospace',
                fontWeight: 600,
                color: '#d4380d',
                backgroundColor: '#fff2e8',
                borderColor: '#ffbb96',
                margin: 0,
                fontSize: 12,
              }}
            >
              {prefix ? `${prefix}-` : '-'}
            </Tag>
          ),
        },
        {
          title: '类型',
          dataIndex: 'isCargoOnly',
          key: 'isCargoOnly',
          width: 100,
          render: (isCargo: boolean) =>
            isCargo ? (
              <Tag color="orange">全货机 (Cargo)</Tag>
            ) : (
              <Tag color="blue">客货混装 (Pax/Cargo)</Tag>
            ),
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
          label: 'IATA 二字码',
          placeholder: '例如：CA、MU、LH (2位字母/数字代码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入IATA二字码' },
            { pattern: /^[A-Za-z0-9]{2}$/, message: '请输入标准的2位IATA代码' },
          ],
        },
        {
          name: 'awbPrefix',
          label: '运单结算前缀 (3位)',
          placeholder: '例如：999 (国航)、781 (东航)、160 (国泰)',
          required: true,
          rules: [
            { required: true, message: '请输入运单结算前缀' },
            { pattern: /^\d{3}$/, message: '运单前缀必须为3位纯数字' },
          ],
        },
        {
          name: 'icaoCode',
          label: 'ICAO 三字码',
          placeholder: '例如：CCA、CES、DLH',
          rules: [{ pattern: /^[A-Za-z0-9]{3}$/, message: '请输入标准的3位 ICAO 航司代码' }],
        },
        {
          name: 'name',
          label: '航司中文简称',
          placeholder: '例如：中国国际航空、德国汉莎航空',
          required: true,
        },
        {
          name: 'nameEn',
          label: '航司英文全称',
          placeholder: '例如：Air China, Lufthansa Cargo',
          required: true,
        },
        {
          name: 'countryCode',
          label: '所属国家代码',
          placeholder: '例如：CN、DE (2位代码)',
          required: true,
          initialValue: 'CN',
        },
        {
          name: 'isCargoOnly',
          label: '承运类型',
          type: 'radio',
          initialValue: false,
          options: [
            { label: '客货混装', value: false },
            { label: '全货机', value: true },
          ],
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
