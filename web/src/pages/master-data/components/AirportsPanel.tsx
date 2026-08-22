import { SendOutlined } from '@ant-design/icons';
import { Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface AirportItem extends BaseMasterDataItem {
  icaoCode: string;
  cityName: string;
  countryCode: string;
  countryName?: string;
}

const INITIAL_AIRPORTS: AirportItem[] = [
  { id: 'PVG', code: 'PVG', icaoCode: 'ZSPD', name: '上海浦东国际机场', nameEn: 'Shanghai Pudong International Airport', cityName: '上海', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SHA', code: 'SHA', icaoCode: 'ZSSS', name: '上海虹桥国际机场', nameEn: 'Shanghai Hongqiao International Airport', cityName: '上海', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'PEK', code: 'PEK', icaoCode: 'ZBAA', name: '北京首都国际机场', nameEn: 'Beijing Capital International Airport', cityName: '北京', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'PKX', code: 'PKX', icaoCode: 'ZBAD', name: '北京大兴国际机场', nameEn: 'Beijing Daxing International Airport', cityName: '北京', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CAN', code: 'CAN', icaoCode: 'ZGGG', name: '广州白云国际机场', nameEn: 'Guangzhou Baiyun International Airport', cityName: '广州', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SZX', code: 'SZX', icaoCode: 'ZGSZ', name: '深圳宝安国际机场', nameEn: 'Shenzhen Baoan International Airport', cityName: '深圳', countryCode: 'CN', countryName: '中国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'HKG', code: 'HKG', icaoCode: 'VHHH', name: '香港国际机场', nameEn: 'Hong Kong International Airport', cityName: '香港', countryCode: 'HK', countryName: '中国香港', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'TPE', code: 'TPE', icaoCode: 'RCTP', name: '台湾桃园国际机场', nameEn: 'Taiwan Taoyuan International Airport', cityName: '台北', countryCode: 'TW', countryName: '中国台湾', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'LAX', code: 'LAX', icaoCode: 'KLAX', name: '洛杉矶国际机场', nameEn: 'Los Angeles International Airport', cityName: '洛杉矶', countryCode: 'US', countryName: '美国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'JFK', code: 'JFK', icaoCode: 'KJFK', name: '纽约肯尼迪国际机场', nameEn: 'John F. Kennedy International Airport', cityName: '纽约', countryCode: 'US', countryName: '美国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'ORD', code: 'ORD', icaoCode: 'KORD', name: '芝加哥奥黑尔国际机场', nameEn: 'O\'Hare International Airport', cityName: '芝加哥', countryCode: 'US', countryName: '美国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'FRA', code: 'FRA', icaoCode: 'EDDF', name: '法兰克福国际机场', nameEn: 'Frankfurt Airport', cityName: '法兰克福', countryCode: 'DE', countryName: '德国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'LHR', code: 'LHR', icaoCode: 'EGLL', name: '伦敦希思罗机场', nameEn: 'London Heathrow Airport', cityName: '伦敦', countryCode: 'GB', countryName: '英国', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'NRT', code: 'NRT', icaoCode: 'RJAA', name: '东京成田国际机场', nameEn: 'Narita International Airport', cityName: '东京', countryCode: 'JP', countryName: '日本', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SIN', code: 'SIN', icaoCode: 'WSSS', name: '新加坡樟宜国际机场', nameEn: 'Singapore Changi Airport', cityName: '新加坡', countryCode: 'SG', countryName: '新加坡', enabled: true, updatedAt: '2026-08-15 12:00:00' },
];

export default function AirportsPanel() {
  const [data, setData] = useState<AirportItem[]>(INITIAL_AIRPORTS);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 5, // Airport
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: AirportItem[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          icaoCode: item.parentCode || '',
          name: item.name || '',
          nameEn: item.nameEn,
          cityName: item.transportMode || '',
          countryCode: 'CN',
          enabled: item.enabled ?? true,
          updatedAt: item.updatedAt,
        }));
        setData(mapped);
      }
    } catch {
      // Keep local preset data
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServerData();
  }, []);

  const handleCreate = async (values: any) => {
    const newAirport: AirportItem = {
      id: values.code?.toUpperCase().trim(),
      code: values.code?.toUpperCase().trim(),
      icaoCode: values.icaoCode?.toUpperCase().trim() || '',
      name: values.name?.trim(),
      nameEn: values.nameEn?.trim(),
      cityName: values.cityName?.trim(),
      countryCode: values.countryCode?.toUpperCase().trim() || 'CN',
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 5,
        code: newAirport.code,
        name: newAirport.name,
        nameEn: newAirport.nameEn,
        parentCode: newAirport.icaoCode,
        transportMode: newAirport.cityName,
      });
    } catch {
      // Fallback
    }

    setData([newAirport, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.trim(),
            icaoCode: values.icaoCode?.toUpperCase().trim() || d.icaoCode,
            cityName: values.cityName?.trim() || d.cityName,
            countryCode: values.countryCode?.toUpperCase().trim() || d.countryCode,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: AirportItem) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
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
          placeholder: '例如：上海浦东国际机场',
          required: true,
        },
        {
          name: 'nameEn',
          label: '英文机场名',
          placeholder: '例如：Shanghai Pudong International Airport',
          required: true,
        },
        {
          name: 'cityName',
          label: '所在城市',
          placeholder: '例如：上海、洛杉矶',
          required: true,
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
