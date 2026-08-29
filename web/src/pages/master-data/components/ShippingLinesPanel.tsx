import { GlobalOutlined, LinkOutlined } from '@ant-design/icons';
import { Button, Tag, Tooltip } from 'antd';
import React from 'react';
import {
  MasterDataTemplate,
  useMasterDataCrud,
} from '@/components/ui/master-data-template';
import {
  masterDataServiceCreateShippingLine,
  masterDataServiceListShippingLines,
  masterDataServiceUpdateShippingLine,
} from '@/services/roncin/masterDataService';
import type { PersistedMasterDataItem } from './masterDataMapper';

export interface ShippingLineItem extends PersistedMasterDataItem {
  containerPrefixes: string[];
  containerPrefixesText: string;
  trackingUrl?: string;
  countryCode: string;
  countryName?: string;
  alliance?: string;
}

const mapShippingLine = (item: API.ShippingLine): ShippingLineItem => {
  if (
    !item.id ||
    !item.scacCode ||
    !item.nameZh ||
    !item.nameEn ||
    !item.countryCode ||
    item.enabled === undefined ||
    item.source === undefined ||
    item.sortOrder === undefined
  ) {
    throw new Error('船司响应缺少必填字段');
  }
  const containerPrefixes = item.containerPrefixes ?? [];
  return {
    id: item.id,
    code: item.scacCode,
    name: item.nameZh,
    nameEn: item.nameEn,
    trackingUrl: item.trackingUrl,
    countryCode: item.countryCode,
    alliance: item.alliance,
    containerPrefixes,
    containerPrefixesText: containerPrefixes.join(', '),
    enabled: item.enabled,
    source: item.source,
    sortOrder: item.sortOrder,
    updatedAt: item.updatedAt,
  };
};

const parseContainerPrefixes = (value?: string) =>
  value
    ?.split(/[,，\s]+/)
    .map((item) => item.trim().toUpperCase())
    .filter(Boolean) ?? [];

export default function ShippingLinesPanel() {
  const fetchShippingLines = React.useCallback(
    (query: import('@/components/ui/master-data-template').MasterDataListQuery) =>
      masterDataServiceListShippingLines(query),
    [],
  );
  const {
    data,
    loading,
    total,
    activeTotal,
    disabledTotal,
    query,
    setQuery,
    reload,
    handleCreate,
    handleUpdate,
    handleToggleActive,
  } = useMasterDataCrud<ShippingLineItem, API.ShippingLine>({
    entityName: '船司',
    fetchList: fetchShippingLines,
    mapItem: mapShippingLine,
    createItem: (values) =>
      masterDataServiceCreateShippingLine({
        scacCode: values.code.toUpperCase().trim(),
        nameZh: values.name.trim(),
        nameEn: values.nameEn.trim(),
        countryCode: values.countryCode.toUpperCase().trim(),
        trackingUrl: values.trackingUrl?.trim() || undefined,
        alliance: values.alliance || undefined,
        containerPrefixes: parseContainerPrefixes(
          values.containerPrefixesText,
        ),
        source: 'manual',
        sortOrder: 100,
      }),
    updateItem: (id, values, enabled, record) =>
      masterDataServiceUpdateShippingLine(
        { id },
        {
          id,
          nameZh: values.name.trim(),
          nameEn: values.nameEn.trim(),
          countryCode: values.countryCode.toUpperCase().trim(),
          trackingUrl: values.trackingUrl?.trim() || undefined,
          alliance: values.alliance || undefined,
          containerPrefixes: parseContainerPrefixes(
            values.containerPrefixesText,
          ),
          source: record.source,
          sortOrder: record.sortOrder,
          enabled,
        },
      ),
  });

  return (
    <MasterDataTemplate<ShippingLineItem>
      title="船公司管理"
      subtitle="维护全球主要班轮船公司 SCAC 代码、联盟分类与官方货柜轨迹追踪入口"
      icon={<GlobalOutlined />}
      codeLabel="SCAC 标准代码"
      items={data}
      loading={loading}
      total={total}
      activeTotal={activeTotal}
      disabledTotal={disabledTotal}
      query={query}
      onQueryChange={setQuery}
      onRefresh={reload}
      searchPlaceholder="搜索 SCAC（如 MAEU）/ BIC 箱主前缀（如 MSKU）/ 船司中英文名称..."
      extraStats={[
        {
          label: '三大班轮联盟',
          value: data.filter(
            (s) => s.alliance && s.alliance !== 'Independent',
          ).length,
          color: '#1677ff',
        },
        {
          label: '独立/近洋船司',
          value: data.filter(
            (s) =>
              !s.alliance ||
              s.alliance === 'Independent' ||
              s.alliance === 'Intra-Asia',
          ).length,
          color: '#52c41a',
        },
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
          title: 'BIC 箱主前缀',
          dataIndex: 'containerPrefixes',
          key: 'containerPrefixes',
          width: 180,
          render: (prefixes: string[]) => (
            <Tag
              style={{
                fontFamily: 'monospace',
                fontWeight: 600,
                color: '#0958d9',
                backgroundColor: '#e6f4ff',
                margin: 0,
              }}
            >
              {prefixes?.join(', ') || '-'}
            </Tag>
          ),
        },
        {
          title: '航运联盟',
          dataIndex: 'alliance',
          key: 'alliance',
          width: 140,
          render: (alliance: string) => {
            if (alliance === 'Ocean Alliance')
              return <Tag color="blue">Ocean Alliance</Tag>;
            if (alliance === 'Gemini') return <Tag color="purple">Gemini</Tag>;
            if (alliance === 'Premier Alliance')
              return <Tag color="cyan">Premier Alliance</Tag>;
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
          label: 'SCAC 标准代码',
          placeholder: '例如：MAEU、COSU',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入 SCAC 标准代码' },
            { pattern: /^[A-Za-z]{2,4}$/, message: 'SCAC 应为2至4位字母' },
          ],
        },
        {
          name: 'containerPrefixesText',
          label: 'BIC 箱主前缀',
          placeholder: '例如：MSKU, MRSU（多个使用逗号分隔）',
          rules: [
            {
              pattern:
                /^\s*$|^[A-Za-z]{3}[UJZujz](\s*[,，]\s*[A-Za-z]{3}[UJZujz])*$/,
              message: '每个箱主前缀应为3位字母加 U/J/Z，并用逗号分隔',
            },
          ],
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
            {
              label: 'Ocean Alliance (海洋联盟 - 中远/达飞/长荣)',
              value: 'Ocean Alliance',
            },
            {
              label: 'Gemini (双子星联盟 - 马士基/赫伯罗特)',
              value: 'Gemini',
            },
            {
              label: 'Premier Alliance (卓越联盟 - ONE/HMM/阳明)',
              value: 'Premier Alliance',
            },
            {
              label: 'Independent (独立运营/近洋船司)',
              value: 'Independent',
            },
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
