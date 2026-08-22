import { GlobalOutlined, LinkOutlined } from '@ant-design/icons';
import { App, Button, Tag, Tooltip } from 'antd';
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

export interface ShippingLineItem extends PersistedMasterDataItem {
  scacCode: string;
  trackingUrl?: string;
  countryCode: string;
  countryName?: string;
  alliance?: string;
}

const mapShippingLine = (item: API.MasterDataItem): ShippingLineItem => ({
  ...mapPersistedMasterDataItem(item),
  scacCode: item.attributes?.scacCode ?? '',
  trackingUrl: item.attributes?.trackingUrl,
  countryCode: item.attributes?.countryCode ?? '',
  alliance: item.attributes?.alliance,
});

export default function ShippingLinesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<ShippingLineItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchServerData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListItems({
        kind: 6,
        transportMode: 'SEA',
        page: 1,
        pageSize: 100,
      });
      setData((response.data ?? []).map(mapShippingLine));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServerData().catch((error: Error) => message.error(error.message || '船司主数据加载失败'));
  }, [fetchServerData, message]);

  const saveResponse = (response: API.MasterDataItemReply) => {
    const saved = mapShippingLine(requireMasterDataResponse(response));
    setData((current) => {
      const exists = current.some((item) => item.id === saved.id);
      return exists
        ? current.map((item) => (item.id === saved.id ? saved : item))
        : [saved, ...current];
    });
  };

  const handleCreate = async (values: any) => {
    const response = await masterDataServiceCreateItem({
      kind: 6,
      code: values.code.toUpperCase().trim(),
      name: values.name.trim(),
      nameEn: values.nameEn.trim(),
      transportMode: 'SEA',
      source: 'manual',
      sortOrder: 100,
      attributes: {
        scacCode: values.scacCode.toUpperCase().trim(),
        trackingUrl: values.trackingUrl?.trim(),
        countryCode: values.countryCode.toUpperCase().trim(),
        alliance: values.alliance,
      },
    });
    saveResponse(response);
  };

  const updateItem = async (record: ShippingLineItem, values: any, enabled: boolean) => {
    const response = await masterDataServiceUpdateItem(
      { id: record.id },
      {
        id: record.id,
        kind: 6,
        name: values.name.trim(),
        nameEn: values.nameEn.trim(),
        transportMode: 'SEA',
        source: record.source,
        sortOrder: record.sortOrder,
        enabled,
        attributes: {
          scacCode: values.scacCode.toUpperCase().trim(),
          trackingUrl: values.trackingUrl?.trim(),
          countryCode: values.countryCode.toUpperCase().trim(),
          alliance: values.alliance,
        },
      },
    );
    saveResponse(response);
  };

  const handleUpdate = async (id: string, values: any) => {
    const record = data.find((item) => item.id === id);
    if (!record) throw new Error('待更新船司不存在');
    await updateItem(record, values, record.enabled);
  };

  const handleToggleActive = async (record: ShippingLineItem) => {
    await updateItem(record, record, !record.enabled);
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
            return <Tag color="default">{alliance || '-'}</Tag>;
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
          required: true,
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
