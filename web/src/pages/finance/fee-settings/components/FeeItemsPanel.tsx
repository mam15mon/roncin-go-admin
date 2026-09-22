import type { ProColumns } from '@ant-design/pro-components';
import React, { useState } from 'react';
import { useAccess } from '@/app/access';
import { SettingTableTemplate } from '@/components/ui';
import { MasterDataKind } from '@/enums.generated';
import { getCurrencies } from '@/features/master-data/currencies';
import {
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceListFeeSettings,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateFeeSetting,
} from '@/services/roncin/feeCatalogService';
import { masterDataServiceListOptions } from '@/services/roncin/masterDataService';
import { toTableRequest, unwrapList } from '@/utils/api';
import FeeSettingFields from './FeeSettingFields';

type FeeSettingFormValues = {
  feeCode: string;
  nameZh: string;
  nameEn?: string;
  aliasName?: string;
  chargeCategoryId?: string;
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
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>(
    [],
  );
  const [currencies, setCurrencies] = useState<API.Currency[]>([]);
  const [serviceTypes, setServiceTypes] = useState<API.MasterDataItem[]>([]);
  const [abnormalCases, setAbnormalCases] = useState<API.MasterDataItem[]>([]);

  // 表头汇总：序号、费用名称、费用名称(英文)、费用代码、归属、币种、计费单位、税率、货物或应税劳务名称、费用大类、对应异常情况、操作
  const columns: ProColumns<API.FeeSetting>[] = [
    {
      title: '序号',
      valueType: 'index',
      width: 60,
    },
    {
      title: '费用名称',
      dataIndex: 'nameZh',
      width: 150,
    },
    {
      title: '费用名称(英文)',
      dataIndex: 'nameEn',
      width: 180,
      ellipsis: true,
      renderText: (value) => value || '-',
    },
    {
      title: '费用代码',
      dataIndex: 'feeCode',
      width: 130,
      copyable: true,
    },
    {
      title: '币种',
      dataIndex: 'defaultCurrency',
      width: 80,
    },
    {
      title: '计费单位',
      dataIndex: 'billingUnitName',
      width: 100,
    },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 90,
      renderText: (value) => `${value ?? '0.00'}%`,
    },
    {
      title: '货物或应税劳务名称',
      dataIndex: 'taxableServiceName',
      width: 180,
      ellipsis: true,
    },
    {
      title: '费用大类',
      dataIndex: 'chargeCategoryName',
      width: 120,
      renderText: (value) => value || '-',
    },
    {
      title: '对应异常情况',
      dataIndex: 'abnormalCaseName',
      width: 130,
      renderText: (value) => value || '不限',
    },
  ];

  return (
    <SettingTableTemplate<API.FeeSetting, FeeSettingFormValues>
      entityName="费用设置"
      columns={columns}
      scroll={{ x: 1460 }}
      modalWidth={840}
      grid
      labelWidth={145}
      canCreate={access.canCreateFeeSettings}
      canUpdate={access.canUpdateFeeSettings}
      pagination={{ pageSize: 20 }}
      query={async (params) => {
        const [
          feeResponse,
          unitResponse,
          taxableResponse,
          currencyResponse,
          optionResponse,
        ] = await Promise.all([
          feeCatalogServiceListFeeSettings({
            page: Number(params?.current ?? 1),
            pageSize: Number(params?.pageSize ?? 20),
          }),
          feeCatalogServiceListBillingUnits({ page: 1, pageSize: 200 }),
          feeCatalogServiceListTaxableServices({ page: 1, pageSize: 200 }),
          getCurrencies(),
          masterDataServiceListOptions(),
        ]);
        setBillingUnits(
          unwrapList(unitResponse).filter((item) => item.enabled),
        );
        setTaxableServices(
          unwrapList(taxableResponse).filter((item) => item.enabled),
        );
        setCurrencies(currencyResponse.filter((item) => item.enabled));
        setServiceTypes(
          unwrapList(optionResponse).filter(
            (item) =>
              item.kind === MasterDataKind.MASTER_DATA_KIND_CHARGE_CATEGORY &&
              item.enabled !== false,
          ),
        );
        setAbnormalCases(
          unwrapList(optionResponse).filter(
            (item) =>
              item.kind === MasterDataKind.MASTER_DATA_KIND_ABNORMAL_CASE &&
              item.enabled !== false,
          ),
        );
        return toTableRequest(feeResponse);
      }}
      createItem={(values) =>
        feeCatalogServiceCreateFeeSetting({
          feeCode: values.feeCode.trim().toUpperCase(),
          nameZh: values.nameZh.trim(),
          nameEn: values.nameEn?.trim() || undefined,
          aliasName: values.aliasName?.trim() || undefined,
          chargeCategoryId: values.chargeCategoryId ?? '',
          defaultCurrency: values.defaultCurrency,
          billingUnitId: values.billingUnitId,
          abnormalCaseId: values.abnormalCaseId || undefined,
          taxRate: values.taxRate,
          taxableServiceId: values.taxableServiceId,
          sortOrder: values.sortOrder ?? 100,
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
            chargeCategoryId: values.chargeCategoryId ?? '',
            defaultCurrency: values.defaultCurrency,
            billingUnitId: values.billingUnitId,
            abnormalCaseId: values.abnormalCaseId || undefined,
            taxRate: values.taxRate,
            taxableServiceId: values.taxableServiceId,
            sortOrder: values.sortOrder ?? 100,
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
              chargeCategoryId: editing.chargeCategoryId,
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
        <FeeSettingFields
          editing={Boolean(editing)}
          billingUnits={billingUnits}
          taxableServices={taxableServices}
          currencies={currencies}
          serviceTypes={serviceTypes}
          abnormalCases={abnormalCases}
        />
      )}
    />
  );
}

export default FeeItemsPanel;
