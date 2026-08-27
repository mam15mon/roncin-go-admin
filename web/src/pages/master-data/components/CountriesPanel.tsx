import { GlobalOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
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

export interface CountryItem extends PersistedMasterDataItem {
  continent?: string;
  currencyCode?: string;
}

const mapCountry = (item: API.MasterDataItem): CountryItem => ({
  ...mapPersistedMasterDataItem(item),
  continent: item.attributes?.continent,
  currencyCode: item.attributes?.currencyCode,
});

export default function CountriesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<CountryItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListItems({ kind: 2, page: 1, pageSize: 200 });
      setData((response.data ?? []).map(mapCountry));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '国家主数据加载失败'));
  }, [fetchServerData, message]);

  const saveResponse = (response: API.CreateItemResponse | API.UpdateItemResponse) => {
    const saved = mapCountry(requireMasterDataResponse(response));
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateItem({
      kind: 2,
      code: values.code.toUpperCase().trim(),
      name: values.name.trim(),
      nameEn: values.nameEn.trim(),
      source: 'manual',
      sortOrder: 100,
      attributes: {
        continent: values.continent,
        currencyCode: values.currencyCode.toUpperCase().trim(),
      },
    });
    saveResponse(response);
  };

  const updateItem = async (record: CountryItem, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateItem(
      { id: record.id },
      {
        id: record.id,
        kind: 2,
        name: values.name.trim(),
        nameEn: values.nameEn.trim(),
        source: record.source,
        sortOrder: record.sortOrder,
        enabled,
        attributes: {
          continent: values.continent,
          currencyCode: values.currencyCode.toUpperCase().trim(),
        },
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新国家不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: CountryItem) => {
    await updateItem(record, record, !record.enabled);
  };

  return (
    <MasterDataTemplate<CountryItem>
      title="国家与地区管理"
      subtitle="维护全球国家/地区 ISO 二字码、所属大洲、官方货币及单证对应属性"
      icon={<GlobalOutlined />}
      codeLabel="ISO 二字码"
      items={data}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索国家代码(如 CN) / 国家中英文名称..."
      extraStats={[
        { label: '亚洲国家', value: data.filter((c) => c.continent === '亚洲').length, color: '#1677ff' },
        { label: '欧美国家', value: data.filter((c) => ['欧洲', '北美洲'].includes(c.continent || '')).length, color: '#722ed1' },
      ]}
      filterOptions={[
        {
          key: 'continent',
          label: '大洲筛选',
          placeholder: '所属大洲',
          options: [
            { label: '全部大洲', value: 'all' },
            { label: '亚洲', value: '亚洲' },
            { label: '欧洲', value: '欧洲' },
            { label: '北美洲', value: '北美洲' },
            { label: '南美洲', value: '南美洲' },
            { label: '大洋洲', value: '大洋洲' },
            { label: '非洲', value: '非洲' },
          ],
          width: 130,
        },
      ]}
      extraColumns={[
        {
          title: '所属大洲',
          dataIndex: 'continent',
          key: 'continent',
          width: 110,
          render: (continent: string) => <Tag color="blue">{continent || '-'}</Tag>,
        },
        {
          title: '官方货币',
          dataIndex: 'currencyCode',
          key: 'currencyCode',
          width: 100,
          render: (curr: string) => (
            <Tag color="gold" style={{ fontFamily: 'monospace', fontWeight: 600 }}>
              {curr || '-'}
            </Tag>
          ),
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: 'ISO 二字码',
          placeholder: '例如：CN、US、DE (2位字母代码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入ISO二字码' },
            { pattern: /^[A-Za-z]{2}$/, message: '请输入标准的2位字母ISO国家代码' },
          ],
        },
        {
          name: 'name',
          label: '中文国名',
          placeholder: '例如：中国、美国、德国',
          required: true,
        },
        {
          name: 'nameEn',
          label: '英文国名',
          placeholder: '例如：China, United States, Germany',
          required: true,
        },
        {
          name: 'continent',
          label: '所属大洲',
          type: 'select',
          required: true,
          initialValue: '亚洲',
          options: [
            { label: '亚洲', value: '亚洲' },
            { label: '欧洲', value: '欧洲' },
            { label: '北美洲', value: '北美洲' },
            { label: '南美洲', value: '南美洲' },
            { label: '大洋洲', value: '大洋洲' },
            { label: '非洲', value: '非洲' },
          ],
        },
        {
          name: 'currencyCode',
          label: '主要结算货币',
          placeholder: '例如：CNY, USD, EUR (3位货币代码)',
          required: true,
          initialValue: 'USD',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
