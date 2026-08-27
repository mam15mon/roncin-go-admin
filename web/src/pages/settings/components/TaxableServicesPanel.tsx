import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Tag } from 'antd';
import React from 'react';
import { SettingTableTemplate } from '@/components/ui';
import {
  feeCatalogServiceCreateTaxableService,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateTaxableService,
} from '@/services/roncin/feeCatalogService';

const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type TaxableServiceFormValues = {
  name: string;
  shortName?: string;
  goodsCode?: string;
  defaultTaxRate: string;
  enabled: boolean;
};

export function TaxableServicesPanel() {
  const access = useAccess();

  const columns: ProColumns<API.TaxableService>[] = [
    {
      title: '序号',
      valueType: 'index',
      width: 60,
    },
    {
      title: '货物或应税劳务名称',
      dataIndex: 'name',
    },
    {
      title: '简称',
      dataIndex: 'shortName',
      width: 160,
      renderText: (value) => value || '-',
    },
    {
      title: '商品编码',
      dataIndex: 'goodsCode',
      width: 160,
      renderText: (value) => value || '-',
    },
    {
      title: '默认税率',
      dataIndex: 'defaultTaxRate',
      width: 110,
      renderText: (value) => `${value ?? '0.00'}%`,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (_, record) =>
        record.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];

  return (
    <SettingTableTemplate<API.TaxableService, TaxableServiceFormValues>
      entityName="货物或应税劳务"
      columns={columns}
      modalWidth={580}
      labelWidth={150}
      query={feeCatalogServiceListTaxableServices}
      canCreate={access.canCreateFeeSettings}
      canUpdate={access.canUpdateFeeSettings}
      createItem={(values) =>
        feeCatalogServiceCreateTaxableService({
          name: values.name.trim(),
          shortName: values.shortName?.trim() || undefined,
          goodsCode: values.goodsCode?.trim() || undefined,
          defaultTaxRate: values.defaultTaxRate,
        })
      }
      updateItem={(record, values) => {
        if (!record.id) return Promise.resolve();
        return feeCatalogServiceUpdateTaxableService(
          { id: record.id },
          {
            id: record.id,
            name: values.name.trim(),
            shortName: values.shortName?.trim() || undefined,
            goodsCode: values.goodsCode?.trim() || undefined,
            defaultTaxRate: values.defaultTaxRate,
            enabled: values.enabled,
          },
        );
      }}
      initialValues={(editing) =>
        editing
          ? {
              name: editing.name,
              shortName: editing.shortName,
              goodsCode: editing.goodsCode,
              defaultTaxRate: editing.defaultTaxRate,
              enabled: editing.enabled ?? true,
            }
          : { defaultTaxRate: '0.00', enabled: true }
      }
      renderFormItems={(editing) => (
        <>
          <ProFormText
            name="name"
            label="货物或应税劳务名称"
            rules={[{ required: true, message: '请输入名称' }]}
            fieldProps={{ maxLength: 128 }}
          />
          <ProFormText
            name="shortName"
            label="简称"
            fieldProps={{ maxLength: 64 }}
          />
          <ProFormText
            name="goodsCode"
            label="商品编码"
            fieldProps={{ maxLength: 64 }}
          />
          <ProFormText
            name="defaultTaxRate"
            label="默认税率（%）"
            rules={[
              { required: true, message: '请输入默认税率' },
              { pattern: taxRatePattern, message: '请输入 0–100，最多两位小数' },
            ]}
            fieldProps={{ inputMode: 'decimal' }}
          />
          {editing && <ProFormSwitch name="enabled" label="启用状态" />}
        </>
      )}
    />
  );
}

export default TaxableServicesPanel;
