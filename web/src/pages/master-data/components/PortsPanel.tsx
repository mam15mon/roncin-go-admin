import { CompassOutlined } from '@ant-design/icons';
import { Space, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface PortItem extends BaseMasterDataItem {
  countryCode: string;
  countryName?: string;
  modes: string[]; // PORT, RAIL, ROAD, AIRPORT
  isBorder?: boolean;
}

const INITIAL_PORTS: PortItem[] = [
  { id: 'CNSHG', code: 'CNSHG', name: '上海港', nameEn: 'SHANGHAI', countryCode: 'CN', countryName: '中国', modes: ['PORT', 'RAIL', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CNNGB', code: 'CNNGB', name: '宁波舟山港', nameEn: 'NINGBO', countryCode: 'CN', countryName: '中国', modes: ['PORT', 'RAIL', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CNYTN', code: 'CNYTN', name: '深圳盐田港', nameEn: 'YANTIAN', countryCode: 'CN', countryName: '中国', modes: ['PORT', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CNTAO', code: 'CNTAO', name: '青岛港', nameEn: 'QINGDAO', countryCode: 'CN', countryName: '中国', modes: ['PORT', 'RAIL'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CNXMN', code: 'CNXMN', name: '厦门港', nameEn: 'XIAMEN', countryCode: 'CN', countryName: '中国', modes: ['PORT'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SGSIN', code: 'SGSIN', name: '新加坡港', nameEn: 'SINGAPORE', countryCode: 'SG', countryName: '新加坡', modes: ['PORT', 'AIRPORT'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'NLRTM', code: 'NLRTM', name: '鹿特丹港', nameEn: 'ROTTERDAM', countryCode: 'NL', countryName: '荷兰', modes: ['PORT', 'RAIL', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'USLAX', code: 'USLAX', name: '洛杉矶港', nameEn: 'LOS ANGELES', countryCode: 'US', countryName: '美国', modes: ['PORT', 'RAIL', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'USLGB', code: 'USLGB', name: '长滩港', nameEn: 'LONG BEACH', countryCode: 'US', countryName: '美国', modes: ['PORT', 'RAIL'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'DEHAM', code: 'DEHAM', name: '汉堡港', nameEn: 'HAMBURG', countryCode: 'DE', countryName: '德国', modes: ['PORT', 'RAIL', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'AEDXB', code: 'AEDXB', name: '迪拜杰贝阿里港', nameEn: 'JEBEL ALI', countryCode: 'AE', countryName: '阿联酋', modes: ['PORT', 'ROAD'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'JPYOK', code: 'JPYOK', name: '横滨港', nameEn: 'YOKOHAMA', countryCode: 'JP', countryName: '日本', modes: ['PORT'], enabled: true, updatedAt: '2026-08-15 12:00:00' },
];

const MODE_LABELS: Record<string, { label: string; color: string }> = {
  PORT: { label: '海港', color: 'blue' },
  RAIL: { label: '铁路枢纽', color: 'volcano' },
  ROAD: { label: '公路集散', color: 'green' },
  AIRPORT: { label: '空运联运', color: 'purple' },
};

export default function PortsPanel() {
  const [data, setData] = useState<PortItem[]>(INITIAL_PORTS);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 4, // Port
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: PortItem[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          name: item.name || '',
          nameEn: item.nameEn,
          countryCode: item.parentCode || 'CN',
          modes: item.transportMode ? item.transportMode.split(',') : ['PORT'],
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
    const newPort: PortItem = {
      id: values.code?.toUpperCase().trim(),
      code: values.code?.toUpperCase().trim(),
      name: values.name?.trim(),
      nameEn: values.nameEn?.toUpperCase().trim(),
      countryCode: values.countryCode?.toUpperCase().trim() || 'CN',
      modes: values.modes || ['PORT'],
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 4,
        code: newPort.code,
        name: newPort.name,
        nameEn: newPort.nameEn,
        parentCode: newPort.countryCode,
        transportMode: newPort.modes.join(','),
      });
    } catch {
      // Fallback local state
    }

    setData([newPort, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.toUpperCase().trim(),
            countryCode: values.countryCode?.toUpperCase().trim() || d.countryCode,
            modes: values.modes || d.modes,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: PortItem) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
  };

  const handleSync = async () => {
    // 模拟同步 UN/LOCODE 港口代码
    await new Promise((resolve) => setTimeout(resolve, 1000));
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
      onSync={handleSync}
    />
  );
}
