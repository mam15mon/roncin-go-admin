import { GlobalOutlined } from '@ant-design/icons';
import { Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface CountryItem extends BaseMasterDataItem {
  continent?: string; // 亚洲, 欧洲, 北美洲, 南美洲, 大洋洲, 非洲
  currencyCode?: string; // CNY, USD, EUR, JPY, GBP
}

const INITIAL_COUNTRIES: CountryItem[] = [
  { id: 'CN', code: 'CN', name: '中国', nameEn: 'China', continent: '亚洲', currencyCode: 'CNY', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'US', code: 'US', name: '美国', nameEn: 'United States', continent: '北美洲', currencyCode: 'USD', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'DE', code: 'DE', name: '德国', nameEn: 'Germany', continent: '欧洲', currencyCode: 'EUR', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'JP', code: 'JP', name: '日本', nameEn: 'Japan', continent: '亚洲', currencyCode: 'JPY', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'GB', code: 'GB', name: '英国', nameEn: 'United Kingdom', continent: '欧洲', currencyCode: 'GBP', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SG', code: 'SG', name: '新加坡', nameEn: 'Singapore', continent: '亚洲', currencyCode: 'SGD', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'NL', code: 'NL', name: '荷兰', nameEn: 'Netherlands', continent: '欧洲', currencyCode: 'EUR', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'FR', code: 'FR', name: '法国', nameEn: 'France', continent: '欧洲', currencyCode: 'EUR', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'AU', code: 'AU', name: '澳大利亚', nameEn: 'Australia', continent: '大洋洲', currencyCode: 'AUD', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CA', code: 'CA', name: '加拿大', nameEn: 'Canada', continent: '北美洲', currencyCode: 'CAD', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'KR', code: 'KR', name: '韩国', nameEn: 'South Korea', continent: '亚洲', currencyCode: 'KRW', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'VN', code: 'VN', name: '越南', nameEn: 'Vietnam', continent: '亚洲', currencyCode: 'VND', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'TH', code: 'TH', name: '泰国', nameEn: 'Thailand', continent: '亚洲', currencyCode: 'THB', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'MY', code: 'MY', name: '马来西亚', nameEn: 'Malaysia', continent: '亚洲', currencyCode: 'MYR', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'AE', code: 'AE', name: '阿联酋', nameEn: 'United Arab Emirates', continent: '亚洲', currencyCode: 'AED', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'BR', code: 'BR', name: '巴西', nameEn: 'Brazil', continent: '南美洲', currencyCode: 'BRL', enabled: true, updatedAt: '2026-08-15 12:00:00' },
];

export default function CountriesPanel() {
  const [data, setData] = useState<CountryItem[]>(INITIAL_COUNTRIES);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 2, // Country
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: CountryItem[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          name: item.name || '',
          nameEn: item.nameEn,
          continent: item.parentCode || '亚洲',
          currencyCode: item.transportMode || 'USD',
          enabled: item.enabled ?? true,
          updatedAt: item.updatedAt,
        }));
        setData(mapped);
      }
    } catch {
      // Fallback
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServerData();
  }, []);

  const handleCreate = async (values: any) => {
    const newCountry: CountryItem = {
      id: values.code?.toUpperCase().trim(),
      code: values.code?.toUpperCase().trim(),
      name: values.name?.trim(),
      nameEn: values.nameEn?.trim(),
      continent: values.continent || '亚洲',
      currencyCode: values.currencyCode?.toUpperCase().trim() || 'USD',
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 2,
        code: newCountry.code,
        name: newCountry.name,
        nameEn: newCountry.nameEn,
        parentCode: newCountry.continent,
        transportMode: newCountry.currencyCode,
      });
    } catch {
      // Fallback
    }

    setData([newCountry, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.trim(),
            continent: values.continent || d.continent,
            currencyCode: values.currencyCode?.toUpperCase().trim() || d.currencyCode,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: CountryItem) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
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
          render: (continent: string) => <Tag color="blue">{continent || '亚洲'}</Tag>,
        },
        {
          title: '官方货币',
          dataIndex: 'currencyCode',
          key: 'currencyCode',
          width: 100,
          render: (curr: string) => (
            <Tag color="gold" style={{ fontFamily: 'monospace', fontWeight: 600 }}>
              {curr || 'USD'}
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
          initialValue: 'USD',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
