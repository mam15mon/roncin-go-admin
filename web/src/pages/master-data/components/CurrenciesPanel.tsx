import { DollarOutlined } from '@ant-design/icons';
import { App, Tag } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import { unwrapList } from '@/utils/api';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';

export interface CurrencyItem extends BaseMasterDataItem {
  symbol?: string;
  minorUnit?: number;
}

export default function CurrenciesPanel() {
  const { message } = App.useApp();
  const [data, setData] = useState<CurrencyItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchCurrencies = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListCurrencies();
      const items: CurrencyItem[] = unwrapList(response).map((item) => ({
        id: item.id || item.code || '',
        code: item.code || '',
        name: item.name || '',
        symbol: item.symbol || '',
        minorUnit: item.minorUnit ?? 2,
        enabled: item.enabled ?? true,
        updatedAt: item.updatedAt,
      }));
      setData(items);
    } catch (err: any) {
      message.error(err.message || '货币主数据加载失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    fetchCurrencies();
  }, [fetchCurrencies]);

  return (
    <MasterDataTemplate<CurrencyItem>
      title="国际货币与币种管理"
      subtitle="统一维护国际标准化组织 ISO 4217 货币代码、中文名称、货币符号及结算小数精度"
      icon={<DollarOutlined />}
      codeLabel="ISO 4217 代码"
      items={data}
      loading={loading}
      onRefresh={fetchCurrencies}
      searchPlaceholder="搜索货币代码 (如 USD, CNY) / 货币中文名称..."
      extraStats={[
        {
          label: '主流结算货币',
          value: data.filter((c) => ['USD', 'CNY', 'EUR', 'HKD', 'JPY', 'GBP'].includes(c.code)).length,
          color: '#1677ff',
        },
        {
          label: '启用币种',
          value: data.filter((c) => c.enabled).length,
          color: '#52c41a',
        },
      ]}
      extraColumns={[
        {
          title: '货币符号',
          dataIndex: 'symbol',
          key: 'symbol',
          width: 100,
          render: (symbol: string) => (
            <Tag color="cyan" style={{ fontFamily: 'monospace', fontSize: 13, fontWeight: 600 }}>
              {symbol || '-'}
            </Tag>
          ),
        },
        {
          title: '小数精度',
          dataIndex: 'minorUnit',
          key: 'minorUnit',
          width: 110,
          render: (unit: number) => `${unit ?? 2} 位小数`,
        },
      ]}
      formFields={[
        {
          name: 'code',
          label: '货币代码 (ISO 4217)',
          placeholder: '例如：USD、CNY、EUR (3位字母代码)',
          required: true,
          disabledOnEdit: true,
          rules: [
            { required: true, message: '请输入3位ISO货币代码' },
            { pattern: /^[A-Za-z]{3}$/, message: '请输入3位字母的ISO 4217货币代码' },
          ],
        },
        {
          name: 'name',
          label: '货币名称',
          placeholder: '例如：美元、人民币、欧元',
          required: true,
        },
        {
          name: 'symbol',
          label: '货币符号',
          placeholder: '例如：$、¥、€',
          required: false,
        },
        {
          name: 'minorUnit',
          label: '小数精度位数',
          type: 'number',
          placeholder: '例如：2',
          required: true,
          initialValue: 2,
        },
      ]}
    />
  );
}
