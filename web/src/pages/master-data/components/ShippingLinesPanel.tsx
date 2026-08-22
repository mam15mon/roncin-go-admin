import { GlobalOutlined, LinkOutlined } from '@ant-design/icons';
import { Button, Tag, Tooltip } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
} from '@/services/roncin/masterDataService';

export interface ShippingLineItem extends BaseMasterDataItem {
  scacCode: string; // SCAC Standard Carrier Alpha Code
  trackingUrl?: string;
  countryCode: string;
  countryName?: string;
  alliance?: string; // 2M, Ocean Alliance, THE Alliance, Premier Alliance
}

const INITIAL_SHIPPING_LINES: ShippingLineItem[] = [
  { id: 'MSKU', code: 'MSKU', scacCode: 'MAEU', name: '马士基航运', nameEn: 'Maersk Line', trackingUrl: 'https://www.maersk.com/tracking/', countryCode: 'DK', countryName: '丹麦', alliance: 'Gemini', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'MSCU', code: 'MSCU', scacCode: 'MSCU', name: '地中海航运', nameEn: 'MSC Mediterranean Shipping Company', trackingUrl: 'https://www.msc.com/en/track-a-shipment', countryCode: 'CH', countryName: '瑞士', alliance: 'Independent', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'COSU', code: 'COSU', scacCode: 'COSU', name: '中远海运集运', nameEn: 'COSCO SHIPPING Lines', trackingUrl: 'https://lines.coscoshipping.com/ebusiness/cargoTracking', countryCode: 'CN', countryName: '中国', alliance: 'Ocean Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'CMDU', code: 'CMDU', scacCode: 'CMDU', name: '达飞轮船', nameEn: 'CMA CGM', trackingUrl: 'https://www.cma-cgm.com/ebusiness/tracking', countryCode: 'FR', countryName: '法国', alliance: 'Ocean Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'HLCU', code: 'HLCU', scacCode: 'HLCU', name: '赫伯罗特', nameEn: 'Hapag-Lloyd', trackingUrl: 'https://www.hapag-lloyd.com/en/online-business/track/track-by-booking-solution.html', countryCode: 'DE', countryName: '德国', alliance: 'Gemini', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'ONEY', code: 'ONEY', scacCode: 'ONEY', name: '海洋网联船务', nameEn: 'Ocean Network Express (ONE)', trackingUrl: 'https://ecomm.one-line.com/one-ecom/manage-shipment/cargo-tracking', countryCode: 'JP', countryName: '日本', alliance: 'Premier Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'EMCU', code: 'EMCU', scacCode: 'EGLV', name: '长荣海运', nameEn: 'Evergreen Marine Corp.', trackingUrl: 'https://www.evergreen-line.com/', countryCode: 'TW', countryName: '中国台湾', alliance: 'Ocean Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'HDMU', code: 'HDMU', scacCode: 'HDMU', name: '韩新海运', nameEn: 'HMM Company Limited', trackingUrl: 'https://www.hmm21.com/cms/company/engn/index.jsp', countryCode: 'KR', countryName: '韩国', alliance: 'Premier Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'YMLU', code: 'YMLU', scacCode: 'YMLU', name: '阳明海运', nameEn: 'Yang Ming Marine Transport', trackingUrl: 'https://www.yangming.com/', countryCode: 'TW', countryName: '中国台湾', alliance: 'Premier Alliance', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'ZIMU', code: 'ZIMU', scacCode: 'ZIMU', name: '以星综合航运', nameEn: 'ZIM Integrated Shipping Services', trackingUrl: 'https://www.zim.com/tools/track-a-shipment', countryCode: 'IL', countryName: '以色列', alliance: 'Independent', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'WHLU', code: 'WHLU', scacCode: 'WHLU', name: '万海航运', nameEn: 'Wan Hai Lines', trackingUrl: 'https://www.wanhai.com/', countryCode: 'TW', countryName: '中国台湾', alliance: 'Independent', enabled: true, updatedAt: '2026-08-15 12:00:00' },
  { id: 'SITC', code: 'SITC', scacCode: 'SITC', name: '海丰国际', nameEn: 'SITC International Holdings', trackingUrl: 'https://www.sitc.com/', countryCode: 'CN', countryName: '中国', alliance: 'Intra-Asia', enabled: true, updatedAt: '2026-08-15 12:00:00' },
];

export default function ShippingLinesPanel() {
  const [data, setData] = useState<ShippingLineItem[]>(INITIAL_SHIPPING_LINES);
  const [loading, setLoading] = useState(false);

  const fetchServerData = async () => {
    setLoading(true);
    try {
      const res = await masterDataServiceListItems({
        kind: 6, // Carrier / Shipping Line
        page: 1,
        pageSize: 100,
      });
      if (res.data && res.data.length > 0) {
        const mapped: ShippingLineItem[] = res.data.map((item) => ({
          id: item.id || item.code || '',
          code: item.code || '',
          scacCode: item.parentCode || item.code || '',
          name: item.name || '',
          nameEn: item.nameEn,
          trackingUrl: item.source || '',
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
    const newLine: ShippingLineItem = {
      id: values.code?.toUpperCase().trim(),
      code: values.code?.toUpperCase().trim(),
      scacCode: values.scacCode?.toUpperCase().trim() || values.code?.toUpperCase().trim(),
      name: values.name?.trim(),
      nameEn: values.nameEn?.trim(),
      trackingUrl: values.trackingUrl?.trim(),
      countryCode: values.countryCode?.toUpperCase().trim() || 'CN',
      alliance: values.alliance || 'Independent',
      enabled: true,
      updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    };

    try {
      await masterDataServiceCreateItem({
        kind: 6,
        code: newLine.code,
        name: newLine.name,
        nameEn: newLine.nameEn,
        parentCode: newLine.scacCode,
        source: newLine.trackingUrl,
      });
    } catch {
      // Fallback
    }

    setData([newLine, ...data]);
  };

  const handleUpdate = async (id: string, values: any) => {
    const next = data.map((d) =>
      d.id === id
        ? {
            ...d,
            name: values.name?.trim(),
            nameEn: values.nameEn?.trim(),
            scacCode: values.scacCode?.toUpperCase().trim() || d.scacCode,
            trackingUrl: values.trackingUrl?.trim() || d.trackingUrl,
            countryCode: values.countryCode?.toUpperCase().trim() || d.countryCode,
            alliance: values.alliance || d.alliance,
            updatedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
          }
        : d,
    );
    setData(next);
  };

  const handleToggleActive = (record: ShippingLineItem) => {
    const next = data.map((d) =>
      d.id === record.id ? { ...d, enabled: !d.enabled } : d,
    );
    setData(next);
  };

  return (
    <MasterDataTemplate<ShippingLineItem>
      title="船公司管理"
      subtitle="维护全球主要班轮船公司 SCAC 代码、联盟分类与官方货柜轨迹追踪入口"
      icon={<GlobalOutlined />}
      codeLabel="船司代码"
      items={data}
      loading={loading}
      onRefresh={fetchServerData}
      searchPlaceholder="搜索船司代码(如 MSKU) / SCAC / 船司中英文名称..."
      extraStats={[
        { label: '三大班轮联盟', value: data.filter((s) => s.alliance && s.alliance !== 'Independent').length, color: '#1677ff' },
        { label: '独立/近洋船司', value: data.filter((s) => !s.alliance || s.alliance === 'Independent' || s.alliance === 'Intra-Asia').length, color: '#52c41a' },
      ]}
      filterOptions={[
        {
          key: 'alliance',
          label: '联盟分类',
          placeholder: '所属联盟',
          options: [
            { label: '全部联盟', value: 'all' },
            { label: 'Ocean Alliance (海洋联盟)', value: 'Ocean Alliance' },
            { label: 'Gemini (双子星联盟)', value: 'Gemini' },
            { label: 'Premier Alliance (卓越联盟)', value: 'Premier Alliance' },
            { label: 'Independent (独立经营)', value: 'Independent' },
          ],
          width: 170,
        },
      ]}
      extraColumns={[
        {
          title: 'SCAC 标准代码',
          dataIndex: 'scacCode',
          key: 'scacCode',
          width: 130,
          render: (scac: string) => (
            <Tag style={{ fontFamily: 'monospace', fontWeight: 600, color: '#0958d9', backgroundColor: '#e6f4ff', margin: 0 }}>
              {scac || '-'}
            </Tag>
          ),
        },
        {
          title: '航运联盟',
          dataIndex: 'alliance',
          key: 'alliance',
          width: 140,
          render: (alliance: string) => {
            if (alliance === 'Ocean Alliance') return <Tag color="blue">Ocean Alliance</Tag>;
            if (alliance === 'Gemini') return <Tag color="purple">Gemini</Tag>;
            if (alliance === 'Premier Alliance') return <Tag color="cyan">Premier Alliance</Tag>;
            return <Tag color="default">{alliance || '独立船司'}</Tag>;
          },
        },
        {
          title: '轨迹追踪',
          dataIndex: 'trackingUrl',
          key: 'trackingUrl',
          width: 100,
          render: (url: string) =>
            url ? (
              <Tooltip title={`打开官网跟踪: ${url}`}>
                <Button
                  type="link"
                  size="small"
                  icon={<LinkOutlined />}
                  onClick={() => window.open(url, '_blank')}
                  style={{ padding: 0 }}
                >
                  去查箱
                </Button>
              </Tooltip>
            ) : (
              <span style={{ color: '#bfbfbf' }}>-</span>
            ),
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: '船司代码',
          placeholder: '例如：MSKU、COSU (4位代码)',
          required: true,
          disabledOnEdit: true,
          rules: [{ required: true, message: '请输入船司代码' }],
        },
        {
          name: 'scacCode',
          label: 'SCAC 代码',
          placeholder: '例如：MAEU、EGLV (美国标准承运人代码)',
          required: true,
        },
        {
          name: 'name',
          label: '船司中文简称',
          placeholder: '例如：马士基航运、中远海运集运',
          required: true,
        },
        {
          name: 'nameEn',
          label: '船司英文全称',
          placeholder: '例如：Maersk Line, COSCO SHIPPING Lines',
          required: true,
        },
        {
          name: 'alliance',
          label: '所属联盟',
          type: 'select',
          initialValue: 'Independent',
          options: [
            { label: 'Ocean Alliance (海洋联盟 - 中远/达飞/长荣)', value: 'Ocean Alliance' },
            { label: 'Gemini (双子星联盟 - 马士基/赫伯罗特)', value: 'Gemini' },
            { label: 'Premier Alliance (卓越联盟 - ONE/HMM/阳明)', value: 'Premier Alliance' },
            { label: 'Independent (独立运营/近洋船司)', value: 'Independent' },
          ],
        },
        {
          name: 'trackingUrl',
          label: '官方追踪网址',
          placeholder: '例如：https://www.maersk.com/tracking/',
        },
        {
          name: 'countryCode',
          label: '总部国家代码',
          placeholder: '例如：DK、CN、FR、CH (2位代码)',
          initialValue: 'CN',
        },
      ]}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onToggleActive={handleToggleActive}
    />
  );
}
