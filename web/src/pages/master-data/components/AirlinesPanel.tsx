import { RocketOutlined } from '@ant-design/icons';
import { Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface AirlineItem extends BaseMasterDataItem {
  awbPrefix: string; // 3位结算运单前缀，如 999, 781, 784
  countryCode: string;
  countryName?: string;
  isCargoOnly?: boolean;
}

const INITIAL_AIRLINES: AirlineItem[] = [
  { id: 'CA', code: 'CA', awbPrefix: '999', name: '中国国际航空', nameEn: 'Air China', countryCode: 'CN', countryName: '中国', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'MU', code: 'MU', awbPrefix: '781', name: '中国东方航空', nameEn: 'China Eastern Airlines', countryCode: 'CN', countryName: '中国', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CZ', code: 'CZ', awbPrefix: '784', name: '中国南方航空', nameEn: 'China Southern Airlines', countryCode: 'CN', countryName: '中国', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'HU', code: 'HU', awbPrefix: '880', name: '海南航空', nameEn: 'Hainan Airlines', countryCode: 'CN', countryName: '中国', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CX', code: 'CX', awbPrefix: '160', name: '国泰航空', nameEn: 'Cathay Pacific', countryCode: 'HK', countryName: '中国香港', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SQ', code: 'SQ', awbPrefix: '618', name: '新加坡航空', nameEn: 'Singapore Airlines', countryCode: 'SG', countryName: '新加坡', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'LH', code: 'LH', awbPrefix: '020', name: '德国汉莎航空', nameEn: 'Lufthansa Cargo', countryCode: 'DE', countryName: '德国', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'EK', code: 'EK', awbPrefix: '176', name: '阿联酋航空', nameEn: 'Emirates SkyCargo', countryCode: 'AE', countryName: '阿联酋', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'QR', code: 'QR', awbPrefix: '157', name: '卡塔尔航空', nameEn: 'Qatar Airways Cargo', countryCode: 'QA', countryName: '卡塔尔', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'NH', code: 'NH', awbPrefix: '205', name: '全日空航空', nameEn: 'All Nippon Airways', countryCode: 'JP', countryName: '日本', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'JL', code: 'JL', awbPrefix: '131', name: '日本航空', nameEn: 'Japan Airlines', countryCode: 'JP', countryName: '日本', isCargoOnly: false, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'FX', code: 'FX', awbPrefix: '023', name: '联邦快递航空', nameEn: 'FedEx Express', countryCode: 'US', countryName: '美国', isCargoOnly: true, enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: '5X', code: '5X', awbPrefix: '406', name: '优比速航空', nameEn: 'UPS Airlines', countryCode: 'US', countryName: '美国', isCargoOnly: true, enabled: true, updatedAt: '2026-08-15 12:00:00' },
];

export default function AirlinesPanel() {
  const [data, setData] = useState<AirlineItem[]>(INITIAL_AIRLINES);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 6, // Carrier / Airline
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: AirlineItem[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          awbPrefix: item.parentCode || '',
          name: item.name || '',
          nameEn: item.nameEn,
          countryCode: 'CN',
          enabled: item.enabled ?? true,
          updatedAt: item.updatedAt,
        }));
        setData(mapped);
      }
    } catch {
      // Keep local preset
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServerData();
  }, []);

  const handleCreate = async (values: any) => {
    const newAirline: AirlineItem = {
      id: values.code?.toUpperCase().trim(),
      code: values.code?.toUpperCase().trim(),
      awbPrefix: values.awbPrefix?.trim(),
      name: values.name?.trim(),
      nameEn: values.nameEn?.trim(),
      countryCode: values.countryCode?.toUpperCase().trim() || 'CN',
      isCargoOnly: values.isCargoOnly || false,
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 6,
        code: newAirline.code,
        name: newAirline.name,
        nameEn: newAirline.nameEn,
        parentCode: newAirline.awbPrefix,
      });
    } catch {
      // Fallback
    }

    setData([newAirline, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.trim(),
            awbPrefix: values.awbPrefix?.trim() || d.awbPrefix,
            countryCode: values.countryCode?.toUpperCase().trim() || d.countryCode,
            isCargoOnly: values.isCargoOnly !== undefined ? values.isCargoOnly : d.isCargoOnly,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: AirlineItem) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
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
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
