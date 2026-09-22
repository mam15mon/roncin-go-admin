import {
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';

const codePattern = /^[A-Z0-9_]{2,32}$/;
const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type Props = {
  editing: boolean;
  template?: boolean;
  billingUnits: API.BillingUnit[];
  taxableServices?: API.TaxableService[];
  currencies: API.Currency[];
  serviceTypes: API.MasterDataItem[];
  abnormalCases: API.MasterDataItem[];
};

export default function FeeSettingFields({
  editing,
  template,
  billingUnits,
  taxableServices = [],
  currencies,
  serviceTypes,
  abnormalCases,
}: Props) {
  return (
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
      <ProFormSearchableSelect
        colProps={{ span: 12 }}
        name="chargeCategoryId"
        label="费用大类"
        options={serviceTypes.map((item) => ({
          label: `${item.name} (${item.code})`,
          value: item.id,
          code: item.code,
          name: item.name,
        }))}
        placeholder="请选择费用大类"
        rules={[{ required: true, message: '请选择费用大类' }]}
      />
      <ProFormSearchableSelect
        colProps={{ span: 12 }}
        name="defaultCurrency"
        label="默认币种"
        rules={[{ required: true, message: '请选择默认币种' }]}
        options={currencies.map((item) => ({
          label: `${item.code} - ${item.name}`,
          value: item.code,
          code: item.code,
          name: item.name,
        }))}
      />
      <ProFormSearchableSelect
        colProps={{ span: 12 }}
        name="billingUnitId"
        label="默认计费单位"
        rules={[{ required: true, message: '请选择计费单位' }]}
        options={billingUnits.map((item) => ({
          label: `${item.name} (${item.code})`,
          value: item.id,
          code: item.code,
          name: item.name,
        }))}
      />
      <ProFormSearchableSelect
        colProps={{ span: 12 }}
        name="abnormalCaseId"
        label="对应异常情况"
        allowClear
        options={abnormalCases.map((item) => ({
          label: `${item.name} (${item.code})`,
          value: item.id,
          code: item.code,
          name: item.name,
        }))}
        placeholder="不选择表示不限异常情况"
      />
      <ProFormText
        colProps={{ span: 12 }}
        name="taxRate"
        label="税率（%）"
        rules={[
          { required: true, message: '请输入税率' },
          {
            pattern: taxRatePattern,
            message: '请输入 0–100，最多两位小数',
          },
        ]}
        fieldProps={{ inputMode: 'decimal' }}
      />
      {template ? (
        <>
          <ProFormText
            colProps={{ span: 12 }}
            name="taxableServiceName"
            label="默认开票项目名称"
            rules={[{ required: true, message: '请输入开票项目名称' }]}
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="taxableServiceShortName"
            label="默认开票项目简称"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="taxableServiceGoodsCode"
            label="默认商品编码"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="taxableServiceDefaultTaxRate"
            label="默认开票税率（%）"
            rules={[
              { required: true, message: '请输入默认开票税率' },
              {
                pattern: taxRatePattern,
                message: '请输入 0–100，最多两位小数',
              },
            ]}
          />
        </>
      ) : (
        <ProFormSearchableSelect
          colProps={{ span: 12 }}
          name="taxableServiceId"
          label="货物或应税劳务名称"
          rules={[{ required: true, message: '请选择货物或应税劳务名称' }]}
          options={taxableServices.map((item) => ({
            label: item.name,
            value: item.id,
            name: item.name,
            code: item.goodsCode,
          }))}
        />
      )}
      <ProFormDigit
        colProps={{ span: 12 }}
        name="sortOrder"
        label="排序权重"
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
  );
}
