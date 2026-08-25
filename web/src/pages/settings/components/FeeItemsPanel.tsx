import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Tag } from 'antd';
import React, { useState } from 'react';
import { SettingTableTemplate } from '@/components/ui';
import {
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceListFeeSettings,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateFeeSetting,
} from '@/services/roncin/feeCatalogService';
import {
  masterDataServiceListCurrencies,
  masterDataServiceListOptions,
} from '@/services/roncin/masterDataService';

const SERVICE_TYPE_KIND = 8;
const ABNORMAL_CASE_KIND = 10;
const codePattern = /^[A-Z0-9_]{2,32}$/;
const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type FeeSettingFormValues = {
  feeCode: string;
  nameZh: string;
  nameEn?: string;
  aliasName?: string;
  serviceTypeId?: string;
  defaultCurrency: string;
  billingUnitId: string;
  abnormalCaseId?: string;
  taxRate: string;
  taxableServiceId: string;
  enabled: boolean;
  sortOrder: number;
};

export function FeeItemsPanel() {
  const access = useAccess();
  const [billingUnits, setBillingUnits] = useState<API.BillingUnit[]>([]);
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>([]);
  const [currencies, setCurrencies] = useState<API.Currency[]>([]);
  const [serviceTypes, setServiceTypes] = useState<API.MasterDataItem[]>([]);
  const [abnormalCases, setAbnormalCases] = useState<API.MasterDataItem[]>([]);

  const columns: ProColumns<API.FeeSetting>[] = [
    { title: '费用代码', dataIndex: 'feeCode', width: 150, copyable: true },
    { title: '费用名称', dataIndex: 'nameZh', width: 150 },
    { title: '英文名称', dataIndex: 'nameEn', width: 180, ellipsis: true },
    {
      title: '服务类型',
      dataIndex: 'serviceTypeName',
      width: 120,
      renderText: (value) => value || '通用',
    },
    { title: '币种', dataIndex: 'defaultCurrency', width: 80 },
    { title: '计费单位', dataIndex: 'billingUnitName', width: 100 },
    {
      title: '异常情况',
      dataIndex: 'abnormalCaseName',
      width: 130,
      renderText: (value) => value || '不限',
    },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 80,
      renderText: (value) => `${value ?? '0.00'}%`,
    },
    {
      title: '货物或应税劳务',
      dataIndex: 'taxableServiceName',
      width: 180,
      ellipsis: true,
    },
    { title: '排序', dataIndex: 'sortOrder', width: 70 },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (_, record) =>
        record.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];

  return (
    <SettingTableTemplate<API.FeeSetting, FeeSettingFormValues>
      entityName="费用设置"
      columns={columns}
      scroll={{ x: 1350 }}
      modalWidth={820}
      grid
      canCreate={access.canCreateFeeSettings}
      canUpdate={access.canUpdateFeeSettings}
      query={async () => {
        const [
          feeResponse,
          unitResponse,
          taxableResponse,
          currencyResponse,
          optionResponse,
        ] = await Promise.all([
          feeCatalogServiceListFeeSettings(),
          feeCatalogServiceListBillingUnits(),
          feeCatalogServiceListTaxableServices(),
          masterDataServiceListCurrencies(),
          masterDataServiceListOptions(),
        ]);
        setBillingUnits((unitResponse.data ?? []).filter((item) => item.enabled));
        setTaxableServices(
          (taxableResponse.data ?? []).filter((item) => item.enabled),
        );
        setCurrencies((currencyResponse.data ?? []).filter((item) => item.enabled));
        setServiceTypes(
          (optionResponse.data ?? []).filter(
            (item) => item.kind === SERVICE_TYPE_KIND,
          ),
        );
        setAbnormalCases(
          (optionResponse.data ?? []).filter(
            (item) => item.kind === ABNORMAL_CASE_KIND,
          ),
        );
        return {
          data: feeResponse.data ?? [],
          success: feeResponse.success ?? true,
        };
      }}
      createItem={(values) =>
        feeCatalogServiceCreateFeeSetting({
          feeCode: values.feeCode.trim().toUpperCase(),
          nameZh: values.nameZh.trim(),
          nameEn: values.nameEn?.trim() || undefined,
          aliasName: values.aliasName?.trim() || undefined,
          serviceTypeId: values.serviceTypeId || undefined,
          defaultCurrency: values.defaultCurrency,
          billingUnitId: values.billingUnitId,
          abnormalCaseId: values.abnormalCaseId || undefined,
          taxRate: values.taxRate,
          taxableServiceId: values.taxableServiceId,
          sortOrder: values.sortOrder,
        })
      }
      updateItem={(record, values) => {
        if (!record.id) return Promise.resolve();
        return feeCatalogServiceUpdateFeeSetting(
          { id: record.id },
          {
            id: record.id,
            feeCode: values.feeCode.trim().toUpperCase(),
            nameZh: values.nameZh.trim(),
            nameEn: values.nameEn?.trim() || undefined,
            aliasName: values.aliasName?.trim() || undefined,
            serviceTypeId: values.serviceTypeId || undefined,
            defaultCurrency: values.defaultCurrency,
            billingUnitId: values.billingUnitId,
            abnormalCaseId: values.abnormalCaseId || undefined,
            taxRate: values.taxRate,
            taxableServiceId: values.taxableServiceId,
            sortOrder: values.sortOrder,
            enabled: values.enabled,
          },
        );
      }}
      initialValues={(editing) =>
        editing
          ? {
              feeCode: editing.feeCode,
              nameZh: editing.nameZh,
              nameEn: editing.nameEn,
              aliasName: editing.aliasName,
              serviceTypeId: editing.serviceTypeId,
              defaultCurrency: editing.defaultCurrency,
              billingUnitId: editing.billingUnitId,
              abnormalCaseId: editing.abnormalCaseId,
              taxRate: editing.taxRate,
              taxableServiceId: editing.taxableServiceId,
              enabled: editing.enabled ?? true,
              sortOrder: editing.sortOrder ?? 100,
            }
          : { enabled: true, sortOrder: 100 }
      }
      renderFormItems={(editing) => (
        <>
          <ProFormText
            colProps={{ span: 12 }}
            name="feeCode"
            label="费用代码"
            disabled={Boolean(editing)}
            rules={[
              { required: true, message: '请输入费用代码' },
              {
                pattern: codePattern,
                message: '请输入 2–32 位大写字母、数字或下划线',
              },
            ]}
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="nameZh"
            label="费用名称"
            rules={[{ required: true, message: '请输入费用名称' }]}
            fieldProps={{ maxLength: 64 }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="nameEn"
            label="费用名称（英文）"
            fieldProps={{ maxLength: 128 }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="aliasName"
            label="费用别名"
            fieldProps={{ maxLength: 64 }}
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="serviceTypeId"
            label="对应服务类型"
            allowClear
            options={serviceTypes.map((item) => ({
              label: `${item.name} (${item.code})`,
              value: item.id,
            }))}
            placeholder="不选择表示通用费用"
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="defaultCurrency"
            label="默认币种"
            rules={[{ required: true, message: '请选择默认币种' }]}
            showSearch
            options={currencies.map((item) => ({
              label: `${item.code} - ${item.name}`,
              value: item.code,
            }))}
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="billingUnitId"
            label="默认计费单位"
            rules={[{ required: true, message: '请选择计费单位' }]}
            showSearch
            options={billingUnits.map((item) => ({
              label: `${item.name} (${item.code})`,
              value: item.id,
            }))}
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="abnormalCaseId"
            label="对应异常情况"
            allowClear
            options={abnormalCases.map((item) => ({
              label: `${item.name} (${item.code})`,
              value: item.id,
            }))}
            placeholder="不选择表示不限异常情况"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="taxRate"
            label="税率（%）"
            rules={[
              { required: true, message: '请输入税率' },
              { pattern: taxRatePattern, message: '请输入 0–100，最多两位小数' },
            ]}
            fieldProps={{ inputMode: 'decimal' }}
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="taxableServiceId"
            label="货物或应税劳务名称"
            rules={[{ required: true, message: '请选择货物或应税劳务名称' }]}
            showSearch
            options={taxableServices.map((item) => ({
              label: item.name,
              value: item.id,
            }))}
          />
          <ProFormDigit
            colProps={{ span: 12 }}
            name="sortOrder"
            label="排序"
            min={0}
            fieldProps={{ precision: 0 }}
          />
          {editing && (
            <ProFormSwitch
              colProps={{ span: 12 }}
              name="enabled"
              label="启用状态"
            />
          )}
        </>
      )}
    />
  );
}

export default FeeItemsPanel;
