import { CompassOutlined } from '@ant-design/icons';
import { Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface CityData extends BaseMasterDataItem {
  level: number; // 1: 省, 2: 市, 3: 区县
  parentCode?: string;
  parentName?: string;
}

const INITIAL_CITIES: CityData[] = [
  { id: '110000', code: '110000', name: '北京市', nameEn: 'Beijing', level: 1, enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '110100', code: '110100', name: '市辖区', nameEn: 'Beijing City', level: 2, parentCode: '110000', parentName: '北京市', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '110101', code: '110101', name: '东城区', nameEn: 'Dongcheng', level: 3, parentCode: '110100', parentName: '市辖区', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '110105', code: '110105', name: '朝阳区', nameEn: 'Chaoyang', level: 3, parentCode: '110100', parentName: '市辖区', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '310000', code: '310000', name: '上海市', nameEn: 'Shanghai', level: 1, enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '310100', code: '310100', name: '市辖区', nameEn: 'Shanghai City', level: 2, parentCode: '310000', parentName: '上海市', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '310115', code: '310115', name: '浦东新区', nameEn: 'Pudong New Area', level: 3, parentCode: '310100', parentName: '市辖区', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '440000', code: '440000', name: '广东省', nameEn: 'Guangdong', level: 1, enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '440100', code: '440100', name: '广州市', nameEn: 'Guangzhou', level: 2, parentCode: '440000', parentName: '广东省', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '440300', code: '440300', name: '深圳市', nameEn: 'Shenzhen', level: 2, parentCode: '440000', parentName: '广东省', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '330000', code: '330000', name: '浙江省', nameEn: 'Zhejiang', level: 1, enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '330100', code: '330100', name: '杭州市', nameEn: 'Hangzhou', level: 2, parentCode: '330000', parentName: '浙江省', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '330200', code: '330200', name: '宁波市', nameEn: 'Ningbo', level: 2, parentCode: '330000', parentName: '浙江省', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '510000', code: '510000', name: '四川省', nameEn: 'Sichuan', level: 1, enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '510100', code: '510100', name: '成都市', nameEn: 'Chengdu', level: 2, parentCode: '510000', parentName: '四川省', enabled: true, updatedAt: '2026-08-01 10:00:00' },
  { id: '510107', code: '510107', name: '武侯区', nameEn: 'Wuhou', level: 3, parentCode: '510100', parentName: '成都市', enabled: true, updatedAt: '2026-08-01 10:00:00' },
];

export default function CitiesPanel() {
  const [data, setData] = useState<CityData[]>(INITIAL_CITIES);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 3, // Region / City
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: CityData[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          name: item.name || '',
          nameEn: item.nameEn,
          level: item.parentCode ? 2 : 1,
          parentCode: item.parentCode,
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
    const newCity: CityData = {
      id: values.code?.trim(),
      code: values.code?.trim(),
      name: values.name?.trim(),
      nameEn: values.nameEn?.trim(),
      parentCode: values.parentCode || undefined,
      level: values.parentCode ? 2 : 1,
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 3,
        code: newCity.code,
        name: newCity.name,
        nameEn: newCity.nameEn,
        parentCode: newCity.parentCode,
      });
    } catch {
      // Fallback
    }

    setData([newCity, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.trim(),
            parentCode: values.parentCode || d.parentCode,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: CityData) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
  };

  const handleSync = async () => {
    await new Promise((resolve) => setTimeout(resolve, 1000));
  };

  return (
    <MasterDataTemplate<CityData>
      title="城市与行政区划管理"
      subtitle="维护中国民政部 6 位标准行政区划代码、省市区县层级与货代中英文映射"
      icon={<CompassOutlined />}
      codeLabel="行政代码"
      items={data}
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
          name: 'parentCode',
          label: '所属上级代码',
          placeholder: '例如：330000 (留空为省直辖)',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
      onSync={handleSync}
    />
  );
}
