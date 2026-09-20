import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DollarOutlined,
} from '@ant-design/icons';
import { useAccess } from '@/app/access';
import { App, Badge, Space, Tag } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { MasterDataTemplate } from '@/components/ui/master-data-template/MasterDataTemplate';
import type { BaseMasterDataItem } from '@/components/ui/master-data-template/types';
import {
  masterDataServiceListCurrencies,
  masterDataServiceSetCurrencyEnabled,
} from '@/services/roncin/masterDataService';
import { unwrapList } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { getCurrencies } from '@/utils/options';

export interface CurrencyItem extends BaseMasterDataItem {
  symbol?: string;
  minorUnit?: number;
  isBaseCurrency?: boolean;
}

export default function CurrenciesPanel() {
  const { message } = App.useApp();
  const access = useAccess();
  const [data, setData] = useState<CurrencyItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchCurrencies = useCallback(async () => {
    setLoading(true);
    try {
      const response = await masterDataServiceListCurrencies({
        enabledOnly: false,
      });
      const items: CurrencyItem[] = unwrapList(response).map((item) => ({
        id: item.id || item.code || '',
        code: item.code || '',
        name: item.name || '',
        symbol: item.symbol || '',
        minorUnit: item.minorUnit ?? 2,
        enabled: Boolean(item.enabled),
        isBaseCurrency: Boolean(item.isBaseCurrency),
        updatedAt: item.updatedAt,
      }));
      setData(items);
    } catch (err) {
      message.error(getErrorMessage(err, '货币主数据加载失败'));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    fetchCurrencies();
  }, [fetchCurrencies]);

  const handleToggleActive = async (record: CurrencyItem) => {
    if (record.isBaseCurrency && record.enabled) {
      message.warning('本位币不可停用');
      return;
    }
    try {
      await masterDataServiceSetCurrencyEnabled(
        { code: record.code },
        { code: record.code, enabled: !record.enabled },
      );
      message.success(
        `已${record.enabled ? '停用' : '启用'}货币【${record.code}】`,
      );
      await fetchCurrencies();
      getCurrencies(true);
    } catch (err) {
      message.error(getErrorMessage(err, '操作失败'));
    }
  };

  const baseCurrencyCode = data.find((c) => c.isBaseCurrency)?.code || 'CNY';
  const enabledCount = data.filter((c) => c.enabled).length;
  const disabledCount = data.filter((c) => !c.enabled).length;

  return (
    <MasterDataTemplate<CurrencyItem>
      title="国际货币与币种管理"
      subtitle="统一维护国际标准化组织 ISO 4217 货币代码、符号及结算小数精度。分公司按需启用业务币种，本位币默认锁定启用。"
      icon={<DollarOutlined />}
      codeLabel="ISO 4217 代码"
      items={data}
      loading={loading}
      onRefresh={fetchCurrencies}
      onToggleActive={
        access.canUpdateMasterDataCurrencies ? handleToggleActive : undefined
      }
      canEditRecord={(record) => !record.isBaseCurrency}
      showUpdatedAt={false}
      nameWidth={240}
      searchPlaceholder="搜索货币代码 (如 USD, CNY) / 货币中文名称..."
      customStats={[
        {
          label: '标准货币主库',
          value: data.length,
          color: '#262626',
        },
        {
          label: access.isHeadquartersOrganization
            ? '有效主币'
            : '本组织已启用',
          value: enabledCount,
          color: '#52c41a',
          prefix: <CheckCircleOutlined style={{ fontSize: 14 }} />,
        },
        {
          label: '未启用币种',
          value: disabledCount,
          color: '#8c8c8c',
          prefix: <CloseCircleOutlined style={{ fontSize: 14 }} />,
        },
        {
          label: '组织本位币',
          value: baseCurrencyCode,
          color: '#1677ff',
        },
      ]}
      renderCode={(record, defaultDom) => (
        <Space size={6}>
          {defaultDom}
          {record.isBaseCurrency && (
            <Tag
              color="blue"
              variant="filled"
              style={{
                fontWeight: 600,
                fontSize: 11,
                margin: 0,
                padding: '0 6px',
                borderRadius: 4,
              }}
            >
              本位币 (锁定)
            </Tag>
          )}
        </Space>
      )}
      renderStatus={(record) =>
        record.isBaseCurrency ? (
          <Tag
            color="blue"
            variant="filled"
            style={{ fontWeight: 600, margin: 0, padding: '1px 8px' }}
          >
            锁定启用
          </Tag>
        ) : (
          <Badge
            status={record.enabled ? 'success' : 'default'}
            text={record.enabled ? '已启用' : '未启用'}
          />
        )
      }
      extraColumns={[
        {
          title: '货币符号',
          dataIndex: 'symbol',
          key: 'symbol',
          width: 100,
          render: (symbol: string) => (
            <span
              style={{
                fontFamily: 'monospace',
                fontSize: 14,
                fontWeight: 600,
                color: '#262626',
              }}
            >
              {symbol || '-'}
            </span>
          ),
        },
        {
          title: '小数精度',
          dataIndex: 'minorUnit',
          key: 'minorUnit',
          width: 110,
          render: (unit: number) => (
            <Tag
              variant="filled"
              style={{
                color: '#595959',
                backgroundColor: '#f5f5f5',
                fontSize: 12,
                borderRadius: 4,
                margin: 0,
              }}
            >
              {unit ?? 2} 位小数
            </Tag>
          ),
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
            {
              pattern: /^[A-Za-z]{3}$/,
              message: '请输入3位字母的ISO 4217货币代码',
            },
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
