import type { ProColumns } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import React from 'react';
import { useAccess } from '@/app/access';
import { SettingTableTemplate } from '@/components/ui';
import { MasterDataKind } from '@/enums.generated';
import { currencyQueryOptions } from '@/features/master-data/currencies';
import {
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceListFeeSettings,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateFeeSetting,
} from '@/services/roncin/feeCatalogService';
import { masterDataServiceListOptions } from '@/services/roncin/masterDataService';
import { ensureListResponse, toTableRequest, unwrapList } from '@/utils/api';
import FeeSettingFields from './FeeSettingFields';

type FeeSettingFormValues = {
  feeCode: string;
  nameZh: string;
  nameEn?: string;
  aliasName?: string;
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
  const currencyQuery = useQuery(currencyQueryOptions());
  const referenceQuery = useQuery({
    queryKey: ['finance', 'fee-settings-reference-options', {}],
    queryFn: async ({ signal }) => {
      const [units, taxable, items] = await Promise.all([
        feeCatalogServiceListBillingUnits({ signal, skipErrorHandler: true }),
        feeCatalogServiceListTaxableServices({
          signal,
          skipErrorHandler: true,
        }),
        masterDataServiceListOptions({ signal, skipErrorHandler: true }),
      ]);
      return {
        billingUnits: unwrapList(
          ensureListResponse(units, '计费单位加载失败'),
        ).filter((item) => item.enabled),
        taxableServices: unwrapList(
          ensureListResponse(taxable, '应税服务加载失败'),
        ).filter((item) => item.enabled),
        abnormalCases: unwrapList(
          ensureListResponse(items, '异常类型加载失败'),
        ).filter(
          (item) =>
            item.kind === MasterDataKind.MASTER_DATA_KIND_ABNORMAL_CASE &&
            item.enabled !== false,
        ),
      };
    },
    meta: { errorMessage: '费用设置候选加载失败' },
  });
  const billingUnits = referenceQuery.data?.billingUnits ?? [];
  const taxableServices = referenceQuery.data?.taxableServices ?? [];
  const abnormalCases = referenceQuery.data?.abnormalCases ?? [];
  const currencies = currencyQuery.data?.filter((item) => item.enabled) ?? [];

  // 表头汇总：序号、费用名称、费用名称(英文)、费用代码、归属、币种、计费单位、税率、货物或应税劳务名称、对应异常情况、操作
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
      modalWidth={840}
      grid
      labelWidth={145}
      canCreate={access.canCreateFeeSettings}
      canUpdate={access.canUpdateFeeSettings}
      pagination={{ pageSize: 20 }}
      query={async (params) =>
        toTableRequest(
          ensureListResponse(
            await feeCatalogServiceListFeeSettings({
              page: Number(params?.current ?? 1),
              pageSize: Number(params?.pageSize ?? 20),
            }),
            '费用设置列表加载失败',
          ),
        )
      }
      createItem={(values) =>
        feeCatalogServiceCreateFeeSetting({
          feeCode: values.feeCode.trim().toUpperCase(),
          nameZh: values.nameZh.trim(),
          nameEn: values.nameEn?.trim() || undefined,
          aliasName: values.aliasName?.trim() || undefined,
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
          abnormalCases={abnormalCases}
        />
      )}
    />
  );
}

export default FeeItemsPanel;
